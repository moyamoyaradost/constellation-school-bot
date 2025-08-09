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

// Настройка тестовой БД
func setupTestDB(t *testing.T) *sql.DB {
	ctx := context.Background()
	
	postgresContainer, err := postgres.Run(ctx, "postgres:16",
postgres.WithDatabase("testdb"),
postgres.WithUsername("testuser"), 
postgres.WithPassword("testpass"),
testcontainers.WithWaitStrategy(
wait.ForLog("database system is ready to accept connections").
WithOccurrence(2).WithStartupTimeout(30*time.Second)),
)
	require.NoError(t, err)

	connectionString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("postgres", connectionString)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, createTables(db))
	require.NoError(t, insertDefaultSubjects(db))

	t.Cleanup(func() {
		db.Close()
		postgresContainer.Terminate(ctx)
	})
	return db
}

// Тест создания пользователей
func TestCreateUsers(t *testing.T) {
	db := setupTestDB(t)

	query := "INSERT INTO users (tg_id, role, full_name, phone) VALUES ($1, $2, $3, $4)"
	_, err := db.Exec(query, "123456789", "student", "Тест Студент", "+79001234567")
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE tg_id = '123456789'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// Тест записи на урок
func TestEnrollmentFlow(t *testing.T) {
	db := setupTestDB(t)

	// Создание пользователей
	_, err := db.Exec("INSERT INTO users (tg_id, role, full_name, phone) VALUES ($1, $2, $3, $4)", 
"teacher1", "teacher", "Учитель", "+79001111111")
	require.NoError(t, err)
	
	_, err = db.Exec("INSERT INTO users (tg_id, role, full_name, phone) VALUES ($1, $2, $3, $4)", 
"student1", "student", "Студент", "+79002222222")
	require.NoError(t, err)

	// Создание учителя
	var teacherUserID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = 'teacher1'").Scan(&teacherUserID)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO teachers (user_id) VALUES ($1)", teacherUserID)
	require.NoError(t, err)

	var teacherID int
	err = db.QueryRow("SELECT id FROM teachers WHERE user_id = $1", teacherUserID).Scan(&teacherID)
	require.NoError(t, err)

	// Создание урока
	var lessonID int
	err = db.QueryRow("INSERT INTO lessons (teacher_id, subject_id, start_time, max_students, status) VALUES ($1, 1, NOW() + INTERVAL '1 day', 10, 'active') RETURNING id", teacherID).Scan(&lessonID)
	require.NoError(t, err)

	// Создание студента
	var studentUserID int
	err = db.QueryRow("SELECT id FROM users WHERE tg_id = 'student1'").Scan(&studentUserID)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO students (user_id) VALUES ($1)", studentUserID)
	require.NoError(t, err)

	var studentID int
	err = db.QueryRow("SELECT id FROM students WHERE user_id = $1", studentUserID).Scan(&studentID)
	require.NoError(t, err)

	// Запись на урок
	_, err = db.Exec("INSERT INTO enrollments (student_id, lesson_id, status) VALUES ($1, $2, 'enrolled')", studentID, lessonID)
	assert.NoError(t, err)

	// Проверка
	var enrolledCount int
	err = db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE lesson_id = $1 AND status = 'enrolled'", lessonID).Scan(&enrolledCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, enrolledCount)
}

// Тест безопасности при параллельных операциях
func TestConcurrentSafety(t *testing.T) {
	db := setupTestDB(t)

	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			query := "INSERT INTO users (tg_id, role, full_name, phone) VALUES ($1, $2, $3, $4)"
			_, err := db.Exec(query, 
fmt.Sprintf("concurrent_%d", id), "student", 
fmt.Sprintf("Concurrent User %d", id), 
fmt.Sprintf("+7900000%04d", id))
			errors <- err
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		assert.NoError(t, err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE tg_id LIKE 'concurrent_%'").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
