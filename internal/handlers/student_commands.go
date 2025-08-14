package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Команда записи на урок (для inline-кнопок)
func handleEnrollCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем, зарегистрирован ли пользователь
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Вы не зарегистрированы. Используйте /register")
		return
	}

	// Парсим команду
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		sendMessage(bot, message.Chat.ID, "❌ Укажите ID урока: /enroll <lesson_id>")
		return
	}

	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}

	// Проверяем, существует ли урок
	var subjectName, teacherName string
	var startTime string
	var maxStudents int
	err = db.QueryRow(`
		SELECT s.name, u.full_name, l.start_time::text, l.max_students
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1 AND l.soft_deleted = false`, lessonID).Scan(&subjectName, &teacherName, &startTime, &maxStudents)

	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска урока")
		return
	}

	// Получаем ID студента
	var studentID int
	err = db.QueryRow("SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&studentID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Вы не являетесь студентом")
		return
	}

	// Проверяем, не записан ли уже студент на этот урок
	var existingEnrollment int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND lesson_id = $2", studentID, lessonID).Scan(&existingEnrollment)
	if err == nil && existingEnrollment > 0 {
		sendMessage(bot, message.Chat.ID, "❌ Вы уже записаны на этот урок")
		return
	}

	// Проверяем количество записанных студентов
	var enrolledCount int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'enrolled'", lessonID).Scan(&enrolledCount)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка проверки количества записанных")
		return
	}

	if enrolledCount >= maxStudents {
		// Проверяем, есть ли место в листе ожидания
		var waitlistPosition int
		err = db.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM waitlist WHERE lesson_id = $1", lessonID).Scan(&waitlistPosition)
		if err != nil {
			waitlistPosition = 1
		}

		// Добавляем в лист ожидания
		_, err = db.Exec("INSERT INTO waitlist (student_id, lesson_id, position) VALUES ($1, $2, $3)", studentID, lessonID, waitlistPosition)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка добавления в лист ожидания")
			return
		}

		// Логируем добавление в лист ожидания
		LogUserAction(db, "waitlist_added", userID, fmt.Sprintf("Урок %d (%s), позиция: %d", lessonID, subjectName, waitlistPosition))

		resultText := fmt.Sprintf("⏳ **Добавлено в лист ожидания**\n\n"+
			"📚 Урок: %s\n"+
			"👨‍🏫 Преподаватель: %s\n"+
			"⏰ Время: %s\n"+
			"📋 Позиция в очереди: %d\n\n"+
			"Вы будете уведомлены, если освободится место.", subjectName, teacherName, startTime[:16], waitlistPosition)

		msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}

	// Записываем на урок
	_, err = db.Exec("INSERT INTO enrollments (student_id, lesson_id, status) VALUES ($1, $2, 'enrolled')", studentID, lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка записи на урок")
		return
	}

	// Логируем запись на урок
	LogUserAction(db, "lesson_enrolled", userID, fmt.Sprintf("Урок %d (%s)", lessonID, subjectName))

	resultText := fmt.Sprintf("✅ **Вы записаны на урок!**\n\n"+
		"📚 Урок: %s\n"+
		"👨‍🏫 Преподаватель: %s\n"+
		"⏰ Время: %s\n"+
		"👥 Записано: %d/%d\n\n"+
		"Не забудьте подготовиться к уроку!", subjectName, teacherName, startTime[:16], enrolledCount+1, maxStudents)

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Команда отписки от урока
func handleUnenrollCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем, зарегистрирован ли пользователь
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Вы не зарегистрированы. Используйте /register")
		return
	}

	// Парсим команду
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		sendMessage(bot, message.Chat.ID, "❌ Укажите ID урока: /unenroll <lesson_id>")
		return
	}

	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
		return
	}

	// Получаем ID студента
	var studentID int
	err = db.QueryRow("SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&studentID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Вы не являетесь студентом")
		return
	}

	// Проверяем, записан ли студент на этот урок
	var enrollmentID int
	err = db.QueryRow("SELECT id FROM enrollments WHERE student_id = $1 AND lesson_id = $2 AND status = 'enrolled'", studentID, lessonID).Scan(&enrollmentID)
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Вы не записаны на этот урок")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка проверки записи")
		return
	}

	// Отписываем от урока
	_, err = db.Exec("UPDATE enrollments SET status = 'cancelled' WHERE id = $1", enrollmentID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка отписки от урока")
		return
	}

	// Удаляем из листа ожидания, если там есть
	_, err = db.Exec("DELETE FROM waitlist WHERE student_id = $1 AND lesson_id = $2", studentID, lessonID)
	// Игнорируем ошибку, если записи в листе ожидания не было

	// Получаем информацию об уроке для уведомления
	var subjectName, teacherName string
	var startTime string
	err = db.QueryRow(`
		SELECT s.name, u.full_name, l.start_time::text
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE l.id = $1`, lessonID).Scan(&subjectName, &teacherName, &startTime)

	// Логируем отписку от урока
	LogUserAction(db, "lesson_unenrolled", userID, fmt.Sprintf("Урок %d (%s)", lessonID, subjectName))

	resultText := fmt.Sprintf("❌ **Вы отписались от урока**\n\n"+
		"📚 Урок: %s\n"+
		"👨‍🏫 Преподаватель: %s\n"+
		"⏰ Время: %s\n\n"+
		"Место освобождено для других студентов.", subjectName, teacherName, startTime[:16])

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
