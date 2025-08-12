package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Деактивация студента (отсутствующая команда SuperUser)
func handleDeactivateStudentCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для деактивации студентов")
		return
	}

	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "🚫 **Деактивация студента**\n\n" +
			"**Формат:** `/deactivate_student <user_id>`\n\n" +
			"**Примеры:**\n" +
			"• `/deactivate_student 123456789`\n" +
			"• `/deactivate_student 987654321`\n\n" +
			"**Что произойдет:**\n" +
			"• Деактивация пользователя (is_active = false)\n" +
			"• Отмена всех записей на уроки\n" +
			"• Удаление из листов ожидания\n" +
			"• Логирование действия\n\n" +
			"**См. также:**\n" +
			"• `/activate_student` - активация студента\n" +
			"• `/stats` - статистика пользователей"

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}

	// Получаем user_id
	studentUserIDStr := args[1]
	studentUserID, err := strconv.Atoi(studentUserIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID пользователя")
		return
	}

	// Проверяем, существует ли пользователь
	var fullName, currentRole string
	var isActive bool
	err = db.QueryRow("SELECT full_name, role, is_active FROM users WHERE id = $1", studentUserID).Scan(&fullName, &currentRole, &isActive)

	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска пользователя")
		return
	}

	if currentRole != "student" {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь не является студентом")
		return
	}

	if !isActive {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь уже деактивирован")
		return
	}

	// Получаем количество активных записей
	var activeEnrollments int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM enrollments e
		JOIN students s ON e.student_id = s.id
		WHERE s.user_id = $1 AND e.status = 'enrolled'`, studentUserID).Scan(&activeEnrollments)
	if err != nil {
		activeEnrollments = 0
	}

	// Получаем количество записей в листе ожидания
	var waitlistEntries int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM waitlist w
		JOIN students s ON w.student_id = s.id
		WHERE s.user_id = $1`, studentUserID).Scan(&waitlistEntries)
	if err != nil {
		waitlistEntries = 0
	}

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка начала транзакции")
		return
	}
	defer tx.Rollback()

	// Деактивируем пользователя
	_, err = tx.Exec("UPDATE users SET is_active = false WHERE id = $1", studentUserID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка деактивации пользователя")
		return
	}

	// Отменяем все записи на уроки
	_, err = tx.Exec(`
		UPDATE enrollments 
		SET status = 'cancelled' 
		WHERE student_id IN (
			SELECT id FROM students WHERE user_id = $1
		) AND status = 'enrolled'`, studentUserID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка отмены записей")
		return
	}

	// Удаляем из листов ожидания
	_, err = tx.Exec(`
		DELETE FROM waitlist 
		WHERE student_id IN (
			SELECT id FROM students WHERE user_id = $1
		)`, studentUserID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка удаления из листа ожидания")
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка фиксации транзакции")
		return
	}

	// Логируем деактивацию студента
	LogSystemAction(db, "student_deactivated", fmt.Sprintf("Студент %s (ID: %d) деактивирован, отменено записей: %d, удалено из листа ожидания: %d", fullName, studentUserID, activeEnrollments, waitlistEntries))

	// Отчет администратору
	resultText := "✅ **Студент деактивирован**\n\n" +
		"👤 Студент: " + fullName + "\n" +
		"🆔 ID: " + strconv.Itoa(studentUserID) + "\n\n" +
		"📊 Действия:\n" +
		"• 🚫 Пользователь деактивирован\n" +
		"• 📝 Отменено записей: " + strconv.Itoa(activeEnrollments) + "\n" +
		"• 🗑️ Удалено из листа ожидания: " + strconv.Itoa(waitlistEntries) + "\n\n" +
		"💾 Изменения сохранены в БД"

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Активация студента (отсутствующая команда SuperUser)
func handleActivateStudentCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID

	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", userID).Scan(&role)

	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для активации студентов")
		return
	}

	// Парсинг сообщения
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		helpText := "✅ **Активация студента**\n\n" +
			"**Формат:** `/activate_student <user_id>`\n\n" +
			"**Примеры:**\n" +
			"• `/activate_student 123456789`\n" +
			"• `/activate_student 987654321`\n\n" +
			"**Что произойдет:**\n" +
			"• Активация пользователя (is_active = true)\n" +
			"• Восстановление доступа к боту\n" +
			"• Логирование действия\n\n" +
			"**См. также:**\n" +
			"• `/deactivate_student` - деактивация студента\n" +
			"• `/stats` - статистика пользователей"

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}

	// Получаем user_id
	studentUserIDStr := args[1]
	studentUserID, err := strconv.Atoi(studentUserIDStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Некорректный ID пользователя")
		return
	}

	// Проверяем, существует ли пользователь
	var fullName, currentRole string
	var isActive bool
	err = db.QueryRow("SELECT full_name, role, is_active FROM users WHERE id = $1", studentUserID).Scan(&fullName, &currentRole, &isActive)

	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь не найден")
		return
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка поиска пользователя")
		return
	}

	if currentRole != "student" {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь не является студентом")
		return
	}

	if isActive {
		sendMessage(bot, message.Chat.ID, "❌ Пользователь уже активирован")
		return
	}

	// Активируем пользователя
	_, err = db.Exec("UPDATE users SET is_active = true WHERE id = $1", studentUserID)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка активации пользователя")
		return
	}

	// Логируем активацию студента
	LogSystemAction(db, "student_activated", fmt.Sprintf("Студент %s (ID: %d) активирован", fullName, studentUserID))

	// Отчет администратору
	resultText := "✅ **Студент активирован**\n\n" +
		"👤 Студент: " + fullName + "\n" +
		"🆔 ID: " + strconv.Itoa(studentUserID) + "\n\n" +
		"📊 Действия:\n" +
		"• ✅ Пользователь активирован\n" +
		"• 🔓 Доступ к боту восстановлен\n\n" +
		"💾 Изменения сохранены в БД"

	msg := tgbotapi.NewMessage(message.Chat.ID, resultText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
