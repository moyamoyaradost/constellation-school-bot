package config

import (
	"os"
	"strconv"
)

type Config struct {
	BotToken      string
	SuperUserID   int64
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
}

func Load() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	superUserID, _ := strconv.ParseInt(getEnv("BOT_SUPERUSER_ID", "0"), 10, 64)

	return &Config{
		BotToken:      getEnv("BOT_TOKEN", ""),
		SuperUserID:   superUserID,
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "constellation_user"),
		DBPassword:    getEnv("DB_PASSWORD", "constellation_pass"),
		DBName:        getEnv("DB_NAME", "constellation_db"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
