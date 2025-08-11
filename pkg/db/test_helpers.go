package db

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// TestDBContainer wraps PostgreSQL container for testing
type TestDBContainer struct {
	container *postgres.PostgresContainer
	DB        *DB
}

// SetupTestDB creates a PostgreSQL container and initializes the database
func SetupTestDB(t *testing.T) *TestDBContainer {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// ログ出力しないlogger。test用
	logger := zap.NewNop()

	// Create connection pool using pgx
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Create DB wrapper
	db := &DB{pool: pool, log: logger}

	// Initialize schema
	if err := initializeSchema(db.pool); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return &TestDBContainer{
		container: pgContainer,
		DB:        db,
	}
}

// Cleanup terminates the container and closes the DB connection
func (tdb *TestDBContainer) Cleanup(t *testing.T) {
	t.Helper()

	if tdb.DB != nil {
		tdb.DB.Close()
	}

	if tdb.container != nil {
		ctx := context.Background()
		if err := tdb.container.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate container: %v", err)
		}
	}
}

// initializeSchema reads and executes the schema SQL file
func initializeSchema(pool *pgxpool.Pool) error {
	// Get the path to the schema file
	schemaPath := filepath.Join("..", "..", "db", "schema", "init.sql")

	// Read the schema file
	schemaFile, err := os.Open(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to open schema file %s: %w", schemaPath, err)
	}
	defer func() {
		if closeErr := schemaFile.Close(); closeErr != nil {
			// In a real implementation, you might want to log this error
			_ = closeErr
		}
	}()

	schemaSQL, err := io.ReadAll(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute the schema
	if _, err := pool.Exec(context.Background(), string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// ClearTables clears all data from test tables
func (tdb *TestDBContainer) ClearTables(t *testing.T) {
	t.Helper()

	// Clear domain_ips first due to foreign key constraint
	if _, err := tdb.DB.pool.Exec(context.Background(), "DELETE FROM domain_ips"); err != nil {
		t.Fatalf("Failed to clear domain_ips table: %v", err)
	}

	// Clear domains
	if _, err := tdb.DB.pool.Exec(context.Background(), "DELETE FROM domains"); err != nil {
		t.Fatalf("Failed to clear domains table: %v", err)
	}
}
