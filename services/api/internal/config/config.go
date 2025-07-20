package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/tokane888/router-manager-go/pkg/logger"
	"github.com/tokane888/router-manager-go/services/api/internal/router"
)

// Config 環境変数を読み取り、各struct向けのConfigを保持
type Config struct {
	Env          string
	RouterConfig router.RouterConfig
	Logger       logger.LoggerConfig
	// 必要に応じてDatabaseConfig等各structへ注入する設定追加
}

// LoadConfig loads environment variables into Config
func LoadConfig(version string) (*Config, error) {
	env := getEnv("ENV", "local")
	envFile := ".env/.env." + env
	err := godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", envFile, err)
	}

	port, err := getIntEnv("API_PORT", 8080)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Env: env,
		RouterConfig: router.RouterConfig{
			Port: port,
		},
		Logger: logger.LoggerConfig{
			AppName:    getEnv("APP_NAME", ""),
			AppVersion: version,
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "local"),
		},
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) (int, error) {
	if s, exists := os.LookupEnv(key); exists {
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("invalid value for environment variable %s: %q (expected integer): %w", key, s, err)
		}
		return i, nil
	}
	return fallback, nil
}
