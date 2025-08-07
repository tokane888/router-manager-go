package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Default values for database connection pool configuration
const (
	DefaultMaxOpenConns = 25              // 25% of PostgreSQL default max_connections (100)
	DefaultMaxIdleConns = 5               // Reasonable number of idle connections
	DefaultMaxLifetime  = 5 * time.Minute // Connection lifetime
	DefaultMaxIdleTime  = 1 * time.Minute // Idle connection timeout
	DefaultSSLMode      = "disable"       // For local development; use "require" for production
)

// DB represents a database connection
type DB struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

// Config holds database connection configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string

	// Connection pool settings
	MaxOpenConns int           // Maximum number of open connections
	MaxIdleConns int           // Maximum number of idle connections
	MaxLifetime  time.Duration // Maximum amount of time a connection may be reused
	MaxIdleTime  time.Duration // Maximum amount of time a connection may be idle
}

// NewDefaultConfig creates a new Config with sensible default values
func NewDefaultConfig() Config {
	return Config{
		SSLMode:      DefaultSSLMode,
		MaxOpenConns: DefaultMaxOpenConns,
		MaxIdleConns: DefaultMaxIdleConns,
		MaxLifetime:  DefaultMaxLifetime,
		MaxIdleTime:  DefaultMaxIdleTime,
	}
}

// NewDB creates a new database connection pool using pgx
func NewDB(config Config, log *zap.Logger) (*DB, error) {
	// Build PostgreSQL connection string for pgx
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.DBName, config.SSLMode)

	// Set default values if not specified
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = DefaultMaxOpenConns
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = DefaultMaxIdleConns
	}
	if config.MaxLifetime == 0 {
		config.MaxLifetime = DefaultMaxLifetime
	}
	if config.MaxIdleTime == 0 {
		config.MaxIdleTime = DefaultMaxIdleTime
	}

	// Configure pgxpool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error("Failed to parse database config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Set connection pool parameters
	poolConfig.MaxConns = int32(config.MaxOpenConns)
	poolConfig.MinConns = int32(config.MaxIdleConns)
	poolConfig.MaxConnLifetime = config.MaxLifetime
	poolConfig.MaxConnIdleTime = config.MaxIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Error("Failed to create connection pool", zap.Error(err))
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Error("Failed to ping database", zap.Error(err))
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection pool established",
		zap.Int("maxConns", config.MaxOpenConns),
		zap.Int("minConns", config.MaxIdleConns),
		zap.Duration("maxLifetime", config.MaxLifetime),
		zap.Duration("maxIdleTime", config.MaxIdleTime))

	return &DB{pool: pool, log: log}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}
