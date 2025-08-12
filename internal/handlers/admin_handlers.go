package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Обработчик команд для администраторов
func handleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав администратора")
		return
	}
	
	// Маршрутизация команд администратора
	switch message.Command() {
	case "add_teacher":
		handleAddTeacherCommand(bot, message, db)
	case "delete_teacher":
		handleDeleteTeacherCommand(bot, message, db)
	case "notify_students":
		handleNotifyStudentsCommand(bot, message, db)
	case "cancel_with_notification":
		handleCancelWithNotificationCommand(bot, message, db)
	case "reschedule_with_notify":
		handleRescheduleWithNotifyCommand(bot, message, db)
	case "list_teachers":
		handleListTeachersCommand(bot, message, db)
	case "my_students":
		handleMyStudentsCommand(bot, message, db)
	case "restore_lesson":
		handleRestoreLessonCommand(bot, message, db)
	case "restore_teacher":
		handleRestoreTeacherCommand(bot, message, db)
	case "rate_limit_stats":
		handleRateLimitStatsCommand(bot, message, db)
	case "stats":
		handleStatsCommand(bot, message, db)
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда администратора")
	}
}

// Добавление преподавателя
func handleAddTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для добавления преподавателей")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 4 {
		helpText := "📝 **Добавление преподавателя**\n\n" +
			"**Формат:** `/add_teacher <Telegram ID> <Имя> <Фамилия>`\n\n" +
			"**Пример:** `/add_teacher 999999999 Анна Петрова`\n\n" +
			"**Как узнать Telegram ID:**\n" +
			"Попросите преподавателя написать боту @userinfobot"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	tgID := args[1]
	firstName := args[2]
	lastName := strings.Join(args[3:], " ")
	
	// Проверяем, что Telegram ID корректный
	_, err = strconv.ParseInt(tgID, 10, 64)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный Telegram ID")
		return
	}
	
	// Проверяем, не существует ли уже пользователь с таким tg_id
	var existingUser string
	err = db.QueryRow("SELECT tg_id FROM users WHERE tg_id = $1", tgID).Scan(&existingUser)
	if err == nil {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь с таким Telegram ID уже существует")
		return
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания преподавателя")
		return
	}
	defer tx.Rollback()
	
	// Создаем пользователя
	var userID_new int
	err = tx.QueryRow(`
		INSERT INTO users (tg_id, full_name, role, created_at) 
		VALUES ($1, $2, 'teacher', NOW()) 
		RETURNING id`,
		tgID, firstName+" "+lastName).Scan(&userID_new)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания пользователя")
		return
	}
	
	// Создаем запись в таблице teachers
	_, err = tx.Exec(`
		INSERT INTO teachers (user_id, created_at)
		VALUES ($1, NOW())`,
		userID_new)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания преподавателя")
		return
	}
	
	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения данных")
		return
	}
	
	successText := "✅ **Преподаватель успешно добавлен!**\n\n" +
		"👨‍🏫 Имя: " + firstName + " " + lastName + "\n" +
		"🆔 Telegram ID: " + tgID + "\n\n" +
		"Преподаватель может начать использовать бота!"
		
	msg := tgbotapi.NewMessage(message.Chat.ID, successText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// КРИТИЧЕСКАЯ ФУНКЦИЯ: Удаление преподавателя (Шаг 8.1 ROADMAP)
func handleDeleteTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для удаления преподавателей")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "⚠️ **КРИТИЧЕСКАЯ ФУНКЦИЯ: Удаление преподавателя**\n\n" +
			"**Формат:** `/delete_teacher <teacher_id>`\n\n" +
			"**Пример:** `/delete_teacher 5`\n\n" +
			"**⚠️ ВНИМАНИЕ:**\n" +
			"• Все активные уроки преподавателя будут отменены\n" +
			"• Все студенты получат уведомления об отмене\n" +
			"• Преподаватель будет деактивирован\n" +
			"• Лист ожидания будет очищен\n\n" +
			"**Получить список преподавателей:** /list_teachers"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	teacherIDStr := args[1]
	teacherID, err := strconv.Atoi(teacherIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID преподавателя")
		return
	}
	
	// Проверяем, существует ли преподаватель
	var teacherName string
	var teacherTelegramID int64
	var userID_teacher int
	err = db.QueryRow(`
		SELECT u.full_name, u.tg_id, u.id
		FROM users u 
		JOIN teachers t ON u.id = t.user_id 
		WHERE t.id = $1 AND u.role = 'teacher' AND u.is_active = true`,
		teacherID).Scan(&teacherName, &teacherTelegramID, &userID_teacher)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Активный преподаватель не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска преподавателя")
		return
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
		return
	}
	defer tx.Rollback()
	
	// 1. Получаем список всех активных уроков преподавателя
	rows, err := tx.Query(`
		SELECT l.id, l.start_time, s.name 
		FROM lessons l 
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.teacher_id = $1 AND l.status = 'active' 
			AND l.start_time > NOW() AND l.soft_deleted = false`,
		teacherID)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения уроков преподавателя")
		return
	}
	
	var lessonIDs []int
	var lessonInfo []string
	
	for rows.Next() {
		var lessonID int
		var startTime string
		var subjectName string
		
		if err := rows.Scan(&lessonID, &startTime, &subjectName); err != nil {
			continue
		}
		
		lessonIDs = append(lessonIDs, lessonID)
		lessonInfo = append(lessonInfo, fmt.Sprintf("📅 %s - %s", startTime[:16], subjectName))
	}
	rows.Close()
	
	if len(lessonIDs) == 0 {
		sendMessage(bot, message.Chat.ID, "ℹ️ У преподавателя нет активных уроков для отмены")
	}
	
	// 2. Soft-delete всех уроков преподавателя
	for _, lessonID := range lessonIDs {
		_, err = tx.Exec(`
			UPDATE lessons 
			SET status = 'cancelled', soft_deleted = true, updated_at = NOW()
			WHERE id = $1`, lessonID)
		
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены урока ID "+strconv.Itoa(lessonID))
			return
		}
		
		// Отменяем все записи студентов на урок
		_, err = tx.Exec(`
			UPDATE enrollments 
			SET status = 'cancelled'
			WHERE lesson_id = $1 AND status = 'enrolled'`, lessonID)
		
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены записей на урок ID "+strconv.Itoa(lessonID))
			return
		}
		
		// Очищаем лист ожидания для урока
		_, err = tx.Exec(`DELETE FROM waitlist WHERE lesson_id = $1`, lessonID)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка очистки листа ожидания для урока ID "+strconv.Itoa(lessonID))
			return
		}
	}
	
	// 3. Деактивируем преподавателя
	_, err = tx.Exec(`
		UPDATE users 
		SET is_active = false, updated_at = NOW()
		WHERE id = $1`, userID_teacher)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка деактивации преподавателя")
		return
	}
	
	// 4. Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения изменений")
		return
	}
	
	// 5. Уведомляем всех пострадавших студентов
	if len(lessonIDs) > 0 {
		go notifyStudentsAboutTeacherDeletion(bot, db, lessonIDs, teacherName)
	}
	
	// 6. Уведомляем самого преподавателя
	teacherNotification := "❌ **Ваш доступ к системе приостановлен**\n\n" +
		"Ваша учетная запись преподавателя была деактивирована администратором.\n" +
		"Все ваши активные уроки отменены.\n\n" +
		"По вопросам восстановления обращайтесь к администратору."
	
	teacherMsg := tgbotapi.NewMessage(teacherTelegramID, teacherNotification)
	teacherMsg.ParseMode = "Markdown"
	bot.Send(teacherMsg)
	
	// 7. Отчет администратору
	reportText := "✅ **Преподаватель успешно удален**\n\n" +
		"👨‍🏫 Преподаватель: " + teacherName + "\n" +
		"🆔 ID: " + teacherIDStr + "\n" +
		"📚 Отменено уроков: " + strconv.Itoa(len(lessonIDs)) + "\n\n"
	
	if len(lessonInfo) > 0 {
		reportText += "**Отмененные уроки:**\n" + strings.Join(lessonInfo, "\n") + "\n\n"
	}
	
	reportText += "📢 Уведомления отправлены всем пострадавшим студентам\n" +
		"🔒 Преподаватель деактивирован\n" +
		"⏳ Лист ожидания очищен"
		
	msg := tgbotapi.NewMessage(message.Chat.ID, reportText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

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
	
	// Парсинг аргументов
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "� **Отмена урока с уведомлением**\n\n" +
			"**Формат:** `/cancel_with_notification <lesson_id> <причина>`\n\n" +
			"**Примеры:**\n" +
			"• `/cancel_with_notification 15 Болезнь преподавателя`\n" +
			"• `/cancel_with_notification 22 Технические проблемы`\n\n" +
			"**Действия:**\n" +
			"• Урок будет отменен\n" +
			"• Студенты получат уведомление с причиной\n" +
			"• Лист ожидания будет очищен"
		
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
	
	reason := strings.Join(args[2:], " ")
	
	// Проверяем и отменяем урок
	if cancelLessonWithNotification(bot, db, lessonID, reason, role, userID) {
		sendMessage(bot, message.Chat.ID, "✅ Урок отменен, студенты уведомлены")
	}
}

