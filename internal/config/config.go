package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerIP           string
	APIPort            int
	Username           string
	Password           string
	DatabasePath       string
	AutoResetInterval  int
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		ServerIP:          getEnv("SERVER_IP", "localhost"),
		APIPort:           getEnvAsInt("API_PORT", 8080),
		Username:          getEnv("PROXY_USERNAME", "admin"),
		Password:          getEnv("PROXY_PASSWORD", ""),
		DatabasePath:      getEnv("DATABASE_PATH", "./data/proxies.db"),
		AutoResetInterval: getEnvAsInt("AUTO_RESET_INTERVAL", 10),
	}

	// Validate required fields
	if cfg.Password == "" {
		return nil, fmt.Errorf("PROXY_PASSWORD environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
