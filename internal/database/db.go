package database

import (
	"database/sql"
	"fmt"
	"log"
	
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
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			specializations TEXT[],
			description TEXT,
			max_students_per_lesson INTEGER DEFAULT 10
		)`,
		
		`CREATE TABLE IF NOT EXISTS students (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			selected_subjects INTEGER[]
		)`,
		
		`CREATE TABLE IF NOT EXISTS subjects (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			code VARCHAR(50) UNIQUE NOT NULL,
			category VARCHAR(50) NOT NULL,
			default_duration INTEGER DEFAULT 90,
			description TEXT,
			competencies JSONB,
			is_active BOOLEAN DEFAULT true
		)`,
		
		`CREATE TABLE IF NOT EXISTS lessons (
			id SERIAL PRIMARY KEY,
			teacher_id INTEGER REFERENCES teachers(id),
			subject_id INTEGER REFERENCES subjects(id),
			start_time TIMESTAMP NOT NULL,
			duration_minutes INTEGER DEFAULT 90,
			max_students INTEGER DEFAULT 10,
			status VARCHAR(30) DEFAULT 'scheduled',
			created_by_superuser_id INTEGER REFERENCES users(id),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS enrollments (
			id SERIAL PRIMARY KEY,
			student_id INTEGER REFERENCES students(id) ON DELETE CASCADE,
			lesson_id INTEGER REFERENCES lessons(id) ON DELETE CASCADE,
			status VARCHAR(30) DEFAULT 'scheduled',
			enrolled_at TIMESTAMP DEFAULT NOW(),
			confirmed_at TIMESTAMP,
			cancellation_reason TEXT,
			feedback TEXT
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("ошибка создания таблицы: %w", err)
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
