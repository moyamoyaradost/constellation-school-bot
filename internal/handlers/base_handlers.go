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
			helpText = "🆘 **Помощь для студентов**\n\n" +
				"📚 **Основные команды:**\n" +
				"• `/start` - главное меню с кнопками\n" +
				"• `/menu` - быстрый вызов главного меню\n" +
				"• `/schedule` - расписание уроков школы\n" +
				"• `/my_lessons` - мои записи на уроки\n" +
				"• `/enroll` - записаться на урок\n" +
				"• `/waitlist` - лист ожидания\n" +
				"• `/help` - эта справка\n\n" +
				"🎯 **Как записаться на урок:**\n" +
				"1. Нажмите кнопку 'Записаться' в главном меню\n" +
				"2. Выберите предмет кнопками\n" +
				"3. Выберите урок из доступных\n" +
				"4. Подтвердите запись\n\n" +
				"💡 **Подсказка:** ID урока отображается в расписании как #123"
				
		case "teacher":
			helpText = "🆘 **Помощь для преподавателей**\n\n" +
				"👨‍🏫 **Основные команды:**\n" +
				"• `/start` - главное меню с кнопками\n" +
				"• `/menu` - быстрый вызов главного меню\n" +
				"• `/my_schedule` - мое расписание на неделю\n" +
				"• `/my_students` - студенты моих уроков\n" +
				"• `/create_lesson` - создать новый урок\n" +
				"• `/cancel_lesson` - отменить урок\n" +
				"• `/help_teacher` - расширенная справка\n" +
				"• `/help` - эта справка\n\n" +
				"🎯 **Как создать урок:**\n" +
				"1. Нажмите кнопку '➕ Создать урок'\n" +
				"2. Выберите предмет кнопками\n" +
				"3. Укажите дату и время\n" +
				"4. Подтвердите создание\n\n" +
				"🗑️ **Как отменить урок:**\n" +
				"1. Нажмите кнопку '🗑️ Отменить урок'\n" +
				"2. Выберите предмет и урок\n" +
				"3. Студенты получат уведомления автоматически"
				
		case "superuser":
			helpText = "🆘 **Помощь для администраторов**\n\n" +
				"🔧 **Управление учителями:**\n" +
				"• `/add_teacher` - добавить преподавателя\n" +
				"• `/delete_teacher` - удалить преподавателя\n" +
				"• `/restore_teacher` - восстановить преподавателя\n" +
				"• `/list_teachers` - список всех преподавателей\n\n" +
				"📚 **Управление уроками:**\n" +
				"• `/create_lesson` - создать урок\n" +
				"• `/delete_lesson` - удалить урок\n" +
				"• `/restore_lesson` - восстановить урок\n" +
				"• `/reschedule_lesson` - перенести урок\n\n" +
				"👥 **Управление студентами:**\n" +
				"• `/deactivate_student` - заблокировать студента\n" +
				"• `/activate_student` - разблокировать студента\n\n" +
				"📢 **Уведомления:**\n" +
				"• `/notify_all` - уведомить всех пользователей\n" +
				"• `/notify_students` - уведомить студентов урока\n" +
				"• `/remind_all` - напомнить о предстоящих уроках\n" +
				"• `/cancel_with_notification` - отменить урок с уведомлением\n\n" +
				"📊 **Статистика и логи:**\n" +
				"• `/stats` - общая статистика системы\n" +
				"• `/rate_limit_stats` - статистика операций\n" +
				"• `/log_recent_errors` - последние ошибки системы\n\n" +
				"• `/help` - эта справка"
				
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
