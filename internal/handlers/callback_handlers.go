package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Структура для callback данных
type CallbackData struct {
	Action   string
	LessonID int
	Extra    string
}

// Парсинг callback данных
func parseCallbackData(data string) (*CallbackData, error) {
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return nil, fmt.Errorf("неверный формат callback данных: %s", data)
	}

	result := &CallbackData{
		Action: parts[0],
	}

	// Для действий с lesson_id
	if len(parts) >= 3 && parts[1] == "lesson" {
		lessonID, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("неверный lesson_id: %s", parts[2])
		}
		result.LessonID = lessonID
		
		// Дополнительные параметры
		if len(parts) > 3 {
			result.Extra = parts[3]
		}
	}

	return result, nil
}

// Новый роутер для callback запросов (заменяет существующий)
func handleNewCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Убрать индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("Ошибка callback ответа: %v", err)
	}

	// Парсинг callback данных
	callbackData, err := parseCallbackData(query.Data)
	if err != nil {
		log.Printf("Ошибка парсинга callback: %v", err)
		sendMessage(bot, query.Message.Chat.ID, "❌ Неверный формат команды")
		return
	}

	// Получение роли пользователя
	userRole, err := getUserRole(db, query.From.ID)
	if err != nil {
		sendMessage(bot, query.Message.Chat.ID, "❌ Ошибка определения роли пользователя")
		return
	}

	// Маршрутизация в зависимости от действия
	switch callbackData.Action {
	case "enroll":
		handleEnrollCallback(bot, query, db, callbackData, userRole)
	case "unenroll":
		handleUnenrollCallback(bot, query, db, callbackData, userRole)
	case "waitlist":
		handleWaitlistCallback(bot, query, db, callbackData, userRole)
	case "cancel":
		handleNewCancelLessonCallback(bot, query, db, callbackData, userRole)
	case "confirm":
		handleConfirmLessonCallback(bot, query, db, callbackData, userRole)
	case "schedule":
		handleScheduleCallback(bot, query, db, callbackData, userRole)
	case "info":
		handleLessonInfoCallback(bot, query, db, callbackData, userRole)
	default:
		sendMessage(bot, query.Message.Chat.ID, "❓ Неизвестное действие")
	}
}

// Запись на урок через callback
func handleEnrollCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	// Только студенты могут записываться
	if userRole != "student" {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Только студенты могут записываться на уроки")
		bot.Request(callbackResponse)
		return
	}

	// ========================= RATE LIMITING =========================
	userID := query.From.ID
	if GlobalRateLimiter != nil {
		allowed, reason := GlobalRateLimiter.IsOperationAllowed(userID, OPERATION_ENROLL, data.LessonID)
		if !allowed {
			callbackResponse := tgbotapi.NewCallback(query.ID, reason)
			bot.Request(callbackResponse)
			return
		}
		
		// Регистрируем начало операции
		if err := GlobalRateLimiter.StartOperation(userID, OPERATION_ENROLL, data.LessonID); err != nil {
			callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Системная ошибка. Попробуйте позже.")
			bot.Request(callbackResponse)
			return
		}
		
		// Завершаем операцию в конце функции
		defer func() {
			if err := GlobalRateLimiter.FinishOperation(userID, OPERATION_ENROLL, data.LessonID); err != nil {
				log.Printf("Ошибка завершения операции rate limiting: %v", err)
			}
		}()
	}

	// Проверка валидности урока
	if !isLessonValid(db, data.LessonID) {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Урок больше недоступен")
		bot.Request(callbackResponse)
		updateMessageWithExpiredLesson(bot, query.Message)
		return
	}

	// Получение student_id
	studentID, err := getStudentID(db, int(query.From.ID))
	if err != nil {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка определения студента")
		bot.Request(callbackResponse)
		return
	}

	// Проверка, не записан ли уже
	if isStudentEnrolled(db, studentID, data.LessonID) {
		callbackResponse := tgbotapi.NewCallback(query.ID, "ℹ️ Вы уже записаны на этот урок")
		bot.Request(callbackResponse)
		return
	}

	// Проверка наличия мест
	if !hasAvailableSpots(db, data.LessonID) {
		// Предложить лист ожидания
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Мест нет. Добавить в лист ожидания?")
		bot.Request(callbackResponse)
		
		// Создаем кнопку для листа ожидания
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏳ В лист ожидания", fmt.Sprintf("waitlist_lesson_%d", data.LessonID)),
			),
		)
		
		editMsg := tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
		bot.Send(editMsg)
		return
	}

	// Запись на урок
	err = enrollStudentInDB(db, studentID, data.LessonID)
	if err != nil {
		log.Printf("Ошибка записи на урок: %v", err)
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка записи на урок")
		bot.Request(callbackResponse)
		return
	}

	callbackResponse := tgbotapi.NewCallback(query.ID, "✅ Вы успешно записались на урок!")
	bot.Request(callbackResponse)

	// Обновляем сообщение с актуальной информацией
	updateLessonMessage(bot, query.Message, db, data.LessonID)
}

