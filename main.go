package main

import (
	"carwash-bot/internal/storage"
	"log"
	"os"

	"carwash-bot/config"
	"carwash-bot/internal/bot"
	"github.com/joho/godotenv"
)

func main() {
	// Загружаем конфигурацию
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	// Используем их в коде
	log.Println("Bot Token:", botToken)
	log.Println("Chat ID:", chatID)
	cfg := config.Load()

	db, err := storage.NewSQLiteStorage("carwash.db")
	if err != nil {
		log.Fatal("Ошибка инициализации БД:", err)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		log.Fatal("Ошибка создания таблиц:", err)
	}
	// Создаем бота
	carWashBot, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	log.Println("Бот успешно запущен!")
	carWashBot.Start()
}
