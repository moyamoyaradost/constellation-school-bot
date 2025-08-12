package handlers

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Простой роутер - перенаправляет на handlers.go
func HandleUpdateSimple(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	// Используем полнофункциональный обработчик из handlers.go
	HandleUpdate(bot, update, db)
}
