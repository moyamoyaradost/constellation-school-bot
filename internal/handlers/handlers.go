package handlers

import (
"database/sql"
"log"

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
		handleTextMessage(bot, message, db)
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
		sendMessage(bot, message.Chat.ID, "🆘 Используйте /start для начала работы")
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда. Используйте /start")
	}
}

// Обработка callback запросов
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := bot.Request(callback); err != nil {
		log.Printf("Ошибка callback ответа: %v", err)
	}
	sendMessage(bot, query.Message.Chat.ID, "⚙️ Функция в разработке")
}

// Вспомогательная функция отправки сообщений
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
