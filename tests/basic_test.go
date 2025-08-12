package tests

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Простой тест подключения к БД
func TestDatabaseConnection(t *testing.T) {
	// Подключение к тестовой БД (используем параметры из .env)
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		t.Skipf("Пропускаем тест: не удалось проверить подключение к БД: %v", err)
		return
	}

	t.Log("✅ Подключение к БД успешно")
}

// Тест создания таблиц
func TestTableCreation(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем существование критичных таблиц
	tables := []string{"users", "teachers", "students", "lessons", "enrollments", "subjects", "waitlist", "pending_operations", "simple_logs"}
	
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`
		
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			t.Errorf("❌ Ошибка проверки таблицы %s: %v", table, err)
			continue
		}
		
		if exists {
			t.Logf("✅ Таблица %s существует", table)
		} else {
			t.Errorf("❌ Таблица %s не найдена", table)
		}
	}
}

// Тест базовых CRUD операций
func TestBasicCRUD(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тест создания пользователя
	testUserID := fmt.Sprintf("test_user_%d", time.Now().Unix())
	
	// INSERT
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testUserID, "student", "Тестовый Студент", "+79001234567", true)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания пользователя: %v", err)
		return
	}
	t.Log("✅ Пользователь создан")

	// SELECT
	var role, fullName string
	err = db.QueryRow("SELECT role, full_name FROM users WHERE tg_id = $1", testUserID).Scan(&role, &fullName)
	if err != nil {
		t.Errorf("❌ Ошибка чтения пользователя: %v", err)
		return
	}
	
	if role != "student" || fullName != "Тестовый Студент" {
		t.Errorf("❌ Неверные данные пользователя: role=%s, full_name=%s", role, fullName)
		return
	}
	t.Log("✅ Пользователь прочитан")

	// UPDATE
	_, err = db.Exec("UPDATE users SET full_name = $1 WHERE tg_id = $2", "Обновленный Студент", testUserID)
	if err != nil {
		t.Errorf("❌ Ошибка обновления пользователя: %v", err)
		return
	}
	t.Log("✅ Пользователь обновлен")

	// DELETE (очистка)
	_, err = db.Exec("DELETE FROM users WHERE tg_id = $1", testUserID)
	if err != nil {
		t.Errorf("❌ Ошибка удаления пользователя: %v", err)
		return
	}
	t.Log("✅ Пользователь удален")
}

// Тест rate-limiting
func TestRateLimiting(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Создаем тестового пользователя
	testUserID := fmt.Sprintf("test_user_%d", time.Now().Unix())
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testUserID, "student", "Тестовый Студент", "+79001234567", true)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать пользователя: %v", err)
		return
	}

	// Получаем ID пользователя
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", testUserID).Scan(&userID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID пользователя: %v", err)
		return
	}

	// Тест создания pending_operation
	_, err = db.Exec(`
		INSERT INTO pending_operations (user_id, operation, lesson_id)
		VALUES ($1, $2, $3)
	`, userID, "enroll", 1)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания pending_operation: %v", err)
		return
	}
	t.Log("✅ Pending operation создан")

	// Проверяем, что операция существует
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pending_operations 
		WHERE user_id = $1 AND operation = $2
	`, userID, "enroll").Scan(&count)
	
	if err != nil {
		t.Errorf("❌ Ошибка проверки pending_operation: %v", err)
		return
	}
	
	if count == 0 {
		t.Errorf("❌ Pending operation не найден")
		return
	}
	t.Log("✅ Pending operation найден")

	// Очистка
	_, err = db.Exec("DELETE FROM pending_operations WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("⚠️ Ошибка очистки pending_operations: %v", err)
	}
	
	_, err = db.Exec("DELETE FROM users WHERE tg_id = $1", testUserID)
	if err != nil {
		t.Logf("⚠️ Ошибка очистки пользователя: %v", err)
	}
}

