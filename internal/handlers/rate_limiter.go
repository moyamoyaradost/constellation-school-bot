package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ========================= БЕЛОЕ ПЯТНО #3: RATE-LIMITING =========================

const (
	OPERATION_ENROLL    = "enroll"
	OPERATION_WAITLIST  = "waitlist"
	OPERATION_CANCEL    = "cancel"
	TIMEOUT_MINUTES     = 5 // Таймаут для операций в минутах
)

// RateLimiter - структура для управления rate limiting
type RateLimiter struct {
	db *sql.DB
}

// NewRateLimiter - создает новый instance rate limiter
func NewRateLimiter(db *sql.DB) *RateLimiter {
	return &RateLimiter{db: db}
}

// IsOperationAllowed - проверяет можно ли выполнить операцию
func (rl *RateLimiter) IsOperationAllowed(userID int64, operation string, lessonID int) (bool, string) {
	// Получаем user_id из БД по tg_id
	var dbUserID int
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userID).Scan(&dbUserID)
	if err != nil {
		log.Printf("Ошибка получения user_id: %v", err)
		return false, "Ошибка проверки прав доступа"
	}
	
	// Проверяем есть ли активная операция этого типа от этого пользователя
	var pendingCount int
	err = rl.db.QueryRow(`
		SELECT COUNT(*) FROM pending_operations 
		WHERE user_id = $1 AND operation = $2 AND created_at > NOW() - INTERVAL '$3 minutes'`,
		dbUserID, operation, TIMEOUT_MINUTES).Scan(&pendingCount)
	
	if err != nil {
		log.Printf("Ошибка проверки pending операций: %v", err)
		return false, "Ошибка проверки системы"
	}
	
	if pendingCount > 0 {
		return false, fmt.Sprintf("⏳ Пожалуйста, подождите. У вас есть незавершенная операция '%s'.\n"+
			"Повторите команду через несколько секунд.", getOperationName(operation))
	}
	
	// Проверяем специфичную для урока операцию (предотвращаем дубли записи на один урок)
	if lessonID > 0 {
		var lessonSpecificCount int
		err = rl.db.QueryRow(`
			SELECT COUNT(*) FROM pending_operations 
			WHERE user_id = $1 AND lesson_id = $2 AND created_at > NOW() - INTERVAL '$3 minutes'`,
			dbUserID, lessonID, TIMEOUT_MINUTES).Scan(&lessonSpecificCount)
		
		if err != nil {
			log.Printf("Ошибка проверки lesson-specific операций: %v", err)
			return false, "Ошибка проверки системы"
		}
		
		if lessonSpecificCount > 0 {
			return false, "⏳ У вас уже есть незавершенная операция для этого урока. Подождите несколько секунд."
		}
	}
	
	return true, ""
}

// StartOperation - регистрирует начало операции
func (rl *RateLimiter) StartOperation(userID int64, operation string, lessonID int) error {
	// Получаем user_id из БД по tg_id
	var dbUserID int
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userID).Scan(&dbUserID)
	if err != nil {
		return fmt.Errorf("ошибка получения user_id: %w", err)
	}
	
	// Очищаем старые операции перед добавлением новой
	rl.CleanupExpiredOperations()
	
	// Добавляем новую операцию
	var insertLessonID interface{}
	if lessonID > 0 {
		insertLessonID = lessonID
	} else {
		insertLessonID = nil
	}
	
	_, err = rl.db.Exec(`
		INSERT INTO pending_operations (user_id, operation, lesson_id) 
		VALUES ($1, $2, $3)`,
		dbUserID, operation, insertLessonID)
	
	if err != nil {
		return fmt.Errorf("ошибка регистрации операции: %w", err)
	}
	
	log.Printf("Rate limiter: начата операция %s для пользователя %d (урок %d)", operation, userID, lessonID)
	return nil
}

