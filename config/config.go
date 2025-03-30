package config

import (
	"os"
	"strconv"
)

type Config struct {
	BotToken  string
	AdminID   int64
	StartTime int // 8 (8:00)
	EndTime   int // 20 (20:00)
}

func Load() *Config {
	return &Config{
		BotToken:  getEnv("7611375727:AAHKbtGJDOlhP5YJKJg8mNTAQRjNumYx1c8", "7611375727:AAHKbtGJDOlhP5YJKJg8mNTAQRjNumYx1c8"),
		AdminID:   getEnvAsInt64("ADMIN_CHAT_ID", 0),
		StartTime: 8,
		EndTime:   20,
	}
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
