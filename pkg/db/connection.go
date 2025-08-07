package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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
	conn *sql.DB
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

// NewDB creates a new database connection
func NewDB(config Config, log *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("Failed to open database connection", zap.Error(err))
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
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

	conn.SetMaxOpenConns(config.MaxOpenConns)
	conn.SetMaxIdleConns(config.MaxIdleConns)
	conn.SetConnMaxLifetime(config.MaxLifetime)
	conn.SetConnMaxIdleTime(config.MaxIdleTime)

	log.Info("Connection pool configured",
		zap.Int("maxOpenConns", config.MaxOpenConns),
		zap.Int("maxIdleConns", config.MaxIdleConns),
		zap.Duration("maxLifetime", config.MaxLifetime),
		zap.Duration("maxIdleTime", config.MaxIdleTime))

	if err := conn.PingContext(context.Background()); err != nil {
		log.Error("Failed to ping database", zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established")
	return &DB{conn: conn, log: log}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}
