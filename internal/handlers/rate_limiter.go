package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
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
	mu sync.RWMutex
}

// NewRateLimiter - создает новый instance rate limiter
func NewRateLimiter(db *sql.DB) *RateLimiter {
	return &RateLimiter{db: db}
}

// IsOperationAllowed - проверяет можно ли выполнить операцию
func (rl *RateLimiter) IsOperationAllowed(userID int64, operation string, lessonID int) (bool, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Получаем user_id из БД по tg_id
	var dbUserID int
	userIDStr := strconv.FormatInt(userID, 10)
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userIDStr).Scan(&dbUserID)
	if err != nil {
		log.Printf("Ошибка получения user_id: %v", err)
		return false, errors.New("Ошибка проверки прав доступа")
	}
	
	// Проверяем есть ли активная операция этого типа от этого пользователя
	var pendingCount int
	err = rl.db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*) FROM pending_operations 
		WHERE user_id = $1 AND operation = $2 AND created_at > NOW() - INTERVAL '%d minutes'`, TIMEOUT_MINUTES),
		dbUserID, operation).Scan(&pendingCount)
	
	if err != nil {
		log.Printf("Ошибка проверки pending operations: %v", err)
		return false, errors.New("Ошибка проверки системы")
	}

	if pendingCount > 0 {
		return false, fmt.Errorf("⏳ Пожалуйста, подождите. У вас есть незавершенная операция '%s'.\n"+
			"Повторите команду через несколько секунд.", getOperationName(operation))
	}
	
	// Проверяем есть ли активная операция для конкретного урока (любого типа)
	if lessonID > 0 {
		var lessonPendingCount int
		err = rl.db.QueryRow(fmt.Sprintf(`
			SELECT COUNT(*) FROM pending_operations 
			WHERE user_id = $1 AND lesson_id = $2 AND created_at > NOW() - INTERVAL '%d minutes'`, TIMEOUT_MINUTES),
			dbUserID, lessonID).Scan(&lessonPendingCount)
		
		if err != nil {
			log.Printf("Ошибка проверки lesson-specific pending operations: %v", err)
			return false, errors.New("Ошибка проверки системы")
		}

		if lessonPendingCount > 0 {
			return false, errors.New("⏳ У вас уже есть незавершенная операция для этого урока. Подождите несколько секунд.")
		}
	}
	
	return true, nil
}

// StartOperation - регистрирует начало операции
func (rl *RateLimiter) StartOperation(userID int64, operation string, lessonID int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Получаем user_id из БД по tg_id
	var dbUserID int
	userIDStr := strconv.FormatInt(userID, 10)
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userIDStr).Scan(&dbUserID)
	if err != nil {
		log.Printf("Ошибка получения user_id для StartOperation: %v", err)
		return errors.New("Ошибка системы")
	}

	// Добавляем запись о начале операции
	_, err = rl.db.Exec(`
		INSERT INTO pending_operations (user_id, operation, lesson_id, created_at)
		VALUES ($1, $2, $3, NOW())`,
		dbUserID, operation, lessonID)
	
	if err != nil {
		log.Printf("Ошибка добавления pending operation: %v", err)
		return errors.New("Ошибка системы")
	}
	
	log.Printf("🔄 Начата операция: user=%d, operation=%s, lesson=%d", userID, operation, lessonID)
	return nil
}

// FinishOperation - завершает операцию
func (rl *RateLimiter) FinishOperation(userID int64, operation string, lessonID int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Получаем user_id из БД по tg_id
	var dbUserID int
	userIDStr := strconv.FormatInt(userID, 10)
	err := rl.db.QueryRow("SELECT id FROM users WHERE tg_id = $1", userIDStr).Scan(&dbUserID)
	if err != nil {
		log.Printf("Ошибка получения user_id для FinishOperation: %v", err)
		return errors.New("Ошибка системы")
	}

	// Удаляем запись об операции
	_, err = rl.db.Exec(`
		DELETE FROM pending_operations 
		WHERE user_id = $1 AND operation = $2 AND lesson_id = $3`,
		dbUserID, operation, lessonID)
	
	if err != nil {
		log.Printf("Ошибка удаления pending operation: %v", err)
		return errors.New("Ошибка системы")
	}
	
	log.Printf("✅ Завершена операция: user=%d, operation=%s, lesson=%d", userID, operation, lessonID)
	return nil
}

// CleanupExpiredOperations - очищает истекшие операции
func (rl *RateLimiter) CleanupExpiredOperations() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	result, err := rl.db.Exec(fmt.Sprintf(`
		DELETE FROM pending_operations 
		WHERE created_at < NOW() - INTERVAL '%d minutes'`, TIMEOUT_MINUTES))
	
	if err != nil {
		log.Printf("Ошибка очистки expired operations: %v", err)
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("🗑️ Очищено %d истекших операций", rowsAffected)
	}
	
	return nil
}

// Периодическая очистка истекших операций
func (rl *RateLimiter) StartCleanupWorker() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute) // Очистка каждые 2 минуты
		defer ticker.Stop()
		
		for range ticker.C {
			err := rl.CleanupExpiredOperations()
			if err != nil {
				log.Printf("⚠️ Ошибка периодической очистки: %v", err)
			}
		}
	}()
}

// getOperationName - получает человеческое название операции
func getOperationName(operation string) string {
	switch operation {
	case OPERATION_ENROLL:
		return "запись на урок"
	case OPERATION_WAITLIST:
		return "лист ожидания"
	case OPERATION_CANCEL:
		return "отмена записи"
	default:
		return operation
	}
}

// ========================= ИНТЕГРАЦИЯ С ОБРАБОТЧИКАМИ =========================

// Глобальный rate limiter
var globalRateLimiter *RateLimiter

// InitializeRateLimiter - инициализирует глобальный rate limiter
func InitializeRateLimiter(db *sql.DB) {
	globalRateLimiter = NewRateLimiter(db)
	globalRateLimiter.StartCleanupWorker()
	log.Println("🚀 Rate Limiter инициализирован")
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
