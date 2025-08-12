package handlers

import (
	"database/sql"
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
	default:
		sendMessage(bot, message.Chat.ID, "❓ Неизвестная команда преподавателя")
	}
}

// Создание урока
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
	
	// Парсинг аргументов команды
	args := strings.Fields(message.Text)
	if len(args) < 4 {
		helpText := "📝 **Создание урока**\n\n" +
			"**Формат:** `/create_lesson <предмет> <дата> <время>`\n\n" +
			"**Пример:** `/create_lesson 3D-моделирование 15.08.2025 14:30`\n\n" +
			"**Доступные предметы:**\n" +
			"• 3D-моделирование\n" +
			"• Геймдев\n" +
			"• VFX-дизайн\n" +
			"• Графический дизайн\n" +
			"• Веб-разработка\n" +
			"• Компьютерная грамотность"
		
		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return
	}
	
	subjectName := args[1]
	dateStr := args[2]
	timeStr := args[3]
	
	// Парсинг даты и времени
	datetimeStr := dateStr + " " + timeStr
	startTime, err := time.Parse("02.01.2006 15:04", datetimeStr)
	if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Неверный формат даты или времени. Используйте DD.MM.YYYY HH:MM")
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

// Отмена урока  
func handleCancelLessonCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем роль пользователя
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE tg_id = $1", 
		strconv.FormatInt(userID, 10)).Scan(&role)
	
	if err != nil || (role != "teacher" && role != "superuser") {
		sendMessage(bot, message.Chat.ID, "❌ У вас нет прав для отмены уроков")
		return
	}
	
	// Пока заглушка - будет реализована позже
	helpText := "📝 **Отмена урока**\n\n" +
		"Выберите урок для отмены из ваших активных уроков.\n\n" +
		"⚙️ Функция в разработке"
		
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
