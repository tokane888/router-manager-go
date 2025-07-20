package main

import (
	"fmt"
	"log"
	"net/http"

	// TODO: import元調整
	"github.com/gin-gonic/gin"
	pkglogger "github.com/tokane888/go-repository-template/pkg/logger"
	"github.com/tokane888/go-repository-template/services/api/internal/config"
	"go.uber.org/zap"
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

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	err = r.Run(fmt.Sprintf(":%d", cfg.RouterConfig.Port))
	if err != nil {
		logger.Error("failed to start API server", zap.Error(err))
		return
	}
}
