package handlers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Глобальный rate limiter (инициализируется в main)
var GlobalRateLimiter *RateLimiter

// Обработчик команд студентов
func handleStudentCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	switch message.Command() {
	case "subjects":
		handleSubjectsCommand(bot, message, db)
	case "schedule":
		handleScheduleCommand(bot, message, db)
	case "enroll":
		// Применяем rate limiting для записи на урок
		lessonID := ExtractLessonIDFromMessage(message)
		handleEnrollWithRateLimit(bot, message, db, lessonID)
	case "waitlist":
		// Применяем rate limiting для записи в очередь
		lessonID := ExtractLessonIDFromMessage(message)
		handleWaitlistWithRateLimit(bot, message, db, lessonID)
	case "my_lessons":
		handleMyLessonsCommand(bot, message, db)
	}
}

// Показ доступных предметов
func handleSubjectsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	rows, err := db.Query("SELECT name, description, category FROM subjects WHERE is_active = true ORDER BY name")
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки предметов")
		return
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var name, description, category string
		if err := rows.Scan(&name, &description, &category); err != nil {
			continue
		}
		subjects = append(subjects, fmt.Sprintf("📚 **%s** (%s)\n%s", name, category, description))
	}

	if len(subjects) == 0 {
		sendMessage(bot, message.Chat.ID, "📚 Пока нет доступных предметов")
		return
	}

	text := "🎯 **Доступные предметы:**\n\n" + strings.Join(subjects, "\n\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Расписание уроков на неделю с кнопками
func handleScheduleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Получаем роль пользователя
	userRole, err := getUserRole(db, message.From.ID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка определения роли пользователя")
		return
	}

	// Используем новую функцию с кнопками
	sendScheduleWithButtons(bot, message.Chat.ID, db, userRole)
}

// Запись на урок (используется функция из student_commands.go)

// Лист ожидания - показ переполненных уроков
func handleWaitlistCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Показываем уроки, где нет мест (для добавления в лист ожидания)
	rows, err := db.Query(`
		SELECT l.id, l.start_time, s.name, u.full_name, l.max_students,
			COUNT(e.id) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.start_time > NOW() AND l.start_time < NOW() + INTERVAL '7 days'
			AND l.soft_deleted = false AND l.status = 'active'
		GROUP BY l.id, l.start_time, s.name, u.full_name, l.max_students
		HAVING COUNT(e.id) >= l.max_students
		ORDER BY l.start_time LIMIT 5`)
		
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки переполненных уроков")
		return
	}
	defer rows.Close()

	hasLessons := false
	for rows.Next() {
		hasLessons = true
		var lessonID int
		var startTime time.Time
		var subjectName, teacherName string
		var maxStudents, enrolledCount int
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName, &teacherName, &maxStudents, &enrolledCount); err != nil {
			continue
		}
		
		text := fmt.Sprintf("📅 **%s**\n📚 %s\n👨‍🏫 %s\n🔴 Мест нет (%d/%d)", 
			startTime.Format("02.01.2006 15:04"), subjectName, teacherName, enrolledCount, maxStudents)

		// Кнопка для добавления в лист ожидания
		buttons := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏳ В лист ожидания", fmt.Sprintf("waitlist_lesson_%d", lessonID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробнее", fmt.Sprintf("info_lesson_%d", lessonID)),
			),
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		bot.Send(msg)
	}

	if !hasLessons {
		sendMessage(bot, message.Chat.ID, "⏳ Все уроки на ближайшую неделю имеют свободные места!\n\nИспользуйте /enroll для записи")
	}
}

