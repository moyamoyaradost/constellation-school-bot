package tests

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Простой мок для Telegram Bot API
type MockBot struct {
	sentMessages []string
}

func (b *MockBot) Send(msg interface{}) error {
	// Простая заглушка - просто сохраняем сообщение
	if message, ok := msg.(string); ok {
		b.sentMessages = append(b.sentMessages, message)
	}
	return nil
}

func (b *MockBot) GetSentMessages() []string {
	return b.sentMessages
}

// Тест полного сценария регистрации студента
func TestStudentRegistrationFlow(t *testing.T) {
	dsn := "host=localhost port=5432 user=postgres password=password dbname=constellation_test sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Очищаем тестовые данные
	testUserID := fmt.Sprintf("test_student_%d", time.Now().Unix())
	_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", testUserID)

	// Шаг 1: Создание студента
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testUserID, "student", "Тестовый Студент", "+79001234567", true)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания студента: %v", err)
		return
	}
	t.Log("✅ Студент создан")

	// Шаг 2: Проверяем, что студент существует
	var role string
	err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", testUserID).Scan(&role)
	if err != nil {
		t.Errorf("❌ Ошибка проверки студента: %v", err)
		return
	}
	
	if role != "student" {
		t.Errorf("❌ Неверная роль студента: %s", role)
		return
	}
	t.Log("✅ Роль студента подтверждена")

	// Шаг 3: Создаем запись в таблице students
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", testUserID).Scan(&userID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID пользователя: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", userID)
	if err != nil {
		t.Errorf("❌ Ошибка создания записи студента: %v", err)
		return
	}
	t.Log("✅ Запись студента создана")

	// Очистка
	_, _ = db.Exec("DELETE FROM students WHERE user_id = $1", userID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", testUserID)
}

// Тест сценария создания преподавателя и урока
func TestTeacherLessonFlow(t *testing.T) {
	dsn := "host=localhost port=5432 user=postgres password=password dbname=constellation_test sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Очищаем тестовые данные
	testTeacherID := fmt.Sprintf("test_teacher_%d", time.Now().Unix())
	_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", testTeacherID)

	// Шаг 1: Создание преподавателя
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testTeacherID, "teacher", "Тестовый Преподаватель", "+79001234568", true)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания преподавателя: %v", err)
		return
	}
	t.Log("✅ Преподаватель создан")

	// Шаг 2: Создание записи в таблице teachers
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", testTeacherID).Scan(&userID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID пользователя: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", userID)
	if err != nil {
		t.Errorf("❌ Ошибка создания записи преподавателя: %v", err)
		return
	}
	t.Log("✅ Запись преподавателя создана")

	// Шаг 3: Получаем ID преподавателя и предмета
	var teacherID int
	err = db.QueryRow("SELECT id FROM teachers WHERE user_id = $1", userID).Scan(&teacherID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID преподавателя: %v", err)
		return
	}

	var subjectID int
	err = db.QueryRow("SELECT id FROM subjects WHERE code = $1", "WEB_DEV").Scan(&subjectID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID предмета: %v", err)
		return
	}

	// Шаг 4: Создание урока
	lessonTime := time.Now().Add(24 * time.Hour) // Завтра
	_, err = db.Exec(`
		INSERT INTO lessons (teacher_id, subject_id, start_time, duration_minutes, max_students, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, teacherID, subjectID, lessonTime, 90, 10, "active")
	
	if err != nil {
		t.Errorf("❌ Ошибка создания урока: %v", err)
		return
	}
	t.Log("✅ Урок создан")

	// Шаг 5: Проверяем, что урок существует
	var lessonCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM lessons 
		WHERE teacher_id = $1 AND subject_id = $2 AND status = 'active'
	`, teacherID, subjectID).Scan(&lessonCount)
	
	if err != nil {
		t.Errorf("❌ Ошибка проверки урока: %v", err)
		return
	}
	
	if lessonCount == 0 {
		t.Errorf("❌ Урок не найден")
		return
	}
	t.Log("✅ Урок найден")

	// Очистка
	_, _ = db.Exec("DELETE FROM lessons WHERE teacher_id = $1", teacherID)
	_, _ = db.Exec("DELETE FROM teachers WHERE user_id = $1", userID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", testTeacherID)
}