// Перенос урока с уведомлением (Шаг 8.2 ROADMAP)
func handleRescheduleWithNotifyCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || (role != "admin" && role != "teacher") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для переноса уроков")
		return
	}
	
	// Парсинг аргументов
	args := strings.Fields(message.Text)
	if len(args) < 3 {
		helpText := "🔄 **Перенос урока с уведомлением**\n\n" +
			"**Формат:** `/reschedule_with_notify <lesson_id> <новое_время>`\n\n" +
			"**Примеры:**\n" +
			"• `/reschedule_with_notify 15 2025-08-15 14:00`\n" +
			"• `/reschedule_with_notify 22 завтра 16:00`\n\n" +
			"**Действия:**\n" +
			"• Время урока будет изменено\n" +
			"• Студенты получат уведомление о переносе\n" +
			"• Записи останутся актуальными"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	lessonIDStr := args[1]
	newTimeStr := strings.Join(args[2:], " ")
	
	// Преобразование lessonID в int
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока: "+lessonIDStr)
		return
	}
	
	// Парсинг нового времени
	newStartTime, err := parseTimeInput(newTimeStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный формат времени: "+err.Error())
		return
	}
	
	// Проверяем существование урока и права доступа
	var currentLessonData struct {
		ID          int
		SubjectName string
		TeacherName string
		StartTime   string
		TeacherID   int
	}
	
	query := `
		SELECT l.id, s.name as subject_name, u.full_name as teacher_name, 
			   l.start_time, t.user_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		JOIN teachers t ON l.teacher_id = t.id
		JOIN users u ON t.user_id = u.id
		WHERE l.id = $1 AND l.is_active = true`
	
	err = db.QueryRow(query, lessonID).Scan(
		&currentLessonData.ID,
		&currentLessonData.SubjectName,
		&currentLessonData.TeacherName,
		&currentLessonData.StartTime,
		&currentLessonData.TeacherID,
	)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок не найден или уже отменен")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка доступа к данным урока")
		return
	}
	
	// Проверка прав (учитель может переносить только свои уроки)
	if role == "teacher" {
		var currentUserID int
		err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userID).Scan(&currentUserID)
		if err != nil || currentUserID != currentLessonData.TeacherID {
			sendMessage(bot, message.Chat.ID, "❌ Вы можете переносить только свои уроки")
			return
		}
	}
	
	// Обновляем время урока
	_, err = db.Exec("UPDATE lessons SET start_time = $1 WHERE id = $2", newStartTime, lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка при переносе урока")
		return
	}
	
	// Уведомляем студентов о переносе
	sent, failed := notifyStudentsOfLesson(bot, db, lessonID, 
		"🔄 **ПЕРЕНОС УРОКА**\n\n"+
		"📚 Предмет: "+currentLessonData.SubjectName+"\n"+
		"👨‍🏫 Преподаватель: "+currentLessonData.TeacherName+"\n\n"+
		"⏰ **Старое время:** "+currentLessonData.StartTime[:16]+"\n"+
		"🕒 **НОВОЕ ВРЕМЯ:** "+newStartTime.Format("2006-01-02 15:04")+"\n\n"+
		"✅ Ваша запись остается активной",
		currentLessonData.SubjectName, currentLessonData.TeacherName, currentLessonData.StartTime)
	
	// Отчет администратору/учителю
	reportText := fmt.Sprintf(
		"✅ **Урок успешно перенесен**\n\n"+
		"🆔 ID урока: %d\n"+
		"📚 Предмет: %s\n"+
		"👨‍🏫 Преподаватель: %s\n\n"+
		"⏰ Было: %s\n"+
		"🕒 Стало: %s\n\n"+
		"📨 Уведомлено студентов: %d\n"+
		"❌ Ошибок доставки: %d",
		lessonID, currentLessonData.SubjectName, currentLessonData.TeacherName,
		currentLessonData.StartTime[:16], newStartTime.Format("2006-01-02 15:04"),
		sent, failed)
	
	sendMessage(bot, message.Chat.ID, reportText)
}

