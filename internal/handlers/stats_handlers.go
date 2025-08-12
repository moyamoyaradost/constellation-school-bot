package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Статистика rate limiting
func handleRateLimitStatsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра статистики")
		return
	}
	
	// Получаем детальную статистику rate limiting
	stats := getRateLimitDetailedStats(db)
	
	// Формируем отчет
	reportText := "📊 **Статистика Rate Limiting**\n\n"
	
	if len(stats) == 0 {
		reportText += "📭 Нет активных операций"
	} else {
		reportText += "🔄 **Активные операции:**\n\n"
		
		for _, stat := range stats {
			reportText += fmt.Sprintf("👤 **Пользователь:** %s\n", stat.UserName)
			reportText += fmt.Sprintf("🆔 **Telegram ID:** %s\n", stat.TelegramID)
			reportText += fmt.Sprintf("📝 **Операция:** %s\n", stat.OperationType)
			reportText += fmt.Sprintf("⏰ **Начата:** %s\n", stat.StartedAt.Format("02.01.2006 15:04:05"))
			reportText += fmt.Sprintf("⏱️ **Длительность:** %s\n", time.Since(stat.StartedAt).Round(time.Second))
			reportText += "---\n"
		}
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, reportText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Общая статистика системы
func handleStatsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра статистики")
		return
	}
	
	// Получаем базовую статистику системы
	stats := getBasicSystemStats(db)
	
	// Формируем отчет
	reportText := "📊 **Общая статистика системы**\n\n"
	
	reportText += fmt.Sprintf("👥 **Пользователи:**\n")
	reportText += fmt.Sprintf("• Всего: %d\n", stats.TotalUsers)
	reportText += fmt.Sprintf("• Активных: %d\n", stats.ActiveUsers)
	reportText += fmt.Sprintf("• Студентов: %d\n", stats.Students)
	reportText += fmt.Sprintf("• Преподавателей: %d\n", stats.Teachers)
	reportText += fmt.Sprintf("• Администраторов: %d\n\n", stats.Admins)
	
	reportText += fmt.Sprintf("📚 **Уроки:**\n")
	reportText += fmt.Sprintf("• Всего: %d\n", stats.TotalLessons)
	reportText += fmt.Sprintf("• Активных: %d\n", stats.ActiveLessons)
	reportText += fmt.Sprintf("• Отмененных: %d\n\n", stats.CancelledLessons)
	
	reportText += fmt.Sprintf("📝 **Записи:**\n")
	reportText += fmt.Sprintf("• Всего: %d\n", stats.TotalEnrollments)
	reportText += fmt.Sprintf("• Активных: %d\n", stats.ActiveEnrollments)
	reportText += fmt.Sprintf("• Отмененных: %d\n\n", stats.CancelledEnrollments)
	
	reportText += fmt.Sprintf("⏰ **Листы ожидания:**\n")
	reportText += fmt.Sprintf("• Всего записей: %d\n\n", stats.WaitlistEntries)
	
	reportText += fmt.Sprintf("🔄 **Rate Limiting:**\n")
	reportText += fmt.Sprintf("• Активных операций: %d\n", stats.ActiveRateLimitOperations)
	
	msg := tgbotapi.NewMessage(message.Chat.ID, reportText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Структура для статистики rate limiting
type RateLimitStat struct {
	UserName      string
	TelegramID    string
	OperationType string
	StartedAt     time.Time
}

// Получение детальной статистики rate limiting
func getRateLimitDetailedStats(db *sql.DB) []RateLimitStat {
	rows, err := db.Query(`
		SELECT u.full_name, u.tg_id, po.operation_type, po.started_at
		FROM pending_operations po
		JOIN users u ON po.user_id = u.id
		WHERE po.finished_at IS NULL
		ORDER BY po.started_at DESC`)
	
	if err != nil {
		return []RateLimitStat{}
	}
	defer rows.Close()
	
	var stats []RateLimitStat
	for rows.Next() {
		var stat RateLimitStat
		if err := rows.Scan(&stat.UserName, &stat.TelegramID, &stat.OperationType, &stat.StartedAt); err != nil {
			continue
		}
		stats = append(stats, stat)
	}
	
	return stats
}

// Структура для базовой статистики системы
type BasicSystemStats struct {
	TotalUsers                    int
	ActiveUsers                   int
	Students                      int
	Teachers                      int
	Admins                        int
	TotalLessons                  int
	ActiveLessons                 int
	CancelledLessons              int
	TotalEnrollments              int
	ActiveEnrollments             int
	CancelledEnrollments          int
	WaitlistEntries               int
	ActiveRateLimitOperations     int
}

// Получение базовой статистики системы
func getBasicSystemStats(db *sql.DB) BasicSystemStats {
	var stats BasicSystemStats
	
	// Статистика пользователей
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&stats.ActiveUsers)
	db.QueryRow("SELECT COUNT(*) FROM students WHERE soft_deleted = false").Scan(&stats.Students)
	db.QueryRow("SELECT COUNT(*) FROM teachers WHERE soft_deleted = false").Scan(&stats.Teachers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'superuser'").Scan(&stats.Admins)
	
	// Статистика уроков
	db.QueryRow("SELECT COUNT(*) FROM lessons").Scan(&stats.TotalLessons)
	db.QueryRow("SELECT COUNT(*) FROM lessons WHERE soft_deleted = false").Scan(&stats.ActiveLessons)
	db.QueryRow("SELECT COUNT(*) FROM lessons WHERE soft_deleted = true").Scan(&stats.CancelledLessons)
	
	// Статистика записей
	db.QueryRow("SELECT COUNT(*) FROM enrollments").Scan(&stats.TotalEnrollments)
	db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE status = 'enrolled'").Scan(&stats.ActiveEnrollments)
	db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE status = 'cancelled'").Scan(&stats.CancelledEnrollments)
	
	// Статистика листов ожидания
	db.QueryRow("SELECT COUNT(*) FROM waitlist").Scan(&stats.WaitlistEntries)
	
	// Статистика rate limiting
	db.QueryRow("SELECT COUNT(*) FROM pending_operations WHERE finished_at IS NULL").Scan(&stats.ActiveRateLimitOperations)
	
	return stats
}
