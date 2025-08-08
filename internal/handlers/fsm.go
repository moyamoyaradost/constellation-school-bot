package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// FSM состояния для регистрации студентов
type UserState string

const (
	StateIdle            UserState = "idle"
	StateWaitingName     UserState = "waiting_name"
	StateWaitingPhone    UserState = "waiting_phone"
	StateSelectingSubjects UserState = "selecting_subjects"
	StateRegistered      UserState = "registered"
)

// Хранилище состояний пользователей (в продакшене использовать Redis)
var userStates = make(map[int64]UserState)
var userData = make(map[int64]map[string]interface{})

// Инициализация данных пользователя
func initUserData(userID int64) {
	if userData[userID] == nil {
		userData[userID] = make(map[string]interface{})
	}
}

// Получить состояние пользователя
func getUserState(userID int64) UserState {
	if state, exists := userStates[userID]; exists {
		return state
	}
	return StateIdle
}

// Установить состояние пользователя
func setUserState(userID int64, state UserState) {
	userStates[userID] = state
}

// Обработчик команды /start
func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверить, зарегистрирован ли пользователь
	var existingUser int
	err := db.QueryRow("SELECT id FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&existingUser)
	
	var msg tgbotapi.MessageConfig
	
	if err == sql.ErrNoRows {
		// Новый пользователь
		msg = tgbotapi.NewMessage(message.Chat.ID, 
			"👋 Добро пожаловать в Constellation School Bot!\n\n"+
			"Я помогу вам записаться на курсы Центра Цифрового Творчества.\n\n"+
			"Для начала работы нужно пройти регистрацию.\n\n"+
			"Нажмите /register для регистрации или /help для получения помощи.")
		
		// Inline кнопки для удобства
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Регистрация", "cmd_register"),
				tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "cmd_help"),
			),
		)
		msg.ReplyMarkup = keyboard
		
	} else if err != nil {
		log.Printf("Ошибка проверки пользователя: %v", err)
		msg = tgbotapi.NewMessage(message.Chat.ID, "❌ Произошла ошибка. Попробуйте позже.")
	} else {
		// Пользователь уже зарегистрирован
		var role string
		db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
		
		switch role {
		case "student":
			msg = tgbotapi.NewMessage(message.Chat.ID,
				"🎓 Добро пожаловать обратно!\n\n"+
				"Доступные команды для студентов:\n"+
				"/subjects - выбор предметов\n"+
				"/schedule - расписание уроков\n"+
				"/my_lessons - мои уроки\n"+
				"/help - помощь")
		case "teacher":
			msg = tgbotapi.NewMessage(message.Chat.ID,
				"👨‍🏫 Добро пожаловать, преподаватель!\n\n"+
				"Доступные команды:\n"+
				"/my_lessons - мои уроки\n"+
				"/my_students - мои студенты\n"+
				"/help_teacher - помощь")
		case "superuser":
			msg = tgbotapi.NewMessage(message.Chat.ID,
				"🔧 Добро пожаловать, администратор!\n\n"+
				"Доступные команды:\n"+
				"/add_teacher - добавить преподавателя\n"+
				"/create_lesson - создать урок\n"+
				"/system_stats - статистика системы")
		default:
			msg = tgbotapi.NewMessage(message.Chat.ID, "👋 Добро пожаловать обратно!")
		}
	}
	
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// Обработчик команды /register
func handleRegister(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверить, не зарегистрирован ли уже пользователь
	var existingUser int
	err := db.QueryRow("SELECT id FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&existingUser)
	
	if err == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "✅ Вы уже зарегистрированы в системе!")
		bot.Send(msg)
		return
	}
	
	// Начать процесс регистрации
	initUserData(userID)
	setUserState(userID, StateWaitingName)
	
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"📝 Начинаем регистрацию!\n\n"+
		"Шаг 1 из 3: Введите ваше полное имя (Фамилия Имя Отчество)")
	
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// Обработчик текстовых сообщений для FSM
func handleTextMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	state := getUserState(userID)
	
	switch state {
	case StateWaitingName:
		handleNameInput(bot, message, db)
	case StateWaitingPhone:
		handlePhoneInput(bot, message, db)
	case StateSelectingSubjects:
		handleSubjectSelection(bot, message, db)
	default:
		// Если пользователь не в процессе регистрации, показать помощь
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🤔 Не понимаю команду. Используйте /help для получения списка доступных команд.")
		bot.Send(msg)
	}
}

// Обработка ввода имени
func handleNameInput(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	name := strings.TrimSpace(message.Text)
	
	// Валидация имени
	if len(name) < 2 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Имя слишком короткое. Введите полное имя (минимум 2 символа):")
		bot.Send(msg)
		return
	}
	
	// Сохранить имя
	initUserData(userID)
	userData[userID]["full_name"] = name
	setUserState(userID, StateWaitingPhone)
	
	msg := tgbotapi.NewMessage(message.Chat.ID,
		fmt.Sprintf("✅ Имя сохранено: %s\n\n"+
		"Шаг 2 из 3: Введите ваш номер телефона (например: +7 999 123 45 67)", name))
	
	bot.Send(msg)
}

