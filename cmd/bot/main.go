package main

import (
	"log"

	"github.com/joho/godotenv"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"constellation-school-bot/internal/config"
	"constellation-school-bot/internal/database"
	"constellation-school-bot/internal/handlers"
)

func main() {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем системные переменные")
	}
	
	cfg := config.Load()
	
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Инициализируем rate limiter
	handlers.InitializeRateLimiter(db)
	log.Println("Rate limiter инициализирован")

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	bot.Debug = true // Включаем debug режим
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handlers.HandleUpdate(bot, update, db)
	}
}