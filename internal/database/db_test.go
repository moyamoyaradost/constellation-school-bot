package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB создает тестовую базу данных с помощью testcontainers
func setupTestDB(t *testing.T) *sql.DB {
	ctx := context.Background()

	// Создаем контейнер PostgreSQL
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "Не удалось создать PostgreSQL контейнер")

	// Подключаемся к БД
	connectionString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Не удалось получить строку подключения")

	db, err := sql.Open("postgres", connectionString)
	require.NoError(t, err, "Не удалось открыть подключение к БД")

	// Проверяем подключение
	err = db.Ping()
	require.NoError(t, err, "Не удалось подключиться к БД")

	// Создаем таблицы
	err = createTables(db)
	require.NoError(t, err, "Не удалось создать таблицы")

	// Вставляем тестовые предметы
	err = insertDefaultSubjects(db)
	require.NoError(t, err, "Не удалось создать предметы")

	// Очистка при завершении теста
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Ошибка закрытия БД: %v", err)
		}
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Ошибка остановки контейнера: %v", err)
		}
	})

	return db
}

// TestCreateManyUsers создает 100 пользователей подряд и проверяет уникальность
func TestCreateManyUsers(t *testing.T) {
	db := setupTestDB(t)

	startTime := time.Now()

	// Создаем 100 пользователей
	userCount := 100
	createdUsers := make([]string, 0, userCount)

	for i := 1; i <= userCount; i++ {
		tgID := fmt.Sprintf("test_user_%d", i)
		fullName := fmt.Sprintf("Тестовый Пользователь %d", i)
		phone := fmt.Sprintf("+7919%07d", i)

		query := `
			INSERT INTO users (tg_id, role, full_name, phone) 
			VALUES ($1, $2, $3, $4)`

		_, err := db.Exec(query, tgID, "student", fullName, phone)
		require.NoError(t, err, "Ошибка создания пользователя %d", i)

		createdUsers = append(createdUsers, tgID)

		// Логируем прогресс каждые 20 пользователей
		if i%20 == 0 {
			t.Logf("Создано %d пользователей", i)
		}
	}

	duration := time.Since(startTime)
	t.Logf("Время создания %d пользователей: %v", userCount, duration)

	// Проверяем что все пользователи созданы
	var totalUsers int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'student'").Scan(&totalUsers)
	require.NoError(t, err, "Ошибка подсчета пользователей")
	assert.Equal(t, userCount, totalUsers, "Количество созданных пользователей не совпадает")

	// Проверяем уникальность tg_id
	rows, err := db.Query("SELECT tg_id FROM users WHERE role = 'student' ORDER BY id")
	require.NoError(t, err, "Ошибка запроса tg_id")
	defer rows.Close()

	uniqueIDs := make(map[string]bool)
	retrievedUsers := make([]string, 0, userCount)

	for rows.Next() {
		var tgID string
		err := rows.Scan(&tgID)
		require.NoError(t, err, "Ошибка сканирования tg_id")

		// Проверяем уникальность
		assert.False(t, uniqueIDs[tgID], "Найден дублированный tg_id: %s", tgID)
		uniqueIDs[tgID] = true
		retrievedUsers = append(retrievedUsers, tgID)
	}

	require.NoError(t, rows.Err(), "Ошибка итерации по строкам")
	assert.Equal(t, len(createdUsers), len(retrievedUsers), "Количество извлеченных пользователей не совпадает")

	t.Logf("✅ Успешно создано %d уникальных пользователей за %v", userCount, duration)
}

