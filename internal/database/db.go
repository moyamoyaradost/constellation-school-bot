package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	
	_ "github.com/lib/pq"
	"constellation-school-bot/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения с БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("ошибка создания таблиц: %w", err)
	}

	if err := insertDefaultSubjects(db); err != nil {
		return nil, fmt.Errorf("ошибка заполнения предметов: %w", err)
	}

	if err := migrateStatuses(db); err != nil {
		return nil, fmt.Errorf("ошибка миграции статусов: %w", err)
	}

	if err := removeRedundantFields(db); err != nil {
		return nil, fmt.Errorf("ошибка удаления избыточных полей: %w", err)
	}

	log.Println("База данных подключена и таблицы созданы")
	return db, nil
}

func createTables(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			tg_id VARCHAR(100) UNIQUE NOT NULL,
			role VARCHAR(20) NOT NULL,
			full_name VARCHAR(255) NOT NULL,
			phone VARCHAR(20),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS teachers (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS students (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS subjects (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE NOT NULL,
			category VARCHAR(50) NOT NULL,
			description TEXT,
			is_active BOOLEAN DEFAULT true
		)`,
		
		`CREATE TABLE IF NOT EXISTS lessons (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER REFERENCES teachers(id),
			subject_id INTEGER REFERENCES subjects(id),
			start_time TIMESTAMP NOT NULL,
			duration_minutes INTEGER DEFAULT 90,
			max_students INTEGER DEFAULT 10,
			status VARCHAR(20) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS enrollments (
			id SERIAL PRIMARY KEY,
			student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
			lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
			status VARCHAR(20) DEFAULT 'enrolled',
			enrolled_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS waitlist (
			id SERIAL PRIMARY KEY,
			student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
			lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS pending_operations (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			operation VARCHAR(50) NOT NULL,
			lesson_id INTEGER,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS simple_logs (
			id SERIAL PRIMARY KEY,
			action VARCHAR(100) NOT NULL,
			user_id INTEGER REFERENCES users(id),
			details TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("ошибка создания таблицы: %w", err)
		}
	}

	// Создаем только критичные индексы для малого бизнеса (50-100 пользователей)
	constraints := []string{
		`CREATE INDEX idx_users_tg_id ON users(tg_id)`,           // КРИТИЧЕН: каждый запрос к боту
		`CREATE INDEX idx_lessons_start_time ON lessons(start_time)`, // КРИТИЧЕН: расписание и сортировка
		`CREATE INDEX idx_pending_operations_user_operation ON pending_operations(user_id, operation)`, // КРИТИЧЕН: rate-limiting
		`CREATE INDEX idx_simple_logs_created_at ON simple_logs(created_at)`, // КРИТИЧЕН: логирование ошибок
		// Убраны избыточные индексы для упрощения (NO OVER-ENGINEERING):
		// - idx_enrollments_lesson_id (можно восстановить при росте >100 пользователей)
		// - idx_waitlist_lesson_id (используется редко, только при переполнении)
	}
	
	for _, constraint := range constraints {
		if _, err := db.Exec(constraint); err != nil {
			// Игнорируем ошибку, если индекс уже существует
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("ошибка создания ограничения: %w", err)
			}
		}
	}

	return nil
}

func insertDefaultSubjects(db *sql.DB) error {
	subjects := []struct {
		name, code, category, description string
	}{
		{
			name:        "3D-моделирование",
			code:        "3D_MODELING",
			category:    "digital_design",
			description: "Основы 3D-моделирования и визуализации",
		},
		{
			name:        "Геймдев",
			code:        "GAMEDEV",
			category:    "programming",
			description: "Разработка компьютерных игр",
		},
		{
			name:        "VFX-дизайн",
			code:        "VFX_DESIGN",
			category:    "digital_design",
			description: "Визуальные эффекты и постобработка",
		},
		{
			name:        "Графический дизайн",
			code:        "GRAPHIC_DESIGN",
			category:    "design",
			description: "Основы графического дизайна",
		},
		{
			name:        "Веб-разработка",
			code:        "WEB_DEV",
			category:    "programming",
			description: "Создание веб-сайтов и приложений",
		},
		{
			name:        "Компьютерная грамотность",
			code:        "COMPUTER_LITERACY",
			category:    "basics",
			description: "Основы работы с компьютером",
		},
	}

	for _, subject := range subjects {
		query := `INSERT INTO subjects (name, code, category, description) 
				  VALUES ($1, $2, $3, $4) 
				  ON CONFLICT (code) DO NOTHING`
		_, err := db.Exec(query, subject.name, subject.code, subject.category, subject.description)
		if err != nil {
			return fmt.Errorf("ошибка добавления предмета %s: %w", subject.name, err)
		}
	}

	return nil
}

func migrateStatuses(db *sql.DB) error {
	// Миграция статусов уроков: 'scheduled', 'confirmed' -> 'active', остальные -> 'cancelled'
	_, err := db.Exec(`
		UPDATE lessons 
		SET status = CASE 
			WHEN status IN ('scheduled', 'confirmed') THEN 'active'
			ELSE 'cancelled'
		END
		WHERE status NOT IN ('active', 'cancelled')
	`)
	if err != nil {
		return fmt.Errorf("ошибка миграции статусов уроков: %w", err)
	}

	// Миграция статусов записей: все кроме 'cancelled' -> 'enrolled'
	_, err = db.Exec(`
		UPDATE enrollments 
		SET status = CASE 
			WHEN status LIKE '%cancelled%' THEN 'cancelled'
			ELSE 'enrolled'
		END
		WHERE status NOT IN ('enrolled', 'cancelled')
	`)
	if err != nil {
		return fmt.Errorf("ошибка миграции статусов записей: %w", err)
	}

	return nil
}

func removeRedundantFields(db *sql.DB) error {
	// Удаляем неиспользуемое поле default_duration из subjects
	_, err := db.Exec(`
		ALTER TABLE subjects 
		DROP COLUMN IF EXISTS default_duration
	`)
	if err != nil {
		return fmt.Errorf("ошибка удаления поля default_duration: %w", err)
	}

	// Обновляем default значения для статусов
	_, err = db.Exec(`
		ALTER TABLE lessons ALTER COLUMN status SET DEFAULT 'active'
	`)
	if err != nil {
		return fmt.Errorf("ошибка обновления default для lessons.status: %w", err)
	}

	_, err = db.Exec(`
		ALTER TABLE enrollments ALTER COLUMN status SET DEFAULT 'enrolled'  
	`)
	if err != nil {
		return fmt.Errorf("ошибка обновления default для enrollments.status: %w", err)
	}

	return nil
}

// CancelLesson отменяет урок и обновляет все связанные записи студентов
// Автор: Maksim Novihin
func CancelLesson(db *sql.DB, lessonID int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Обновляем статус урока на 'cancelled'
	_, err = tx.Exec(`
		UPDATE lessons 
		SET status = 'cancelled' 
		WHERE id = $1 AND status = 'active'
	`, lessonID)
	if err != nil {
		return fmt.Errorf("ошибка отмены урока: %w", err)
	}

	// Обновляем статусы всех записей студентов на 'cancelled'
	_, err = tx.Exec(`
		UPDATE enrollments 
		SET status = 'cancelled' 
		WHERE lesson_id = $1 AND status = 'enrolled'
	`, lessonID)
	if err != nil {
		return fmt.Errorf("ошибка отмены записей студентов: %w", err)
	}

	return tx.Commit()
}

// EnrollStudent записывает студента на урок
// Автор: Maksim Novihin  
func EnrollStudent(db *sql.DB, studentID, lessonID int) error {
	_, err := db.Exec(`
		INSERT INTO enrollments (student_id, lesson_id, status) 
		VALUES ($1, $2, 'enrolled')
	`, studentID, lessonID)
	
	if err != nil {
		return fmt.Errorf("ошибка записи студента на урок: %w", err)
	}
	
	return nil
}

// NotifyStudentsLessonCancelled получает список студентов для уведомления об отмене урока
// Автор: Maksim Novihin
func NotifyStudentsLessonCancelled(db *sql.DB, lessonID int) ([]int, error) {
	query := `
		SELECT DISTINCT u.id 
		FROM users u
		JOIN students s ON u.id = s.user_id  
		JOIN enrollments e ON s.id = e.student_id
		WHERE e.lesson_id = $1 AND e.status = 'cancelled'
	`
	
	rows, err := db.Query(query, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка студентов: %w", err)
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("ошибка сканирования ID пользователя: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по результатам: %w", err)
	}

	return userIDs, nil
}
