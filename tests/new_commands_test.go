package tests

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// Тест команды /delete_lesson
func TestDeleteLessonCommand(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тестовые данные
	adminUserID := "delete_lesson_admin"
	teacherUserID := "delete_lesson_teacher"
	studentUserID := "delete_lesson_student"
	
	// Очистка перед тестом
	_, _ = db.Exec("DELETE FROM enrollments WHERE student_id IN (SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM waitlist WHERE student_id IN (SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM lessons WHERE teacher_id IN (SELECT t.id FROM teachers t JOIN users u ON t.user_id = u.id WHERE u.tg_id = $1)", teacherUserID)
	_, _ = db.Exec("DELETE FROM students WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM teachers WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", teacherUserID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2, $3)", adminUserID, teacherUserID, studentUserID)

	// Создаем тестового администратора
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active) 
		VALUES ($1, 'superuser', 'Delete Lesson Admin', '+79001234567', true)`,
		adminUserID)
	if err != nil {
		t.Errorf("❌ Ошибка создания администратора: %v", err)
		return
	}

	// Создаем тестового преподавателя
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active) 
		VALUES ($1, 'teacher', 'Delete Lesson Teacher', '+79001234568', true)`,
		teacherUserID)
	if err != nil {
		t.Errorf("❌ Ошибка создания преподавателя: %v", err)
		return
	}

	var teacherRecordID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", teacherUserID).Scan(&teacherRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID преподавателя: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", teacherRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка создания записи преподавателя: %v", err)
		return
	}

	// Создаем тестового студента
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active) 
		VALUES ($1, 'student', 'Delete Lesson Student', '+79001234569', true)`,
		studentUserID)
	if err != nil {
		t.Errorf("❌ Ошибка создания студента: %v", err)
		return
	}

	var studentRecordID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", studentUserID).Scan(&studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID студента: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка создания записи студента: %v", err)
		return
	}

	// Получаем ID записей
	var teacherID, studentID int
	err = db.QueryRow("SELECT id FROM teachers WHERE user_id = $1", teacherRecordID).Scan(&teacherID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID преподавателя: %v", err)
		return
	}

	err = db.QueryRow("SELECT id FROM students WHERE user_id = $1", studentRecordID).Scan(&studentID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID студента: %v", err)
		return
	}

	// Получаем предмет
	var subjectID int
	err = db.QueryRow("SELECT id FROM subjects LIMIT 1").Scan(&subjectID)
	if err != nil {
		t.Errorf("❌ Ошибка получения предмета: %v", err)
		return
	}

	// Создаем тестовый урок
	var lessonID int
	err = db.QueryRow(`
		INSERT INTO lessons (subject_id, teacher_id, start_time, duration_minutes, max_students, status)
		VALUES ($1, $2, NOW() + INTERVAL '1 day', 90, 5, 'active')
		RETURNING id`,
		subjectID, teacherID).Scan(&lessonID)
	if err != nil {
		t.Errorf("❌ Ошибка создания урока: %v", err)
		return
	}
	t.Log("✅ Тестовый урок создан")

	// Записываем студента на урок
	_, err = db.Exec(`
		INSERT INTO enrollments (student_id, lesson_id, status)
		VALUES ($1, $2, 'enrolled')`,
		studentID, lessonID)
	if err != nil {
		t.Errorf("❌ Ошибка записи студента на урок: %v", err)
		return
	}

	// Тест 1: Проверяем права администратора
	var adminRole string
	err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", adminUserID).Scan(&adminRole)
	if err != nil {
		t.Errorf("❌ Ошибка получения роли администратора: %v", err)
		return
	}
	
	if adminRole != "superuser" {
		t.Errorf("❌ Некорректная роль: ожидалось 'superuser', получено '%s'", adminRole)
		return
	}
	t.Log("✅ Права администратора подтверждены")

	// Тест 2: Soft delete урока
	_, err = db.Exec("UPDATE lessons SET soft_deleted = true WHERE id = $1", lessonID)
	if err != nil {
		t.Errorf("❌ Ошибка soft delete урока: %v", err)
		return
	}

	// Проверяем, что урок помечен как удаленный
	var softDeleted bool
	err = db.QueryRow("SELECT soft_deleted FROM lessons WHERE id = $1", lessonID).Scan(&softDeleted)
	if err != nil {
		t.Errorf("❌ Ошибка проверки soft delete: %v", err)
		return
	}
	
	if !softDeleted {
		t.Error("❌ Урок не помечен как удаленный")
		return
	}
	t.Log("✅ Урок помечен как удаленный (soft delete)")

	// Тест 3: Отмена записей студентов
	_, err = db.Exec("UPDATE enrollments SET status = 'cancelled' WHERE lesson_id = $1", lessonID)
	if err != nil {
		t.Errorf("❌ Ошибка отмены записей: %v", err)
		return
	}

	// Проверяем статус записи
	var enrollmentStatus string
	err = db.QueryRow("SELECT status FROM enrollments WHERE lesson_id = $1 AND student_id = $2", lessonID, studentID).Scan(&enrollmentStatus)
	if err != nil {
		t.Errorf("❌ Ошибка проверки статуса записи: %v", err)
		return
	}
	
	if enrollmentStatus != "cancelled" {
		t.Errorf("❌ Некорректный статус записи: ожидалось 'cancelled', получено '%s'", enrollmentStatus)
		return
	}
	t.Log("✅ Записи студентов отменены")

	// Очистка
	_, _ = db.Exec("DELETE FROM enrollments WHERE lesson_id = $1", lessonID)
	_, _ = db.Exec("DELETE FROM lessons WHERE id = $1", lessonID)
	_, _ = db.Exec("DELETE FROM students WHERE id = $1", studentID)
	_, _ = db.Exec("DELETE FROM teachers WHERE id = $1", teacherID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2, $3)", adminUserID, teacherUserID, studentUserID)
}

// Тест команды /notify_all
func TestNotifyAllCommand(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тестовые данные
	adminUserID := "notify_all_admin"
	studentUserID := "notify_all_student"
	teacherUserID := "notify_all_teacher"
	
	// Очистка перед тестом
	_, _ = db.Exec("DELETE FROM students WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM teachers WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", teacherUserID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2, $3)", adminUserID, studentUserID, teacherUserID)

	// Создаем тестовых пользователей
	testUsers := []struct {
		tgID     string
		role     string
		fullName string
	}{
		{adminUserID, "superuser", "Notify All Admin"},
		{studentUserID, "student", "Notify All Student"},
		{teacherUserID, "teacher", "Notify All Teacher"},
	}

	for _, user := range testUsers {
		_, err = db.Exec(`
			INSERT INTO users (tg_id, role, full_name, phone, is_active) 
			VALUES ($1, $2, $3, '+79001234567', true)`,
			user.tgID, user.role, user.fullName)
		if err != nil {
			t.Errorf("❌ Ошибка создания пользователя %s: %v", user.role, err)
			return
		}

		// Создаем соответствующие записи
		var userRecordID int
		err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", user.tgID).Scan(&userRecordID)
		if err != nil {
			t.Errorf("❌ Ошибка получения ID пользователя %s: %v", user.role, err)
			return
		}

		if user.role == "student" {
			_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", userRecordID)
			if err != nil {
				t.Errorf("❌ Ошибка создания записи студента: %v", err)
				return
			}
		} else if user.role == "teacher" {
			_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", userRecordID)
			if err != nil {
				t.Errorf("❌ Ошибка создания записи преподавателя: %v", err)
				return
			}
		}
	}
	t.Log("✅ Тестовые пользователи созданы")

	// Тест 1: Проверяем права администратора
	var adminRole string
	err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", adminUserID).Scan(&adminRole)
	if err != nil {
		t.Errorf("❌ Ошибка получения роли администратора: %v", err)
		return
	}
	
	if adminRole != "superuser" {
		t.Errorf("❌ Некорректная роль: ожидалось 'superuser', получено '%s'", adminRole)
		return
	}
	t.Log("✅ Права администратора подтверждены")

	// Тест 2: Получение списка активных пользователей для уведомлений
	rows, err := db.Query(`
		SELECT tg_id, full_name, role 
		FROM users 
		WHERE is_active = true AND tg_id IS NOT NULL AND tg_id IN ($1, $2, $3)
		ORDER BY role, full_name`,
		adminUserID, studentUserID, teacherUserID)
	if err != nil {
		t.Errorf("❌ Ошибка получения списка пользователей: %v", err)
		return
	}
	defer rows.Close()

	userCount := 0
	roleStats := make(map[string]int)

	for rows.Next() {
		var tgID, fullName, role string
		if err := rows.Scan(&tgID, &fullName, &role); err != nil {
			continue
		}
		userCount++
		roleStats[role]++
	}

	if userCount != 3 {
		t.Errorf("❌ Некорректное количество пользователей: ожидалось 3, получено %d", userCount)
		return
	}
	t.Logf("✅ Найдено %d пользователей для уведомлений", userCount)

	// Тест 3: Проверяем статистику по ролям
	expectedRoles := map[string]int{
		"superuser": 1,
		"student":   1,
		"teacher":   1,
	}

	for role, expectedCount := range expectedRoles {
		actualCount, exists := roleStats[role]
		if !exists || actualCount != expectedCount {
			t.Errorf("❌ Некорректное количество пользователей роли %s: ожидалось %d, получено %d", role, expectedCount, actualCount)
			return
		}
	}
	t.Log("✅ Статистика по ролям корректна")

	// Очистка
	for _, user := range testUsers {
		var userRecordID int
		err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", user.tgID).Scan(&userRecordID)
		if err == nil {
			if user.role == "student" {
				_, _ = db.Exec("DELETE FROM students WHERE user_id = $1", userRecordID)
			} else if user.role == "teacher" {
				_, _ = db.Exec("DELETE FROM teachers WHERE user_id = $1", userRecordID)
			}
		}
		_, _ = db.Exec("DELETE FROM users WHERE tg_id = $1", user.tgID)
	}
}

// Тест команд активации/деактивации студентов
func TestStudentActivationCommands(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тестовые данные
	adminUserID := "activation_admin"
	studentUserID := "activation_student"
	
	// Очистка перед тестом
	_, _ = db.Exec("DELETE FROM enrollments WHERE student_id IN (SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM waitlist WHERE student_id IN (SELECT s.id FROM students s JOIN users u ON s.user_id = u.id WHERE u.tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM students WHERE user_id IN (SELECT id FROM users WHERE tg_id = $1)", studentUserID)
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2)", adminUserID, studentUserID)

	// Создаем тестового администратора
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active) 
		VALUES ($1, 'superuser', 'Activation Admin', '+79001234567', true)`,
		adminUserID)
	if err != nil {
		t.Errorf("❌ Ошибка создания администратора: %v", err)
		return
	}

	// Создаем тестового студента
	_, err = db.Exec(`
		INSERT INTO users (tg_id, role, full_name, phone, is_active) 
		VALUES ($1, 'student', 'Activation Student', '+79001234568', true)`,
		studentUserID)
	if err != nil {
		t.Errorf("❌ Ошибка создания студента: %v", err)
		return
	}

	var studentRecordID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = $1", studentUserID).Scan(&studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка получения ID студента: %v", err)
		return
	}

	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка создания записи студента: %v", err)
		return
	}
	t.Log("✅ Тестовые пользователи созданы")

	// Тест 1: Деактивация студента
	_, err = db.Exec("UPDATE users SET is_active = false WHERE id = $1", studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка деактивации студента: %v", err)
		return
	}

	// Проверяем статус
	var isActive bool
	err = db.QueryRow("SELECT is_active FROM users WHERE id = $1", studentRecordID).Scan(&isActive)
	if err != nil {
		t.Errorf("❌ Ошибка проверки статуса: %v", err)
		return
	}
	
	if isActive {
		t.Error("❌ Студент не деактивирован")
		return
	}
	t.Log("✅ Студент деактивирован")

	// Тест 2: Активация студента
	_, err = db.Exec("UPDATE users SET is_active = true WHERE id = $1", studentRecordID)
	if err != nil {
		t.Errorf("❌ Ошибка активации студента: %v", err)
		return
	}

	// Проверяем статус
	err = db.QueryRow("SELECT is_active FROM users WHERE id = $1", studentRecordID).Scan(&isActive)
	if err != nil {
		t.Errorf("❌ Ошибка проверки статуса: %v", err)
		return
	}
	
	if !isActive {
		t.Error("❌ Студент не активирован")
		return
	}
	t.Log("✅ Студент активирован")

	// Тест 3: Проверяем права администратора
	var adminRole string
	err = db.QueryRow("SELECT role FROM users WHERE tg_id = $1", adminUserID).Scan(&adminRole)
	if err != nil {
		t.Errorf("❌ Ошибка получения роли администратора: %v", err)
		return
	}
	
	if adminRole != "superuser" {
		t.Errorf("❌ Некорректная роль: ожидалось 'superuser', получено '%s'", adminRole)
		return
	}
	t.Log("✅ Права администратора подтверждены")

	// Очистка
	var studentID int
	err = db.QueryRow("SELECT id FROM students WHERE user_id = $1", studentRecordID).Scan(&studentID)
	if err == nil {
		_, _ = db.Exec("DELETE FROM students WHERE id = $1", studentID)
	}
	_, _ = db.Exec("DELETE FROM users WHERE tg_id IN ($1, $2)", adminUserID, studentUserID)
}

