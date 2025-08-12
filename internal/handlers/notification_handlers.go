package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Расширенные уведомления студентам (Шаг 8.2 ROADMAP)
func handleNotifyStudentsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отправки уведомлений")
		return
	}
	
	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "📢 **Уведомления студентам урока** (Шаг 8.2)\n\n" +
			"**Формат:** `/notify_students <lesson_id> <сообщение>`\n\n" +
			"**Примеры:**\n" +
			"• `/notify_students 15 Урок переносится на час позже`\n" +
			"• `/notify_students 22 Не забудьте принести материалы`\n\n" +
			"**Получат уведомление:** Все студенты, записанные на указанный урок\n\n" +
			"**См. также:**\n" +
			"• `/cancel_with_notification` - отмена с объяснением\n" +
			"• `/reschedule_with_notify` - перенос с уведомлениями"
		
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
		SELECT s.name, u.full_name, l.start_time::text
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
	
	// Отправляем уведомления студентам урока с retry-механизмом
	sentCount, failedCount := notifyStudentsOfLesson(bot, db, lessonID, notificationText, subjectName, teacherName, startTime)
	
	// Логируем отправку уведомлений
	LogSystemAction(db, "notifications_sent", fmt.Sprintf("Урок %d (%s), отправлено: %d, ошибок: %d", lessonID, subjectName, sentCount, failedCount))
	
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

// Отмена урока с уведомлением (Шаг 8.2 ROADMAP)
func handleCancelWithNotificationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || (role != "admin" && role != "teacher") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отмены уроков")
		return
	}
	
	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "❌ **Отмена урока с уведомлением** (Шаг 8.2)\n\n" +
			"**Формат:** `/cancel_with_notification <lesson_id> <причина>`\n\n" +
			"**Примеры:**\n" +
			"• `/cancel_with_notification 15 Преподаватель заболел`\n" +
			"• `/cancel_with_notification 22 Технические проблемы`\n\n" +
			"**Что произойдет:**\n" +
			"• Отмена урока\n" +
			"• Уведомления всех записанных студентов\n" +
			"• Очистка листа ожидания\n\n" +
			"**См. также:**\n" +
			"• `/notify_students` - произвольные уведомления\n" +
			"• `/reschedule_with_notify` - перенос с уведомлениями"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	// Получаем lesson_id и причину
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}
	
	reason := strings.Join(args[2:], " ")
	
	// Проверяем, существует ли урок
	var subjectName, teacherName string
	var startTime string
	var teacherID int
	err = db.QueryRow(`
		SELECT s.name, u.full_name, l.start_time::text, l.teacher_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1 AND l.soft_deleted = false`, lessonID).Scan(&subjectName, &teacherName, &startTime, &teacherID)
		
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска урока")
		return
	}
	
	// Проверяем права доступа (учитель может отменять только свои уроки)
	if role == "teacher" {
		var currentTeacherID int
		err = db.QueryRow(`
			SELECT t.id FROM teachers t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.tg_id = $1`, userID).Scan(&currentTeacherID)
		
		if err != nil || currentTeacherID != teacherID {
			sendMessage(bot, message.Chat.ID, "❌ Вы можете отменять только свои уроки")
			return
		}
	}
	
	// Отменяем урок с уведомлениями
	cancelLessonWithNotification(bot, db, lessonID, subjectName, teacherName, startTime, reason, message.Chat.ID)
}