// Обработка ввода телефона
func handlePhoneInput(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	phone := strings.TrimSpace(message.Text)
	
	// Простая валидация телефона
	if len(phone) < 10 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Некорректный номер телефона. Введите корректный номер:")
		bot.Send(msg)
		return
	}
	
	// Сохранить телефон
	userData[userID]["phone"] = phone
	setUserState(userID, StateSelectingSubjects)
	
	// Показать предметы для выбора
	showSubjectsKeyboard(bot, message, db)
}

// Показать клавиатуру выбора предметов
func showSubjectsKeyboard(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Получить список предметов из БД
	rows, err := db.Query("SELECT id, name, description FROM subjects WHERE is_active = true ORDER BY name")
	if err != nil {
		log.Printf("Ошибка получения предметов: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ошибка загрузки предметов")
		bot.Send(msg)
		return
	}
	defer rows.Close()
	
	var buttons [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	
	for rows.Next() {
		var id int
		var name, description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			continue
		}
		
		button := tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("subject_%d", id))
		row = append(row, button)
		
		// Создаем ряды по 2 кнопки
		if len(row) == 2 {
			buttons = append(buttons, row)
			row = []tgbotapi.InlineKeyboardButton{}
		}
	}
	
	// Добавить последний ряд если есть
	if len(row) > 0 {
		buttons = append(buttons, row)
	}
	
	// Добавить кнопку "Завершить выбор"
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Завершить регистрацию", "finish_registration"),
	))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"🎯 Шаг 3 из 3: Выберите интересующие вас предметы\n\n"+
		"Вы можете выбрать несколько предметов, затем нажать 'Завершить регистрацию'")
	msg.ReplyMarkup = keyboard
	
	bot.Send(msg)
}

// Обработка выбора предметов (через callback)
func handleSubjectSelection(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	// Эта функция будет обрабатывать callback запросы
	msg := tgbotapi.NewMessage(message.Chat.ID,
		"🎯 Используйте кнопки выше для выбора предметов.")
	bot.Send(msg)
}

// Завершение регистрации студента
func finishStudentRegistration(bot *tgbotapi.BotAPI, userID int64, chatID int64, db *sql.DB) {
	// Получить данные пользователя
	data := userData[userID]
	fullName := data["full_name"].(string)
	phone := data["phone"].(string)
	
	var selectedSubjects []int
	if subjects, exists := data["selected_subjects"]; exists {
		selectedSubjects = subjects.([]int)
	}
	
	// Создать пользователя в БД
	var userDBID int
	err := db.QueryRow(`
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ($1, 'student', $2, $3) 
		RETURNING id`,
		strconv.FormatInt(userID, 10), fullName, phone).Scan(&userDBID)
	
	if err != nil {
		log.Printf("Ошибка создания пользователя: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при регистрации. Попробуйте позже.")
		bot.Send(msg)
		return
	}
	
	// Создать запись студента
	var studentDBID int
	err = db.QueryRow(`
		INSERT INTO students (user_id, selected_subjects) 
		VALUES ($1, $2) 
		RETURNING id`,
		userDBID, fmt.Sprintf("{%s}", strings.Join(intSliceToStringSlice(selectedSubjects), ","))).Scan(&studentDBID)
	
	if err != nil {
		log.Printf("Ошибка создания студента: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при регистрации. Попробуйте позже.")
		bot.Send(msg)
		return
	}
	
	// Очистить данные FSM
	delete(userStates, userID)
	delete(userData, userID)
	
	// Отправить подтверждение
	var subjectNames []string
	for _, subjectID := range selectedSubjects {
		var name string
		db.QueryRow("SELECT name FROM subjects WHERE id = $1", subjectID).Scan(&name)
		subjectNames = append(subjectNames, name)
	}
	
	msg := tgbotapi.NewMessage(chatID,
		"🎉 Регистрация завершена!\n\n"+
		fmt.Sprintf("👤 Имя: %s\n", fullName)+
		fmt.Sprintf("📞 Телефон: %s\n", phone)+
		fmt.Sprintf("🎯 Выбранные предметы: %s\n\n", strings.Join(subjectNames, ", "))+
		"Теперь вы можете:\n"+
		"/schedule - посмотреть расписание уроков\n"+
		"/my_lessons - мои записи на уроки\n"+
		"/subjects - изменить выбор предметов")
	
	bot.Send(msg)
}

// Вспомогательная функция для конвертации []int в []string
func intSliceToStringSlice(ints []int) []string {
	strings := make([]string, len(ints))
	for i, v := range ints {
		strings[i] = strconv.Itoa(v)
	}
	return strings
}