// Список преподавателей (Шаг 8.1 ROADMAP)
func handleListTeachersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра списка преподавателей")
		return
	}
	
	// Получаем всех преподавателей
	rows, err := db.Query(`
		SELECT t.id, u.full_name, u.tg_id, u.is_active,
			COUNT(l.id) as lessons_count
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN lessons l ON t.id = l.teacher_id AND l.status = 'active' 
			AND l.start_time > NOW() AND l.soft_deleted = false
		GROUP BY t.id, u.full_name, u.tg_id, u.is_active
		ORDER BY u.is_active DESC, u.full_name`)
		
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка преподавателей")
		return
	}
	defer rows.Close()
	
	var teachersText strings.Builder
	teachersText.WriteString("👥 **Список преподавателей**\n\n")
	
	teacherCount := 0
	for rows.Next() {
		var teacherID int
		var name string
		var telegramID int64
		var isActive bool
		var lessonsCount int
		
		if err := rows.Scan(&teacherID, &name, &telegramID, &isActive, &lessonsCount); err != nil {
			continue
		}
		
		teacherCount++
		statusIcon := "✅"
		if !isActive {
			statusIcon = "❌"
		}
		
		teachersText.WriteString(fmt.Sprintf("%s **%s**\n", statusIcon, name))
		teachersText.WriteString(fmt.Sprintf("   🆔 ID: %d | TG: %d\n", teacherID, telegramID))
		teachersText.WriteString(fmt.Sprintf("   📚 Активных уроков: %d\n\n", lessonsCount))
	}
	
	if teacherCount == 0 {
		teachersText.WriteString("Пока нет зарегистрированных преподавателей")
	} else {
		teachersText.WriteString(fmt.Sprintf("**Всего преподавателей:** %d", teacherCount))
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, teachersText.String())
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
func cancelLessonWithNotification(bot *tgbotapi.BotAPI, db *sql.DB, lessonID int, reason, userRole string, userID int64) bool {
	// Проверяем урок
	var subjectName, teacherName, startTime string
	var teacherTelegramID int64
	err := db.QueryRow(`
		SELECT s.name, ut.full_name, l.start_time::text, ut.tg_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users ut ON t.user_id = ut.id
		WHERE l.id = $1 AND l.status = 'active' AND l.soft_deleted = false`,
		lessonID).Scan(&subjectName, &teacherName, &startTime, &teacherTelegramID)
		
	if err != nil {
		return false
	}
	
	// Для учителей - проверяем, что это их урок
	if userRole == "teacher" {
		var checkTeacherID int64
		err = db.QueryRow(`
			SELECT ut.tg_id
			FROM lessons l
			JOIN teachers t ON l.teacher_id = t.id
			JOIN users ut ON t.user_id = ut.id
			WHERE l.id = $1`, lessonID).Scan(&checkTeacherID)
			
		if err != nil || checkTeacherID != userID {
			return false
		}
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()
	
	// Отменяем урок
	_, err = tx.Exec("UPDATE lessons SET status = 'cancelled' WHERE id = $1", lessonID)
	if err != nil {
		return false
	}
	
	// Отменяем записи
	_, err = tx.Exec("UPDATE enrollments SET status = 'cancelled' WHERE lesson_id = $1", lessonID)
	if err != nil {
		return false
	}
	
	// Очищаем лист ожидания
	_, err = tx.Exec("DELETE FROM waitlist WHERE lesson_id = $1", lessonID)
	if err != nil {
		return false
	}
	
	if err = tx.Commit(); err != nil {
		return false
	}
	
	// Уведомляем студентов
	notifyStudentsAboutCancellationWithReason(bot, db, lessonID, subjectName, teacherName, startTime, reason)
	
	return true
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

// Просмотр всех студентов (обновленная версия)
func handleMyStudentsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра студентов")
		return
	}
	
	// Получаем всех студентов
	rows, err := db.Query(`
		SELECT u.full_name, u.phone, u.created_at,
			COUNT(e.id) as enrollments_count
		FROM users u
		LEFT JOIN students s ON u.id = s.user_id
		LEFT JOIN enrollments e ON s.id = e.student_id AND e.status = 'enrolled'
		WHERE u.role = 'student' AND u.is_active = true
		GROUP BY u.id, u.full_name, u.phone, u.created_at
		ORDER BY u.created_at DESC`)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка студентов")
		return
	}
	defer rows.Close()
	
	var studentsText strings.Builder
	studentsText.WriteString("👥 **Список всех студентов**\n\n")
	
	studentCount := 0
	for rows.Next() {
		var name, phone string
		var createdAt string
		var enrollmentsCount int
		
		if err := rows.Scan(&name, &phone, &createdAt, &enrollmentsCount); err != nil {
			continue
		}
		
		studentCount++
		studentsText.WriteString(fmt.Sprintf("%d. **%s**", studentCount, name))
		if phone != "" {
			studentsText.WriteString(fmt.Sprintf(" (%s)", phone))
		}
		studentsText.WriteString(fmt.Sprintf("\n   📚 Активных записей: %d\n", enrollmentsCount))
		studentsText.WriteString(fmt.Sprintf("   📅 Регистрация: %s\n\n", createdAt[:10]))
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

// Уведомление студентов об удалении преподавателя (Шаг 8.1 ROADMAP)
func notifyStudentsAboutTeacherDeletion(bot *tgbotapi.BotAPI, db *sql.DB, lessonIDs []int, teacherName string) {
	if len(lessonIDs) == 0 {
		return
	}
	
	// Создаем строку с ID уроков для SQL запроса
	lessonIDsStr := make([]string, len(lessonIDs))
	for i, id := range lessonIDs {
		lessonIDsStr[i] = strconv.Itoa(id)
	}
	
	// Получаем всех студентов, записанных на отмененные уроки
	query := fmt.Sprintf(`
		SELECT DISTINCT u.tg_id, u.full_name,
			STRING_AGG(TO_CHAR(l.start_time, 'DD.MM.YYYY HH24:MI') || ' - ' || s.name, E'\n') as lessons_info
		FROM enrollments e
		JOIN students st ON e.student_id = st.id
		JOIN users u ON st.user_id = u.id
		JOIN lessons l ON e.lesson_id = l.id
		JOIN subjects s ON l.subject_id = s.id
		WHERE e.lesson_id IN (%s) AND e.status = 'enrolled'
		GROUP BY u.tg_id, u.full_name`, 
		strings.Join(lessonIDsStr, ","))
	
	rows, err := db.Query(query)
	if err != nil {
		return // Не удалось получить студентов
	}
	defer rows.Close()
	
	var sentCount int
	for rows.Next() {
		var studentTelegramID int64
		var studentName, lessonsInfo string
		
		if err := rows.Scan(&studentTelegramID, &studentName, &lessonsInfo); err != nil {
			continue
		}
		
		// Формируем уведомление студенту
		notificationText := "❌ **Отмена уроков**\n\n" +
			"К сожалению, ваши уроки были отменены в связи с приостановкой работы преподавателя **" + teacherName + "**.\n\n" +
			"**Отмененные уроки:**\n" + lessonsInfo + "\n\n" +
			"💔 Приносим извинения за неудобства.\n" +
			"📞 Для получения дополнительной информации обратитесь к администрации.\n\n" +
			"🔄 Вы можете записаться на уроки других преподавателей командой /schedule"
		
		msg := tgbotapi.NewMessage(studentTelegramID, notificationText)
		msg.ParseMode = "Markdown"
		
		if _, err := bot.Send(msg); err == nil {
			sentCount++
		}
	}
}

// parseTimeInput - парсинг времени из различных форматов
func parseTimeInput(input string) (time.Time, error) {
	now := time.Now()
	input = strings.ToLower(strings.TrimSpace(input))
	
	// Регулярные выражения для разных форматов
	patterns := []struct {
		regex string
		parse func([]string) (time.Time, error)
	}{
		// Полный формат: 2025-08-15 14:00
		{
			regex: `^(\d{4})-(\d{1,2})-(\d{1,2})\s+(\d{1,2}):(\d{2})$`,
			parse: func(matches []string) (time.Time, error) {
				year, _ := strconv.Atoi(matches[1])
				month, _ := strconv.Atoi(matches[2])
				day, _ := strconv.Atoi(matches[3])
				hour, _ := strconv.Atoi(matches[4])
				minute, _ := strconv.Atoi(matches[5])
				return time.Date(year, time.Month(month), day, hour, minute, 0, 0, now.Location()), nil
			},
		},
		// Краткий формат: завтра 16:00, сегодня 14:30
		{
			regex: `^(завтра|сегодня|послезавтра)\s+(\d{1,2}):(\d{2})$`,
			parse: func(matches []string) (time.Time, error) {
				hour, _ := strconv.Atoi(matches[2])
				minute, _ := strconv.Atoi(matches[3])
				
				var targetDate time.Time
				switch matches[1] {
				case "сегодня":
					targetDate = now
				case "завтра":
					targetDate = now.AddDate(0, 0, 1)
				case "послезавтра":
					targetDate = now.AddDate(0, 0, 2)
				}
				
				return time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 
					hour, minute, 0, 0, now.Location()), nil
			},
		},
		// Формат дд.мм чч:мм
		{
			regex: `^(\d{1,2})\.(\d{1,2})\s+(\d{1,2}):(\d{2})$`,
			parse: func(matches []string) (time.Time, error) {
				day, _ := strconv.Atoi(matches[1])
				month, _ := strconv.Atoi(matches[2])
				hour, _ := strconv.Atoi(matches[3])
				minute, _ := strconv.Atoi(matches[4])
				
				year := now.Year()
				// Если дата в прошлом, берем следующий год
				targetDate := time.Date(year, time.Month(month), day, hour, minute, 0, 0, now.Location())
				if targetDate.Before(now) {
					targetDate = targetDate.AddDate(1, 0, 0)
				}
				
				return targetDate, nil
			},
		},
	}
	
	// Пробуем каждый паттерн
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern.regex)
		if matches := re.FindStringSubmatch(input); matches != nil {
			return pattern.parse(matches)
		}
	}
	
	return time.Time{}, fmt.Errorf("неподдерживаемый формат времени. Используйте: '2025-08-15 14:00', 'завтра 16:00', '15.08 14:00'")
}

