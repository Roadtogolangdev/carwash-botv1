package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken  string
	AdminID   int64
	StartTime int
	EndTime   int
	Days      map[string]string // Новое поле для дней недели
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := &Config{
		BotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		AdminID:   getEnvAsInt64("ADMIN_CHAT_ID", 0),
		StartTime: 8,
		EndTime:   20,
		Days: map[string]string{ // Русские названия дней
			"Monday":    "Понедельник",
			"Tuesday":   "Вторник",
			"Wednesday": "Среда",
			"Thursday":  "Четверг",
			"Friday":    "Пятница",
			"Saturday":  "Суббота",
			"Sunday":    "Воскресенье",
		},
	}

	return cfg
}
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			return num
		}
	}
	return defaultValue
}