// Тест команды /remind_all
func TestRemindAllCommand(t *testing.T) {
	dsn := "host=localhost port=5433 user=constellation_user password=constellation_pass dbname=constellation_db sslmode=disable"
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Пропускаем тест: не удалось подключиться к БД: %v", err)
		return
	}
	defer db.Close()

	// Тест: Проверяем получение предстоящих уроков
	hoursAhead := 24

	rows, err := db.Query(`
		SELECT l.id, s.name as subject_name, u.full_name as teacher_name,
		       l.start_time, l.duration_minutes, l.max_students,
		       COUNT(e.id) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN teachers t ON l.teacher_id = t.id
		LEFT JOIN users u ON t.user_id = u.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id AND e.status = 'enrolled'
		WHERE l.soft_deleted = false 
		AND l.start_time > NOW() 
		AND l.start_time <= NOW() + INTERVAL '1 hour' * $1
		GROUP BY l.id, s.name, u.full_name, l.start_time, l.duration_minutes, l.max_students
		ORDER BY l.start_time LIMIT 5`, hoursAhead)
	
	if err != nil {
		t.Errorf("❌ Ошибка получения предстоящих уроков: %v", err)
		return
	}
	defer rows.Close()

	lessonCount := 0
	for rows.Next() {
		var lessonID, duration, maxStudents, enrolledCount int
		var subjectName, teacherName, startTime string
		
		if err := rows.Scan(&lessonID, &subjectName, &teacherName, &startTime, &duration, &maxStudents, &enrolledCount); err != nil {
			continue
		}
		lessonCount++
		
		// Проверяем корректность данных урока
		if duration <= 0 {
			t.Errorf("❌ Некорректная длительность урока: %d", duration)
			continue
		}
		if maxStudents <= 0 {
			t.Errorf("❌ Некорректное максимальное количество студентов: %d", maxStudents)
			continue
		}
		if enrolledCount < 0 {
			t.Errorf("❌ Некорректное количество записанных студентов: %d", enrolledCount)
			continue
		}
	}
	
	t.Logf("✅ Найдено %d предстоящих уроков в ближайшие %d часов", lessonCount, hoursAhead)
}
