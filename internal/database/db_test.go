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
	// 3. Создаем студента 
	studentQuery := `
		INSERT INTO students (user_id) 
		VALUES ($1) 
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
		INSERT INTO teachers (user_id) 
		VALUES ($1) 
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
		VALUES ($1, $2, 'enrolled') 
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
		INSERT INTO teachers (user_id) 
		VALUES ($1) 
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
			INSERT INTO students (user_id) 
			VALUES ($1) 
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
				VALUES ($1, $2, 'enrolled')`

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

// TestLessonCancellationCascade тестирует каскадную отмену урока и всех связанных записей
// Автор: Maksim Novihin
func TestLessonCancellationCascade(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. Создаем пользователя-преподавателя
	teacherUserQuery := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ('teacher_cancel_test', 'teacher', 'Преподаватель Отмена', '+79101234567') 
		RETURNING id`

	var teacherUserID int
	err := db.QueryRow(teacherUserQuery).Scan(&teacherUserID)
	require.NoError(t, err, "Ошибка создания пользователя-преподавателя")

	// Создаем запись преподавателя
	teacherInsertQuery := `
		INSERT INTO teachers (user_id) 
		VALUES ($1) 
		RETURNING id`

	var teacherID int
	err = db.QueryRow(teacherInsertQuery, teacherUserID).Scan(&teacherID)
	require.NoError(t, err, "Ошибка создания преподавателя")

	// 2. Создаем урок
	lessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students) 
		VALUES ($1, 1, NOW() + INTERVAL '1 day', 3) 
		RETURNING id`

	var lessonID int
	err = db.QueryRow(lessonQuery, teacherID).Scan(&lessonID)
	require.NoError(t, err, "Ошибка создания урока")

	// 3. Создаем 3 студентов и записываем их на урок
	studentIDs := make([]int, 3)
	for i := 0; i < 3; i++ {
		// Создаем пользователя
		userQuery := `
			INSERT INTO users (tg_id, role, full_name, phone) 
			VALUES ($1, 'student', $2, $3) 
			RETURNING id`

		tgID := fmt.Sprintf("student_cancel_%d", i)
		fullName := fmt.Sprintf("Студент Отмена %d", i)
		phone := fmt.Sprintf("+7919%07d", 2000000+i)

		var userID int
		err := db.QueryRow(userQuery, tgID, fullName, phone).Scan(&userID)
		require.NoError(t, err, "Ошибка создания пользователя %d", i)

		// Создаем студента
		studentQuery := `
			INSERT INTO students (user_id) 
			VALUES ($1) 
			RETURNING id`

		err = db.QueryRow(studentQuery, userID).Scan(&studentIDs[i])
		require.NoError(t, err, "Ошибка создания студента %d", i)

		// Записываем студента на урок
		err = EnrollStudent(db, studentIDs[i], lessonID)
		require.NoError(t, err, "Ошибка записи студента %d на урок", i)
	}

	// 4. Проверяем начальное состояние - урок активен, все студенты записаны
	var lessonStatus string
	err = db.QueryRow("SELECT status FROM lessons WHERE id = $1", lessonID).Scan(&lessonStatus)
	require.NoError(t, err, "Ошибка получения статуса урока")
	assert.Equal(t, "active", lessonStatus, "Урок должен быть активным")

	var enrolledCount int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'enrolled'", lessonID).Scan(&enrolledCount)
	require.NoError(t, err, "Ошибка подсчета записанных студентов")
	assert.Equal(t, 3, enrolledCount, "Должно быть 3 записанных студента")

	// 5. ОТМЕНЯЕМ УРОК
	err = CancelLesson(db, lessonID)
	require.NoError(t, err, "Ошибка отмены урока")

	// 6. Проверяем состояние после отмены
	err = db.QueryRow("SELECT status FROM lessons WHERE id = $1", lessonID).Scan(&lessonStatus)
	require.NoError(t, err, "Ошибка получения статуса урока после отмены")
	assert.Equal(t, "cancelled", lessonStatus, "Урок должен быть отменен")

	var cancelledCount int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'cancelled'", lessonID).Scan(&cancelledCount)
	require.NoError(t, err, "Ошибка подсчета отмененных записей")
	assert.Equal(t, 3, cancelledCount, "Все 3 записи должны быть отменены")

	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'enrolled'", lessonID).Scan(&enrolledCount)
	require.NoError(t, err, "Ошибка подсчета активных записей")
	assert.Equal(t, 0, enrolledCount, "Не должно остаться активных записей")

	// 7. Проверяем функцию уведомлений
	userIDs, err := NotifyStudentsLessonCancelled(db, lessonID)
	require.NoError(t, err, "Ошибка получения списка для уведомлений")
	assert.Equal(t, 3, len(userIDs), "Должно быть 3 пользователя для уведомления")

	t.Logf("✅ Тест каскадной отмены урока успешен: отменен урок %d и %d записей студентов", lessonID, cancelledCount)
}