// Тест сценария записи на урок
func TestEnrollmentFlow(t *testing.T) {
	dsn := "host=localhost port=5432 user=postgres password=password dbname=constellation_test sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Создаем тестовые данные
	testStudentID := fmt.Sprintf("test_student_%d", time.Now().Unix())
	testTeacherID := fmt.Sprintf("test_teacher_%d", time.Now().Unix())
	
	// Очищаем
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2)", testStudentID, testTeacherID)

	// Создаем студента
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testStudentID, "student", "Тестовый Студент", "+79001234567", true)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать студента: %v", err)
		return
	}

	// Создаем преподавателя
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testTeacherID, "teacher", "Тестовый Преподаватель", "+79001234568", true)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать преподавателя: %v", err)
		return
	}

	// Получаем ID
	var studentUserID, teacherUserID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", testStudentID).Scan(&studentUserID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID студента: %v", err)
		return
	}

	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", testTeacherID).Scan(&teacherUserID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID преподавателя: %v", err)
		return
	}

	// Создаем записи в таблицах
	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", studentUserID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать запись студента: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", teacherUserID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать запись преподавателя: %v", err)
		return
	}

	// Получаем ID преподавателя и предмета
	var teacherID, subjectID int
	err = db.QueryRow("SELECT id FROM teachers WHERE user_id = $1", teacherUserID).Scan(&teacherID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID преподавателя: %v", err)
		return
	}

	err = db.QueryRow("SELECT id FROM subjects WHERE code = $1", "WEB_DEV").Scan(&subjectID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID предмета: %v", err)
		return
	}

	// Создаем урок
	lessonTime := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO lessons (teacher_id, subject_id, start_time, duration_minutes, max_students, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, teacherID, subjectID, lessonTime, 90, 10, "active")
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось создать урок: %v", err)
		return
	}

	// Получаем ID урока
	var lessonID int
	err = db.QueryRow(`
		SELECT id FROM lessons 
		WHERE teacher_id = $1 AND subject_id = $2 AND status = 'active'
		ORDER BY id DESC LIMIT 1
	`, teacherID, subjectID).Scan(&lessonID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID урока: %v", err)
		return
	}

	// Получаем ID студента
	var studentID int
	err = db.QueryRow("SELECT id FROM students WHERE user_id = $1", studentUserID).Scan(&studentID)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось получить ID студента: %v", err)
		return
	}

	// Шаг 1: Создаем pending_operation (rate-limiting)
	_, err = db.Exec(`
		INSERT INTO pending_operations (user_id, operation, lesson_id)
		VALUES ($1, $2, $3)
	`, studentUserID, "enroll", lessonID)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания pending_operation: %v", err)
		return
	}
	t.Log("✅ Pending operation создан")

	// Шаг 2: Создаем запись на урок
	_, err = db.Exec(`
		INSERT INTO enrollments (student_id, lesson_id, status)
		VALUES ($1, $2, $3)
	`, studentID, lessonID, "enrolled")
	
	if err != nil {
		t.Errorf("❌ Ошибка записи на урок: %v", err)
		return
	}
	t.Log("✅ Запись на урок создана")

	// Шаг 3: Проверяем, что запись существует
	var enrollmentCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM enrollments 
		WHERE student_id = $1 AND lesson_id = $2 AND status = 'enrolled'
	`, studentID, lessonID).Scan(&enrollmentCount)
	
	if err != nil {
		t.Errorf("❌ Ошибка проверки записи: %v", err)
		return
	}
	
	if enrollmentCount == 0 {
		t.Errorf("❌ Запись на урок не найдена")
		return
	}
	t.Log("✅ Запись на урок найдена")

	// Очистка
	_, _ = db.Exec("DELETE FROM enrollments WHERE student_id = $1", studentID)
	_, _ = db.Exec("DELETE FROM pending_operations WHERE user_id = $1", studentUserID)
	_, _ = db.Exec("DELETE FROM lessons WHERE id = $1", lessonID)
	_, _ = db.Exec("DELETE FROM students WHERE user_id = $1", studentUserID)
	_, _ = db.Exec("DELETE FROM teachers WHERE user_id = $1", teacherUserID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2)", testStudentID, testTeacherID)
}

// Тест обработки ошибок
func TestErrorHandling(t *testing.T) {
	dsn := "host=localhost port=5432 user=postgres password=password dbname=constellation_test sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тест 1: Попытка создать пользователя с дублирующимся tg_id
	testUserID := fmt.Sprintf("test_duplicate_%d", time.Now().Unix())
	
	// Первый пользователь
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testUserID, "student", "Первый Студент", "+79001234567", true)
	
	if err != nil {
		t.Errorf("❌ Ошибка создания первого пользователя: %v", err)
		return
	}

	// Второй пользователь с тем же tg_id (должен вызвать ошибку)
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, testUserID, "student", "Второй Студент", "+79001234568", true)
	
	if err == nil {
		t.Errorf("❌ Ожидалась ошибка дублирования, но её не было")
		return
	}
	t.Log("✅ Ошибка дублирования корректно обработана")

	// Тест 2: Попытка создать запись с несуществующим lesson_id
	_, err = db.Exec(`
		INSERT INTO enrollments (student_id, lesson_id, status)
		VALUES ($1, $2, $3)
	`, 999999, 999999, "enrolled")
	
	if err == nil {
		t.Errorf("❌ Ожидалась ошибка внешнего ключа, но её не было")
		return
	}
	t.Log("✅ Ошибка внешнего ключа корректно обработана")

	// Очистка
	_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", testUserID)
}