// FinishOperation - завершает операцию
func (rl *RateLimiter) FinishOperation(userID int64, operation string, lessonID int) error {
	// Получаем user_id из БД по tg_id
	var dbUserID int
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userID).Scan(&dbUserID)
	if err != nil {
		return fmt.Errorf("ошибка получения user_id: %w", err)
	}
	
	// Удаляем операцию
	var result sql.Result
	if lessonID > 0 {
		result, err = rl.db.Exec(`
			DELETE FROM pending_operations 
			WHERE user_id = $1 AND operation = $2 AND lesson_id = $3`,
			dbUserID, operation, lessonID)
	} else {
		result, err = rl.db.Exec(`
			DELETE FROM pending_operations 
			WHERE user_id = $1 AND operation = $2 AND lesson_id IS NULL`,
			dbUserID, operation)
	}
	
	if err != nil {
		return fmt.Errorf("ошибка завершения операции: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Rate limiter: завершена операция %s для пользователя %d (урок %d), удалено записей: %d", 
		operation, userID, lessonID, rowsAffected)
	
	return nil
}

// CleanupExpiredOperations - удаляет устаревшие операции
func (rl *RateLimiter) CleanupExpiredOperations() {
	result, err := rl.db.Exec(`
		DELETE FROM pending_operations 
		WHERE created_at < NOW() - INTERVAL '$1 minutes'`, TIMEOUT_MINUTES)
	
	if err != nil {
		log.Printf("Ошибка очистки устаревших операций: %v", err)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Rate limiter: очищено %d устаревших операций", rowsAffected)
	}
}

// GetActiveOperationsCount - получить количество активных операций (для статистики)
func (rl *RateLimiter) GetActiveOperationsCount() (int, error) {
	var count int
	err := rl.db.QueryRow(`
		SELECT COUNT(*) FROM pending_operations 
		WHERE created_at > NOW() - INTERVAL '$1 minutes'`, TIMEOUT_MINUTES).Scan(&count)
	
	return count, err
}

// getOperationName - человекочитаемое название операции
func getOperationName(operation string) string {
	switch operation {
	case OPERATION_ENROLL:
		return "запись на урок"
	case OPERATION_WAITLIST:
		return "запись в очередь"
	case OPERATION_CANCEL:
		return "отмена записи"
	default:
		return operation
	}
}

// ================ ИНТЕГРАЦИЯ С СУЩЕСТВУЮЩИМИ ОБРАБОТЧИКАМИ ================

// WithRateLimit - обертка для обработчиков с rate limiting
func WithRateLimit(rateLimiter *RateLimiter, operation string, lessonID int, handler func(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB)) func(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
	return func(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *sql.DB) {
		userID := message.From.ID
		
		// Проверяем можно ли выполнить операцию
		allowed, reason := rateLimiter.IsOperationAllowed(userID, operation, lessonID)
		if !allowed {
			sendMessage(bot, message.Chat.ID, reason)
			return
		}
		
		// Регистрируем начало операции
		if err := rateLimiter.StartOperation(userID, operation, lessonID); err != nil {
			log.Printf("Ошибка регистрации операции: %v", err)
			sendMessage(bot, message.Chat.ID, "❌ Системная ошибка. Попробуйте позже.")
			return
		}
		
		// Выполняем операцию
		handler(bot, message, db)
		
		// Завершаем операцию
		if err := rateLimiter.FinishOperation(userID, operation, lessonID); err != nil {
			log.Printf("Ошибка завершения операции: %v", err)
		}
	}
}

// ExtractLessonIDFromMessage - извлекает lesson_id из сообщения
func ExtractLessonIDFromMessage(message *tgbotapi.Message) int {
	if message.Text == "" {
		return 0
	}
	
	// Пробуем извлечь из аргументов команды
	args := message.CommandArguments()
	if args != "" {
		if id, err := strconv.Atoi(args); err == nil {
			return id
		}
	}
	
	return 0
}

// ===================== ПЕРИОДИЧЕСКАЯ ОЧИСТКА =========================

// StartCleanupWorker - запускает фоновый процесс очистки устаревших операций
func (rl *RateLimiter) StartCleanupWorker() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute) // Очистка каждые 2 минуты
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				rl.CleanupExpiredOperations()
			}
		}
	}()
	
	log.Println("Rate limiter: фоновый процесс очистки запущен")
}
