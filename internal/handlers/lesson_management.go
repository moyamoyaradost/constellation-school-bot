package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Удаление урока (отсутствующая команда SuperUser)
func handleDeleteLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для удаления уроков")
		return
	}

	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "❌ **Удаление урока**\n\n" +
			"**Формат:** `/delete_lesson <lesson_id>`\n\n" +
			"**Примеры:**\n" +
			"• `/delete_lesson 15`\n" +
			"• `/delete_lesson 22`\n\n" +
			"**Что произойдет:**\n" +
			"• Удаление урока (soft delete)\n" +
			"• Уведомления всех записанных студентов\n" +
			"• Очистка листа ожидания\n" +
			"• Логирование действия\n\n" +
			"**См. также:**\n" +
			"• `/restore_lesson` - восстановление урока\n" +
			"• `/cancel_with_notification` - отмена с уведомлением"

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}

	// Получаем lesson_id
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
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

	// Получаем список студентов для уведомления
	var studentIDs []int
	rows, err := db.Query(`
		SELECT DISTINCT u.id
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE e.lesson_id = $1 AND e.status = 'enrolled'`, lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка студентов")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var studentID int
		if err := rows.Scan(&studentID); err != nil {
			continue
		}
		studentIDs = append(studentIDs, studentID)
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
		return
	}
	defer tx.Rollback()

	// Удаляем урок (soft delete)
	_, err = tx.Exec("UPDATE lessons SET soft_deleted = true WHERE id = $1", lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка удаления урока")
		return
	}

	// Отменяем все записи на урок
	_, err = tx.Exec("UPDATE enrollments SET status = 'cancelled' WHERE lesson_id = $1", lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены записей")
		return
	}

	// Очищаем лист ожидания
	_, err = tx.Exec("DELETE FROM waitlist WHERE lesson_id = $1", lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка очистки листа ожидания")
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка фиксации транзакции")
		return
	}

	// Уведомляем студентов
	notificationText := fmt.Sprintf("❌ **Урок отменен**\n\n"+
		"📚 Урок: %s\n"+
		"👨‍🏫 Преподаватель: %s\n"+
		"⏰ Время: %s\n\n"+
		"Урок был удален администратором.", subjectName, teacherName, startTime[:16])

	sentCount := 0
	failedCount := 0

	for _, studentID := range studentIDs {
		// Получаем tg_id студента
		var tgID int64
		err := db.QueryRow("SELECT tg_id FROM users WHERE id = $1", studentID).Scan(&tgID)
		if err != nil {
			failedCount++
			continue
		}

		msg := tgbotapi.NewMessage(tgID, notificationText)
		msg.ParseMode = "Markdown"
		if _, err := bot.Send(msg); err != nil {
			failedCount++
		} else {
			sentCount++
		}
	}

	// Логируем удаление урока
	LogSystemAction(db, "lesson_deleted", fmt.Sprintf("Урок %d (%s) удален, уведомлено студентов: %d, ошибок: %d", lessonID, subjectName, sentCount, failedCount))

	// Отчет администратору
	resultText := "✅ **Урок удален**\n\n" +
		"📚 Урок: " + subjectName + " (" + startTime[:16] + ")\n" +
		"👨‍🏫 Преподаватель: " + teacherName + "\n\n" +
		"📤 Уведомлений отправлено: " + strconv.Itoa(sentCount) + "\n" +
		"❌ Ошибок отправки: " + strconv.Itoa(failedCount) + "\n\n" +
		"💾 Урок помечен как удаленный (soft delete)\n" +
		"📝 Записи отменены\n" +
		"🗑️ Лист ожидания очищен"

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
