package handlers

import (
	"database/sql"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Справка для преподавателей (отсутствующая команда Teacher)
func handleHelpTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра справки преподавателя")
		return
	}

	helpText := "👨‍🏫 **Справка для преподавателей**\n\n" +
		"**📋 Доступные команды:**\n\n" +
		"**📅 Управление уроками:**\n" +
		"• `/create_lesson <subject_code> <date> <time> [max_students]` - создать урок\n" +
		"• `/reschedule_lesson <lesson_id> <new_date> <new_time>` - перенести урок\n" +
		"• `/cancel_lesson <lesson_id>` - отменить урок\n\n" +
		"**👥 Управление студентами:**\n" +
		"• `/my_students` - список моих студентов\n" +
		"• `/my_lessons` - мои уроки\n\n" +
		"**📚 Доступные предметы:**\n" +
		"• `3D_MODELING` - 3D-моделирование\n" +
		"• `GAMEDEV` - Геймдев\n" +
		"• `VFX_DESIGN` - VFX-дизайн\n" +
		"• `GRAPHIC_DESIGN` - Графический дизайн\n" +
		"• `WEB_DEV` - Веб-разработка\n" +
		"• `COMPUTER_LITERACY` - Компьютерная грамотность\n\n" +
		"**📝 Примеры команд:**\n" +
		"• `/create_lesson WEB_DEV 2025-08-15 18:00 10`\n" +
		"• `/reschedule_lesson 15 2025-08-16 19:00`\n" +
		"• `/cancel_lesson 22`\n\n" +
		"**ℹ️ Дополнительная информация:**\n" +
		"• Максимальное количество студентов по умолчанию: 10\n" +
		"• Длительность урока по умолчанию: 90 минут\n" +
		"• Формат даты: YYYY-MM-DD\n" +
		"• Формат времени: HH:MM (24-часовой)\n\n" +
		"**🆘 Поддержка:**\n" +
		"При возникновении проблем обращайтесь к администратору."

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
