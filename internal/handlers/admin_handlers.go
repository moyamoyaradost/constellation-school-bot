package handlers

import (
	"database/sql"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Обработчик команд для администраторов
func handleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав администратора")
		return
	}
	
	// Маршрутизация команд администратора
	switch message.Command() {
	case "add_teacher":
		handleAddTeacherCommand(bot, message, db)
	case "delete_teacher":
		handleDeleteTeacherCommand(bot, message, db)
	case "notify_students":
		handleNotifyStudentsCommand(bot, message, db)
	case "cancel_with_notification":
		handleCancelWithNotificationCommand(bot, message, db)
	case "reschedule_with_notify":
		handleRescheduleWithNotifyCommand(bot, message, db)
	case "list_teachers":
		handleListTeachersCommand(bot, message, db)
	case "my_students":
		handleMyStudentsCommand(bot, message, db)
	case "restore_lesson":
		handleRestoreLessonCommand(bot, message, db)
	case "restore_teacher":
		handleRestoreTeacherCommand(bot, message, db)
	case "rate_limit_stats":
		handleRateLimitStatsCommand(bot, message, db)
	case "stats":
		handleStatsCommand(bot, message, db)
	case "log_recent_errors":
		handleLogRecentErrorsCommand(bot, message, db)
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда администратора")
	}
}
