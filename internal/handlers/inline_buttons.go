package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Создание inline-клавиатуры для главного меню студента
func createStudentMainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Расписание", "schedule"),
			tgbotapi.NewInlineKeyboardButtonData("📚 Мои уроки", "my_lessons"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "help"),
			tgbotapi.NewInlineKeyboardButtonData("👤 Профиль", "profile"),
		),
	)
}

// Создание inline-клавиатуры для главного меню преподавателя
func createTeacherMainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Мои уроки", "my_lessons"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Мои студенты", "my_students"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать урок", "create_lesson"),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Отменить урок", "cancel_lesson"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "help_teacher"),
		),
	)
}

// Создание inline-клавиатуры для главного меню администратора
func createAdminMainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨‍🏫 Преподаватели", "teachers"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Уведомления", "notifications"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Логи", "logs"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "help_admin"),
		),
	)
}

// Создание inline-клавиатуры для списка предметов
func createSubjectsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎮 Геймдев", "subject_GAMEDEV"),
			tgbotapi.NewInlineKeyboardButtonData("🌐 Веб-разработка", "subject_WEB_DEV"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎨 Графический дизайн", "subject_GRAPHIC_DESIGN"),
			tgbotapi.NewInlineKeyboardButtonData("🎬 VFX-дизайн", "subject_VFX_DESIGN"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎯 3D-моделирование", "subject_3D_MODELING"),
			tgbotapi.NewInlineKeyboardButtonData("💻 Компьютерная грамотность", "subject_COMPUTER_LITERACY"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back_to_main"),
		),
	)
}

// Создание inline-клавиатуры для действий с уроком
func createLessonActionsKeyboard(lessonID int, canEnroll bool, canUnenroll bool) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if canEnroll {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Записаться", fmt.Sprintf("enroll_%d", lessonID)),
		))
	}

	if canUnenroll {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отписаться", fmt.Sprintf("unenroll_%d", lessonID)),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📋 Подробнее", fmt.Sprintf("lesson_info_%d", lessonID)),
		tgbotapi.NewInlineKeyboardButtonData("⏰ Напомнить", fmt.Sprintf("remind_%d", lessonID)),
	))

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к расписанию", "back_to_schedule"),
	))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: buttons}
}

// Создание inline-клавиатуры для подтверждения действий
func createConfirmationKeyboard(action string, id int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("confirm_%s_%d", action, id)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel_action"),
		),
	)
}

// Создание inline-клавиатуры для навигации
func createNavigationKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "main_menu"),
		),
	)
}

// Обработка inline-кнопок
func handleInlineButton(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	// Убираем индикатор загрузки
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callback)

	data := query.Data

	switch {
	case strings.HasPrefix(data, "create_lesson:") || strings.HasPrefix(data, "delete_lesson:"):
		handleLessonSubjectCallback(bot, query, db)
	case data == "main_menu":
		handleMainMenu(bot, query.Message, db)
	case data == "create_lesson":
		handleCreateLessonButton(bot, query.Message, db)
	case data == "cancel_lesson":
		handleCancelLessonButton(bot, query.Message, db)
	case data == "schedule":
		handleScheduleButton(bot, query.Message, db)
	case data == "my_lessons":
		handleMyLessonsButton(bot, query.Message, db)
	case data == "help":
		handleHelpButton(bot, query.Message, db)
	case data == "profile":
		handleProfileButton(bot, query.Message, db)
	case data == "teachers":
		handleTeachersButton(bot, query.Message, db)
	case data == "stats":
		handleStatsButton(bot, query.Message, db)
	case data == "notifications":
		handleNotificationsButton(bot, query.Message, db)
	case data == "logs":
		handleLogsButton(bot, query.Message, db)
	case data == "help_teacher":
		handleHelpTeacherButton(bot, query.Message, db)
	case data == "help_admin":
		handleHelpAdminButton(bot, query.Message, db)
	case data == "back_to_main":
		handleMainMenu(bot, query.Message, db)
	case data == "back_to_schedule":
		handleScheduleButton(bot, query.Message, db)
	case data == "back":
		handleBackButton(bot, query.Message, db)
	case data == "cancel_action":
		handleCancelAction(bot, query.Message, db)
	// Новые студенческие кнопки
	case data == "student_dashboard":
		showStudentMainMenu(bot, query.Message, db)
	case data == "enroll_subjects":
		showSubjectsForEnrollment(bot, query.Message.Chat.ID, db)
	case data == "my_lessons_menu":
		handleMyLessonsCommand(bot, query.Message, db)
	case data == "school_schedule":
		handleScheduleCommand(bot, query.Message, db)
	case data == "my_waitlist":
		handleWaitlistCommand(bot, query.Message, db)
	case data == "help_student":
		sendMessage(bot, query.Message.Chat.ID, 
			"📚 **Справка для студентов:**\n\n"+
			"🎓 Главное меню: /start\n"+
			"📚 Записаться на урок: используйте кнопки\n"+
			"📅 Мои уроки: показывает ваши записи\n"+
			"📆 Расписание: все уроки школы\n"+
			"⏳ Лист ожидания: очередь на популярные уроки\n\n"+
			"❓ Возникли вопросы? Обратитесь к администратору.")
	default:
		// Обработка callback'ов с предметами для записи
		if strings.HasPrefix(data, "enroll_subject:") {
			parts := strings.Split(data, ":")
			if len(parts) == 2 {
				subjectID, err := strconv.Atoi(parts[1])
				if err == nil {
					showAvailableLessonsForSubject(bot, query, db, subjectID)
					return
				}
			}
		}
		
		// Обработка динамических кнопок
		handleDynamicButton(bot, query, db)
	}
}

