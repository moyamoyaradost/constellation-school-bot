package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// Подключение к БД
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://constellation_user:constellation_pass@localhost:5433/constellation_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Ошибка соединения с БД: %v", err)
	}

	fmt.Println("✅ Подключение к базе данных успешно")

	// Проверим, что у нас есть правильная схема
	checkSchema(db)
	
	// Проверим уроки и их статусы
	checkLessons(db)
	
	// Проверим записи и их статусы
	checkEnrollments(db)
}

func checkSchema(db *sql.DB) {
	fmt.Println("\n📋 Проверка схемы базы данных:")
	
	// Проверим таблицы
	tables := []string{"users", "students", "subjects", "teachers", "lessons", "enrollments"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ Ошибка проверки таблицы %s: %v\n", table, err)
		} else {
			fmt.Printf("✅ Таблица %s: %d записей\n", table, count)
		}
	}
}

func checkLessons(db *sql.DB) {
	fmt.Println("\n📚 Проверка уроков:")
	
	rows, err := db.Query(`
		SELECT l.id, s.name, l.start_time, l.status, l.max_students,
			COUNT(CASE WHEN e.status IN ('pending', 'confirmed') THEN e.id END) as enrolled_count
		FROM lessons l
		JOIN subjects s ON l.subject_id = s.id
		LEFT JOIN enrollments e ON l.id = e.lesson_id
		WHERE l.status IN ('scheduled', 'cancelled')
		GROUP BY l.id, s.name, l.start_time, l.status, l.max_students
		ORDER BY l.start_time
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("❌ Ошибка получения уроков: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-5s %-15s %-20s %-10s %-8s %-10s\n", "ID", "Предмет", "Время", "Статус", "Макс", "Записано")
	fmt.Println(strings.Repeat("-", 70))

	for rows.Next() {
		var id, maxStudents, enrolledCount int
		var subject, status string
		var startTime string
		
		err := rows.Scan(&id, &subject, &startTime, &status, &maxStudents, &enrolledCount)
		if err != nil {
			fmt.Printf("❌ Ошибка чтения урока: %v\n", err)
			continue
		}
		
		fmt.Printf("%-5d %-15s %-20s %-10s %-8d %-10d\n", 
			id, subject, startTime[:16], status, maxStudents, enrolledCount)
	}
}

func checkEnrollments(db *sql.DB) {
	fmt.Println("\n👥 Проверка записей:")
	
	rows, err := db.Query(`
		SELECT e.id, u.full_name, s.name, l.start_time, e.status
		FROM enrollments e
		JOIN students st ON e.student_id = st.id
		JOIN users u ON st.user_id = u.id
		JOIN lessons l ON e.lesson_id = l.id
		JOIN subjects s ON l.subject_id = s.id
		ORDER BY l.start_time
		LIMIT 10
	`)
	if err != nil {
		fmt.Printf("❌ Ошибка получения записей: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("%-5s %-20s %-15s %-20s %-10s\n", "ID", "Студент", "Предмет", "Время", "Статус")
	fmt.Println(strings.Repeat("-", 80))

	for rows.Next() {
		var id int
		var student, subject, status string
		var startTime string
		
		err := rows.Scan(&id, &student, &subject, &startTime, &status)
		if err != nil {
			fmt.Printf("❌ Ошибка чтения записи: %v\n", err)
			continue
		}
		
		fmt.Printf("%-5d %-20s %-15s %-20s %-10s\n", 
			id, student, subject, startTime[:16], status)
	}
}
