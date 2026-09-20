package config

import (
	"os"
)

type Config struct {
	AppName string
	AppPort string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	RedisHost string
	RedisPort string

	LogLevel string
}

func Load() Config {
	return Config{
		AppName: getEnv("APP_NAME", "order-service"),
		AppPort: getEnv("APP_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "orderflow"),
		DBUser:     getEnv("DB_USER", "orderflow"),
		DBPassword: getEnv("DB_PASSWORD", "orderflow_password"),

		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		LogLevel: getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)

	if !exists || value == "" {
		return fallback
	}

	return value
}
