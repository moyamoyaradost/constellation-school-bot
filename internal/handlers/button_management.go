package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ================ КНОПОЧНОЕ УПРАВЛЕНИЕ ПРЕПОДАВАТЕЛЯМИ ================

// Обработчик меню управления преподавателями
func handleTeachersMenuButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Проверяем права администратора
	userID := message.From.ID
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
	if err != nil || role != "superuser" {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для управления преподавателями")
		return
	}
	
	// Показываем меню управления преподавателями
	text := "👨‍🏫 **Управление преподавателями**\n\n" +
		"Выберите действие:"
	
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 Список преподавателей", "list_teachers"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить преподавателя", "delete_teacher_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔄 Восстановить преподавателя", "restore_teacher_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "main_menu"),
		},
	}
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// Показать список преподавателей для удаления с кнопками
func showDeleteTeacherButtons(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	rows, err := db.Query(`
		SELECT t.id, u.full_name, 
			(SELECT COUNT(*) FROM lessons WHERE teacher_id = t.id AND soft_deleted = false AND start_time > NOW()) as active_lessons
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE t.soft_deleted = false
		ORDER BY u.full_name`)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка преподавателей")
		return
	}
	defer rows.Close()
	
	var buttons [][]tgbotapi.InlineKeyboardButton
	count := 0
	
	for rows.Next() {
		var teacherID int
		var fullName string
		var activeLessons int
		
		if err := rows.Scan(&teacherID, &fullName, &activeLessons); err != nil {
			continue
		}
		
		count++
		buttonText := fmt.Sprintf("👨‍🏫 %s (📚%d)", fullName, activeLessons)
		callbackData := fmt.Sprintf("confirm_delete_teacher_%d", teacherID)
		
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	
	if count == 0 {
		sendMessage(bot, message.Chat.ID, "❌ Нет преподавателей для удаления")
		return
	}
	
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "teachers"),
	})
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	text := "🗑️ **Выберите преподавателя для удаления:**\n\n" +
		"ℹ️ Цифра показывает активные уроки"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// Показать список удаленных преподавателей для восстановления
func showRestoreTeacherButtons(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	rows, err := db.Query(`
		SELECT t.id, u.full_name, t.updated_at
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE t.soft_deleted = true
		ORDER BY t.updated_at DESC`)
	
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка получения списка удаленных преподавателей")
		return
	}
	defer rows.Close()
	
	var buttons [][]tgbotapi.InlineKeyboardButton
	count := 0
	
	for rows.Next() {
		var teacherID int
		var fullName string
		var updatedAt time.Time
		
		if err := rows.Scan(&teacherID, &fullName, &updatedAt); err != nil {
			continue
		}
		
		count++
		buttonText := fmt.Sprintf("👨‍🏫 %s (%s)", fullName, updatedAt.Format("02.01"))
		callbackData := fmt.Sprintf("restore_teacher_%d", teacherID)
		
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{button})
	}
	
	if count == 0 {
		sendMessage(bot, message.Chat.ID, "❌ Нет удаленных преподавателей")
		return
	}
	
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "teachers"),
	})
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	text := "🔄 **Выберите преподавателя для восстановления:**"
	
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// Подтверждение удаления преподавателя
func handleConfirmDeleteTeacher(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	parts := strings.Split(query.Data, "_")
	if len(parts) != 4 {
		return
	}
	
	teacherID, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}
	
	// Получаем информацию о преподавателе
	var fullName string
	var activeLessons int
	err = db.QueryRow(`
		SELECT u.full_name,
			(SELECT COUNT(*) FROM lessons WHERE teacher_id = $1 AND soft_deleted = false AND start_time > NOW()) as active_lessons
		FROM teachers t
		JOIN users u ON t.user_id = u.id
		WHERE t.id = $1`, teacherID).Scan(&fullName, &activeLessons)
	
	if err != nil {
		sendMessage(bot, query.Message.Chat.ID, "❌ Преподаватель не найден")
		return
	}
	
	confirmText := fmt.Sprintf("⚠️ **Подтверждение удаления**\n\n"+
		"👨‍🏫 **Преподаватель:** %s\n"+
		"📚 **Активных уроков:** %d\n\n"+
		"❗️ При удалении все уроки будут отменены!\n"+
		"Продолжить?", fullName, activeLessons)
	
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Удалить", fmt.Sprintf("execute_delete_teacher_%d", teacherID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "delete_teacher_menu"),
		},
	}
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, confirmText)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	bot.Send(editMsg)
}

// Выполнение удаления преподавателя
func handleExecuteDeleteTeacher(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	parts := strings.Split(query.Data, "_")
	if len(parts) != 4 {
		return
	}
	
	teacherID, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}
	
	// Создаем временное сообщение для вызова существующей функции
	tempMessage := *query.Message
	tempMessage.Text = fmt.Sprintf("/delete_teacher %d", teacherID)
	tempMessage.From = query.From
	
	handleDeleteTeacherCommand(bot, &tempMessage, db)
}

// Восстановление преподавателя
func handleRestoreTeacherAction(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, db *sql.DB) {
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		return
	}
	
	teacherID, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	
	// Создаем временное сообщение для вызова существующей функции
	tempMessage := *query.Message
	tempMessage.Text = fmt.Sprintf("/restore_teacher %d", teacherID)
	tempMessage.From = query.From
	
	handleRestoreTeacherCommand(bot, &tempMessage, db)
}

// ================ КНОПОЧНОЕ УПРАВЛЕНИЕ УРОКАМИ ================

// Обработчик меню управления уроками для админов
func handleAdminLessonsMenuButton(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	text := "📚 **Управление уроками (Администратор)**\n\n" +
		"Выберите действие:"
	
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить урок", "admin_delete_lesson"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 Все уроки", "schedule"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "main_menu"),
		},
	}
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}
