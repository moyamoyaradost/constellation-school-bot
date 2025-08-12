package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
		sendMessage(bot, message.Chat.ID, "❌ Ошибка базы данных")
		return
	}
	defer tx.Rollback()
	
	// Создаем пользователя
	var userIDResult int
	err = tx.QueryRow(`
		INSERT INTO users (tg_id, full_name, role, is_active, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id`,
		tgID, firstName+" "+lastName, "teacher", true).Scan(&userIDResult)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания пользователя")
		return
	}
	
	// Создаем запись преподавателя
	_, err = tx.Exec(`
		INSERT INTO teachers (user_id, created_at)
		VALUES ($1, NOW())`,
		userIDResult)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка создания преподавателя")
		return
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения данных")
		return
	}
	
	successText := "✅ **Преподаватель успешно добавлен**\n\n" +
		"👤 **Имя:** " + firstName + " " + lastName + "\n" +
		"🆔 **Telegram ID:** " + tgID + "\n" +
		"📅 **Дата добавления:** " + time.Now().Format("02.01.2006 15:04") + "\n\n" +
		"Преподаватель может начать создавать уроки командой /create_lesson"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, successText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Удаление преподавателя
func handleDeleteTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для удаления преподавателей")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🗑️ **Удаление преподавателя**\n\n" +
			"**Формат:** `/delete_teacher <teacher_id>`\n\n" +
			"**Пример:** `/delete_teacher 5`\n\n" +
			"**Внимание:** Это действие отменит ВСЕ уроки преподавателя и уведомит студентов!\n\n" +
			"**См. список преподавателей:** `/list_teachers`"
		
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
	
	// Получаем информацию о преподавателе
	var teacherName string
	err = db.QueryRow(`
		SELECT u.full_name 
		FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE t.id = $1 AND t.soft_deleted = false`, teacherID).Scan(&teacherName)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска преподавателя")
		return
	}
	
	// Получаем все уроки преподавателя
	rows, err := db.Query(`
		SELECT id FROM lessons 
		WHERE teacher_id = $1 AND soft_deleted = false`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения уроков")
		return
	}
	defer rows.Close()
	
	var lessonIDs []int
	for rows.Next() {
		var lessonID int
		if err := rows.Scan(&lessonID); err != nil {
			continue
		}
		lessonIDs = append(lessonIDs, lessonID)
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка базы данных")
		return
	}
	defer tx.Rollback()
	
	// Soft delete всех уроков преподавателя
	if len(lessonIDs) > 0 {
		_, err = tx.Exec(`
			UPDATE lessons 
			SET soft_deleted = true, updated_at = NOW() 
			WHERE teacher_id = $1 AND soft_deleted = false`, teacherID)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены уроков")
			return
		}
		
		// Очищаем записи на отмененные уроки
		_, err = tx.Exec(`
			UPDATE enrollments 
			SET status = 'cancelled', updated_at = NOW() 
			WHERE lesson_id = ANY($1)`, lessonIDs)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены записей")
			return
		}
		
		// Очищаем листы ожидания
		_, err = tx.Exec(`
			DELETE FROM waitlist 
			WHERE lesson_id = ANY($1)`, lessonIDs)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка очистки листов ожидания")
			return
		}
	}
	
	// Soft delete преподавателя
	_, err = tx.Exec(`
		UPDATE teachers 
		SET soft_deleted = true, updated_at = NOW() 
		WHERE id = $1`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка удаления преподавателя")
		return
	}
	
	// Деактивируем пользователя
	_, err = tx.Exec(`
		UPDATE users 
		SET is_active = false, updated_at = NOW() 
		WHERE id = (SELECT user_id FROM teachers WHERE id = $1)`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка деактивации пользователя")
		return
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения данных")
		return
	}
	
	// Логируем удаление преподавателя
	LogSystemAction(db, "teacher_deleted", fmt.Sprintf("Преподаватель %s (ID: %d) удален, отменено уроков: %d", teacherName, teacherID, len(lessonIDs)))
	
	// Отправляем уведомления студентам в отдельной горутине
	go notifyStudentsAboutTeacherDeletion(bot, db, lessonIDs, teacherName)
	
	// Отчет об удалении
	resultText := "✅ **Преподаватель удален**\n\n" +
		"👤 **Имя:** " + teacherName + "\n" +
		"📊 **Результаты:**\n" +
		"• Отменено уроков: " + strconv.Itoa(len(lessonIDs)) + "\n" +
		"• Очищены листы ожидания\n" +
		"• Деактивирован аккаунт\n\n" +
		"📢 **Уведомления:** Студенты получат сообщения об отмене уроков"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Просмотр списка преподавателей
func handleListTeachersCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для просмотра списка преподавателей")
		return
	}
	
	// Получаем список всех преподавателей
	rows, err := db.Query(`
		SELECT t.id, u.full_name, u.tg_id, u.is_active,
			COUNT(l.id) as active_lessons
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN lessons l ON t.id = l.teacher_id AND l.soft_deleted = false
		WHERE t.soft_deleted = false
		GROUP BY t.id, u.full_name, u.tg_id, u.is_active
		ORDER BY u.full_name`)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка преподавателей")
		return
	}
	defer rows.Close()
	
	var teachersText strings.Builder
	teachersText.WriteString("👨‍🏫 **Список преподавателей**\n\n")
	
	var teacherCount int
	for rows.Next() {
		var id int
		var fullName, tgID string
		var isActive bool
		var activeLessons int
		
		if err := rows.Scan(&id, &fullName, &tgID, &isActive, &activeLessons); err != nil {
			continue
		}
		
		status := "✅ Активен"
		if !isActive {
			status = "❌ Неактивен"
		}
		
		teachersText.WriteString(fmt.Sprintf("**%d.** %s\n", id, fullName))
		teachersText.WriteString(fmt.Sprintf("   🆔 ID: %s\n", tgID))
		teachersText.WriteString(fmt.Sprintf("   📊 Статус: %s\n", status))
		teachersText.WriteString(fmt.Sprintf("   📚 Активных уроков: %d\n\n", activeLessons))
		
		teacherCount++
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

// Восстановление преподавателя
func handleRestoreTeacherCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для восстановления преподавателей")
		return
	}
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🔄 **Восстановление преподавателя**\n\n" +
			"**Формат:** `/restore_teacher <teacher_id>`\n\n" +
			"**Пример:** `/restore_teacher 5`\n\n" +
			"**Что произойдет:**\n" +
			"• Восстановление аккаунта преподавателя\n" +
			"• Восстановление всех уроков\n" +
			"• Уведомления студентов о восстановлении\n\n" +
			"**См. список преподавателей:** `/list_teachers`"
		
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
	
	// Получаем информацию о преподавателе
	var teacherData struct {
		ID   int
		Name string
	}
	err = db.QueryRow(`
		SELECT t.id, u.full_name 
		FROM teachers t 
		JOIN users u ON t.user_id = u.id 
		WHERE t.id = $1`, teacherID).Scan(&teacherData.ID, &teacherData.Name)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Преподаватель не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска преподавателя")
		return
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка базы данных")
		return
	}
	defer tx.Rollback()
	
	// Восстанавливаем преподавателя
	_, err = tx.Exec(`
		UPDATE teachers 
		SET soft_deleted = false, updated_at = NOW() 
		WHERE id = $1`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления преподавателя")
		return
	}
	
	// Активируем пользователя
	_, err = tx.Exec(`
		UPDATE users 
		SET is_active = true, updated_at = NOW() 
		WHERE id = (SELECT user_id FROM teachers WHERE id = $1)`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка активации пользователя")
		return
	}
	
	// Восстанавливаем уроки преподавателя
	_, err = tx.Exec(`
		UPDATE lessons 
		SET soft_deleted = false, updated_at = NOW() 
		WHERE teacher_id = $1 AND soft_deleted = true`, teacherID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка восстановления уроков")
		return
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка сохранения данных")
		return
	}
	
	// Логируем восстановление преподавателя
	LogSystemAction(db, "teacher_restored", fmt.Sprintf("Преподаватель %s (ID: %d) восстановлен", teacherData.Name, teacherID))
	
	// Отправляем уведомления студентам
	sent, failed := notifyStudentsAboutTeacherRestoration(bot, db, teacherID, teacherData.Name)
	
	// Отчет о восстановлении
	resultText := "✅ **Преподаватель восстановлен**\n\n" +
		"👤 **Имя:** " + teacherData.Name + "\n" +
		"📊 **Результаты:**\n" +
		"• Восстановлен аккаунт\n" +
		"• Восстановлены все уроки\n" +
		"• Уведомлено студентов: " + strconv.Itoa(sent) + "\n" +
		"• Ошибок отправки: " + strconv.Itoa(failed) + "\n\n" +
		"🎉 Преподаватель может снова создавать уроки!"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// notifyStudentsAboutTeacherDeletion - уведомление студентов об удалении преподавателя
func notifyStudentsAboutTeacherDeletion(bot *tgbotapi.BotAPI, db *sql.DB, lessonIDs []int, teacherName string) {
	if len(lessonIDs) == 0 {
		return
	}
	
	// Получаем всех студентов с группировкой по студенту
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
		WHERE l.id = ANY($1) AND e.status = 'enrolled'
		GROUP BY u.tg_id, u.full_name`
	
	rows, err := db.Query(query, lessonIDs)
	if err != nil {
		return
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
		notificationText := "❌ **Отмена уроков**\n\n" +
			"К сожалению, преподаватель **" + teacherName + "** больше не работает в школе.\n\n" +
			"📚 **Отмененные уроки:**\n" +
			"• " + lessonsInfo + "\n\n" +
			"💔 Приносим извинения за неудобства.\n" +
			"🔄 Вы можете записаться на другие уроки командой /schedule"
		
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
