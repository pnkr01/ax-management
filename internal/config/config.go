package config

import (
	"os"
	"sync"
)

type Config struct {
	AppPort     string
	PostgresDSN string
	RedisAddr   string
	Environment string
	JWTSecret   string
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns a singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{
			AppPort:     getEnv("APP_PORT", "8081"),
			PostgresDSN: getEnv("DATABASE_URL", "host=localhost user=ax_admin password=ax_password dbname=ax_management port=5432 sslmode=disable"),
			RedisAddr:   getEnv("REDIS_URL", "localhost:6379"),
			Environment: os.Getenv("APP_ENV"),    // "production" or "development"
			JWTSecret:   os.Getenv("JWT_SECRET"), // Must be set in terminal
		}
	})
	return instance
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
