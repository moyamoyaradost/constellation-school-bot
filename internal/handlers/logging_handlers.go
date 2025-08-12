package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Структура для логов
type SimpleLog struct {
	ID        int
	Action    string
	UserID    *int
	Details   string
	CreatedAt time.Time
}

// Логирование действия в БД
func LogAction(db *sql.DB, action string, userID *int, details string) error {
	_, err := db.Exec(`
		INSERT INTO simple_logs (action, user_id, details, created_at)
		VALUES ($1, $2, $3, NOW())`,
		action, userID, details)
	return err
}

// Получение последних ошибок
func GetRecentErrors(db *sql.DB, limit int) ([]SimpleLog, error) {
	rows, err := db.Query(`
		SELECT id, action, user_id, details, created_at
		FROM simple_logs
		WHERE action LIKE '%error%' OR action LIKE '%ошибка%'
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []SimpleLog
	for rows.Next() {
		var log SimpleLog
		if err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.Details, &log.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, log)
	}
	
	return logs, nil
}

// Команда для просмотра последних ошибок
func handleLogRecentErrorsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра логов")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	limit := 10 // по умолчанию 10 записей
	
	if len(args) >= 2 {
		if parsedLimit, err := strconv.Atoi(args[1]); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}
	
	// Получаем последние ошибки
	logs, err := GetRecentErrors(db, limit)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения логов")
		return
	}
	
	if len(logs) == 0 {
		sendMessage(bot, message.Chat.ID, "✅ Ошибок в логах не найдено")
		return
	}
	
	// Формируем отчет
	var report strings.Builder
	report.WriteString(fmt.Sprintf("📋 **Последние ошибки (%d записей)**\n\n", len(logs)))
	
	for i, log := range logs {
		report.WriteString(fmt.Sprintf("**%d.** %s\n", i+1, log.Action))
		if log.UserID != nil {
			report.WriteString(fmt.Sprintf("   👤 Пользователь: %d\n", *log.UserID))
		}
		if log.Details != "" {
			// Обрезаем длинные детали
			details := log.Details
			if len(details) > 100 {
				details = details[:100] + "..."
			}
			report.WriteString(fmt.Sprintf("   📝 Детали: %s\n", details))
		}
		report.WriteString(fmt.Sprintf("   ⏰ %s\n\n", log.CreatedAt.Format("02.01.2006 15:04:05")))
	}
	
	// Разбиваем на части, если сообщение слишком длинное
	reportText := report.String()
	if len(reportText) > 4000 {
		reportText = reportText[:4000] + "\n\n... (сообщение обрезано)"
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, reportText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Логирование ошибки с автоматическим определением пользователя
func LogError(db *sql.DB, action string, userID int64, details string) {
	userIDInt := int(userID)
	LogAction(db, "ERROR: "+action, &userIDInt, details)
}

// Логирование действия пользователя
func LogUserAction(db *sql.DB, action string, userID int64, details string) {
	userIDInt := int(userID)
	LogAction(db, action, &userIDInt, details)
}

// Логирование системного действия
func LogSystemAction(db *sql.DB, action string, details string) {
	LogAction(db, "SYSTEM: "+action, nil, details)
}
