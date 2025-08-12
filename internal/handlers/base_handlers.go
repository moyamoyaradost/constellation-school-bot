package handlers

import (
	"database/sql"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Команда помощи
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
				"/subjects - доступные предметы\n" +
				"/schedule - расписание уроков\n" +
				"/enroll - записаться на урок\n" +
				"/my_lessons - мои записи\n" +
				"/help - эта справка"
				
		case "teacher":
			helpText = "🆘 Помощь для преподавателей\n\n" +
				"👨‍🏫 Доступные команды:\n" +
				"/create_lesson - создать урок\n" +
				"/cancel_lesson - отменить урок\n" +
				"/my_students - мои студенты\n" +
				"/help - эта справка"
				
		case "superuser":
			helpText = "🆘 Помощь для администраторов\n\n" +
				"🔧 Доступные команды:\n" +
				"/add_teacher - добавить преподавателя\n" +
				"/delete_teacher - удалить преподавателя\n" +
				"/notify_students - уведомить студентов\n" +
				"/create_lesson - создать урок\n" +
				"/help - эта справка"
				
		default:
			helpText = "🆘 Помощь\n\nИспользуйте /start для начала работы"
		}
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	bot.Send(msg)
}

// Обработчик callback для студентов
func handleStudentCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Базовая обработка callback запросов от студентов
	sendMessage(bot, query.Message.Chat.ID, "⚙️ Функция в разработке")
}

// Обработчик callback отмены уроков
func handleCancelLessonCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Базовая обработка отмены уроков
	sendMessage(bot, query.Message.Chat.ID, "⚙️ Отмена урока - функция в разработке")
}
