package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tokane888/router-manager-go/pkg/db"
	"github.com/tokane888/router-manager-go/pkg/logger"
)

// Config represents the application configuration
type Config struct {
	Env        string
	Logger     logger.LoggerConfig
	Database   db.Config
	DNS        DNSConfig
	Firewall   FirewallConfig
	Processing ProcessingConfig
}

// DNSConfig contains DNS resolution configuration
type DNSConfig struct {
	Timeout           time.Duration
	RetryAttempts     int
	DiscoveryWaitTime time.Duration
}

// FirewallConfig contains firewall management configuration
type FirewallConfig struct {
	DryRun         bool
	CommandTimeout time.Duration
	Table          string
	Chain          string
}

// ProcessingConfig contains domain processing configuration
type ProcessingConfig struct {
	MaxConcurrency int // Configurable via environment variable, default 10
	DomainTimeout  time.Duration
}

// LoadConfig loads environment variables into Config
func LoadConfig(version string) (*Config, error) {
	env := getEnv("ENV", "local")
	envFile := ".env/.env." + env
	err := godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", envFile, err)
	}

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

	discoveryWaitTime, err := getDurationEnv("DNS_DISCOVERY_WAIT_TIME", 100*time.Millisecond)
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

	cfg := &Config{
		Env: env,
		Logger: logger.LoggerConfig{
			AppName:    getEnv("APP_NAME", ""),
			AppVersion: version,
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "local"),
		},
		DNS: DNSConfig{
			Timeout:           dnsTimeout,
			RetryAttempts:     dnsRetryAttempts,
			DiscoveryWaitTime: discoveryWaitTime,
		},
		Firewall: FirewallConfig{
			DryRun:         firewallDryRun,
			CommandTimeout: firewallTimeout,
			Table:          getEnv("FIREWALL_TABLE", "ip filter"),
			Chain:          getEnv("FIREWALL_CHAIN", "OUTPUT"),
		},
		Processing: ProcessingConfig{
			MaxConcurrency: maxConcurrency,
			DomainTimeout:  domainTimeout,
		},
		// TODO: Database config will be implemented in subsequent tasks
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
