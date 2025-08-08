package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Основной обработчик сообщений
func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	if update.Message != nil {
		handleMessage(bot, update.Message, db)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update.CallbackQuery, db)
	}
}

// Обработка текстовых сообщений
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	if message.IsCommand() {
		handleCommand(bot, message, db)
	} else {
		// Обработка текста через FSM
		handleTextMessage(bot, message, db)
	}
}

// Обработка команд
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	switch message.Command() {
	case "start":
		handleStart(bot, message, db)
	case "register":
		handleRegister(bot, message, db)
	case "help":
		handleHelp(bot, message, db)
	case "subjects":
		handleSubjectsCommand(bot, message, db)
	case "schedule":
		handleScheduleCommand(bot, message, db)
	case "my_lessons":
		handleMyLessonsCommand(bot, message, db)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, 
			"❓ Неизвестная команда. Используйте /help для получения списка доступных команд.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}

// Обработка callback запросов (inline кнопки)
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Ответить на callback чтобы убрать индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callback)
	
	data := query.Data
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	
	switch {
	case data == "cmd_register":
		// Имитация команды /register
		fakeMessage := &tgbotapi.Message{
			From: query.From,
			Chat: query.Message.Chat,
			Text: "/register",
		}
		handleRegister(bot, fakeMessage, db)
		
	case data == "cmd_help":
		// Имитация команды /help
		fakeMessage := &tgbotapi.Message{
			From: query.From,
			Chat: query.Message.Chat,
			Text: "/help",
		}
		handleHelp(bot, fakeMessage, db)
		
	case strings.HasPrefix(data, "subject_"):
		handleSubjectCallback(bot, query, db)
		
	case data == "finish_registration":
		finishStudentRegistration(bot, userID, chatID, db)
		
	default:
		msg := tgbotapi.NewMessage(chatID, "❓ Неизвестное действие")
		bot.Send(msg)
	}
}

// Обработка выбора предмета через callback
func handleSubjectCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	
	// Проверить состояние пользователя
	if getUserState(userID) != StateSelectingSubjects {
		msg := tgbotapi.NewMessage(chatID, "❌ Сначала начните регистрацию командой /register")
		bot.Send(msg)
		return
	}
	
	// Извлечь ID предмета
	subjectIDStr := strings.TrimPrefix(query.Data, "subject_")
	subjectID, err := strconv.Atoi(subjectIDStr)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка выбора предмета")
		bot.Send(msg)
		return
	}
	
	// Получить текущие выбранные предметы
	initUserData(userID)
	var selectedSubjects []int
	if subjects, exists := userData[userID]["selected_subjects"]; exists {
		selectedSubjects = subjects.([]int)
	}
	
	// Проверить, не выбран ли уже этот предмет
	var alreadySelected bool
	var newSubjects []int
	
	for _, id := range selectedSubjects {
		if id == subjectID {
			alreadySelected = true
		} else {
			newSubjects = append(newSubjects, id)
		}
	}
	
	// Получить название предмета
	var subjectName string
	db.QueryRow("SELECT name FROM subjects WHERE id = $1", subjectID).Scan(&subjectName)
	
	var responseText string
	if alreadySelected {
		// Убрать из выбранных
		userData[userID]["selected_subjects"] = newSubjects
		responseText = fmt.Sprintf("➖ Предмет '%s' убран из списка", subjectName)
	} else {
		// Добавить в выбранные
		newSubjects = append(selectedSubjects, subjectID)
		userData[userID]["selected_subjects"] = newSubjects
		responseText = fmt.Sprintf("➕ Предмет '%s' добавлен в список", subjectName)
	}
	
	// Показать текущий выбор
	var currentSubjects []string
	for _, id := range newSubjects {
		var name string
		db.QueryRow("SELECT name FROM subjects WHERE id = $1", id).Scan(&name)
		currentSubjects = append(currentSubjects, name)
	}
	
	if len(currentSubjects) > 0 {
		responseText += fmt.Sprintf("\n\n✅ Выбранные предметы:\n• %s", 
			strings.Join(currentSubjects, "\n• "))
	} else {
		responseText += "\n\n📝 Пока не выбрано ни одного предмета"
	}
	
	// Отправить уведомление
	msg := tgbotapi.NewMessage(chatID, responseText)
	bot.Send(msg)
}

// Команда /help
func handleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверить роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	var helpText string
	
	if err == sql.ErrNoRows {
		// Незарегистрированный пользователь
		helpText = "🆘 Помощь - Constellation School Bot\n\n" +
			"👋 Добро пожаловать! Для начала работы необходимо зарегистрироваться.\n\n" +
			"📝 Доступные команды:\n" +
			"/start - начальное приветствие\n" +
			"/register - регистрация в системе\n" +
			"/help - эта справка\n\n" +
			"🎯 О Центре Цифрового Творчества:\n" +
			"Мы предлагаем 6 направлений обучения:\n" +
			"• 3D-моделирование\n" +
			"• Геймдев\n" +
			"• VFX-дизайн\n" +
			"• Графический дизайн\n" +
			"• Веб-разработка\n" +
			"• Компьютерная грамотность"
			
	} else if err != nil {
		helpText = "❌ Ошибка получения информации о пользователе"
		
	} else {
		switch role {
		case "student":
			helpText = "🆘 Помощь для студентов\n\n" +
				"📚 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/subjects - выбор/изменение предметов\n" +
				"/schedule - расписание доступных уроков\n" +
				"/my_lessons - мои записи на уроки\n" +
				"/help - эта справка\n\n" +
				"❓ Как записаться на урок:\n" +
				"1. Используйте /schedule для просмотра доступных уроков\n" +
				"2. Нажмите 'Записаться' у интересующего урока\n" +
				"3. Подтвердите запись\n\n" +
				"📞 Нужна помощь? Обратитесь к администратору."
				
		case "teacher":
			helpText = "🆘 Помощь для преподавателей\n\n" +
				"👨‍🏫 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/my_lessons - мои уроки\n" +
				"/my_students - студенты на моих уроках\n" +
				"/help_teacher - подробная справка\n\n" +
				"📝 Управление уроками:\n" +
				"• Просмотр списка своих уроков\n" +
				"• Просмотр записанных студентов\n" +
				"• Отмена уроков при необходимости"
				
		case "superuser":
			helpText = "🆘 Помощь для администраторов\n\n" +
				"🔧 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/add_teacher - добавить нового преподавателя\n" +
				"/create_lesson - создать новый урок\n" +
				"/system_stats - статистика системы\n" +
				"/manage_subjects - управление предметами\n\n" +
				"⚡ Административные функции:\n" +
				"• Управление преподавателями\n" +
				"• Создание и управление уроками\n" +
				"• Просмотр статистики системы"
				
		default:
			helpText = "🆘 Помощь\n\nИспользуйте /start для начала работы"
		}
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	bot.Send(msg)
}

// Заглушки для других команд (будут реализованы в следующих шагах)
func handleSubjectsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	msg := tgbotapi.NewMessage(message.Chat.ID, 
		"🎯 Управление предметами будет доступно в следующих обновлениях")
	bot.Send(msg)
}

func handleScheduleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	msg := tgbotapi.NewMessage(message.Chat.ID, 
		"📅 Просмотр расписания будет доступен в следующих обновлениях")
	bot.Send(msg)
}

func handleMyLessonsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	msg := tgbotapi.NewMessage(message.Chat.ID, 
		"📖 Просмотр ваших уроков будет доступен в следующих обновлениях")
	bot.Send(msg)
}
