package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Восстановление урока
func handleRestoreLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для восстановления уроков")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🔄 **Восстановление урока**\n\n" +
			"**Формат:** `/restore_lesson <lesson_id>`\n\n" +
			"**Пример:** `/restore_lesson 15`\n\n" +
			"**Что произойдет:**\n" +
			"• Восстановление урока\n" +
			"• Восстановление всех записей студентов\n" +
			"• Уведомления студентов о восстановлении\n\n" +
			"**Внимание:** Восстанавливаются только отмененные уроки!"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}
	
	// Получаем информацию об уроке
	var lessonData struct {
		ID          int
		SubjectName string
		TeacherName string
		StartTime   string
		IsActive    bool
		TeacherID   int
	}
	err = db.QueryRow(`
		SELECT l.id, s.name, u.full_name, l.start_time::text, l.is_active, l.teacher_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1`, lessonID).Scan(&lessonData.ID, &lessonData.SubjectName, &lessonData.TeacherName, &lessonData.StartTime, &lessonData.IsActive, &lessonData.TeacherID)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска урока")
		return
	}
	
	// Проверяем, что урок отменен
	if lessonData.IsActive {
		sendMessage(bot, message.Chat.ID, "❌ Урок уже активен и не требует восстановления")
		return
	}
	
	// Проверяем, что преподаватель активен
	var teacherActive bool
	err = db.QueryRow(`
		SELECT t.soft_deleted = false AND u.is_active = true
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE t.id = $1`, lessonData.TeacherID).Scan(&teacherActive)
	
	if err != nil || !teacherActive {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель неактивен. Сначала восстановите преподавателя.")
		return
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка базы данных")
		return
	}
	defer tx.Rollback()
	
	// Восстанавливаем урок
	_, err = tx.Exec(`
		UPDATE lessons 
		SET soft_deleted = false, updated_at = NOW() 
		WHERE id = $1`, lessonID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления урока")
		return
	}
	
	// Восстанавливаем записи студентов
	_, err = tx.Exec(`
		UPDATE enrollments 
		SET status = 'enrolled', updated_at = NOW() 
		WHERE lesson_id = $1 AND status = 'cancelled'`, lessonID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления записей")
		return
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения данных")
		return
	}
	
	// Отправляем уведомления студентам
	sent, failed := notifyPreviouslyEnrolledStudents(bot, db, lessonID, lessonData)
	
	// Отчет о восстановлении
	resultText := "✅ **Урок восстановлен**\n\n" +
		"📚 Урок: " + lessonData.SubjectName + " (" + lessonData.StartTime[:16] + ")\n" +
		"👨‍🏫 Преподаватель: " + lessonData.TeacherName + "\n" +
		"📊 **Результаты:**\n" +
		"• Восстановлен урок\n" +
		"• Восстановлены записи студентов\n" +
		"• Уведомлено студентов: " + strconv.Itoa(sent) + "\n" +
		"• Ошибок отправки: " + strconv.Itoa(failed) + "\n\n" +
		"🎉 Урок снова доступен для записи!"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Просмотр всех студентов (обновленная версия)
func handleMyStudentsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра списка студентов")
		return
	}
	
	// Получаем список всех студентов с их записями
	rows, err := db.Query(`
		SELECT s.id, u.full_name, u.tg_id, u.is_active,
			COUNT(e.id) as total_enrollments,
			COUNT(CASE WHEN e.status = 'enrolled' THEN 1 END) as active_enrollments,
			COUNT(CASE WHEN e.status = 'cancelled' THEN 1 END) as cancelled_enrollments
		FROM students s
		JOIN users u ON s.user_id = u.id
		LEFT JOIN enrollments e ON s.id = e.student_id
		WHERE s.soft_deleted = false
		GROUP BY s.id, u.full_name, u.tg_id, u.is_active
		ORDER BY u.full_name`)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка студентов")
		return
	}
	defer rows.Close()
	
	var studentsText strings.Builder
	studentsText.WriteString("👨‍🎓 **Список всех студентов**\n\n")
	
	var studentCount int
	for rows.Next() {
		var id int
		var fullName, tgID string
		var isActive bool
		var totalEnrollments, activeEnrollments, cancelledEnrollments int
		
		if err := rows.Scan(&id, &fullName, &tgID, &isActive, &totalEnrollments, &activeEnrollments, &cancelledEnrollments); err != nil {
			continue
		}
		
		status := "✅ Активен"
		if !isActive {
			status = "❌ Неактивен"
		}
		
		studentsText.WriteString(fmt.Sprintf("**%d.** %s\n", id, fullName))
		studentsText.WriteString(fmt.Sprintf("   🆔 ID: %s\n", tgID))
		studentsText.WriteString(fmt.Sprintf("   📊 Статус: %s\n", status))
		studentsText.WriteString(fmt.Sprintf("   📚 Записей: %d (активных: %d, отмененных: %d)\n\n", totalEnrollments, activeEnrollments, cancelledEnrollments))
		
		studentCount++
	}
	
	if studentCount == 0 {
		studentsText.WriteString("Пока нет зарегистрированных студентов")
	} else {
		studentsText.WriteString(fmt.Sprintf("**Всего студентов:** %d", studentCount))
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, studentsText.String())
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// notifyPreviouslyEnrolledStudents - уведомление студентов о восстановлении урока
func notifyPreviouslyEnrolledStudents(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int, lessonData struct {
	ID          int
	SubjectName string
	TeacherName string
	StartTime   string
	IsActive    bool
	TeacherID   int
}) (int, int) {
	// Получаем всех записанных студентов
	query := `
		SELECT u.tg_id, u.full_name
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE e.lesson_id = $1 AND e.status = 'enrolled'`
	
	rows, err := db.Query(query, lessonID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	
	sent := 0
	failed := 0
	
	for rows.Next() {
		var studentTelegramID int64
		var studentName string
		
		if err := rows.Scan(&studentTelegramID, &studentName); err != nil {
			failed++
			continue
		}
		
		// Формируем уведомление
		notificationText := "🎉 **УРОК ВОССТАНОВЛЕН!**\n\n" +
			"📚 Предмет: " + lessonData.SubjectName + "\n" +
			"👨‍🏫 Преподаватель: " + lessonData.TeacherName + "\n" +
			"📅 Время: " + lessonData.StartTime[:16] + "\n\n" +
			"✅ Ваша запись остается активной - урок состоится!\n" +
			"🎯 Ждем вас на занятии!"
		
		msg := tgbotapi.NewMessage(studentTelegramID, notificationText)
		msg.ParseMode = "Markdown"
		
		// Retry механизм (3 попытки)
		success := false
		for attempt := 0; attempt < 3; attempt++ {
			if _, err := bot.Send(msg); err == nil {
				success = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		
		if success {
			sent++
		} else {
			failed++
		}
	}
	
	return sent, failed
}
