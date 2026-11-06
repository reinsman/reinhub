package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/reinsman/reinhub/internal/config"
	"gorm.io/gorm"
)

// newServer 组装路由并创建带生产环境基础超时设置的 HTTP 服务器。
func newServer(cfg *config.Config, db *gorm.DB, debug bool) *http.Server {
	return &http.Server{
		Addr:              cfg.Listen,
		Handler:           newRouter(db, debug, cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// runServer 启动服务器，并在 context 取消后执行优雅退出。
func runServer(ctx context.Context, server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	err := <-errCh
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