// Мои уроки с кнопками управления
func handleMyLessonsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Получаем student_id
	studentID, err := getStudentID(db, int(message.From.ID))
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка: вы не зарегистрированы как студент")
		return
	}

	// Запрос активных записей студента
	rows, err := db.Query(`
		SELECT l.id, l.start_time, s.name, u.full_name, e.status
		FROM enrollments e
		JOIN lessons l ON e.lesson_id = l.id
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE e.student_id = $1 AND e.status = 'enrolled' 
			AND l.start_time > NOW() AND l.soft_deleted = false
		ORDER BY l.start_time`, studentID)
		
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки ваших уроков")
		return
	}
	defer rows.Close()

	hasLessons := false
	for rows.Next() {
		hasLessons = true
		var lessonID int
		var startTime time.Time
		var subjectName, teacherName, status string
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName, &teacherName, &status); err != nil {
			continue
		}
		
		text := fmt.Sprintf("📅 **%s**\n📚 %s\n👨‍🏫 %s\n✅ Вы записаны", 
			startTime.Format("02.01.2006 15:04"), subjectName, teacherName)

		// Кнопки управления записью
		buttons := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отменить запись", fmt.Sprintf("unenroll_lesson_%d", lessonID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробнее", fmt.Sprintf("info_lesson_%d", lessonID)),
			),
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		bot.Send(msg)
	}

	if !hasLessons {
		sendMessage(bot, message.Chat.ID, "📚 У вас пока нет записей на уроки\n\nИспользуйте /enroll для записи на урок")
	}

	// Дополнительно показываем лист ожидания
	rows2, err := db.Query(`
		SELECT l.id, l.start_time, s.name, u.full_name
		FROM waitlist w
		JOIN lessons l ON w.lesson_id = l.id
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		WHERE w.student_id = $1 AND l.start_time > NOW() 
			AND l.soft_deleted = false
		ORDER BY l.start_time`, studentID)
		
	if err == nil {
		defer rows2.Close()
		
		waitlistCount := 0
		for rows2.Next() {
			if waitlistCount == 0 {
				sendMessage(bot, message.Chat.ID, "⏳ **Лист ожидания:**")
			}
			waitlistCount++
			
			var lessonID int
			var startTime time.Time
			var subjectName, teacherName string
			
			if err := rows2.Scan(&lessonID, &startTime, &subjectName, &teacherName); err != nil {
				continue
			}
			
			text := fmt.Sprintf("📅 %s\n📚 %s\n👨‍🏫 %s\n⏳ В очереди", 
				startTime.Format("02.01.2006 15:04"), subjectName, teacherName)
			
			sendMessage(bot, message.Chat.ID, text)
		}
	}
}

// ========================= ИНТЕГРАЦИЯ RATE-LIMITING =========================

// handleEnrollWithRateLimit - запись на урок с rate limiting
func handleEnrollWithRateLimit(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, lessonID int) {
	userID := message.From.ID
	
	// Проверяем rate limiting
	if GlobalRateLimiter != nil {
		allowed, reason := GlobalRateLimiter.IsOperationAllowed(userID, OPERATION_ENROLL, lessonID)
		if !allowed {
			sendMessage(bot, message.Chat.ID, reason.Error())
			return
		}
		
		// Регистрируем начало операции
		if err := GlobalRateLimiter.StartOperation(userID, OPERATION_ENROLL, lessonID); err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Системная ошибка. Попробуйте позже.")
			return
		}
		
		// Выполняем операцию
		handleEnrollCommand(bot, message, db)
		
		// Завершаем операцию
		GlobalRateLimiter.FinishOperation(userID, OPERATION_ENROLL, lessonID)
	} else {
		// Fallback если rate limiter не инициализирован
		handleEnrollCommand(bot, message, db)
	}
}

// handleWaitlistWithRateLimit - запись в очередь с rate limiting  
func handleWaitlistWithRateLimit(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, lessonID int) {
	userID := message.From.ID
	
	// Проверяем rate limiting
	if GlobalRateLimiter != nil {
		allowed, reason := GlobalRateLimiter.IsOperationAllowed(userID, OPERATION_WAITLIST, lessonID)
		if !allowed {
			sendMessage(bot, message.Chat.ID, reason.Error())
			return
		}
		
		// Регистrируем начало операции
		if err := GlobalRateLimiter.StartOperation(userID, OPERATION_WAITLIST, lessonID); err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Системная ошибка. Попробуйте позже.")
			return
		}
		
		// Выполняем операцию
		handleWaitlistCommand(bot, message, db)
		
		// Завершаем операцию
		GlobalRateLimiter.FinishOperation(userID, OPERATION_WAITLIST, lessonID)
	} else {
		// Fallback если rate limiter не инициализирован
		handleWaitlistCommand(bot, message, db)
	}
}
