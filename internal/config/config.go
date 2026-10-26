package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerConfig
	AppEnv string
}

type ServerConfig struct {
	SrvPort string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading env : %w", err)
	}
	config := &Config{
		SrvPort: getEnv("PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
	}
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