// ========================= БЕЛОЕ ПЯТНО #1: КОМАНДЫ ВОССТАНОВЛЕНИЯ =========================

// handleRestoreLessonCommand - восстановление отмененного урока
func handleRestoreLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для восстановления уроков")
		return
	}
	
	// Парсинг аргументов
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🔄 **Восстановление отмененного урока**\n\n" +
			"**Формат:** `/restore_lesson <lesson_id>`\n\n" +
			"**Пример:** `/restore_lesson 15`\n\n" +
			"**Действия:**\n" +
			"• Урок становится активным\n" +
			"• Все студенты получают уведомление о восстановлении\n" +
			"• Возобновляется возможность записи\n\n" +
			"⚠️ **Внимание:** убедитесь в отсутствии конфликтов с расписанием!"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	lessonIDStr := args[1]
	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID урока: "+lessonIDStr)
		return
	}
	
	// Проверяем существование урока и его текущий статус
	var lessonData struct {
		ID          int
		SubjectName string
		TeacherName string
		StartTime   string
		IsActive    bool
		TeacherID   int
	}
	
	query := `
		SELECT l.id, s.name as subject_name, u.full_name as teacher_name, 
			   l.start_time, l.is_active, t.id as teacher_id
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		JOIN teachers t ON l.teacher_id = t.id
		JOIN users u ON t.user_id = u.id
		WHERE l.id = $1`
	
	err = db.QueryRow(query, lessonID).Scan(
		&lessonData.ID,
		&lessonData.SubjectName,
		&lessonData.TeacherName,
		&lessonData.StartTime,
		&lessonData.IsActive,
		&lessonData.TeacherID,
	)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Урок с ID "+lessonIDStr+" не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка доступа к данным урока")
		return
	}
	
	if lessonData.IsActive {
		sendMessage(bot, message.Chat.ID, "✅ Урок уже активен, восстановление не требуется")
		return
	}
	
	// Проверяем, активен ли преподаватель
	var teacherActive bool
	err = db.QueryRow("SELECT u.is_active FROM users u JOIN teachers t ON u.id = t.user_id WHERE t.id = $1", 
		lessonData.TeacherID).Scan(&teacherActive)
	
	if err != nil || !teacherActive {
		sendMessage(bot, message.Chat.ID, "❌ Невозможно восстановить урок - преподаватель деактивирован")
		return
	}
	
	// Проверяем конфликты времени (нет ли активного урока в это же время у этого преподавателя)
	var conflictCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM lessons 
		WHERE teacher_id = $1 AND start_time = $2 AND is_active = true AND id != $3`,
		lessonData.TeacherID, lessonData.StartTime, lessonID).Scan(&conflictCount)
	
	if err == nil && conflictCount > 0 {
		sendMessage(bot, message.Chat.ID, 
			"⚠️ **Конфликт расписания!**\n\n"+
			"У преподавателя **"+lessonData.TeacherName+"** уже есть активный урок "+
			"в это время: "+lessonData.StartTime[:16]+"\n\n"+
			"Восстановление невозможно без решения конфликта.")
		return
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
		return
	}
	defer tx.Rollback()
	
	// Восстанавливаем урок
	_, err = tx.Exec("UPDATE lessons SET is_active = true WHERE id = $1", lessonID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления урока")
		return
	}
	
	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения изменений")
		return
	}
	
	// Уведомляем всех ранее записанных студентов
	sent, failed := notifyPreviouslyEnrolledStudents(bot, db, lessonID, lessonData)
	
	// Отчет администратору
	reportText := fmt.Sprintf(
		"✅ **Урок успешно восстановлен**\n\n"+
		"🆔 ID урока: %d\n"+
		"📚 Предмет: %s\n"+
		"👨‍🏫 Преподаватель: %s\n"+
		"📅 Время: %s\n\n"+
		"📨 Уведомлено студентов: %d\n"+
		"❌ Ошибок доставки: %d\n\n"+
		"🔄 Урок снова доступен для записи",
		lessonID, lessonData.SubjectName, lessonData.TeacherName,
		lessonData.StartTime[:16], sent, failed)
	
	sendMessage(bot, message.Chat.ID, reportText)
}

// handleRestoreTeacherCommand - восстановление деактивированного преподавателя
func handleRestoreTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)
	
	if err != nil || role != "admin" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для восстановления преподавателей")
		return
	}
	
	// Парсинг аргументов
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🔄 **Восстановление преподавателя**\n\n" +
			"**Формат:** `/restore_teacher <teacher_id>`\n\n" +
			"**Пример:** `/restore_teacher 5`\n\n" +
			"**Действия:**\n" +
			"• Преподаватель становится активным\n" +
			"• Все его будущие уроки восстанавливаются\n" +
			"• Студенты получают уведомления о возобновлении уроков\n\n" +
			"💡 Используйте `/list_teachers` для просмотра ID преподавателей"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	teacherIDStr := args[1]
	teacherID, err := strconv.Atoi(teacherIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID преподавателя: "+teacherIDStr)
		return
	}
	
	// Проверяем существование преподавателя и его статус
	var teacherData struct {
		ID       int
		Name     string
		IsActive bool
		UserID   int
	}
	
	query := `
		SELECT t.id, u.full_name, u.is_active, u.id as user_id
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE t.id = $1`
	
	err = db.QueryRow(query, teacherID).Scan(
		&teacherData.ID,
		&teacherData.Name,
		&teacherData.IsActive,
		&teacherData.UserID,
	)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель с ID "+teacherIDStr+" не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка доступа к данным преподавателя")
		return
	}
	
	if teacherData.IsActive {
		sendMessage(bot, message.Chat.ID, "✅ Преподаватель **"+teacherData.Name+"** уже активен")
		return
	}
	
	// Получаем количество отмененных уроков преподавателя
	var canceledLessonsCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM lessons 
		WHERE teacher_id = $1 AND is_active = false AND start_time > NOW()`,
		teacherID).Scan(&canceledLessonsCount)
	
	if err != nil {
		canceledLessonsCount = 0
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
		return
	}
	defer tx.Rollback()
	
	// Восстанавливаем преподавателя
	_, err = tx.Exec("UPDATE users SET is_active = true WHERE id = $1", teacherData.UserID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления преподавателя")
		return
	}
	
	// Восстанавливаем все его будущие уроки
	result, err := tx.Exec(`
		UPDATE lessons SET is_active = true 
		WHERE teacher_id = $1 AND is_active = false AND start_time > NOW()`,
		teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления уроков преподавателя")
		return
	}
	
	restoredLessons, _ := result.RowsAffected()
	
	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения изменений")
		return
	}
	
	// Уведомляем студентов о восстановлении всех уроков
	sent, failed := notifyStudentsAboutTeacherRestoration(bot, db, teacherID, teacherData.Name)
	
	// Отчет администратору
	reportText := fmt.Sprintf(
		"✅ **Преподаватель успешно восстановлен**\n\n"+
		"🆔 ID преподавателя: %d\n"+
		"👨‍🏫 Имя: %s\n\n"+
		"📚 Восстановлено уроков: %d из %d\n"+
		"📨 Уведомлено студентов: %d\n"+
		"❌ Ошибок доставки: %d\n\n"+
		"🎯 Преподаватель может снова проводить занятия",
		teacherID, teacherData.Name, restoredLessons, canceledLessonsCount, sent, failed)
	
	sendMessage(bot, message.Chat.ID, reportText)
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