// Отмена записи на урок
func handleUnenrollCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	if userRole != "student" {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Только студенты могут отменять запись")
		bot.Request(callbackResponse)
		return
	}

	// ========================= RATE LIMITING =========================
	userID := query.From.ID
	if GlobalRateLimiter != nil {
		allowed, reason := GlobalRateLimiter.IsOperationAllowed(userID, OPERATION_CANCEL, data.LessonID)
		if !allowed {
			callbackResponse := tgbotapi.NewCallback(query.ID, reason)
			bot.Request(callbackResponse)
			return
		}
		
		// Регистрируем начало операции
		if err := GlobalRateLimiter.StartOperation(userID, OPERATION_CANCEL, data.LessonID); err != nil {
			callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Системная ошибка. Попробуйте позже.")
			bot.Request(callbackResponse)
			return
		}
		
		// Завершаем операцию в конце функции
		defer func() {
			if err := GlobalRateLimiter.FinishOperation(userID, OPERATION_CANCEL, data.LessonID); err != nil {
				log.Printf("Ошибка завершения операции rate limiting: %v", err)
			}
		}()
	}

	studentID, err := getStudentID(db, int(query.From.ID))
	if err != nil {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка определения студента")
		bot.Request(callbackResponse)
		return
	}

	err = unenrollStudentFromDB(db, studentID, data.LessonID)
	if err != nil {
		log.Printf("Ошибка отмены записи: %v", err)
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка отмены записи")
		bot.Request(callbackResponse)
		return
	}

	callbackResponse := tgbotapi.NewCallback(query.ID, "✅ Запись отменена")
	bot.Request(callbackResponse)

	// Обновляем сообщение
	updateLessonMessage(bot, query.Message, db, data.LessonID)

	// Уведомляем следующего в листе ожидания
	notifyNextInWaitlist(bot, db, data.LessonID)
}

// Добавление в лист ожидания
func handleWaitlistCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	if userRole != "student" {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Только студенты могут попадать в лист ожидания")
		bot.Request(callbackResponse)
		return
	}

	// ========================= RATE LIMITING =========================
	userID := query.From.ID
	if GlobalRateLimiter != nil {
		allowed, reason := GlobalRateLimiter.IsOperationAllowed(userID, OPERATION_WAITLIST, data.LessonID)
		if !allowed {
			callbackResponse := tgbotapi.NewCallback(query.ID, reason)
			bot.Request(callbackResponse)
			return
		}
		
		// Регистрируем начало операции
		if err := GlobalRateLimiter.StartOperation(userID, OPERATION_WAITLIST, data.LessonID); err != nil {
			callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Системная ошибка. Попробуйте позже.")
			bot.Request(callbackResponse)
			return
		}
		
		// Завершаем операцию в конце функции
		defer func() {
			if err := GlobalRateLimiter.FinishOperation(userID, OPERATION_WAITLIST, data.LessonID); err != nil {
				log.Printf("Ошибка завершения операции rate limiting: %v", err)
			}
		}()
	}

	studentID, err := getStudentID(db, int(query.From.ID))
	if err != nil {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка определения студента")
		bot.Request(callbackResponse)
		return
	}

	err = addToWaitlist(db, studentID, data.LessonID)
	if err != nil {
		log.Printf("Ошибка добавления в лист ожидания: %v", err)
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка добавления в лист ожидания")
		bot.Request(callbackResponse)
		return
	}

	callbackResponse := tgbotapi.NewCallback(query.ID, "⏳ Вы добавлены в лист ожидания")
	bot.Request(callbackResponse)
}

