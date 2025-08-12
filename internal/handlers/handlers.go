package handlers

import (
	"database/sql"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Основной обработчик обновлений
func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	if update.Message != nil {
		if update.Message.IsCommand() {
			handleCommand(bot, update.Message, db)
		} else {
			// Обработка текста через FSM
			handleTextMessage(bot, update.Message, db)
		}
	} else if update.CallbackQuery != nil {
		handleNewCallbackQuery(bot, update.CallbackQuery, db)
	}
}

// Маршрутизация команд
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	switch message.Command() {
	case "start":
		handleStart(bot, message, db)
	case "register":
		handleRegister(bot, message, db)
	case "help":
		handleHelp(bot, message, db)
	case "subjects", "schedule", "enroll", "waitlist", "my_lessons":
		handleStudentCommand(bot, message, db)
	case "create_lesson", "reschedule_lesson", "cancel_lesson":
		handleTeacherCommand(bot, message, db)
	case "add_teacher", "delete_teacher", "notify_students", "cancel_with_notification", "reschedule_with_notify", "list_teachers", "my_students", "restore_lesson", "restore_teacher", "rate_limit_stats", "stats", "log_recent_errors":
		handleAdminCommand(bot, message, db)
	default:
		sendMessage(bot, message.Chat.ID, 
			"❓ Неизвестная команда. Используйте /help для получения списка доступных команд.")
	}
}

// Обработка callback запросов
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Убрать индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("Ошибка callback ответа: %v", err)
	}

	if query.Data == "cancel_lesson" {
		handleCancelLessonCallback(bot, query, db)
	} else {
		handleStudentCallback(bot, query, db)
	}
}
