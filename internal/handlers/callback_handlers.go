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

	// Парсинг lesson_id если есть
	if len(parts) > 1 {
		if lessonID, err := strconv.Atoi(parts[1]); err == nil {
			result.LessonID = lessonID
		} else {
			result.Extra = parts[1]
		}
		
		if len(parts) > 3 {
			result.Extra = parts[3]
		}
	}

	return result, nil
}

// Обработка callback кнопок выбора предмета для создания/удаления урока
func handleLessonSubjectCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	parts := strings.Split(query.Data, ":")
	if len(parts) != 2 {
		sendMessage(bot, query.Message.Chat.ID, "❌ Неверный формат команды")
		return
	}
	
	action := parts[0] // "create_lesson" или "delete_lesson"
	subjectID, err := strconv.Atoi(parts[1])
	if err != nil {
		sendMessage(bot, query.Message.Chat.ID, "❌ Неверный ID предмета")
		return
	}
	
	// Получаем название предмета
	var subjectName string
	err = db.QueryRow("SELECT name FROM subjects WHERE id = $1", subjectID).Scan(&subjectName)
	if err != nil {
		sendMessage(bot, query.Message.Chat.ID, "❌ Предмет не найден")
		return
	}
	
	// Проверяем права пользователя
	userID := query.From.ID
	var role string
	err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, query.Message.Chat.ID, "❌ У вас нет прав для этого действия")
		return
	}
	
	if action == "create_lesson" {
		// Показываем форму для ввода даты и времени
		text := fmt.Sprintf("📚 **Создание урока: %s**\n\n" +
			"Введите дату и время урока в формате:\n" +
			"`/create_lesson \"%s\" ДД.ММ.ГГГГ ЧЧ:ММ`\n\n" +
			"**Пример:**\n" +
			"`/create_lesson \"%s\" 16.08.2025 16:30`", 
			subjectName, subjectName, subjectName)
			
		editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)
		
	} else if action == "delete_lesson" {
		// Показываем уроки этого предмета для удаления
		showLessonsForDeletion(bot, query, db, subjectID, subjectName)
	}
}

// Показать уроки предмета для удаления
func showLessonsForDeletion(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB, subjectID int, subjectName string) {
	// Получаем уроки этого предмета для учителя
	userID := strconv.FormatInt(query.From.ID, 10)
	
	rows, err := db.Query(`
		SELECT l.id, l.lesson_date, l.lesson_time 
		FROM lessons l 
		JOIN subjects s ON l.subject_id = s.id 
		WHERE l.subject_id = $1 AND l.teacher_id = $2 AND l.is_deleted = false
		ORDER BY l.lesson_date, l.lesson_time`,
		subjectID, userID)
	
	if err != nil {
		sendMessage(bot, query.Message.Chat.ID, "❌ Ошибка при получении уроков")
		return
	}
	defer rows.Close()
	
	var buttons [][]tgbotapi.InlineKeyboardButton
	lessonCount := 0
	
	for rows.Next() {
		var lessonID int
		var lessonDate, lessonTime string
		
		if err := rows.Scan(&lessonID, &lessonDate, &lessonTime); err != nil {
			continue
		}
		
		lessonCount++
		buttonText := fmt.Sprintf("%s %s", lessonDate, lessonTime)
		callbackData := fmt.Sprintf("cancel_lesson:%d", lessonID)
		
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	
	if lessonCount == 0 {
		sendMessage(bot, query.Message.Chat.ID, fmt.Sprintf("📚 У вас нет активных уроков по предмету \"%s\"", subjectName))
		return
	}
	
	// Кнопка "Назад"
	backButton := tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back_to_subjects")
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{backButton})
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	text := fmt.Sprintf("📚 **Выберите урок для отмены (%s):**", subjectName)
	
	editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	
	bot.Send(editMsg)
}

// Новый роутер для callback запросов (заменяет существующий)
func handleNewCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Убрать индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("Ошибка callback ответа: %v", err)
	}

	// Специальная обработка для create_lesson и delete_lesson кнопок
	if strings.HasPrefix(query.Data, "create_lesson:") || strings.HasPrefix(query.Data, "delete_lesson:") {
		handleLessonSubjectCallback(bot, query, db)
		return
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

	// Обработка общих кнопок интерфейса (делегируем в handleInlineButton)
	switch query.Data {
	case "main_menu", "create_lesson", "cancel_lesson", "schedule", "my_lessons", 
		 "help", "profile", "teachers", "stats", "notifications", "logs", 
		 "help_teacher", "help_admin", "back_to_main", "back_to_schedule", 
		 "back", "cancel_action", "student_dashboard", "enroll_subjects", 
		 "my_lessons_menu", "school_schedule":
		handleInlineButton(bot, query, db)
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
		log.Printf("Неизвестное callback действие: %s (данные: %s)", callbackData.Action, query.Data)
		sendMessage(bot, query.Message.Chat.ID, "❓ Неизвестное действие: " + query.Data)
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
			callbackResponse := tgbotapi.NewCallback(query.ID, reason.Error())
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
			callbackResponse := tgbotapi.NewCallback(query.ID, reason.Error())
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
			callbackResponse := tgbotapi.NewCallback(query.ID, reason.Error())
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
