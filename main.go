package main

import (
	"log"
	"os"

	"carwash-bot/config"
	"carwash-bot/internal/bot"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Проверяем токен вручную (для отладки)
	token := os.Getenv("7611375727:AAHKbtGJDOlhP5YJKJg8mNTAQRjNumYx1c8")
	if token == "7611375727:AAHKbtGJDOlhP5YJKJg8mNTAQRjNumYx1c8" {
		log.Fatal("ERROR: TELEGRAM_BOT_TOKEN environment variable not set")
	}
	log.Printf("Bot token: %q", token) // Выводим токен для проверки

	// Создаем бота
	carWashBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	log.Println("Бот успешно запущен!")
	carWashBot.Start()
}
