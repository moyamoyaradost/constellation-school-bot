package main

import (
	"fmt"
	"log"
	
	"github.com/joho/godotenv"
	"constellation-school-bot/internal/config"
	"constellation-school-bot/internal/database"
)

func main() {
	fmt.Println("🔍 Тест подключения к базе данных...")
	
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Файл .env не найден, используем системные переменные")
	}
	
	cfg := config.Load()
	fmt.Printf("📊 Конфигурация БД: %s:%s@%s:%s/%s\n", 
		cfg.DBUser, "***", cfg.DBHost, cfg.DBPort, cfg.DBName)
	
	// Подключаемся к БД
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("❌ Ошибка подключения к БД:", err)
	}
	defer db.Close()
	
	fmt.Println("✅ Подключение успешно!")
	
	// Проверяем количество таблиц
	var tableCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public'`).Scan(&tableCount)
	
	if err != nil {
		fmt.Printf("⚠️  Ошибка подсчета таблиц: %v\n", err)
	} else {
		fmt.Printf("📋 Количество таблиц: %d\n", tableCount)
	}
	
	// Проверяем предметы
	var subjectCount int
	err = db.QueryRow("SELECT COUNT(*) FROM subjects").Scan(&subjectCount)
	if err != nil {
		fmt.Printf("⚠️  Ошибка подсчета предметов: %v\n", err)
	} else {
		fmt.Printf("📚 Количество предметов: %d\n", subjectCount)
	}
	
	// Проверяем пользователей
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		fmt.Printf("⚠️  Ошибка подсчета пользователей: %v\n", err)
	} else {
		fmt.Printf("👥 Количество пользователей: %d\n", userCount)
	}
	
	// Показываем предметы
	fmt.Println("\n📚 Доступные предметы:")
	rows, err := db.Query("SELECT name, code, category FROM subjects ORDER BY id")
	if err != nil {
		fmt.Printf("⚠️  Ошибка получения предметов: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var name, code, category string
			if err := rows.Scan(&name, &code, &category); err != nil {
				continue
			}
			fmt.Printf("  • %s [%s] (%s)\n", name, code, category)
		}
	}
	
	fmt.Println("\n🎯 База данных работает корректно!")
}
