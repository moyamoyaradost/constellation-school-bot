package handlers

import (
	"database/sql"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Проверка валидности урока (не отменен, не прошел)
func isLessonValid(db *sql.DB, lessonID int) bool {
	var status string
	var startTime time.Time
	
	err := db.QueryRow(`
		SELECT status, start_time 
		FROM lessons 
		WHERE id = $1 AND soft_deleted = false`, lessonID).Scan(&status, &startTime)
	
	if err != nil {
		return false
	}
	
	// Урок валиден, если активен и еще не прошел
	return status == "active" && startTime.After(time.Now())
}

// Получение student_id по telegram user_id
func getStudentID(db *sql.DB, telegramUserID int) (int, error) {
	var studentID int
	err := db.QueryRow(`
		SELECT s.id 
		FROM students s 
		JOIN users u ON s.user_id = u.id 
		WHERE u.tg_id = $1`, int64(telegramUserID)).Scan(&studentID)
	return studentID, err
}

// Получение teacher_id по telegram user_id
func getTeacherID(db *sql.DB, telegramUserID int) (int, error) {
	var teacherID int
	err := db.QueryRow(`
		SELECT t.id 
		FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE u.tg_id = $1`, int64(telegramUserID)).Scan(&teacherID)
	return teacherID, err
}

// Проверка, записан ли студент на урок
func isStudentEnrolled(db *sql.DB, studentID, lessonID int) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM enrollments 
		WHERE student_id = $1 AND lesson_id = $2 AND status = 'enrolled'`, 
		studentID, lessonID).Scan(&count)
	
	return err == nil && count > 0
}

// Проверка наличия свободных мест
func hasAvailableSpots(db *sql.DB, lessonID int) bool {
	var maxStudents, enrolledCount int
	
	err := db.QueryRow(`
		SELECT l.max_students, 
			COALESCE(COUNT(e.id), 0) as enrolled_count
		FROM lessons l
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.id = $1 AND l.soft_deleted = false
		GROUP BY l.id, l.max_students`, lessonID).Scan(&maxStudents, &enrolledCount)
		
	if err != nil {
		return false
	}
	
	return enrolledCount < maxStudents
}

// Запись студента на урок в БД
func enrollStudentInDB(db *sql.DB, studentID, lessonID int) error {
	_, err := db.Exec(`
		INSERT INTO enrollments (student_id, lesson_id, status, enrolled_at) 
		VALUES ($1, $2, 'enrolled', NOW())
		ON CONFLICT (student_id, lesson_id) DO NOTHING`, 
		studentID, lessonID)
	return err
}

// Отмена записи студента
func unenrollStudentFromDB(db *sql.DB, studentID, lessonID int) error {
	_, err := db.Exec(`
		UPDATE enrollments 
		SET status = 'cancelled' 
		WHERE student_id = $1 AND lesson_id = $2 AND status = 'enrolled'`, 
		studentID, lessonID)
	return err
}

// Добавление в лист ожидания
func addToWaitlist(db *sql.DB, studentID, lessonID int) error {
	_, err := db.Exec(`
		INSERT INTO waitlist (student_id, lesson_id, created_at) 
		VALUES ($1, $2, NOW())
		ON CONFLICT (student_id, lesson_id) DO NOTHING`, 
		studentID, lessonID)
	return err
}

// Отмена урока в БД
func cancelLessonInDB(db *sql.DB, lessonID int) error {
	_, err := db.Exec(`
		UPDATE lessons 
		SET status = 'cancelled' 
		WHERE id = $1`, lessonID)
	return err
}

// Проверка, что урок принадлежит учителю
func isTeacherLesson(db *sql.DB, teacherID, lessonID int) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM lessons 
		WHERE id = $1 AND teacher_id = $2`, lessonID, teacherID).Scan(&count)
	
	return err == nil && count > 0
}

// Уведомление студентов об отмене урока
func notifyStudentsAboutCancellation(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int) {
	rows, err := db.Query(`
		SELECT u.tg_id, u.full_name
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE e.lesson_id = $1 AND e.status = 'enrolled'`, lessonID)
		
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var telegramID int64
		var fullName string
		if err := rows.Scan(&telegramID, &fullName); err != nil {
			continue
		}
		
		message := "❌ **Уведомление об отмене**\n\nВаш урок был отменен преподавателем. Приносим извинения за неудобства."
		sendMessage(bot, telegramID, message)
	}
}

// Уведомление следующего в листе ожидания
func notifyNextInWaitlist(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int) {
	// Получаем следующего в очереди
	var telegramID int64
	var studentID int
	
	err := db.QueryRow(`
		SELECT u.tg_id, w.student_id
		FROM waitlist w
		JOIN students s ON w.student_id = s.id
		JOIN users u ON s.user_id = u.id
		WHERE w.lesson_id = $1
		ORDER BY w.created_at
		LIMIT 1`, lessonID).Scan(&telegramID, &studentID)
		
	if err != nil {
		return // Никого нет в листе ожидания
	}
	
	// Автоматически записываем на урок
	err = enrollStudentInDB(db, studentID, lessonID)
	if err != nil {
		return
	}
	
	// Удаляем из листа ожидания
	db.Exec(`DELETE FROM waitlist WHERE student_id = $1 AND lesson_id = $2`, studentID, lessonID)
	
	// Уведомляем
	message := "🎉 **Освободилось место!**\n\nВы автоматически записаны на урок из листа ожидания."
	sendMessage(bot, telegramID, message)
}

