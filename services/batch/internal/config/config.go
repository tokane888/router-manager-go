package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tokane888/router-manager-go/pkg/db"
	"github.com/tokane888/router-manager-go/pkg/logger"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/dns"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/firewall"
	"github.com/tokane888/router-manager-go/services/batch/internal/usecase"
)

// Config represents the application configuration
type Config struct {
	Env        string
	Logger     logger.LoggerConfig
	Database   db.Config
	DNS        dns.DNSConfig
	Firewall   firewall.FirewallConfig
	Processing usecase.ProcessingConfig
}

// LoadConfig loads configuration from environment variables and defaults
// Priority: environment variables > defaults
func LoadConfig(version string) (*Config, error) {
	// Determine environment from environment variable
	env := getEnv("ENV", "local")

	// Load environment file (optional)
	envFile := ".env/.env." + env
	_ = godotenv.Load(envFile) // Ignore error if file doesn't exist

	maxConcurrency, err := getIntEnv("MAX_CONCURRENCY", 10)
	if err != nil {
		return nil, err
	}

	dnsTimeout, err := getDurationEnv("DNS_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	dnsRetryAttempts, err := getIntEnv("DNS_RETRY_ATTEMPTS", 3)
	if err != nil {
		return nil, err
	}

	firewallDryRun, err := getBoolEnv("FIREWALL_DRY_RUN", true)
	if err != nil {
		return nil, err
	}

	firewallTimeout, err := getDurationEnv("FIREWALL_COMMAND_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	domainTimeout, err := getDurationEnv("DOMAIN_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}

	// Load configuration from environment variables
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "local")

	cfg := &Config{
		Env: env,
		Logger: logger.LoggerConfig{
			AppName:    getEnv("APP_NAME", ""),
			AppVersion: version,
			Level:      logLevel,
			Format:     logFormat,
		},
		DNS: dns.DNSConfig{
			Timeout:       dnsTimeout,
			RetryAttempts: dnsRetryAttempts,
		},
		Firewall: firewall.FirewallConfig{
			DryRun:         firewallDryRun,
			CommandTimeout: firewallTimeout,
			Table:          getEnv("FIREWALL_TABLE", "ip filter"),
			Chain:          getEnv("FIREWALL_CHAIN", "OUTPUT"),
		},
		Processing: usecase.ProcessingConfig{
			MaxConcurrency: maxConcurrency,
			DomainTimeout:  domainTimeout,
		},
		Database: db.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			DBName:   getEnv("DB_NAME", "router_manager"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
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

func getBoolEnv(key string, fallback bool) (bool, error) {
	if s, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return false, fmt.Errorf("invalid value for environment variable %s: %q (expected boolean): %w", key, s, err)
		}
		return b, nil
	}
	return fallback, nil
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	if s, exists := os.LookupEnv(key); exists {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid value for environment variable %s: %q (expected duration): %w", key, s, err)
		}
		return d, nil
	}
	return fallback, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	// Validate environment
	if cfg.Env != "local" && cfg.Env != "prod" {
		return fmt.Errorf("invalid environment: %s (must be 'local' or 'prod')", cfg.Env)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[cfg.Logger.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logger.Level)
	}

	// Validate log format
	if cfg.Logger.Format != "local" && cfg.Logger.Format != "cloud" {
		return fmt.Errorf("invalid log format: %s (must be 'local' or 'cloud')", cfg.Logger.Format)
	}

	// Validate processing configuration
	if cfg.Processing.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive, got: %d", cfg.Processing.MaxConcurrency)
	}
	if cfg.Processing.MaxConcurrency > 100 {
		return fmt.Errorf("max concurrency too high: %d (maximum: 100)", cfg.Processing.MaxConcurrency)
	}

	// Validate DNS configuration
	if cfg.DNS.Timeout <= 0 {
		return fmt.Errorf("DNS timeout must be positive, got: %v", cfg.DNS.Timeout)
	}
	if cfg.DNS.RetryAttempts < 0 {
		return fmt.Errorf("DNS retry attempts cannot be negative, got: %d", cfg.DNS.RetryAttempts)
	}
	if cfg.DNS.RetryAttempts > 10 {
		return fmt.Errorf("DNS retry attempts too high: %d (maximum: 10)", cfg.DNS.RetryAttempts)
	}

	// Validate firewall configuration
	if cfg.Firewall.CommandTimeout <= 0 {
		return fmt.Errorf("firewall command timeout must be positive, got: %v", cfg.Firewall.CommandTimeout)
	}
	if cfg.Firewall.Table == "" {
		return errors.New("firewall table cannot be empty")
	}
	if cfg.Firewall.Chain == "" {
		return errors.New("firewall chain cannot be empty")
	}

	// Validate domain timeout
	if cfg.Processing.DomainTimeout <= 0 {
		return fmt.Errorf("domain timeout must be positive, got: %v", cfg.Processing.DomainTimeout)
	}

	return nil
}
