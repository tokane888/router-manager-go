package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerConfig struct {
	Level      string // debug, info, warn, error
	Format     string // local(見やすさ重視), cloud(CloudWatch等で解析可能であることを重視)
	Env        string // 環境名(cloudでのみログ出力)
	AppName    string // アプリ名(cloudでのみログ出力)
	AppVersion string // アプリのバージョン(cloudでのみログ出力)
}

func NewLogger(cfg LoggerConfig) *zap.Logger {
	var zapCfg zap.Config
	switch cfg.Format {
	case "local":
		// local環境では読みやすさ重視
		// (非構造化ログ、JST固定、ミリ秒精度)
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			jst := t.In(time.FixedZone("Asia/Tokyo", 9*60*60))
			enc.AppendString(jst.Format("2006-01-02T15:04:05.000Z07:00"))
		}
	case "cloud":
		// cloud環境ではcloud watch等で読まれる前提で解析重視
		// (構造化ログ、UTC、ナノ秒精度)
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)
	default:
		// LOG_FORMATが不正の場合、cloud向けフォーマットで出力
		fmt.Fprintf(os.Stderr, "invalid LOG_FORMAT %q, fallback to 'cloud'\n", cfg.Format)
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)
	}

	// ログレベル設定
	parsedLevel := zapcore.InfoLevel
	if err := parsedLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid LOG_LEVEL %q, fallback to 'info'\n", cfg.Level)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(parsedLevel)

	// error時のみStackTrace出力するよう設定
	zapCfg.DisableStacktrace = true
	logger, _ := zapCfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// cloud向けはフィールド追加
	if cfg.Format == "cloud" {
		logger = logger.With(
			zap.String("app", cfg.AppName),
			zap.String("env", cfg.Env),
			zap.String("version", cfg.AppVersion),
		)
	}

	return logger
}