// Обработка главного меню
func handleMainMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Получаем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения роли пользователя")
		return
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	var welcomeText string

	switch role {
	case "student":
		keyboard = createStudentMainMenu()
		welcomeText = "👋 **Добро пожаловать в главное меню!**\n\nВыберите нужный раздел:"
	case "teacher":
		keyboard = createTeacherMainMenu()
		welcomeText = "👨‍🏫 **Панель преподавателя**\n\nВыберите действие:"
	case "superuser":
		keyboard = createAdminMainMenu()
		welcomeText = "👑 **Панель администратора**\n\nВыберите раздел управления:"
	default:
		sendMessage(bot, message.Chat.ID, "❌ Неизвестная роль пользователя")
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// Обработка кнопки расписания
func handleScheduleButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	text := "📅 **Расписание уроков**\n\nВыберите предмет для просмотра расписания:"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createSubjectsKeyboard()
	bot.Send(msg)
}

// Обработка кнопки "Мои уроки"
func handleMyLessonsButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleMyLessonsCommand(bot, message, db)
}

// Обработка кнопки помощи
func handleHelpButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleHelp(bot, message, db)
}

// Обработка кнопки профиля
func handleProfileButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	var fullName, role, phone string
	var isActive bool
	err := db.QueryRow("SELECT full_name, role, phone, is_active FROM users WHERE tg_id = $1", userID).Scan(&fullName, &role, &phone, &isActive)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения данных профиля")
		return
	}

	status := "✅ Активен"
	if !isActive {
		status = "❌ Деактивирован"
	}

	profileText := fmt.Sprintf("👤 **Ваш профиль**\n\n"+
		"📝 **Имя:** %s\n"+
		"🎭 **Роль:** %s\n"+
		"📱 **Телефон:** %s\n"+
		"🔐 **Статус:** %s\n\n"+
		"Для изменения данных обратитесь к администратору.", fullName, role, phone, status)

	msg := tgbotapi.NewMessage(message.Chat.ID, profileText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createNavigationKeyboard()
	bot.Send(msg)
}

// Обработка кнопки преподавателей (для админов)
func handleTeachersButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleListTeachersCommand(bot, message, db)
}

// Обработка кнопки статистики (для админов)
func handleStatsButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleStatsCommand(bot, message, db)
}

// Обработка кнопки уведомлений (для админов)
func handleNotificationsButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	text := "📢 **Управление уведомлениями**\n\n" +
		"Выберите тип уведомления:\n\n" +
		"• `/notify_students <lesson_id> <текст>` - уведомления студентов урока\n" +
		"• `/notify_all <текст>` - массовые уведомления\n" +
		"• `/remind_all [часы]` - напоминания о уроках\n\n" +
		"Используйте команды напрямую для отправки уведомлений."

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createNavigationKeyboard()
	bot.Send(msg)
}

