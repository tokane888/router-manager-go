package main

import (
	"errors"
	"log"

	// TODO: import元調整
	pkglogger "github.com/tokane888/go-repository-template/pkg/logger"
	"github.com/tokane888/go-repository-template/services/batch/internal/config"
	"go.uber.org/zap"
)

// アプリのversion。デフォルトは開発版。cloud上ではbuild時に-ldflagsフラグ経由でバージョンを埋め込む
var version = "dev"

func main() {
	cfg, err := config.LoadConfig(version)
	if err != nil {
		log.Fatal("failed to load config: ", err)
	}
	logger := pkglogger.NewLogger(cfg.Logger)
	//nolint: errcheck
	defer logger.Sync()

	logger.Info("sample batch info")
	logger.Info("additional field sample", zap.String("key", "value"))
	logger.Warn("sample warn")
	logger.Error("sample error")
	err = errors.New("errorのサンプル")
	logger.Error("DB Connection failed", zap.Error(err))
}
