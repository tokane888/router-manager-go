package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	pkglogger "github.com/tokane888/router-manager-go/pkg/logger"
	"github.com/tokane888/router-manager-go/services/batch/internal/config"
)

// アプリのversion。デフォルトは開発版。cloud上ではbuild時に-ldflagsフラグ経由でバージョンを埋め込む
var version = "dev"

func main() {
	cfg, err := config.LoadConfig(version)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger := pkglogger.NewLogger(cfg.Logger)
	//nolint: errcheck
	defer logger.Sync()

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("Domain IP Blocker batch service starting")

	// TODO: Initialize dependencies and use case
	// This will be implemented in subsequent tasks

	// Use context to prevent unused variable error
	select {
	case <-ctx.Done():
		logger.Info("Service cancelled")
	default:
		logger.Info("Domain IP Blocker batch service completed")
	}
}
