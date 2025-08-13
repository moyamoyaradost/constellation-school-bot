package handlers

import (
	"database/sql"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Обработчик команд для администраторов
func handleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || (role != "admin" && role != "superuser") {
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
		sendMessage(bot, message.Chat.ID, "⚙️ Команда в разработке")
	case "reschedule_with_notify":
		sendMessage(bot, message.Chat.ID, "⚙️ Команда в разработке")
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
	case "delete_lesson":
		handleDeleteLessonCommand(bot, message, db)
	case "notify_all":
		handleNotifyAllCommand(bot, message, db)
	case "remind_all":
		handleRemindAllCommand(bot, message, db)
	case "deactivate_student":
		handleDeactivateStudentCommand(bot, message, db)
	case "activate_student":
		handleActivateStudentCommand(bot, message, db)
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда администратора")
	}
}

// Уведомления студентам урока
func handleNotifyStudentsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя (дополнительная проверка)
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || (role != "admin" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отправки уведомлений")
		return
	}
	
	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "📢 **Уведомления студентам урока**\n\n" +
			"**Формат:** `/notify_students <lesson_id> <сообщение>`\n\n" +
			"**Примеры:**\n" +
			"• `/notify_students 15 Урок переносится на час позже`\n" +
			"• `/notify_students 22 Не забудьте принести материалы`\n\n" +
			"**Получат уведомление:** Все студенты, записанные на указанный урок"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	// Получаем lesson_id и сообщение
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}
	
	notificationText := strings.Join(args[2:], " ")
	
	// Проверяем, существует ли урок
	var subjectName, teacherName string
	var startTime string
	err = db.QueryRow(`
		SELECT s.name, COALESCE(u.full_name, 'Не назначен'), l.start_time::text
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1 AND l.soft_deleted = false`, lessonID).Scan(&subjectName, &teacherName, &startTime)
		
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска урока")
		return
	}
	
	// Отправляем уведомления студентам урока
	sentCount, failedCount := notifyStudentsOfLesson(bot, db, lessonID, notificationText, subjectName, teacherName, startTime)
	
	// Отчет администратору
	resultText := "✅ **Уведомления отправлены**\n\n" +
		"📚 Урок: " + subjectName + " (" + startTime[:16] + ")\n" +
		"👨‍🏫 Преподаватель: " + teacherName + "\n\n" +
		"📤 Отправлено: " + strconv.Itoa(sentCount) + "\n" +
		"❌ Не удалось отправить: " + strconv.Itoa(failedCount) + "\n\n" +
		"💬 Сообщение: " + notificationText
	
	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Вспомогательная функция: отправка уведомлений студентам урока
func notifyStudentsOfLesson(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int, message, subjectName, teacherName, startTime string) (int, int) {
	// Получаем студентов, записанных на урок
	rows, err := db.Query(`
		SELECT u.tg_id, u.full_name
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE e.lesson_id = $1 AND e.status = 'enrolled'`, lessonID)
		
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	
	var sentCount, failedCount int
	
	for rows.Next() {
		var studentTelegramID int64
		var studentName string
		
		if err := rows.Scan(&studentTelegramID, &studentName); err != nil {
			failedCount++
			continue
		}
		
		// Формируем уведомление
		notificationText := "📢 **Уведомление об уроке**\n\n" +
			"📚 Предмет: " + subjectName + "\n" +
			"👨‍🏫 Преподаватель: " + teacherName + "\n" +
			"📅 Время: " + startTime[:16] + "\n\n" +
			"💬 Сообщение: " + message
		
		msg := tgbotapi.NewMessage(studentTelegramID, notificationText)
		msg.ParseMode = "Markdown"
		
		// Retry механизм (3 попытки)
		sent := false
		for i := 0; i < 3; i++ {
			if _, err := bot.Send(msg); err == nil {
				sent = true
				break
			}
		}
		
		if sent {
			sentCount++
		} else {
			failedCount++
		}
	}
	
	return sentCount, failedCount
}