// Отмена урока (только для учителей) - новое имя функции
func handleNewCancelLessonCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	if userRole != "teacher" && userRole != "admin" {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Только учителя могут отменять уроки")
		bot.Request(callbackResponse)
		return
	}

	// Для учителей - проверяем, что это их урок
	if userRole == "teacher" {
		teacherID, err := getTeacherID(db, int(query.From.ID))
		if err != nil || !isTeacherLesson(db, teacherID, data.LessonID) {
			callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Вы можете отменять только свои уроки")
			bot.Request(callbackResponse)
			return
		}
	}

	err := cancelLessonInDB(db, data.LessonID)
	if err != nil {
		log.Printf("Ошибка отмены урока: %v", err)
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка отмены урока")
		bot.Request(callbackResponse)
		return
	}

	callbackResponse := tgbotapi.NewCallback(query.ID, "✅ Урок отменен")
	bot.Request(callbackResponse)

	// Уведомляем всех записанных студентов
	notifyStudentsAboutCancellation(bot, db, data.LessonID)

	// Обновляем сообщение
	updateCancelledLessonMessage(bot, query.Message)
}

// Подтверждение урока (только для учителей)
func handleConfirmLessonCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	if userRole != "teacher" && userRole != "admin" {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Только учителя могут подтверждать уроки")
		bot.Request(callbackResponse)
		return
	}

	callbackResponse := tgbotapi.NewCallback(query.ID, "✅ Урок подтвержден")
	bot.Request(callbackResponse)

	// Логика подтверждения урока
	sendMessage(bot, query.Message.Chat.ID, "✅ Урок подтвержден")
}

// Показ расписания через callback
func handleScheduleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	// Показываем обновленное расписание
	sendScheduleWithButtons(bot, query.Message.Chat.ID, db, userRole)
	
	callbackResponse := tgbotapi.NewCallback(query.ID, "🔄 Расписание обновлено")
	bot.Request(callbackResponse)
}

// Показ информации об уроке
func handleLessonInfoCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, data *CallbackData, userRole string) {
	lessonInfo, err := getLessonInfo(db, data.LessonID)
	if err != nil {
		callbackResponse := tgbotapi.NewCallback(query.ID, "❌ Ошибка загрузки информации")
		bot.Request(callbackResponse)
		return
	}

	sendMessage(bot, query.Message.Chat.ID, lessonInfo)
	
	callbackResponse := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callbackResponse)
}

// Обновление сообщения с уроком
func updateLessonMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, lessonID int) {
	lessonText, keyboard := getLessonWithButtons(db, lessonID, 0) // 0 = любая роль для просмотра
	
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, lessonText)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
}

// Обновление сообщения с отмененным уроком
func updateCancelledLessonMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := "❌ **Урок отменен**\n\nЭтот урок больше недоступен."
	
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
	bot.Send(editMsg)
}

// Обновление сообщения с истекшим уроком
func updateMessageWithExpiredLesson(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	text := "⏰ **Урок недоступен**\n\nЭтот урок больше не принимает записи."
	
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
	bot.Send(editMsg)
}
