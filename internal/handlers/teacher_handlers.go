package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Обработчик команд для преподавателей
func handleTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав преподавателя")
		return
	}
	
	// Маршрутизация команд преподавателя
	switch message.Command() {
	case "create_lesson":
		handleCreateLessonCommand(bot, message, db)
	case "reschedule_lesson":
		handleRescheduleLessonCommand(bot, message, db)
	case "cancel_lesson":
		handleCancelLessonCommand(bot, message, db)
	case "my_schedule":
		handleMyScheduleCommand(bot, message, db)
	case "my_students":
		handleTeacherStudentsCommand(bot, message, db)
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда преподавателя")
	}
}

// Создание урока с интерактивными кнопками
func handleCreateLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для создания уроков")
		return
	}
	
	// Если нет аргументов - показываем кнопки с предметами
	args := message.CommandArguments()
	if args == "" {
		showSubjectButtons(bot, message, db, "create")
		return
	}
	
	// Парсинг аргументов команды  
	argsList := strings.Fields(args)
	if len(argsList) < 3 {
		helpText := "📝 **Создание урока**\n\n" +
			"**Формат:** `/create_lesson <предмет> <дата> <время>`\n\n" +
			"**Примеры:**\n" +
			"• `/create_lesson \"3D-моделирование\" 16.08.2025 16:30`\n" +
			"• `/create_lesson Математика 20.08.2025 10:00`\n\n" +
			"💡 **Совет:** Используйте `/create_lesson` без параметров для выбора предмета кнопками!"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	subjectName := argsList[0]
	dateStr := argsList[1]
	timeStr := argsList[2]
	
	// Если предмет в кавычках, соберем полное название
	if strings.HasPrefix(args, "\"") {
		// Найдем предмет в кавычках
		endQuote := strings.Index(args[1:], "\"")
		if endQuote != -1 {
			subjectName = args[1 : endQuote+1]
			// Парсим оставшиеся аргументы
			remaining := strings.TrimSpace(args[endQuote+2:])
			remainingArgs := strings.Fields(remaining)
			if len(remainingArgs) >= 2 {
				dateStr = remainingArgs[0]
				timeStr = remainingArgs[1]
			}
		}
	}
	
	// Парсинг даты и времени с несколькими форматами
	datetimeStr := dateStr + " " + timeStr
	var startTime time.Time
	var parseErr error
	
	// Попробуем разные форматы
	formats := []string{
		"02.01.2006 15:04",
		"2.01.2006 15:04", 
		"02.1.2006 15:04",
		"2.1.2006 15:04",
		"02.01.2006 15:4",
		"2.01.2006 15:4",
	}
	
	for _, format := range formats {
		startTime, parseErr = time.Parse(format, datetimeStr)
		if parseErr == nil {
			break
		}
	}
	
	if parseErr != nil {
		sendMessage(bot, message.Chat.ID, fmt.Sprintf("❌ Неверный формат даты или времени: '%s'\nИспользуйте DD.MM.YYYY HH:MM или D.M.YYYY H:MM", datetimeStr))
		return
	}
	
	// Проверяем, что урок не в прошлом
	if startTime.Before(time.Now()) {
		sendMessage(bot, message.Chat.ID, "❌ Нельзя создать урок в прошлом")
		return
	}
	
	// Получаем ID предмета
	var subjectID int
	err = db.QueryRow("SELECT id FROM subjects WHERE name = $1", subjectName).Scan(&subjectID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Предмет не найден. Используйте /subjects для просмотра доступных предметов")
		return
	}
	
	// Получаем teacher_id для текущего пользователя
	var teacherID int
	err = db.QueryRow(`
		SELECT t.id FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE u.tg_id = $1`, strconv.FormatInt(userID, 10)).Scan(&teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден в системе")
		return
	}
	
	// Создаем урок
	_, err = db.Exec(`
		INSERT INTO lessons (subject_id, teacher_id, start_time, max_students, status, created_at)
		VALUES ($1, $2, $3, 10, 'active', NOW())`,
		subjectID, teacherID, startTime)
		
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания урока")
		return
	}
	
	successText := "✅ **Урок успешно создан!**\n\n" +
		"📚 Предмет: " + subjectName + "\n" +
		"📅 Дата: " + startTime.Format("02.01.2006 15:04") + "\n" +
		"👥 Максимум студентов: 10\n\n" +
		"Урок уже доступен для записи студентов!"
		
	msg := tgbotapi.NewMessage(message.Chat.ID, successText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Перенос урока
func handleRescheduleLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для переноса уроков")
		return
	}
	
	// Пока заглушка - будет реализована позже
	helpText := "📝 **Перенос урока**\n\n" +
		"**Формат:** `/reschedule_lesson <ID урока> <новая дата> <новое время>`\n\n" +
		"**Пример:** `/reschedule_lesson 123 16.08.2025 15:00`\n\n" +
		"⚙️ Функция в разработке"
		
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Отмена/удаление урока  
func handleCancelLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	args := message.CommandArguments()
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отмены уроков")
		return
	}
	
	// Если нет аргументов - показываем кнопки с предметами
	if args == "" {
		showSubjectButtons(bot, message, db, "delete")
		return
	}
	
	// Если есть ID урока - удаляем конкретный урок
	lessonID, err := strconv.Atoi(args)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока. Используйте: /cancel_lesson [ID]")
		return
	}
	
	// Получаем teacher_id если это преподаватель
	var teacherID int
	if role == "teacher" {
		err = db.QueryRow(`
			SELECT t.id FROM teachers t 
			JOIN users u ON t.user_id = u.id 
			WHERE u.tg_id = $1`, strconv.FormatInt(userID, 10)).Scan(&teacherID)
		
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден в системе")
			return
		}
		
		// Проверяем что урок принадлежит этому преподавателю
		var lessonExists bool
		err = db.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM lessons WHERE id = $1 AND teacher_id = $2 AND soft_deleted = false)`,
			lessonID, teacherID).Scan(&lessonExists)
		
		if err != nil || !lessonExists {
			sendMessage(bot, message.Chat.ID, "❌ Урок не найден или не принадлежит вам")
			return
		}
	}
	
	// Получаем информацию об уроке
	var subjectName string
	var startTime time.Time
	var enrolledCount int
	err = db.QueryRow(`
		SELECT s.name, l.start_time,
		       (SELECT COUNT(*) FROM enrollments e WHERE e.lesson_id = l.id AND e.status = 'confirmed') as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.id = $1 AND l.soft_deleted = false`, lessonID).Scan(&subjectName, &startTime, &enrolledCount)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
		return
	}
	
	// Мягкое удаление урока
	_, err = db.Exec(`
		UPDATE lessons SET soft_deleted = true, updated_at = NOW() 
		WHERE id = $1`, lessonID)
	
	if err != nil {
		log.Printf("Ошибка удаления урока: %v", err)
		sendMessage(bot, message.Chat.ID, "❌ Ошибка при удалении урока")
		return
	}
	
	// Отменяем все записи на урок
	_, err = db.Exec(`
		UPDATE enrollments SET status = 'cancelled', updated_at = NOW()
		WHERE lesson_id = $1 AND status = 'confirmed'`, lessonID)
	
	if err != nil {
		log.Printf("Ошибка отмены записей: %v", err)
	}
	
	// Уведомляем студентов об отмене (если есть записавшиеся)
	if enrolledCount > 0 {
		rows, err := db.Query(`
			SELECT u.tg_id FROM enrollments e
			JOIN users u ON e.user_id = u.id
			WHERE e.lesson_id = $1 AND e.status = 'cancelled'`, lessonID)
		
		if err == nil {
			defer rows.Close()
			notificationText := fmt.Sprintf(
				"❌ **Урок отменен**\n\n"+
				"📚 Предмет: %s\n"+
				"📅 Время: %s\n\n"+
				"Приносим извинения за неудобства.",
				subjectName, startTime.Format("02.01.2006 15:04"))
			
			for rows.Next() {
				var studentTgID string
				if rows.Scan(&studentTgID) == nil {
					studentID, _ := strconv.ParseInt(studentTgID, 10, 64)
					msg := tgbotapi.NewMessage(studentID, notificationText)
					msg.ParseMode = "Markdown"
					bot.Send(msg)
				}
			}
		}
	}
	
	// Подтверждение успешного удаления
	confirmText := fmt.Sprintf(
		"✅ **Урок успешно удален**\n\n"+
		"📚 Предмет: %s\n"+
		"📅 Время: %s\n"+
		"👥 Уведомлено студентов: %d",
		subjectName, startTime.Format("02.01.2006 15:04"), enrolledCount)
		
	msg := tgbotapi.NewMessage(message.Chat.ID, confirmText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Расписание преподавателя - мои уроки
func handleMyScheduleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Получаем teacher_id для текущего пользователя
	var teacherID int
	err := db.QueryRow(`
		SELECT t.id FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE u.tg_id = $1`, strconv.FormatInt(userID, 10)).Scan(&teacherID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден в системе")
		return
	}
	
	// Получаем уроки преподавателя на ближайшую неделю
	rows, err := db.Query(`
		SELECT l.id, l.start_time, s.name, l.max_students,
		       (SELECT COUNT(*) FROM enrollments e WHERE e.lesson_id = l.id AND e.status = 'confirmed') as enrolled_count,
		       l.status
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.teacher_id = $1 
		  AND l.start_time >= NOW() 
		  AND l.start_time <= NOW() + INTERVAL '7 days'
		  AND l.soft_deleted = false
		ORDER BY l.start_time`, teacherID)
	
	if err != nil {
		log.Printf("Ошибка получения расписания преподавателя: %v", err)
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки расписания")
		return
	}
	defer rows.Close()
	
	responseText := "📅 **Мое расписание на неделю**\n\n"
	hasLessons := false
	
	for rows.Next() {
		hasLessons = true
		var lessonID int
		var startTime time.Time
		var subjectName, status string
		var maxStudents, enrolledCount int
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName, &maxStudents, &enrolledCount, &status); err != nil {
			continue
		}
		
		statusIcon := "✅"
		if status == "cancelled" {
			statusIcon = "❌"
		} else if status == "rescheduled" {
			statusIcon = "🔄"
		}
		
		responseText += fmt.Sprintf(
			"%s **%s**\n📅 %s\n👥 Записано: %d/%d\n🆔 ID: %d\n\n",
			statusIcon, subjectName, 
			startTime.Format("02.01.2006 15:04"), 
			enrolledCount, maxStudents, lessonID)
	}
	
	if !hasLessons {
		responseText += "📭 У вас нет запланированных уроков на ближайшую неделю"
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Студенты преподавателя по урокам
func handleTeacherStudentsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	args := message.CommandArguments()
	
	// Получаем teacher_id для текущего пользователя
	var teacherID int
	err := db.QueryRow(`
		SELECT t.id FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE u.tg_id = $1`, strconv.FormatInt(userID, 10)).Scan(&teacherID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден в системе")
		return
	}
	
	if args == "" {
		// Показываем список уроков для выбора
		handleShowTeacherLessonsForStudents(bot, message, db, teacherID)
		return
	}
	
	// Парсим lesson_id из аргументов
	lessonID, err := strconv.Atoi(args)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока. Используйте: /my_students [lesson_id]")
		return
	}
	
	// Проверяем что урок принадлежит данному преподавателю
	var lessonExists bool
	err = db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM lessons WHERE id = $1 AND teacher_id = $2 AND soft_deleted = false)`,
		lessonID, teacherID).Scan(&lessonExists)
	
	if err != nil || !lessonExists {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден или не принадлежит вам")
		return
	}
	
	// Получаем информацию об уроке и студентах
	var subjectName string
	var startTime time.Time
	err = db.QueryRow(`
		SELECT s.name, l.start_time
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.id = $1`, lessonID).Scan(&subjectName, &startTime)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки информации об уроке")
		return
	}
	
	// Получаем список студентов
	rows, err := db.Query(`
		SELECT u.full_name, u.tg_id, e.status, e.enrolled_at
		FROM enrollments e
		JOIN users u ON e.user_id = u.id
		WHERE e.lesson_id = $1
		ORDER BY e.enrolled_at`, lessonID)
	
	if err != nil {
		log.Printf("Ошибка получения студентов урока: %v", err)
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки студентов")
		return
	}
	defer rows.Close()
	
	responseText := fmt.Sprintf("� **Студенты урока**\n\n📚 Урок: %s\n📅 %s\n\n", 
		subjectName, startTime.Format("02.01.2006 15:04"))
	
	studentCount := 0
	for rows.Next() {
		var fullName, tgID, status string
		var enrolledAt time.Time
		
		if err := rows.Scan(&fullName, &tgID, &status, &enrolledAt); err != nil {
			continue
		}
		
		studentCount++
		statusIcon := "✅"
		if status == "waitlist" {
			statusIcon = "⏳"
		} else if status == "cancelled" {
			statusIcon = "❌"
		}
		
		responseText += fmt.Sprintf("%d. %s %s\n📞 @%s\n📅 Записался: %s\n\n",
			studentCount, statusIcon, fullName, tgID, enrolledAt.Format("02.01.2006 15:04"))
	}
	
	if studentCount == 0 {
		responseText += "👤 На урок пока никто не записался"
	} else {
		responseText += fmt.Sprintf("👥 **Всего студентов: %d**", studentCount)
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Показать уроки преподавателя для выбора студентов
func handleShowTeacherLessonsForStudents(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, teacherID int) {
	// Получаем активные уроки преподавателя
	rows, err := db.Query(`
		SELECT l.id, l.start_time, s.name,
		       (SELECT COUNT(*) FROM enrollments e WHERE e.lesson_id = l.id AND e.status IN ('confirmed', 'waitlist')) as student_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.teacher_id = $1 
		  AND l.start_time >= NOW() - INTERVAL '1 day'
		  AND l.soft_deleted = false
		ORDER BY l.start_time`, teacherID)
	
	if err != nil {
		log.Printf("Ошибка получения уроков преподавателя: %v", err)
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки уроков")
		return
	}
	defer rows.Close()
	
	responseText := "👥 **Выберите урок для просмотра студентов**\n\n"
	responseText += "Используйте команду: `/my_students [ID урока]`\n\n"
	
	hasLessons := false
	for rows.Next() {
		hasLessons = true
		var lessonID int
		var startTime time.Time
		var subjectName string
		var studentCount int
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName, &studentCount); err != nil {
			continue
		}
		
		responseText += fmt.Sprintf("🆔 **%d** - %s\n📅 %s\n👥 Студентов: %d\n\n",
			lessonID, subjectName, startTime.Format("02.01.2006 15:04"), studentCount)
	}
	
	if !hasLessons {
		responseText += "📭 У вас нет активных уроков"
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Показ кнопок с предметами для создания/удаления урока
func showSubjectButtons(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, action string) {
	// Получаем все предметы из базы
	rows, err := db.Query("SELECT id, name FROM subjects ORDER BY name")
	if err != nil {
		log.Printf("Ошибка получения предметов: %v", err)
		sendMessage(bot, message.Chat.ID, "❌ Ошибка загрузки предметов")
		return
	}
	defer rows.Close()
	
	var keyboard [][]tgbotapi.InlineKeyboardButton
	
	for rows.Next() {
		var subjectID int
		var subjectName string
		if err := rows.Scan(&subjectID, &subjectName); err != nil {
			continue
		}
		
		callbackData := fmt.Sprintf("%s_lesson:%d", action, subjectID)
		button := tgbotapi.NewInlineKeyboardButtonData(subjectName, callbackData)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}
	
	if len(keyboard) == 0 {
		sendMessage(bot, message.Chat.ID, "❌ В базе нет предметов. Обратитесь к администратору.")
		return
	}
	
	var headerText string
	if action == "create" {
		headerText = "📚 **Выберите предмет для создания урока:**"
	} else {
		headerText = "📚 **Выберите предмет для удаления урока:**"
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, headerText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	bot.Send(msg)
}
