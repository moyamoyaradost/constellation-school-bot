package handlers

import (
"database/sql"
"fmt"
"log"
"strconv"
"strings"
"unicode"

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

// Timeout для состояний (в минутах)
const StateTimeoutMinutes = 15

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

// Сбросить состояние пользователя
func resetUserState(userID int64) {
	delete(userStates, userID)
	delete(userData, userID)
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
		// Показываем главное меню в зависимости от роли
		var role string
		db.QueryRow("SELECT role FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&role)
		
		if role == "student" {
			showStudentMainMenu(bot, message, db)
		} else {
			handleMainMenu(bot, message, db)
		}
	}
}

// Команда /register
func handleRegister(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	
	// Проверяем, не зарегистрирован ли уже пользователь
	var existingUser int
	err := db.QueryRow("SELECT id FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&existingUser)
	if err == nil {
		sendMessage(bot, message.Chat.ID, "✅ Вы уже зарегистрированы в системе!")
		return
	}
	
	setUserState(userID, StateWaitingName)
	
	if userData[userID] == nil {
		userData[userID] = make(map[string]interface{})
	}
	
	sendMessage(bot, message.Chat.ID, "📝 Введите ваше полное имя:\n\n💡 Для отмены регистрации используйте команду /cancel")
}

// Команда /cancel - отмена регистрации
func handleCancel(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID
	state := getUserState(userID)
	
	if state != StateIdle {
		resetUserState(userID)
		sendMessage(bot, message.Chat.ID, "❌ Регистрация отменена. Для начала регистрации используйте /register")
	} else {
		sendMessage(bot, message.Chat.ID, "📝 Нет активного процесса регистрации")
	}
}

// Обработка текстовых сообщений через FSM
func handleTextMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	userID := message.From.ID
	state := getUserState(userID)
	
	switch state {
	case StateWaitingName:
		// Улучшенная валидация имени
		fullName := strings.TrimSpace(message.Text)
		if len(fullName) < 2 {
			sendMessage(bot, message.Chat.ID, "❌ Имя должно содержать минимум 2 символа")
			return
		}
		
		if len(fullName) > 100 {
			sendMessage(bot, message.Chat.ID, "❌ Имя не должно превышать 100 символов")
			return
		}
		
		// Проверяем, что имя содержит хотя бы одну букву
		hasLetter := false
		for _, r := range fullName {
			if unicode.IsLetter(r) {
				hasLetter = true
				break
			}
		}
		
		if !hasLetter {
			sendMessage(bot, message.Chat.ID, "❌ Имя должно содержать хотя бы одну букву")
			return
		}
		
		userData[userID]["full_name"] = fullName
		setUserState(userID, StateWaitingPhone)
		sendMessage(bot, message.Chat.ID, "📱 Введите ваш номер телефона (формат: +79001234567):")
		
	case StateWaitingPhone:
		// Улучшенная валидация номера телефона
		phone := strings.TrimSpace(message.Text)
		if len(phone) < 10 || len(phone) > 15 {
			sendMessage(bot, message.Chat.ID, "❌ Некорректный номер телефона. Введите номер в формате +79001234567")
			return
		}
		
		// Проверяем, что номер начинается с + или цифры
		if !strings.HasPrefix(phone, "+") && !unicode.IsDigit(rune(phone[0])) {
			sendMessage(bot, message.Chat.ID, "❌ Номер телефона должен начинаться с + или цифры")
			return
		}
		
		userData[userID]["phone"] = phone
		
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
	// Проверяем наличие данных
	if userData[userID] == nil {
		return fmt.Errorf("пользовательские данные не найдены")
	}
	
	fullNameInterface, ok := userData[userID]["full_name"]
	if !ok {
		return fmt.Errorf("имя пользователя не указано")
	}
	
	phoneInterface, ok := userData[userID]["phone"]
	if !ok {
		return fmt.Errorf("телефон пользователя не указан")
	}
	
	fullName, ok := fullNameInterface.(string)
	if !ok {
		return fmt.Errorf("некорректное имя пользователя")
	}
	
	phone, ok := phoneInterface.(string)
	if !ok {
		return fmt.Errorf("некорректный номер телефона")
	}
	
	// Проверяем, не существует ли уже пользователь с таким tg_id
	var existingCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE tg_id = $1", strconv.FormatInt(userID, 10)).Scan(&existingCount)
	if err != nil {
		log.Printf("Ошибка проверки существующего пользователя: %v", err)
		return fmt.Errorf("ошибка проверки пользователя")
	}
	
	if existingCount > 0 {
		return fmt.Errorf("пользователь уже зарегистрирован")
	}
	
	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		return fmt.Errorf("ошибка базы данных")
	}
	defer tx.Rollback()
	
	// Создаем пользователя
	var userRecordID int
	err = tx.QueryRow(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW()) 
		RETURNING id`,
		strconv.FormatInt(userID, 10), "student", fullName, phone, true).Scan(&userRecordID)
	if err != nil {
		log.Printf("Ошибка создания пользователя: %v", err)
		return fmt.Errorf("ошибка создания пользователя")
	}
	
	// Создание записи студента
	_, err = tx.Exec("INSERT INTO students (user_id, created_at) VALUES ($1, NOW())", userRecordID)
	if err != nil {
		log.Printf("Ошибка создания записи студента: %v", err)
		return fmt.Errorf("ошибка создания студента")
	}
	
	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		log.Printf("Ошибка подтверждения транзакции: %v", err)
		return fmt.Errorf("ошибка сохранения данных")
	}
	
	// Очищаем временные данные после успешной регистрации
	delete(userData, userID)
	
	// Логируем успешную регистрацию
	log.Printf("Пользователь %d (%s) успешно зарегистрирован", userID, fullName)
	
	return nil
}