// Получение информации об уроке
func getLessonInfo(db *sql.DB, lessonID int) (string, error) {
	var startTime time.Time
	var subjectName, teacherName string
	var maxStudents, enrolledCount int
	
	err := db.QueryRow(`
		SELECT l.start_time, s.name, u.full_name, l.max_students,
			COUNT(e.id) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.id = $1
		GROUP BY l.id, l.start_time, s.name, u.full_name, l.max_students`, lessonID).Scan(
		&startTime, &subjectName, &teacherName, &maxStudents, &enrolledCount)
		
	if err != nil {
		return "", err
	}
	
	freeSpots := maxStudents - enrolledCount
	status := fmt.Sprintf("(%d/%d мест)", enrolledCount, maxStudents)
	if freeSpots == 0 {
		status += " 🔴"
	} else if freeSpots <= 2 {
		status += " 🟡" 
	} else {
		status += " 🟢"
	}
	
	return fmt.Sprintf("📅 **%s**\n📚 %s\n👨‍🏫 %s\n%s", 
		startTime.Format("02.01.2006 15:04"), subjectName, teacherName, status), nil
}

// Создание урока с кнопками
func getLessonWithButtons(db *sql.DB, lessonID int, userRole int) (string, tgbotapi.InlineKeyboardMarkup) {
	lessonText, err := getLessonInfo(db, lessonID)
	if err != nil {
		return "❌ Ошибка загрузки информации об уроке", tgbotapi.NewInlineKeyboardMarkup()
	}
	
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Кнопки для студентов
	if userRole == 0 || userRole == 1 { // 0=любой, 1=студент
		if hasAvailableSpots(db, lessonID) {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Записаться", fmt.Sprintf("enroll_lesson_%d", lessonID)),
			))
		} else {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏳ В лист ожидания", fmt.Sprintf("waitlist_lesson_%d", lessonID)),
			))
		}
		
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить запись", fmt.Sprintf("unenroll_lesson_%d", lessonID)),
		))
	}
	
	// Кнопки для учителей/админов
	if userRole == 0 || userRole == 2 || userRole == 3 { // 0=любой, 2=учитель, 3=админ
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить урок", fmt.Sprintf("cancel_lesson_%d", lessonID)),
		))
	}
	
	// Общие кнопки
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробнее", fmt.Sprintf("info_lesson_%d", lessonID)),
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", fmt.Sprintf("schedule_lesson_%d", lessonID)),
	))
	
	return lessonText, tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// Отправка расписания с кнопками
func sendScheduleWithButtons(bot *tgbotapi.BotAPI, chatID int64, db *sql.DB, userRole string) {
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
		ORDER BY l.start_time LIMIT 5`)
		
	if err != nil {
		sendMessage(bot, chatID, "❌ Ошибка загрузки расписания")
		return
	}
	defer rows.Close()

	lessons := []struct {
		ID            int
		StartTime     time.Time
		SubjectName   string
		TeacherName   string
		MaxStudents   int
		EnrolledCount int
	}{}

	for rows.Next() {
		var lesson struct {
			ID            int
			StartTime     time.Time
			SubjectName   string
			TeacherName   string
			MaxStudents   int
			EnrolledCount int
		}
		
		if err := rows.Scan(&lesson.ID, &lesson.StartTime, &lesson.SubjectName, 
			&lesson.TeacherName, &lesson.MaxStudents, &lesson.EnrolledCount); err != nil {
			continue
		}
		lessons = append(lessons, lesson)
	}

	if len(lessons) == 0 {
		sendMessage(bot, chatID, "📅 На ближайшую неделю уроков не запланировано")
		return
	}

	// Отправляем каждый урок отдельным сообщением с кнопками
	for _, lesson := range lessons {
		freeSpots := lesson.MaxStudents - lesson.EnrolledCount
		status := fmt.Sprintf("(%d/%d мест)", lesson.EnrolledCount, lesson.MaxStudents)
		if freeSpots == 0 {
			status += " 🔴"
		} else if freeSpots <= 2 {
			status += " 🟡" 
		} else {
			status += " 🟢"
		}
		
		text := fmt.Sprintf("� **%s** (ID: #%d)\n� %s\n👨‍🏫 %s\n%s", 
			lesson.SubjectName, lesson.ID,
			lesson.StartTime.Format("02.01.2006 15:04"), 
			lesson.TeacherName, status)

		var buttons [][]tgbotapi.InlineKeyboardButton
		
		// Кнопки в зависимости от роли
		if userRole == "student" {
			if freeSpots > 0 {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Записаться", fmt.Sprintf("enroll_lesson_%d", lesson.ID)),
				))
			} else {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⏳ В лист ожидания", fmt.Sprintf("waitlist_lesson_%d", lesson.ID)),
				))
			}
			
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отменить запись", fmt.Sprintf("unenroll_lesson_%d", lesson.ID)),
			))
		}
		
		if userRole == "teacher" || userRole == "admin" {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить урок", fmt.Sprintf("cancel_lesson_%d", lesson.ID)),
			))
		}
		
		// Общие кнопки
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробнее", fmt.Sprintf("info_lesson_%d", lesson.ID)),
		))

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		bot.Send(msg)
	}
}
