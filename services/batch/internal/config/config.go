package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/tokane888/router-manager-go/pkg/logger"
)

// Config 環境変数を読み取り、各struct向けのConfigを保持
type Config struct {
	Env    string
	Logger logger.LoggerConfig
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

	cfg := &Config{
		Env: env,
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