// notifyStudentsAboutTeacherRestoration - уведомление о восстановлении преподавателя
func notifyStudentsAboutTeacherRestoration(bot *tgbotapi.BotAPI, db *sql.DB, teacherID int, teacherName string) (int, int) {
	// Получаем всех студентов с активными записями к этому преподавателю
	query := `
		SELECT DISTINCT u.tg_id, u.full_name,
			STRING_AGG(
				s.name || ' (' || l.start_time::date || ' ' || l.start_time::time || ')',
				E'\n• '
			) as lessons_info
		FROM enrollments e
		JOIN students st ON e.student_id = st.id
		JOIN users u ON st.user_id = u.id
		JOIN lessons l ON e.lesson_id = l.id
		JOIN subjects s ON l.subject_id = s.id
		WHERE l.teacher_id = $1 AND l.is_active = true AND l.start_time > NOW()
			AND e.status = 'enrolled'
		GROUP BY u.tg_id, u.full_name`
	
	rows, err := db.Query(query, teacherID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	
	sent := 0
	failed := 0
	
	for rows.Next() {
		var studentTelegramID int64
		var studentName, lessonsInfo string
		
		if err := rows.Scan(&studentTelegramID, &studentName, &lessonsInfo); err != nil {
			failed++
			continue
		}
		
		// Формируем уведомление
		notificationText := "🎉 **ОТЛИЧНЫЕ НОВОСТИ!**\n\n" +
			"Преподаватель **" + teacherName + "** возобновляет работу!\n\n" +
			"📚 **Ваши восстановленные уроки:**\n" +
			"• " + lessonsInfo + "\n\n" +
			"✅ Все ваши записи остаются активными\n" +
			"🎯 Ждем вас на занятиях!"
		
		msg := tgbotapi.NewMessage(studentTelegramID, notificationText)
		msg.ParseMode = "Markdown"
		
		// Retry механизм
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

// ========================= БЕЛОЕ ПЯТНО #3: RATE LIMITING СТАТИСТИКА =========================

// handleRateLimitStatsCommand - показывает статистику rate limiting
func handleRateLimitStatsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	if GlobalRateLimiter == nil {
		sendMessage(bot, message.Chat.ID, "❌ Rate limiter не инициализирован")
		return
	}
	
	// Получаем количество активных операций
	activeCount, err := GlobalRateLimiter.GetActiveOperationsCount()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения статистики rate limiting")
		return
	}
	
	// Получаем детальную статистику по типам операций
	stats, err := getRateLimitDetailedStats(db)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения детальной статистики")
		return
	}
	
	// Формируем отчет
	report := "📊 **СТАТИСТИКА RATE LIMITING**\n\n"
	report += fmt.Sprintf("🔄 **Активных операций:** %d\n\n", activeCount)
	
	if len(stats) > 0 {
		report += "📈 **По типам операций:**\n"
		for _, stat := range stats {
			operationName := getOperationName(stat.Operation)
			report += fmt.Sprintf("• %s: %d\n", operationName, stat.Count)
		}
	} else {
		report += "✅ Нет активных операций\n"
	}
	
	report += "\n⏰ **Таймаут:** 5 минут\n"
	report += "🧹 **Автоочистка:** каждые 2 минуты"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, report)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// getRateLimitDetailedStats - получает детальную статистику rate limiting
