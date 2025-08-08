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

// Основной обработчик сообщений
func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *sql.DB) {
	if update.Message != nil {
		handleMessage(bot, update.Message, db)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update.CallbackQuery, db)
	}
}

// Обработка текстовых сообщений
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	if message.IsCommand() {
		handleCommand(bot, message, db)
	} else {
		// Обработка текста через FSM
		handleTextMessage(bot, message, db)
	}
}

// Обработка команд
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	switch message.Command() {
	case "start":
		handleStart(bot, message, db)
	case "register":
		handleRegister(bot, message, db)
	case "help":
		handleHelp(bot, message, db)
	case "subjects":
		handleSubjectsCommand(bot, message, db)
	case "schedule":
		handleScheduleCommand(bot, message, db)
	case "enroll":
		handleEnrollCommand(bot, message, db)
	case "create_lesson":
		handleCreateLessonCommand(bot, message, db)
	case "reschedule_lesson":
		handleRescheduleLessonCommand(bot, message, db)
	case "my_lessons":
		handleMyLessonsCommand(bot, message, db)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, 
			"❓ Неизвестная команда. Используйте /help для получения списка доступных команд.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}

// Обработка callback запросов (inline кнопки)
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Ответить на callback чтобы убрать индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callback)
	
	data := query.Data
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	
	switch {
	case data == "cmd_register":
		// Имитация команды /register
		fakeMessage := &tgbotapi.Message{
			From: query.From,
			Chat: query.Message.Chat,
			Text: "/register",
		}
		handleRegister(bot, fakeMessage, db)
		
	case data == "cmd_help":
		// Имитация команды /help
		fakeMessage := &tgbotapi.Message{
			From: query.From,
			Chat: query.Message.Chat,
			Text: "/help",
		}
		handleHelp(bot, fakeMessage, db)
		
	case strings.HasPrefix(data, "subject_"):
		handleSubjectCallback(bot, query, db)
		
	case strings.HasPrefix(data, "enroll_"):
		handleEnrollCallback(bot, query, db)
		
	case data == "finish_registration":
		finishStudentRegistration(bot, userID, chatID, db)
		
	default:
		msg := tgbotapi.NewMessage(chatID, "❓ Неизвестное действие")
		bot.Send(msg)
	}
}

// Обработка выбора предмета через callback
func handleSubjectCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	
	// Проверить состояние пользователя
	if getUserState(userID) != StateSelectingSubjects {
		msg := tgbotapi.NewMessage(chatID, "❌ Сначала начните регистрацию командой /register")
		bot.Send(msg)
		return
	}
	
	// Извлечь ID предмета
	subjectIDStr := strings.TrimPrefix(query.Data, "subject_")
	subjectID, err := strconv.Atoi(subjectIDStr)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка выбора предмета")
		bot.Send(msg)
		return
	}
	
	// Получить текущие выбранные предметы
	initUserData(userID)
	var selectedSubjects []int
	if subjects, exists := userData[userID]["selected_subjects"]; exists {
		selectedSubjects = subjects.([]int)
	}
	
	// Проверить, не выбран ли уже этот предмет
	var alreadySelected bool
	var newSubjects []int
	
	for _, id := range selectedSubjects {
		if id == subjectID {
			alreadySelected = true
		} else {
			newSubjects = append(newSubjects, id)
		}
	}
	
	// Получить название предмета
	var subjectName string
	db.QueryRow("SELECT name FROM subjects WHERE id = $1", subjectID).Scan(&subjectName)
	
	var responseText string
	if alreadySelected {
		// Убрать из выбранных
		userData[userID]["selected_subjects"] = newSubjects
		responseText = fmt.Sprintf("➖ Предмет '%s' убран из списка", subjectName)
	} else {
		// Добавить в выбранные
		newSubjects = append(selectedSubjects, subjectID)
		userData[userID]["selected_subjects"] = newSubjects
		responseText = fmt.Sprintf("➕ Предмет '%s' добавлен в список", subjectName)
	}
	
	// Показать текущий выбор
	var currentSubjects []string
	for _, id := range newSubjects {
		var name string
		db.QueryRow("SELECT name FROM subjects WHERE id = $1", id).Scan(&name)
		currentSubjects = append(currentSubjects, name)
	}
	
	if len(currentSubjects) > 0 {
		responseText += fmt.Sprintf("\n\n✅ Выбранные предметы:\n• %s", 
			strings.Join(currentSubjects, "\n• "))
	} else {
		responseText += "\n\n📝 Пока не выбрано ни одного предмета"
	}
	
	// Отправить уведомление
	msg := tgbotapi.NewMessage(chatID, responseText)
	bot.Send(msg)
}