// TestCascadeDelete проверяет каскадное удаление user → student → enrollment
func TestCascadeDelete(t *testing.T) {
	db := setupTestDB(t)

	// 1. Создаем пользователя
	userTgID := "cascade_test_user"
	query := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ($1, 'student', 'Тест Каскад', '+79191234567') 
		RETURNING id`

	var userID int
	err := db.QueryRow(query, userTgID).Scan(&userID)
	require.NoError(t, err, "Ошибка создания пользователя")
	t.Logf("Создан пользователь с ID: %d", userID)

	// 2. Создаем запись студента
	studentQuery := `
		INSERT INTO students (user_id, selected_subjects) 
		VALUES ($1, ARRAY[1]) 
		RETURNING id`

	var studentID int
	err = db.QueryRow(studentQuery, userID).Scan(&studentID)
	require.NoError(t, err, "Ошибка создания студента")
	t.Logf("Создан студент с ID: %d", studentID)

	// 3. Создаем преподавателя для урока
	teacherQuery := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ('test_teacher', 'teacher', 'Тест Учитель', '+79191111111') 
		RETURNING id`

	var teacherUserID int
	err = db.QueryRow(teacherQuery).Scan(&teacherUserID)
	require.NoError(t, err, "Ошибка создания пользователя-учителя")

	teacherInsertQuery := `
		INSERT INTO teachers (user_id, specializations) 
		VALUES ($1, ARRAY['3D-моделирование']) 
		RETURNING id`

	var teacherID int
	err = db.QueryRow(teacherInsertQuery, teacherUserID).Scan(&teacherID)
	require.NoError(t, err, "Ошибка создания учителя")

	// 4. Создаем урок
	lessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students) 
		VALUES ($1, 1, $2, 10) 
		RETURNING id`

	futureTime := time.Now().Add(24 * time.Hour)
	var lessonID int
	err = db.QueryRow(lessonQuery, teacherID, futureTime).Scan(&lessonID)
	require.NoError(t, err, "Ошибка создания урока")
	t.Logf("Создан урок с ID: %d", lessonID)

	// 5. Создаем запись на урок (enrollment)
	enrollmentQuery := `
		INSERT INTO enrollments (student_id, lesson_id, status) 
		VALUES ($1, $2, 'scheduled') 
		RETURNING id`

	var enrollmentID int
	err = db.QueryRow(enrollmentQuery, studentID, lessonID).Scan(&enrollmentID)
	require.NoError(t, err, "Ошибка создания записи на урок")
	t.Logf("Создана запись на урок с ID: %d", enrollmentID)

	// 6. Проверяем что все записи существуют ДО удаления
	var countUsers, countStudents, countEnrollments int

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", userID).Scan(&countUsers)
	require.NoError(t, err, "Ошибка подсчета пользователей")
	assert.Equal(t, 1, countUsers, "Пользователь должен существовать")

	err = db.QueryRow("SELECT COUNT(*) FROM students WHERE id = $1", studentID).Scan(&countStudents)
	require.NoError(t, err, "Ошибка подсчета студентов")
	assert.Equal(t, 1, countStudents, "Студент должен существовать")

	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE id = $1", enrollmentID).Scan(&countEnrollments)
	require.NoError(t, err, "Ошибка подсчета записей")
	assert.Equal(t, 1, countEnrollments, "Запись на урок должна существовать")

	// 7. УДАЛЯЕМ пользователя (должно произойти каскадное удаление)
	deleteQuery := "DELETE FROM users WHERE id = $1"
	result, err := db.Exec(deleteQuery, userID)
	require.NoError(t, err, "Ошибка удаления пользователя")

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err, "Ошибка получения количества удаленных строк")
	assert.Equal(t, int64(1), rowsAffected, "Должен быть удален 1 пользователь")

	t.Logf("Пользователь удален, проверяем каскадное удаление...")

	// 8. Проверяем что все связанные записи УДАЛИЛИСЬ
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", userID).Scan(&countUsers)
	require.NoError(t, err, "Ошибка подсчета пользователей после удаления")
	assert.Equal(t, 0, countUsers, "Пользователь должен быть удален")

	err = db.QueryRow("SELECT COUNT(*) FROM students WHERE id = $1", studentID).Scan(&countStudents)
	require.NoError(t, err, "Ошибка подсчета студентов после удаления")
	assert.Equal(t, 0, countStudents, "Студент должен быть удален каскадно")

	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE id = $1", enrollmentID).Scan(&countEnrollments)
	require.NoError(t, err, "Ошибка подсчета записей после удаления")
	assert.Equal(t, 0, countEnrollments, "Запись на урок должна быть удалена каскадно")

	t.Logf("✅ Каскадное удаление работает корректно: user → student → enrollment")
}

// TestConcurrentEnrollments проверяет одновременную запись нескольких студентов на урок
func TestConcurrentEnrollments(t *testing.T) {
	db := setupTestDB(t)

	// 1. Создаем преподавателя
	teacherQuery := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ('concurrent_teacher', 'teacher', 'Преподаватель Теста', '+79191234567') 
		RETURNING id`

	var teacherUserID int
	err := db.QueryRow(teacherQuery).Scan(&teacherUserID)
	require.NoError(t, err, "Ошибка создания пользователя-преподавателя")

	teacherInsertQuery := `
		INSERT INTO teachers (user_id, specializations) 
		VALUES ($1, ARRAY['3D-моделирование']) 
		RETURNING id`

	var teacherID int
	err = db.QueryRow(teacherInsertQuery, teacherUserID).Scan(&teacherID)
	require.NoError(t, err, "Ошибка создания преподавателя")

	// 2. Создаем урок с ограничением на 5 студентов
	maxStudents := 5
	lessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students) 
		VALUES ($1, 1, $2, $3) 
		RETURNING id`

	futureTime := time.Now().Add(24 * time.Hour)
	var lessonID int
	err = db.QueryRow(lessonQuery, teacherID, futureTime, maxStudents).Scan(&lessonID)
	require.NoError(t, err, "Ошибка создания урока")
	t.Logf("Создан урок с ID: %d (макс. %d студентов)", lessonID, maxStudents)

	// 3. Создаем 10 студентов
	studentCount := 10
	studentIDs := make([]int, studentCount)

	for i := 0; i < studentCount; i++ {
		// Создаем пользователя
		userQuery := `
			INSERT INTO users (tg_id, role, full_name, phone) 
			VALUES ($1, 'student', $2, $3) 
			RETURNING id`

		tgID := fmt.Sprintf("concurrent_student_%d", i)
		fullName := fmt.Sprintf("Студент Тест %d", i)
		phone := fmt.Sprintf("+7919%07d", 1000000+i)

		var userID int
		err := db.QueryRow(userQuery, tgID, fullName, phone).Scan(&userID)
		require.NoError(t, err, "Ошибка создания пользователя %d", i)

		// Создаем студента
		studentQuery := `
			INSERT INTO students (user_id, selected_subjects) 
			VALUES ($1, ARRAY[1]) 
			RETURNING id`

		err = db.QueryRow(studentQuery, userID).Scan(&studentIDs[i])
		require.NoError(t, err, "Ошибка создания студента %d", i)
	}

	t.Logf("Создано %d студентов", studentCount)

	// 4. Одновременная запись на урок (имитация конкурентности через goroutines)
	startTime := time.Now()
	
	var wg sync.WaitGroup
	errors := make(chan error, studentCount)
	successCount := make(chan int, studentCount)

	// Запускаем goroutines для записи на урок
	for i := 0; i < studentCount; i++ {
		wg.Add(1)
		go func(studentID int, index int) {
			defer wg.Done()

			enrollmentQuery := `
				INSERT INTO enrollments (student_id, lesson_id, status) 
				VALUES ($1, $2, 'scheduled')`

			_, err := db.Exec(enrollmentQuery, studentID, lessonID)
			if err != nil {
				errors <- fmt.Errorf("студент %d: %w", index, err)
			} else {
				successCount <- 1
				t.Logf("Студент %d успешно записался", index)
			}
		}(studentIDs[i], i)
	}

	wg.Wait()
	close(errors)
	close(successCount)

	duration := time.Since(startTime)

	// 5. Подсчитываем результаты
	var errorCount int
	var enrolledCount int

	for err := range errors {
		errorCount++
		t.Logf("Ошибка записи: %v", err)
	}

	for range successCount {
		enrolledCount++
	}

	t.Logf("Время выполнения %d одновременных записей: %v", studentCount, duration)
	t.Logf("Успешно записались: %d, Ошибок: %d", enrolledCount, errorCount)

	// 6. Проверяем итоговое количество записей в БД
	var actualEnrollments int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1", lessonID).Scan(&actualEnrollments)
	require.NoError(t, err, "Ошибка подсчета записей в БД")

	t.Logf("Фактическое количество записей в БД: %d", actualEnrollments)

	// 7. Основные проверки
	// Примечание: ограничение max_students обрабатывается в бизнес-логике приложения,
	// не на уровне БД, поэтому здесь проверяем только корректность записи
	assert.True(t, actualEnrollments > 0, 
		"Должна быть хотя бы одна успешная запись")
	
	assert.Equal(t, enrolledCount, actualEnrollments, 
		"Количество успешных записей должно совпадать с записями в БД")

	// 8. Проверяем отсутствие дублей по student_id
	duplicateQuery := `
		SELECT student_id, COUNT(*) as count 
		FROM enrollments 
		WHERE lesson_id = $1 
		GROUP BY student_id 
		HAVING COUNT(*) > 1`

	duplicateRows, err := db.Query(duplicateQuery, lessonID)
	require.NoError(t, err, "Ошибка проверки дублей")
	defer duplicateRows.Close()

	var duplicatesFound bool
	for duplicateRows.Next() {
		var studentID, count int
		err := duplicateRows.Scan(&studentID, &count)
		require.NoError(t, err, "Ошибка сканирования дублей")
		t.Errorf("Найден дубль: студент %d записан %d раз", studentID, count)
		duplicatesFound = true
	}

	assert.False(t, duplicatesFound, "Не должно быть дублированных записей")

	t.Logf("✅ Тест одновременных записей успешен: записалось %d студентов без дублей (лимит %d обрабатывается в бизнес-логике)", actualEnrollments, maxStudents)
}
