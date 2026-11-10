package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/reinsman/reinhub/internal/app"
	"github.com/reinsman/reinhub/internal/config"
	applog "github.com/reinsman/reinhub/pkg/log"

	"log/slog"
)

var (
	appConfPath string // 配置文件路径
)

func init() {
	initRunTime()
	initFlag()
}

func initRunTime() {
}

func initFlag() {
	flag.StringVar(&appConfPath, "conf", "./conf.toml", "app config's path")
}

func loadConfig() *config.Config {
	flag.Parse()

	cfg, err := config.Load(appConfPath)
	if err != nil {
		panic(fmt.Errorf("load config %q: %w", appConfPath, err))
	}
	return cfg
}

func initLog(cfg *config.Config) {
	lv := slog.LevelInfo
	if err := lv.UnmarshalText([]byte(cfg.LogLv)); err != nil {
		panic(fmt.Errorf("invalid log level %q: %w", cfg.LogLv, err))
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.MessageKey {
				attr.Key = "message"
			}
			if attr.Key == slog.TimeKey {
				return slog.String(
					slog.TimeKey,
					attr.Value.Time().Format("2006-01-02T15:04:05.000Z07:00"),
				)
			}

			return attr
		},
		Level: lv,
	})

	slog.SetDefault(slog.New(handler))

	slog.SetDefault(slog.New(applog.NewContextHandler(handler)))
}

// @title           go-examplesrv
// @version         0.1.0
// @description     Go 微服务脚手架示例服务。
// @termsOfService

// @contact.name   xhb
// @contact.url    https://chatgpt.com/
// @contact.email  support@examplesrv.com

// @license.name  Private
// @license.url

// @host      localhost:9010
// @BasePath  /

// @securityDefinitions.apikey  ApiKeyAuth
// @in							header
// @name						Authorization
func main() {
	cfg := loadConfig()
	initLog(cfg)

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("initialize application", "err", err)
		panic(err)
	}
	slog.Info("Hub started", "http", cfg.Listen, "grpc", cfg.GRPC.Listen)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		slog.Error("application stopped", "err", err)
		panic(err)
	}
}
