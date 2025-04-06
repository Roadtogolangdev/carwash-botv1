package main

import (
	"log"
	"os"

	"carwash-bot/config"
	"carwash-bot/internal/bot"
	"carwash-bot/internal/storage"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Получаем переменные окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	log.Println("Bot Token:", botToken)
	log.Println("Chat ID:", chatID)

	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем БД
	db := storage.New()
	defer db.DB.Close()

	// Создаём и запускаем бота
	carWashBot, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	log.Println("Бот успешно запущен!")
	carWashBot.Start()
}
