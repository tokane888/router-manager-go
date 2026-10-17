package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/tokane888/router-manager-go/pkg/db"
	pkglogger "github.com/tokane888/router-manager-go/pkg/logger"
	"github.com/tokane888/router-manager-go/services/batch/internal/config"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/dns"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/firewall"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/system"
	"github.com/tokane888/router-manager-go/services/batch/internal/usecase"
)

// アプリのversion。デフォルトは開発版。cloud上ではbuild時に-ldflagsフラグ経由でバージョンを埋め込む
var version = "dev"

func main() {
	cfg, err := config.NewConfig(version)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger := pkglogger.NewLogger(cfg.Logger)

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("Domain IP Blocker batch service starting")

	// Initialize database connection
	database, err := db.NewDB(cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to initialize database connection",
			slog.Any("error", err))
		stop()
		os.Exit(1) //nolint:gocritic // stop()で明示的に後始末済み
	}
	defer database.Close()

	// Initialize DNS resolver
	dnsResolver := dns.NewDNSResolver(cfg.DNS, net.DefaultResolver, logger)

	// Initialize nftables manager
	nftablesManager := firewall.NewNFTablesManager(cfg.NFTables, logger)

	// Initialize reboot detector
	rebootDetector := system.NewRebootDetector(logger)

	// Initialize use case
	domainBlockerUseCase := usecase.NewDomainBlockerUseCase(
		database,
		dnsResolver,
		nftablesManager,
		rebootDetector,
		logger,
		cfg.Processing,
	)

	logger.Info("Starting domain processing")

	// Execute domain processing
	if err := domainBlockerUseCase.ProcessAllDomains(ctx); err != nil {
		logger.Error("Failed to process domains", slog.Any("error", err))
	}

	select {
	case <-ctx.Done():
		logger.Info("Service cancelled")
	default:
		logger.Info("Domain IP Blocker batch service completed")
	}
}