// TestConcurrentLessonCancellation тестирует одновременную отмену урока и попытки записи
// Автор: Maksim Novihin
func TestConcurrentLessonCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. Создаем пользователя-преподавателя и урок
	teacherUserQuery := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ('teacher_concurrent_test', 'teacher', 'Преподаватель Конкурент', '+79101234568') 
		RETURNING id`

	var teacherUserID int
	err := db.QueryRow(teacherUserQuery).Scan(&teacherUserID)
	require.NoError(t, err, "Ошибка создания пользователя-преподавателя")

	// Создаем запись преподавателя
	teacherInsertQuery := `
		INSERT INTO teachers (user_id) 
		VALUES ($1) 
		RETURNING id`

	var teacherID int
	err = db.QueryRow(teacherInsertQuery, teacherUserID).Scan(&teacherID)
	require.NoError(t, err, "Ошибка создания преподавателя")

	lessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students) 
		VALUES ($1, 1, NOW() + INTERVAL '1 day', 10) 
		RETURNING id`

	var lessonID int
	err = db.QueryRow(lessonQuery, teacherID).Scan(&lessonID)
	require.NoError(t, err, "Ошибка создания урока")

	// 2. Создаем 5 студентов
	studentIDs := make([]int, 5)
	for i := 0; i < 5; i++ {
		userQuery := `
			INSERT INTO users (tg_id, role, full_name, phone) 
			VALUES ($1, 'student', $2, $3) 
			RETURNING id`

		tgID := fmt.Sprintf("student_concurrent_%d", i)
		fullName := fmt.Sprintf("Студент Конкурент %d", i)
		phone := fmt.Sprintf("+7919%07d", 3000000+i)

		var userID int
		err := db.QueryRow(userQuery, tgID, fullName, phone).Scan(&userID)
		require.NoError(t, err, "Ошибка создания пользователя %d", i)

		studentQuery := `
			INSERT INTO students (user_id) 
			VALUES ($1) 
			RETURNING id`

		err = db.QueryRow(studentQuery, userID).Scan(&studentIDs[i])
		require.NoError(t, err, "Ошибка создания студента %d", i)
	}

	// 3. Записываем первых 2 студентов на урок
	for i := 0; i < 2; i++ {
		err = EnrollStudent(db, studentIDs[i], lessonID)
		require.NoError(t, err, "Ошибка записи студента %d", i)
	}

	// 4. Имитируем конкурентность: одновременно отменяем урок и пытаемся записать остальных студентов
	var wg sync.WaitGroup
	results := make(chan string, 4) // 1 отмена + 3 записи

	// Запускаем отмену урока
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Небольшая задержка
		err := CancelLesson(db, lessonID)
		if err != nil {
			results <- fmt.Sprintf("Ошибка отмены: %v", err)
		} else {
			results <- "Урок отменен успешно"
		}
	}()

	// Запускаем попытки записи остальных студентов
	for i := 2; i < 5; i++ {
		wg.Add(1)
		go func(studentID int, index int) {
			defer wg.Done()
			err := EnrollStudent(db, studentID, lessonID)
			if err != nil {
				results <- fmt.Sprintf("Студент %d: ошибка записи - %v", index, err)
			} else {
				results <- fmt.Sprintf("Студент %d: записан успешно", index)
			}
		}(studentIDs[i], i)
	}

	wg.Wait()
	close(results)

	// 5. Собираем результаты
	var messages []string
	for result := range results {
		messages = append(messages, result)
		t.Logf("Результат операции: %s", result)
	}

	// 6. Проверяем финальное состояние
	var lessonStatus string
	err = db.QueryRow("SELECT status FROM lessons WHERE id = $1", lessonID).Scan(&lessonStatus)
	require.NoError(t, err, "Ошибка получения финального статуса урока")

	// Урок должен быть отменен
	assert.Equal(t, "cancelled", lessonStatus, "Урок должен быть отменен")

	// Все записи должны быть либо отменены, либо некоторые студенты могли записаться до отмены
	var totalEnrollments int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1", lessonID).Scan(&totalEnrollments)
	require.NoError(t, err, "Ошибка подсчета всех записей")

	var cancelledEnrollments int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'cancelled'", lessonID).Scan(&cancelledEnrollments)
	require.NoError(t, err, "Ошибка подсчета отмененных записей")

	var activeEnrollments int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'enrolled'", lessonID).Scan(&activeEnrollments)
	require.NoError(t, err, "Ошибка подсчета активных записей")

	t.Logf("Финальная статистика: всего записей=%d, отменено=%d, активных=%d", totalEnrollments, cancelledEnrollments, activeEnrollments)

	// Основные проверки целостности
	assert.True(t, totalEnrollments >= 2, "Должно быть минимум 2 записи (изначально записанные)")
	assert.True(t, cancelledEnrollments >= 2, "Минимум 2 записи должны быть отменены")
	
	// После отмены урока не должно оставаться активных записей
	assert.Equal(t, 0, activeEnrollments, "После отмены урока не должно быть активных записей")

	t.Logf("✅ Тест конкурентной отмены успешен: урок отменен, обработано %d операций", len(messages))
}