// Обработка кнопки логов (для админов)
func handleLogsButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleLogRecentErrorsCommand(bot, message, db)
}

// Обработка кнопки помощи преподавателя
func handleHelpTeacherButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Используем существующую функцию
	handleHelpTeacherCommand(bot, message, db)
}

// Обработка кнопки помощи администратора
func handleHelpAdminButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	helpText := "👑 **Справка администратора**\n\n" +
		"**📋 Доступные команды:**\n\n" +
		"**👨‍🏫 Управление преподавателями:**\n" +
		"• `/add_teacher` - добавление преподавателя\n" +
		"• `/delete_teacher` - удаление преподавателя\n" +
		"• `/restore_teacher` - восстановление преподавателя\n" +
		"• `/list_teachers` - список преподавателей\n\n" +
		"**📅 Управление уроками:**\n" +
		"• `/create_lesson` - создание урока\n" +
		"• `/delete_lesson` - удаление урока\n" +
		"• `/restore_lesson` - восстановление урока\n" +
		"• `/reschedule_lesson` - перенос урока\n\n" +
		"**📢 Уведомления:**\n" +
		"• `/notify_students` - уведомления студентов\n" +
		"• `/notify_all` - массовые уведомления\n" +
		"• `/remind_all` - напоминания\n\n" +
		"**👥 Управление студентами:**\n" +
		"• `/deactivate_student` - деактивация студента\n" +
		"• `/activate_student` - активация студента\n\n" +
		"**📊 Аналитика:**\n" +
		"• `/stats` - статистика системы\n" +
		"• `/log_recent_errors` - просмотр логов"

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createNavigationKeyboard()
	bot.Send(msg)
}

// Обработка кнопки "Назад"
func handleBackButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	handleMainMenu(bot, message, db)
}

// Обработка кнопки отмены действия
func handleCancelAction(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	text := "❌ **Действие отменено**\n\nВозвращаемся в главное меню."
	
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createNavigationKeyboard()
	bot.Send(msg)
}

// Обработка динамических кнопок (запись, отписка, информация об уроке)
func handleDynamicButton(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	data := query.Data
	message := query.Message

	// Обработка записи на урок
	if len(data) > 7 && data[:7] == "enroll_" {
		lessonIDStr := data[7:]
		lessonID, err := strconv.Atoi(lessonIDStr)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
			return
		}
		
		// Создаем временное сообщение для обработки
		tempMessage := *message
		tempMessage.Text = fmt.Sprintf("/enroll %d", lessonID)
		handleEnrollCommand(bot, &tempMessage, db)
		return
	}

	// Обработка отписки от урока
	if len(data) > 9 && data[:9] == "unenroll_" {
		lessonIDStr := data[9:]
		lessonID, err := strconv.Atoi(lessonIDStr)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
			return
		}
		
		// Создаем временное сообщение для обработки
		tempMessage := *message
		tempMessage.Text = fmt.Sprintf("/unenroll %d", lessonID)
		// Здесь нужно будет добавить функцию handleUnenrollCommand
		sendMessage(bot, message.Chat.ID, "🔄 Функция отписки от урока в разработке")
		return
	}

	// Обработка информации об уроке
	if len(data) > 12 && data[:12] == "lesson_info_" {
		lessonIDStr := data[12:]
		lessonID, err := strconv.Atoi(lessonIDStr)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока")
			return
		}
		
		// Получаем информацию об уроке
		var subjectName, teacherName, startTime string
		var maxStudents, enrolledCount int
		err = db.QueryRow(`
			SELECT s.name, u.full_name, l.start_time::text, l.max_students,
			       COALESCE(COUNT(e.id), 0) as enrolled_count
			FROM lessons l
			JOIN subjects s ON l.subject_id = s.id
			LEFT JOIN teachers t ON l.teacher_id = t.id
			LEFT JOIN users u ON t.user_id = u.id
			LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
			WHERE l.id = $1 AND l.soft_deleted = false
			GROUP BY s.name, u.full_name, l.start_time, l.max_students`, lessonID).Scan(&subjectName, &teacherName, &startTime, &maxStudents, &enrolledCount)
		
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Урок не найден")
			return
		}

		infoText := fmt.Sprintf("📋 **Информация об уроке**\n\n"+
			"📚 **Предмет:** %s\n"+
			"👨‍🏫 **Преподаватель:** %s\n"+
			"⏰ **Время:** %s\n"+
			"👥 **Записано:** %d/%d\n"+
			"⏱️ **Длительность:** 90 минут", 
			subjectName, teacherName, startTime[:16], enrolledCount, maxStudents)

		msg := tgbotapi.NewMessage(message.Chat.ID, infoText)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = createNavigationKeyboard()
		bot.Send(msg)
		return
	}

	// Обработка предметов
	if len(data) > 8 && data[:8] == "subject_" {
		subjectCode := data[8:]
		handleSubjectSelection(bot, message, db, subjectCode)
		return
	}

	// Обработка подтверждений
	if len(data) > 8 && data[:8] == "confirm_" {
		actionData := data[8:]
		handleConfirmation(bot, message, db, actionData)
		return
	}

	sendMessage(bot, message.Chat.ID, "❌ Неизвестная кнопка")
}

