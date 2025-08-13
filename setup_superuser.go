package main

import (
	"database/sql"
	"fmt"
	"log"

	"constellation-school-bot/internal/config"
	"constellation-school-bot/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()
	
	// Подключаемся к БД
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Ваш Telegram ID
	userID := int64(7231695922)
	
	// Проверяем есть ли уже пользователь
	var existingRole string
	err = db.QueryRow("SELECT role FROM users WHERE telegram_id = $1", userID).Scan(&existingRole)
	
	if err == sql.ErrNoRows {
		// Добавляем нового superuser
		_, err = db.Exec(`
			INSERT INTO users (telegram_id, full_name, role, is_active, created_at) 
			VALUES ($1, $2, $3, $4, NOW())`,
			userID, "Kate (Superuser)", "superuser", true)
		
		if err != nil {
			log.Fatal("Ошибка добавления superuser:", err)
		}
		fmt.Printf("✅ Superuser добавлен: ID=%d\n", userID)
	} else if err != nil {
		log.Fatal("Ошибка проверки пользователя:", err)
	} else {
		fmt.Printf("ℹ️ Пользователь уже существует: ID=%d, роль=%s\n", userID, existingRole)
		
		// Обновляем роль до superuser если это не так
		if existingRole != "superuser" {
			_, err = db.Exec("UPDATE users SET role = 'superuser' WHERE telegram_id = $1", userID)
			if err != nil {
				log.Fatal("Ошибка обновления роли:", err)
			}
			fmt.Printf("✅ Роль обновлена до superuser: ID=%d\n", userID)
		}
	}
	
	// Тестируем функцию проверки роли
	var role string
	err = db.QueryRow("SELECT role FROM users WHERE telegram_id = $1", userID).Scan(&role)
	if err != nil {
		log.Fatal("Ошибка проверки роли:", err)
	}
	
	fmt.Printf("🔍 Проверка роли: telegram_id=%d, role='%s'\n", userID, role)
	
	// Симулируем проверку как в handleAdminCommand
	if err == nil && (role == "admin" || role == "superuser") {
		fmt.Printf("✅ Проверка админских прав: УСПЕШНА\n")
	} else {
		fmt.Printf("❌ Проверка админских прав: ОШИБКА (err=%v, role='%s')\n", err, role)
	}
	
	fmt.Println("\n✨ Готово! Теперь вы можете использовать команду /notify_students")
}