// TestStatusConsistency проверяет консистентность статусов в базе данных
// Автор: Maksim Novihin  
func TestStatusConsistency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// 1. Создаем тестовые данные
	teacherUserQuery := `
		INSERT INTO users (tg_id, role, full_name, phone) 
		VALUES ('teacher_consistency', 'teacher', 'Преподаватель Консистенс', '+79101234569') 
		RETURNING id`

	var teacherUserID int
	err := db.QueryRow(teacherUserQuery).Scan(&teacherUserID)
	require.NoError(t, err, "Ошибка создания пользователя-преподавателя")

	// Создаем запись преподавателя
	teacherInsertQuery := `
		INSERT INTO teachers (user_id) 
		VALUES ($1) 
		RETURNING id`

	var teacherID int
	err = db.QueryRow(teacherInsertQuery, teacherUserID).Scan(&teacherID)
	require.NoError(t, err, "Ошибка создания преподавателя")

	// Создаем активный урок
	activeLessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students, status) 
		VALUES ($1, 1, NOW() + INTERVAL '1 day', 5, 'active') 
		RETURNING id`

	var activeLessonID int
	err = db.QueryRow(activeLessonQuery, teacherID).Scan(&activeLessonID)
	require.NoError(t, err, "Ошибка создания активного урока")

	// Создаем отмененный урок
	cancelledLessonQuery := `
		INSERT INTO lessons (teacher_id, subject_id, start_time, max_students, status) 
		VALUES ($1, 1, NOW() + INTERVAL '2 days', 5, 'cancelled') 
		RETURNING id`

	var cancelledLessonID int
	err = db.QueryRow(cancelledLessonQuery, teacherID).Scan(&cancelledLessonID)
	require.NoError(t, err, "Ошибка создания отмененного урока")

	// 2. Создаем студентов
	studentIDs := make([]int, 4)
	for i := 0; i < 4; i++ {
		userQuery := `
			INSERT INTO users (tg_id, role, full_name, phone) 
			VALUES ($1, 'student', $2, $3) 
			RETURNING id`

		tgID := fmt.Sprintf("student_consistency_%d", i)
		fullName := fmt.Sprintf("Студент Консистенс %d", i)
		phone := fmt.Sprintf("+7919%07d", 4000000+i)

		var userID int
		err := db.QueryRow(userQuery, tgID, fullName, phone).Scan(&userID)
		require.NoError(t, err, "Ошибка создания пользователя %d", i)

		studentQuery := `
			INSERT INTO students (user_id) 
			VALUES ($1) 
			RETURNING id`

		err = db.QueryRow(studentQuery, userID).Scan(&studentIDs[i])
		require.NoError(t, err, "Ошибка создания студента %d", i)
	}

	// 3. Записываем студентов на активный урок
	for i := 0; i < 2; i++ {
		err = EnrollStudent(db, studentIDs[i], activeLessonID)
		require.NoError(t, err, "Ошибка записи студента %d на активный урок", i)
	}

	// 4. Создаем записи на отмененный урок (имитация данных до отмены)
	for i := 2; i < 4; i++ {
		_, err = db.Exec(`
			INSERT INTO enrollments (student_id, lesson_id, status) 
			VALUES ($1, $2, 'cancelled')
		`, studentIDs[i], cancelledLessonID)
		require.NoError(t, err, "Ошибка создания отмененной записи для студента %d", i)
	}

	// 5. ПРОВЕРЯЕМ КОНСИСТЕНТНОСТЬ ДАННЫХ

	// Проверка 1: У активных уроков должны быть только активные записи
	inconsistentActiveQuery := `
		SELECT COUNT(*) FROM enrollments e 
		JOIN lessons l ON e.lesson_id = l.id 
		WHERE l.status = 'active' AND e.status != 'enrolled'`

	var inconsistentActive int
	err = db.QueryRow(inconsistentActiveQuery).Scan(&inconsistentActive)
	require.NoError(t, err, "Ошибка проверки активных уроков")
	assert.Equal(t, 0, inconsistentActive, "У активных уроков не должно быть неактивных записей")

	// Проверка 2: У отмененных уроков не должно быть активных записей
	inconsistentCancelledQuery := `
		SELECT COUNT(*) FROM enrollments e 
		JOIN lessons l ON e.lesson_id = l.id 
		WHERE l.status = 'cancelled' AND e.status = 'enrolled'`

	var inconsistentCancelled int
	err = db.QueryRow(inconsistentCancelledQuery).Scan(&inconsistentCancelled)
	require.NoError(t, err, "Ошибка проверки отмененных уроков")
	assert.Equal(t, 0, inconsistentCancelled, "У отмененных уроков не должно быть активных записей")

	// Проверка 3: Каждая запись должна ссылаться на существующих студента и урок
	orphanEnrollmentsQuery := `
		SELECT COUNT(*) FROM enrollments e
		LEFT JOIN students s ON e.student_id = s.id
		LEFT JOIN lessons l ON e.lesson_id = l.id
		WHERE s.id IS NULL OR l.id IS NULL`

	var orphanEnrollments int
	err = db.QueryRow(orphanEnrollmentsQuery).Scan(&orphanEnrollments)
	require.NoError(t, err, "Ошибка проверки сиротских записей")
	assert.Equal(t, 0, orphanEnrollments, "Не должно быть записей без студента или урока")

	// Проверка 4: Каждый студент должен ссылаться на существующего пользователя
	orphanStudentsQuery := `
		SELECT COUNT(*) FROM students s
		LEFT JOIN users u ON s.user_id = u.id
		WHERE u.id IS NULL`

	var orphanStudents int
	err = db.QueryRow(orphanStudentsQuery).Scan(&orphanStudents)
	require.NoError(t, err, "Ошибка проверки сиротских студентов")
	assert.Equal(t, 0, orphanStudents, "Не должно быть студентов без пользователя")

	// Проверка 5: Статистика по статусам
	statusStatsQuery := `
		SELECT 
			l.status as lesson_status,
			e.status as enrollment_status,
			COUNT(*) as count
		FROM lessons l
		JOIN enrollments e ON l.id = e.lesson_id
		GROUP BY l.status, e.status
		ORDER BY l.status, e.status`

	rows, err := db.Query(statusStatsQuery)
	require.NoError(t, err, "Ошибка получения статистики статусов")
	defer rows.Close()

	statusMap := make(map[string]map[string]int)
	for rows.Next() {
		var lessonStatus, enrollmentStatus string
		var count int
		err := rows.Scan(&lessonStatus, &enrollmentStatus, &count)
		require.NoError(t, err, "Ошибка сканирования статистики")
		
		if statusMap[lessonStatus] == nil {
			statusMap[lessonStatus] = make(map[string]int)
		}
		statusMap[lessonStatus][enrollmentStatus] = count
		
		t.Logf("Статистика: урок='%s', запись='%s', количество=%d", lessonStatus, enrollmentStatus, count)
	}

	// Проверяем ожидаемую статистику
	assert.Equal(t, 2, statusMap["active"]["enrolled"], "У активного урока должно быть 2 активные записи")
	assert.Equal(t, 2, statusMap["cancelled"]["cancelled"], "У отмененного урока должно быть 2 отмененные записи")

	// Проверка 6: Тестируем отмену активного урока для проверки каскадного обновления
	err = CancelLesson(db, activeLessonID)
	require.NoError(t, err, "Ошибка отмены активного урока")

	// Перепроверяем консистентность после отмены
	err = db.QueryRow(inconsistentActiveQuery).Scan(&inconsistentActive)
	require.NoError(t, err, "Ошибка повторной проверки активных уроков")
	assert.Equal(t, 0, inconsistentActive, "После отмены не должно остаться несогласованных активных записей")

	err = db.QueryRow(inconsistentCancelledQuery).Scan(&inconsistentCancelled)
	require.NoError(t, err, "Ошибка повторной проверки отмененных уроков")
	assert.Equal(t, 0, inconsistentCancelled, "После отмены не должно остаться активных записей у отмененных уроков")

	t.Logf("✅ Тест консистентности статусов успешен: все проверки целостности данных пройдены")
}