// Команда /help
func handleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверить роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	var helpText string
	
	if err == sql.ErrNoRows {
		// Незарегистрированный пользователь
		helpText = "🆘 Помощь - Constellation School Bot\n\n" +
			"👋 Добро пожаловать! Для начала работы необходимо зарегистрироваться.\n\n" +
			"📝 Доступные команды:\n" +
			"/start - начальное приветствие\n" +
			"/register - регистрация в системе\n" +
			"/help - эта справка\n\n" +
			"🎯 О Центре Цифрового Творчества:\n" +
			"Мы предлагаем 6 направлений обучения:\n" +
			"• 3D-моделирование\n" +
			"• Геймдев\n" +
			"• VFX-дизайн\n" +
			"• Графический дизайн\n" +
			"• Веб-разработка\n" +
			"• Компьютерная грамотность"
			
	} else if err != nil {
		helpText = "❌ Ошибка получения информации о пользователе"
		
	} else {
		switch role {
		case "student":
			helpText = "🆘 Помощь для студентов\n\n" +
				"📚 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/subjects - выбор/изменение предметов\n" +
				"/schedule - расписание доступных уроков\n" +
				"/my_lessons - мои записи на уроки\n" +
				"/help - эта справка\n\n" +
				"❓ Как записаться на урок:\n" +
				"1. Используйте /schedule для просмотра доступных уроков\n" +
				"2. Нажмите 'Записаться' у интересующего урока\n" +
				"3. Подтвердите запись\n\n" +
				"📞 Нужна помощь? Обратитесь к администратору."
				
		case "teacher":
			helpText = "🆘 Помощь для преподавателей\n\n" +
				"👨‍🏫 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/my_lessons - мои уроки\n" +
				"/my_students - студенты на моих уроках\n" +
				"/help_teacher - подробная справка\n\n" +
				"📝 Управление уроками:\n" +
				"• Просмотр списка своих уроков\n" +
				"• Просмотр записанных студентов\n" +
				"• Отмена уроков при необходимости"
				
		case "superuser":
			helpText = "🆘 Помощь для администраторов\n\n" +
				"🔧 Доступные команды:\n" +
				"/start - главное меню\n" +
				"/add_teacher - добавить нового преподавателя\n" +
				"/create_lesson - создать новый урок\n" +
				"/system_stats - статистика системы\n" +
				"/manage_subjects - управление предметами\n\n" +
				"⚡ Административные функции:\n" +
				"• Управление преподавателями\n" +
				"• Создание и управление уроками\n" +
				"• Просмотр статистики системы"
				
		default:
			helpText = "🆘 Помощь\n\nИспользуйте /start для начала работы"
		}
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	bot.Send(msg)
}

// Заглушки для других команд (будут реализованы в следующих шагах)
func handleSubjectsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Получаем все активные предметы
	rows, err := db.Query("SELECT name, description, category FROM subjects WHERE is_active = true ORDER BY name")
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка загрузки предметов")
		bot.Send(msg)
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
		msg := tgbotapi.NewMessage(message.Chat.ID, "📚 Пока нет доступных предметов")
		bot.Send(msg)
		return
	}

	text := "🎯 **Доступные предметы:**\n\n" + strings.Join(subjects, "\n\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func handleScheduleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Получаем уроки на ближайшие 7 дней
	rows, err := db.Query(`
		SELECT l.start_time, s.name, u.full_name, l.max_students,
			COUNT(e.id) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'confirmed'
		WHERE l.start_time > NOW() AND l.start_time < NOW() + INTERVAL '7 days'
			AND l.soft_deleted = false
		GROUP BY l.id, l.start_time, s.name, u.full_name, l.max_students
		ORDER BY l.start_time
		LIMIT 10
	`)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка загрузки расписания")
		bot.Send(msg)
		return
	}
	defer rows.Close()

	var lessons []string
	for rows.Next() {
		var startTime time.Time
		var subjectName, teacherName string
		var maxStudents, enrolledCount int
		
		if err := rows.Scan(&startTime, &subjectName, &teacherName, &maxStudents, &enrolledCount); err != nil {
			continue
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
		
		lesson := fmt.Sprintf("📅 %s\n📚 %s\n👨‍🏫 %s\n%s", 
			startTime.Format("02.01.2006 15:04"), subjectName, teacherName, status)
		lessons = append(lessons, lesson)
	}

	if len(lessons) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📅 На ближайшую неделю уроков не запланировано")
		bot.Send(msg)
		return
	}

	text := "📅 **Расписание на неделю:**\n\n" + strings.Join(lessons, "\n\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func handleEnrollCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем, что пользователь зарегистрирован как студент
	var studentID int
	err := db.QueryRow("SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&studentID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Сначала зарегистрируйтесь как студент с помощью /register")
		bot.Send(msg)
		return
	}

	// Получаем доступные для записи уроки
	rows, err := db.Query(`
		SELECT l.id, l.start_time, s.name, u.full_name, l.max_students,
			COUNT(e.id) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id  
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'confirmed'
		WHERE l.start_time > NOW() AND l.soft_deleted = false
		GROUP BY l.id, l.start_time, s.name, u.full_name, l.max_students
		HAVING COUNT(e.id) < l.max_students
		ORDER BY l.start_time
		LIMIT 10
	`)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка загрузки доступных уроков")
		bot.Send(msg)
		return
	}
	defer rows.Close()

	var keyboard [][]tgbotapi.InlineKeyboardButton
	var lessons []string
	
	for rows.Next() {
		var lessonID, maxStudents, enrolledCount int
		var startTime time.Time
		var subjectName, teacherName string
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName, &teacherName, &maxStudents, &enrolledCount); err != nil {
			continue
		}
		
		freeSpots := maxStudents - enrolledCount
		lesson := fmt.Sprintf("📅 %s\n📚 %s\n👨‍🏫 %s\n🆓 %d мест свободно", 
			startTime.Format("02.01.2006 15:04"), subjectName, teacherName, freeSpots)
		lessons = append(lessons, lesson)
		
		// Кнопка для записи
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("Записаться на %s", subjectName),
			fmt.Sprintf("enroll_%d", lessonID))
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{btn})
	}

	if len(lessons) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📝 Нет доступных для записи уроков")
		bot.Send(msg)
		return
	}

	text := "📝 **Выберите урок для записи:**\n\n" + strings.Join(lessons, "\n\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	bot.Send(msg)
}

func handleMyLessonsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	msg := tgbotapi.NewMessage(message.Chat.ID, 
		"📖 Просмотр ваших уроков будет доступен в следующих обновлениях")
	bot.Send(msg)
}

// Обработка записи на урок через callback
func handleEnrollCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	
	// Извлекаем ID урока из callback data
	lessonIDStr := strings.TrimPrefix(query.Data, "enroll_")
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка: неверный ID урока")
		bot.Send(msg)
		return
	}
	
	// Получаем ID студента
	var studentID int
	err = db.QueryRow("SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&studentID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Сначала зарегистрируйтесь как студент")
		bot.Send(msg)
		return
	}
	
	// Проверяем, не записан ли уже студент на этот урок
	var existingEnrollment int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE student_id = $1 AND lesson_id = $2 AND status = 'confirmed'", 
		studentID, lessonID).Scan(&existingEnrollment)
	if err == nil && existingEnrollment > 0 {
		msg := tgbotapi.NewMessage(chatID, "ℹ️ Вы уже записаны на этот урок")
		bot.Send(msg)
		return
	}
	
	// Проверяем свободные места
	var enrolledCount, maxStudents int
	var subjectName string
	var startTime time.Time
	err = db.QueryRow(`
		SELECT COUNT(e.id), l.max_students, s.name, l.start_time
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'confirmed'
		WHERE l.id = $1 AND l.soft_deleted = false
		GROUP BY l.id, l.max_students, s.name, l.start_time
	`, lessonID).Scan(&enrolledCount, &maxStudents, &subjectName, &startTime)
	
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Урок не найден или отменен")
		bot.Send(msg)
		return
	}
	
	if enrolledCount >= maxStudents {
		msg := tgbotapi.NewMessage(chatID, "❌ На урок нет свободных мест")
		bot.Send(msg)
		return
	}
	
	// Записываем студента на урок
	_, err = db.Exec("INSERT INTO enrollments (student_id, lesson_id, status, enrolled_at) VALUES ($1, $2, 'confirmed', NOW())", 
		studentID, lessonID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при записи на урок")
		bot.Send(msg)
		return
	}
	
	// Подтверждение записи
	text := fmt.Sprintf("✅ **Успешная запись!**\n\n📚 Урок: %s\n📅 Дата: %s\n\n💡 Не забудьте прийти вовремя!", 
		subjectName, startTime.Format("02.01.2006 15:04"))
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Команда создания урока для teachers/superusers
func handleCreateLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	chatID := message.Chat.ID
	
	// Проверяем роль (teacher или superuser)
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
	if err != nil || (role != "teacher" && role != "superuser") {
		msg := tgbotapi.NewMessage(chatID, "❌ У вас нет прав для создания уроков")
		bot.Send(msg)
		return
	}
	
	msg := tgbotapi.NewMessage(chatID, "🔧 **Создание урока**\n\n" +
		"Формат: `/create_lesson <предмет> <дата> <время>`\n" +
		"Пример: `/create_lesson математика 15.08.2025 14:30`\n\n" +
		"📝 Доступные предметы: /subjects")
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Команда переноса урока для teachers/superusers  
func handleRescheduleLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	chatID := message.Chat.ID
	
	// Проверяем роль (teacher или superuser)
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
	if err != nil || (role != "teacher" && role != "superuser") {
		msg := tgbotapi.NewMessage(chatID, "❌ У вас нет прав для переноса уроков")
		bot.Send(msg)
		return
	}
	
	msg := tgbotapi.NewMessage(chatID, "📅 **Перенос урока**\n\n" +
		"Формат: `/reschedule_lesson <ID урока> <новая дата> <новое время>`\n" +
		"Пример: `/reschedule_lesson 123 16.08.2025 15:00`\n\n" +
		"📋 Ваши уроки: /my_lessons")
	msg.ParseMode = "Markdown" 
	bot.Send(msg)
}