// Обработка выбора предмета
func handleSubjectSelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, subjectCode string) {
	// Получаем уроки по предмету
	rows, err := db.Query(`
		SELECT l.id, l.start_time::text, u.full_name, l.max_students,
		       COALESCE(COUNT(e.id), 0) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE s.code = $1 AND l.soft_deleted = false AND l.start_time > NOW()
		GROUP BY l.id, l.start_time, u.full_name, l.max_students
		ORDER BY l.start_time`, subjectCode)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения расписания")
		return
	}
	defer rows.Close()

	var lessons []struct {
		id            int
		startTime     string
		teacherName   string
		maxStudents   int
		enrolledCount int
	}

	for rows.Next() {
		var lesson struct {
			id            int
			startTime     string
			teacherName   string
			maxStudents   int
			enrolledCount int
		}
		if err := rows.Scan(&lesson.id, &lesson.startTime, &lesson.teacherName, &lesson.maxStudents, &lesson.enrolledCount); err != nil {
			continue
		}
		lessons = append(lessons, lesson)
	}

	if len(lessons) == 0 {
		text := "📅 **Расписание пусто**\n\nНа данный момент нет запланированных уроков по этому предмету."
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = createNavigationKeyboard()
		bot.Send(msg)
		return
	}

	// Формируем список уроков
	var text string
	switch subjectCode {
	case "GAMEDEV":
		text = "🎮 **Расписание: Геймдев**\n\n"
	case "WEB_DEV":
		text = "🌐 **Расписание: Веб-разработка**\n\n"
	case "GRAPHIC_DESIGN":
		text = "🎨 **Расписание: Графический дизайн**\n\n"
	case "VFX_DESIGN":
		text = "🎬 **Расписание: VFX-дизайн**\n\n"
	case "3D_MODELING":
		text = "🎯 **Расписание: 3D-моделирование**\n\n"
	case "COMPUTER_LITERACY":
		text = "💻 **Расписание: Компьютерная грамотность**\n\n"
	default:
		text = "📅 **Расписание**\n\n"
	}

	for _, lesson := range lessons {
		available := lesson.maxStudents - lesson.enrolledCount
		status := "✅"
		if available <= 0 {
			status = "⏳"
		}
		
		text += fmt.Sprintf("%s **Урок %d**\n", status, lesson.id)
		text += fmt.Sprintf("⏰ %s\n", lesson.startTime[:16])
		text += fmt.Sprintf("👨‍🏫 %s\n", lesson.teacherName)
		text += fmt.Sprintf("👥 %d/%d мест\n\n", lesson.enrolledCount, lesson.maxStudents)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = createNavigationKeyboard()
	bot.Send(msg)
}

// Обработчик кнопки "Создать урок"
func handleCreateLessonButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Вызываем функцию показа предметов для создания урока
	showSubjectButtons(bot, message, db, "create")
}

// Обработчик кнопки "Отменить урок"
func handleCancelLessonButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Вызываем функцию показа предметов для удаления урока
	showSubjectButtons(bot, message, db, "delete")
}

// Обработка подтверждений действий
func handleConfirmation(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB, actionData string) {
	// Здесь можно добавить обработку подтверждений различных действий
	sendMessage(bot, message.Chat.ID, "✅ Действие подтверждено!")
}
