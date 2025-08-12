package handlers

import (
"database/sql"
"log"
"strconv"

tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// FSM состояния регистрации
type UserState string

const (
StateIdle        UserState = "idle"
StateWaitingName UserState = "waiting_name" 
StateWaitingPhone UserState = "waiting_phone"
StateRegistered  UserState = "registered"
)

// Хранилище состояний (в продакшене - Redis)
var userStates = make(map[int64]UserState)
var userData = make(map[int64]map[string]interface{})

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

// Команда /start
func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	var existingUser int
	err := db.QueryRow("SELECT id FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&existingUser)
	
	if err == sql.ErrNoRows {
		sendMessage(bot, message.Chat.ID, 
"👋 Добро пожаловать в Constellation School!\n\n"+
"Для начала работы зарегистрируйтесь командой /register")
	} else if err != nil {
		sendMessage(bot, message.Chat.ID, "❌ Ошибка проверки регистрации")
	} else {
		// Пользователь уже зарегистрирован - показываем главное меню с кнопками
		handleMainMenu(bot, message, db)
	}
}

// Команда /register
func handleRegister(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	setUserState(userID, StateWaitingName)
	
	if userData[userID] == nil {
		userData[userID] = make(map[string]interface{})
	}
	
	sendMessage(bot, message.Chat.ID, "📝 Введите ваше полное имя:")
}

// Обработка текстовых сообщений через FSM
func handleTextMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	state := getUserState(userID)
	
	switch state {
	case StateWaitingName:
		if len(message.Text) < 2 {
			sendMessage(bot, message.Chat.ID, "❌ Имя должно содержать минимум 2 символа")
			return
		}
		userData[userID]["full_name"] = message.Text
		setUserState(userID, StateWaitingPhone)
		sendMessage(bot, message.Chat.ID, "📱 Введите ваш номер телефона (формат: +79001234567):")
		
	case StateWaitingPhone:
		if len(message.Text) < 10 {
			sendMessage(bot, message.Chat.ID, "❌ Некорректный номер телефона")
			return
		}
		userData[userID]["phone"] = message.Text
		
		// Завершение регистрации
		err := finishRegistration(userID, message.Chat.ID, db)
		if err != nil {
			sendMessage(bot, message.Chat.ID, "❌ Ошибка регистрации")
			log.Printf("Ошибка регистрации: %v", err)
		} else {
			sendMessage(bot, message.Chat.ID, "✅ Регистрация завершена! Используйте /help для просмотра команд")
		}
		setUserState(userID, StateRegistered)
		
	default:
		sendMessage(bot, message.Chat.ID, "❓ Используйте команды бота или /help для получения справки")
	}
}

// Завершение регистрации
func finishRegistration(userID int64, chatID int64, db *sql.DB) error {
	fullName := userData[userID]["full_name"].(string)
	phone := userData[userID]["phone"].(string)
	
	query := "INSERT INTO users (tg_id, role, full_name, phone) VALUES ($1, $2, $3, $4)"
	_, err := db.Exec(query, strconv.FormatInt(userID, 10), "student", fullName, phone)
	if err != nil {
		return err
	}
	
	// Создание записи студента
	var userRecordID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&userRecordID)
	if err != nil {
		return err
	}
	
	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", userRecordID)
	return err
}
