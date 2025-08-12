package handlers

import (
	"database/sql"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Отправка сообщения с обработкой ошибок
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// Получение роли пользователя
func getUserRole(db *sql.DB, telegramID int64) (string, error) {
	var role string
	query := "SELECT role FROM users WHERE tg_id = $1 AND is_active = true"
	err := db.QueryRow(query, strconv.FormatInt(telegramID, 10)).Scan(&role)
	return role, err
}

// Получение ID пользователя по Telegram ID
func getUserID(db *sql.DB, telegramID int64) (int, error) {
	var userID int
	query := "SELECT id FROM users WHERE tg_id = $1 AND is_active = true"
	err := db.QueryRow(query, strconv.FormatInt(telegramID, 10)).Scan(&userID)
	return userID, err
}

// Проверка существования пользователя
func userExists(db *sql.DB, telegramID int64) bool {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE tg_id = $1 AND is_active = true"
	err := db.QueryRow(query, strconv.FormatInt(telegramID, 10)).Scan(&count)
	if err != nil {
		log.Printf("Ошибка проверки существования пользователя: %v", err)
		return false
	}
	return count > 0
}
