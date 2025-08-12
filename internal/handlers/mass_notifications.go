package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Массовые уведомления всем пользователям (отсутствующая команда SuperUser)
func handleNotifyAllCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для массовых уведомлений")
		return
	}

	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "📢 **Массовые уведомления**\n\n" +
			"**Формат:** `/notify_all <сообщение>`\n\n" +
			"**Примеры:**\n" +
			"• `/notify_all Новый курс по веб-разработке!`\n" +
			"• `/notify_all Технические работы 15.08 с 20:00 до 22:00`\n\n" +
			"**Что произойдет:**\n" +
			"• Уведомление всех активных пользователей\n" +
			"• Логирование действия\n" +
			"• Отчет о результатах\n\n" +
			"**См. также:**\n" +
			"• `/notify_students` - уведомления студентов конкретного урока\n" +
			"• `/remind_all` - напоминания о предстоящих уроках"

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}

	// Получаем текст уведомления
	notificationText := strings.Join(args[1:], " ")

	// Получаем всех активных пользователей
	rows, err := db.Query(`
		SELECT tg_id, full_name, role 
		FROM users 
		WHERE is_active = true AND tg_id IS NOT NULL
		ORDER BY role, full_name`)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка пользователей")
		return
	}
	defer rows.Close()

	var users []struct {
		tgID  int64
		name  string
		role  string
	}

	for rows.Next() {
		var user struct {
			tgID  int64
			name  string
			role  string
		}
		if err := rows.Scan(&user.tgID, &user.name, &user.role); err != nil {
			continue
		}
		users = append(users, user)
	}

	if len(users) == 0 {
		sendMessage(bot, message.Chat.ID, "❌ Нет активных пользователей для уведомления")
		return
	}

	// Формируем сообщение
	messageText := fmt.Sprintf("📢 **Массовое уведомление**\n\n%s", notificationText)

	// Отправляем уведомления
	sentCount := 0
	failedCount := 0
	studentsCount := 0
	teachersCount := 0
	adminsCount := 0

	for _, user := range users {
		msg := tgbotapi.NewMessage(user.tgID, messageText)
		msg.ParseMode = "Markdown"
		
		if _, err := bot.Send(msg); err != nil {
			failedCount++
		} else {
			sentCount++
			switch user.role {
			case "student":
				studentsCount++
			case "teacher":
				teachersCount++
			case "superuser":
				adminsCount++
			}
		}
	}

	// Логируем массовое уведомление
	LogSystemAction(db, "mass_notification_sent", fmt.Sprintf("Массовое уведомление: '%s', отправлено: %d, ошибок: %d", notificationText[:50], sentCount, failedCount))

	// Отчет администратору
	resultText := "✅ **Массовое уведомление отправлено**\n\n" +
		"📢 Сообщение: " + notificationText + "\n\n" +
		"📊 Статистика:\n" +
		"• 📤 Отправлено: " + strconv.Itoa(sentCount) + "\n" +
		"• ❌ Ошибок: " + strconv.Itoa(failedCount) + "\n\n" +
		"👥 По ролям:\n" +
		"• 👨‍🎓 Студенты: " + strconv.Itoa(studentsCount) + "\n" +
		"• 👨‍🏫 Преподаватели: " + strconv.Itoa(teachersCount) + "\n" +
		"• 👑 Администраторы: " + strconv.Itoa(adminsCount)

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Напоминания о предстоящих уроках (отсутствующая команда SuperUser)
func handleRemindAllCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отправки напоминаний")
		return
	}

	// Парсинг сообщения
	args := strings.Fields(message.Text)
	hoursAhead := 24 // по умолчанию напоминаем за 24 часа

	if len(args) >= 2 {
		if parsedHours, err := strconv.Atoi(args[1]); err == nil && parsedHours > 0 && parsedHours <= 168 {
			hoursAhead = parsedHours
		}
	}

	// Получаем предстоящие уроки
	rows, err := db.Query(`
		SELECT 
			l.id,
			s.name as subject_name,
			u.full_name as teacher_name,
			l.start_time,
			l.duration_minutes,
			COUNT(e.id) as enrolled_count,
			l.max_students
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.soft_deleted = false 
		AND l.start_time > NOW() 
		AND l.start_time <= NOW() + INTERVAL '1 hour' * $1
		GROUP BY l.id, s.name, u.full_name, l.start_time, l.duration_minutes, l.max_students
		ORDER BY l.start_time`, hoursAhead)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения предстоящих уроков")
		return
	}
	defer rows.Close()

	var lessons []struct {
		id            int
		subjectName   string
		teacherName   string
		startTime     string
		duration      int
		enrolledCount int
		maxStudents   int
	}

	for rows.Next() {
		var lesson struct {
			id            int
			subjectName   string
			teacherName   string
			startTime     string
			duration      int
			enrolledCount int
			maxStudents   int
		}
		if err := rows.Scan(&lesson.id, &lesson.subjectName, &lesson.teacherName, &lesson.startTime, &lesson.duration, &lesson.enrolledCount, &lesson.maxStudents); err != nil {
			continue
		}
		lessons = append(lessons, lesson)
	}

	if len(lessons) == 0 {
		sendMessage(bot, message.Chat.ID, fmt.Sprintf("📅 Нет предстоящих уроков в ближайшие %d часов", hoursAhead))
		return
	}

	// Отправляем напоминания для каждого урока
	totalSent := 0
	totalFailed := 0

	for _, lesson := range lessons {
		// Получаем студентов урока
		studentRows, err := db.Query(`
			SELECT u.tg_id, u.full_name
			FROM enrollments e
			JOIN students s ON e.student_id = s.id
			JOIN users u ON s.user_id = u.id
			WHERE e.lesson_id = $1 AND e.status = 'enrolled'`, lesson.id)
		
		if err != nil {
			continue
		}

		reminderText := fmt.Sprintf("⏰ **Напоминание о уроке**\n\n"+
			"📚 Урок: %s\n"+
			"👨‍🏫 Преподаватель: %s\n"+
			"⏰ Время: %s\n"+
			"⏱️ Длительность: %d минут\n"+
			"👥 Записано: %d/%d\n\n"+
			"Не забудьте подготовиться к уроку!", 
			lesson.subjectName, lesson.teacherName, lesson.startTime[:16], 
			lesson.duration, lesson.enrolledCount, lesson.maxStudents)

		// Отправляем напоминания студентам
		for studentRows.Next() {
			var tgID int64
			var fullName string
			if err := studentRows.Scan(&tgID, &fullName); err != nil {
				continue
			}

			msg := tgbotapi.NewMessage(tgID, reminderText)
			msg.ParseMode = "Markdown"
			if _, err := bot.Send(msg); err != nil {
				totalFailed++
			} else {
				totalSent++
			}
		}
		studentRows.Close()
	}

	// Логируем отправку напоминаний
	LogSystemAction(db, "reminders_sent", fmt.Sprintf("Напоминания за %d часов, уроков: %d, отправлено: %d, ошибок: %d", hoursAhead, len(lessons), totalSent, totalFailed))

	// Отчет администратору
	resultText := fmt.Sprintf("✅ **Напоминания отправлены**\n\n"+
		"⏰ Период: ближайшие %d часов\n"+
		"📅 Уроков: %d\n"+
		"📤 Напоминаний отправлено: %d\n"+
		"❌ Ошибок: %d", hoursAhead, len(lessons), totalSent, totalFailed)

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