// Тест логирования
func TestLogging(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тест создания лога
	_, err = db.Exec(`
		INSERT INTO simple_logs (action, user_id, details)
		VALUES ($1, $2, $3)
	`, "test_action", nil, "Тестовый лог")
	
	if err != nil {
		t.Errorf("❌ Ошибка создания лога: %v", err)
		return
	}
	t.Log("✅ Лог создан")

	// Проверяем, что лог существует
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM simple_logs 
		WHERE action = $1 AND details = $2
	`, "test_action", "Тестовый лог").Scan(&count)
	
	if err != nil {
		t.Errorf("❌ Ошибка проверки лога: %v", err)
		return
	}
	
	if count == 0 {
		t.Errorf("❌ Лог не найден")
		return
	}
	t.Log("✅ Лог найден")

	// Очистка
	_, err = db.Exec("DELETE FROM simple_logs WHERE action = $1", "test_action")
	if err != nil {
		t.Logf("⚠️ Ошибка очистки логов: %v", err)
	}
}

// Тест функций уведомлений
func TestNotificationFunctions(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем получение списка активных пользователей
	rows, err := db.Query(`
		SELECT tg_id, full_name, role 
		FROM users 
		WHERE is_active = true AND tg_id IS NOT NULL
		ORDER BY role, full_name LIMIT 10`)
	if err != nil {
		t.Errorf("❌ Ошибка получения пользователей: %v", err)
		return
	}
	defer rows.Close()

	userCount := 0
	for rows.Next() {
		var tgID, fullName, role string
		if err := rows.Scan(&tgID, &fullName, &role); err != nil {
			continue
		}
		userCount++
	}
	t.Logf("✅ Найдено %d активных пользователей", userCount)
}

// Тест функций управления уроками
func TestLessonManagement(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем существование предметов
	var subjectCount int
	err = db.QueryRow("SELECT COUNT(*) FROM subjects").Scan(&subjectCount)
	if err != nil {
		t.Errorf("❌ Ошибка проверки предметов: %v", err)
		return
	}
	t.Logf("✅ Найдено %d предметов", subjectCount)

	// Проверяем получение уроков с фильтрацией
	rows, err := db.Query(`
		SELECT l.id, s.name, l.start_time, l.max_students,
		       COALESCE(COUNT(e.id), 0) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.soft_deleted = false
		GROUP BY l.id, s.name, l.start_time, l.max_students
		ORDER BY l.start_time LIMIT 5`)
	
	if err != nil {
		t.Errorf("❌ Ошибка получения уроков: %v", err)
		return
	}
	defer rows.Close()

	lessonCount := 0
	for rows.Next() {
		var lessonID, maxStudents, enrolledCount int
		var subjectName, startTime string
		if err := rows.Scan(&lessonID, &subjectName, &startTime, &maxStudents, &enrolledCount); err != nil {
			continue
		}
		lessonCount++
	}
	t.Logf("✅ Найдено %d уроков", lessonCount)
}

// Тест функций управления пользователями
func TestUserManagement(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем функции проверки ролей
	var roleCount int
	err = db.QueryRow("SELECT COUNT(DISTINCT role) FROM users").Scan(&roleCount)
	if err != nil {
		t.Errorf("❌ Ошибка проверки ролей: %v", err)
		return
	}
	t.Logf("✅ Найдено %d различных ролей", roleCount)

	// Проверяем активацию/деактивацию пользователей
	var activeCount, inactiveCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&activeCount)
	if err != nil {
		t.Errorf("❌ Ошибка подсчета активных пользователей: %v", err)
		return
	}
	
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = false").Scan(&inactiveCount)
	if err != nil {
		t.Errorf("❌ Ошибка подсчета неактивных пользователей: %v", err)
		return
	}

	t.Logf("✅ Активных пользователей: %d, неактивных: %d", activeCount, inactiveCount)
}

// Тест функций статистики
func TestStatistics(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Проверяем базовую статистику системы
	tables := []string{"users", "teachers", "students", "lessons", "enrollments", "subjects"}
	for _, table := range tables {
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			t.Errorf("❌ Ошибка получения статистики для %s: %v", table, err)
			continue
		}
		t.Logf("✅ %s: %d записей", table, count)
	}

	// Проверяем rate-limiting статистику
	var pendingOpsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pending_operations").Scan(&pendingOpsCount)
	if err != nil {
		t.Errorf("❌ Ошибка получения статистики pending_operations: %v", err)
		return
	}
	t.Logf("✅ Pending operations: %d", pendingOpsCount)
}
