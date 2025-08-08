package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"constellation-school-bot/internal/config"
	"constellation-school-bot/internal/database"
	"constellation-school-bot/internal/handlers"
)

func main() {
	cfg := config.Load()
	
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	log.Printf("Бот запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handlers.HandleUpdate(bot, update, db)
	}
}