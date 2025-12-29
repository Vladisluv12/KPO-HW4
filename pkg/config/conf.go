package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    string
	DatabaseURL   string
	RunMigrations bool
	MigrationsDir string
	RabbitMQURL   string
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnv("PORT", "8081"),
		DatabaseURL:   getEnv("DB_CONNECTION_STRING", "postgres://user:password@postgres:5432/file_storage?sslmode=disable"),
		RunMigrations: parseBool(getEnv("RUN_MIGRATIONS", "true")),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "/migrations"),
		RabbitMQURL:   getEnv("RABBITMQ_URL", "amqp://user:password@localhost:5672/"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseBool(value string) bool {
	if i, err := strconv.ParseBool(value); err == nil {
		return i
	}
	return true
}