func getRateLimitDetailedStats(db *sql.DB) ([]RateLimitStat, error) {
	query := `
		SELECT operation, COUNT(*) as count
		FROM pending_operations 
		WHERE created_at > NOW() - INTERVAL '5 minutes'
		GROUP BY operation
		ORDER BY count DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []RateLimitStat
	for rows.Next() {
		var stat RateLimitStat
		if err := rows.Scan(&stat.Operation, &stat.Count); err != nil {
			continue
		}
		stats = append(stats, stat)
	}
	
	return stats, nil
}

// RateLimitStat - структура для статистики rate limiting
type RateLimitStat struct {
	Operation string
	Count     int
}

// ========================= БЕЛОЕ ПЯТНО #4: ОБЩАЯ СТАТИСТИКА =========================

// handleStatsCommand - показывает общую статистику системы
func handleStatsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	stats, err := getBasicSystemStats(db)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения статистики системы")
		return
	}
	
	// Формируем отчет
	report := "📊 **СТАТИСТИКА СИСТЕМЫ**\n\n"
	report += fmt.Sprintf("📚 **Активных уроков:** %d\n", stats.ActiveLessons)
	report += fmt.Sprintf("👥 **Всего записей:** %d\n", stats.TotalEnrollments)
	report += fmt.Sprintf("⏳ **В листе ожидания:** %d\n", stats.WaitlistCount)
	report += fmt.Sprintf("👨‍🏫 **Активных преподавателей:** %d\n", stats.ActiveTeachers)
	report += fmt.Sprintf("🎓 **Активных студентов:** %d\n", stats.ActiveStudents)
	
	// Добавляем статистику rate limiting если доступна
	if GlobalRateLimiter != nil {
		if activeOps, err := GlobalRateLimiter.GetActiveOperationsCount(); err == nil {
			report += fmt.Sprintf("🔄 **Активных операций:** %d\n", activeOps)
		}
	}
	
	msg := tgbotapi.NewMessage(message.Chat.ID, report)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// getBasicSystemStats - получает базовую статистику системы
func getBasicSystemStats(db *sql.DB) (*BasicSystemStats, error) {
	stats := &BasicSystemStats{}
	
	// Активные уроки
	err := db.QueryRow(`
		SELECT COUNT(*) FROM lessons 
		WHERE start_time > NOW() AND status = 'active' AND soft_deleted = false`).Scan(&stats.ActiveLessons)
	if err != nil {
		return nil, err
	}
	
	// Всего записей
	err = db.QueryRow(`
		SELECT COUNT(*) FROM enrollments 
		WHERE status = 'enrolled'`).Scan(&stats.TotalEnrollments)
	if err != nil {
		return nil, err
	}
	
	// В листе ожидания
	err = db.QueryRow(`
		SELECT COUNT(*) FROM waitlist`).Scan(&stats.WaitlistCount)
	if err != nil {
		return nil, err
	}
	
	// Активные преподаватели
	err = db.QueryRow(`
		SELECT COUNT(*) FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE u.is_active = true`).Scan(&stats.ActiveTeachers)
	if err != nil {
		return nil, err
	}
	
	// Активные студенты
	err = db.QueryRow(`
		SELECT COUNT(*) FROM students s
		JOIN users u ON s.user_id = u.id
		WHERE u.is_active = true`).Scan(&stats.ActiveStudents)
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// BasicSystemStats - структура для базовой статистики системы
type BasicSystemStats struct {
	ActiveLessons    int
	TotalEnrollments int
	WaitlistCount    int
	ActiveTeachers   int
	ActiveStudents   int
}