// Перенос урока с уведомлениями (Шаг 8.2 ROADMAP)
func handleRescheduleWithNotifyCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || (role != "admin" && role != "teacher") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для переноса уроков")
		return
	}
	
	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "🔄 **Перенос урока с уведомлениями** (Шаг 8.2)\n\n" +
			"**Формат:** `/reschedule_with_notify <lesson_id> <новое_время>`\n\n" +
			"**Примеры:**\n" +
			"• `/reschedule_with_notify 15 2025-08-15 18:00`\n" +
			"• `/reschedule_with_notify 22 2025-08-16 19:30`\n\n" +
			"**Что произойдет:**\n" +
			"• Перенос урока на новое время\n" +
			"• Уведомления всех записанных студентов\n" +
			"• Обновление записей в базе\n\n" +
			"**См. также:**\n" +
			"• `/notify_students` - произвольные уведомления\n" +
			"• `/cancel_with_notification` - отмена с уведомлениями"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	// Получаем lesson_id и новое время
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}
	
	newTimeStr := args[2] + " " + args[3]
	newTime, err := time.Parse("2006-01-02 15:04", newTimeStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный формат времени. Используйте: YYYY-MM-DD HH:MM")
		return
	}
	
	// Проверяем, что новое время в будущем
	if newTime.Before(time.Now()) {
		sendMessage(bot, message.Chat.ID, "❌ Новое время должно быть в будущем")
		return
	}
	
	// Проверяем, существует ли урок
	var subjectName, teacherName string
	var startTime string
	var teacherID int
	err = db.QueryRow(`
		SELECT s.name, u.full_name, l.start_time::text, l.teacher_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1 AND l.soft_deleted = false`, lessonID).Scan(&subjectName, &teacherName, &startTime, &teacherID)
		
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска урока")
		return
	}
	
	// Проверяем права доступа (учитель может переносить только свои уроки)
	if role == "teacher" {
		var currentTeacherID int
		err = db.QueryRow(`
			SELECT t.id FROM teachers t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.tg_id = $1`, userID).Scan(&currentTeacherID)
		
		if err != nil || currentTeacherID != teacherID {
			sendMessage(bot, message.Chat.ID, "❌ Вы можете переносить только свои уроки")
			return
		}
	}
	
	// Переносим урок
	_, err = db.Exec(`
		UPDATE lessons 
		SET start_time = $1, updated_at = NOW() 
		WHERE id = $2`, newTime, lessonID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка переноса урока")
		return
	}
	
	// Отправляем уведомления студентам
	notificationText := "🔄 Урок перенесен на " + newTime.Format("02.01.2006 15:04")
	sentCount, failedCount := notifyStudentsOfLesson(bot, db, lessonID, notificationText, subjectName, teacherName, newTime.Format("2006-01-02 15:04:05"))
	
	// Отчет администратору
	resultText := "✅ **Урок перенесен**\n\n" +
		"📚 Урок: " + subjectName + "\n" +
		"👨‍🏫 Преподаватель: " + teacherName + "\n" +
		"🕐 Новое время: " + newTime.Format("02.01.2006 15:04") + "\n\n" +
		"📤 Уведомлений отправлено: " + strconv.Itoa(sentCount) + "\n" +
		"❌ Ошибок отправки: " + strconv.Itoa(failedCount)
	
	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Вспомогательная функция: отправка уведомлений студентам урока с retry-механизмом
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

// Вспомогательная функция: отмена урока с уведомлением
func cancelLessonWithNotification(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int, subjectName, teacherName, startTime, reason string, chatID int64) {
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, chatID, "❌ Ошибка базы данных")
		return
	}
	defer tx.Rollback()
	
	// Отменяем урок
	_, err = tx.Exec(`
		UPDATE lessons 
		SET soft_deleted = true, updated_at = NOW() 
		WHERE id = $1`, lessonID)
	
	if err != nil {
		sendMessage(bot, chatID, "❌ Ошибка отмены урока")
		return
	}
	
	// Отменяем записи
	_, err = tx.Exec(`
		UPDATE enrollments 
		SET status = 'cancelled', updated_at = NOW() 
		WHERE lesson_id = $1`, lessonID)
	
	if err != nil {
		sendMessage(bot, chatID, "❌ Ошибка отмены записей")
		return
	}
	
	// Очищаем лист ожидания
	_, err = tx.Exec(`
		DELETE FROM waitlist 
		WHERE lesson_id = $1`, lessonID)
	
	if err != nil {
		sendMessage(bot, chatID, "❌ Ошибка очистки листа ожидания")
		return
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, chatID, "❌ Ошибка сохранения данных")
		return
	}
	
	// Отправляем уведомления студентам
	notifyStudentsAboutCancellationWithReason(bot, db, lessonID, subjectName, teacherName, startTime, reason)
	
	// Отчет администратору
	resultText := "✅ **Урок отменен**\n\n" +
		"📚 Урок: " + subjectName + " (" + startTime[:16] + ")\n" +
		"👨‍🏫 Преподаватель: " + teacherName + "\n" +
		"📝 Причина: " + reason + "\n\n" +
		"📢 Все записанные студенты получили уведомления"
	
	msg := tgbotapi.NewMessage(chatID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Уведомление студентов об отмене урока с причиной (расширенная версия)
func notifyStudentsAboutCancellationWithReason(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int, subjectName, teacherName, startTime, reason string) {
	rows, err := db.Query(`
		SELECT u.tg_id, u.full_name
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE e.lesson_id = $1`, lessonID)
		
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var studentTelegramID int64
		var studentName string
		
		if err := rows.Scan(&studentTelegramID, &studentName); err != nil {
			continue
		}
		
		notificationText := "❌ **Отмена урока**\n\n" +
			"📚 Предмет: " + subjectName + "\n" +
			"👨‍🏫 Преподаватель: " + teacherName + "\n" +
			"📅 Время: " + startTime[:16] + "\n\n" +
			"📝 Причина отмены: " + reason + "\n\n" +
			"💔 Приносим извинения за неудобства.\n" +
			"🔄 Вы можете записаться на другие уроки командой /schedule"
		
		msg := tgbotapi.NewMessage(studentTelegramID, notificationText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}
