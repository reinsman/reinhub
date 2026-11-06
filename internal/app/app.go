package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/reinsman/reinhub/internal/config"
	"github.com/reinsman/reinhub/internal/dbs"
	"google.golang.org/grpc"
)

// Application 持有应用运行所需的进程级资源。
type Application struct {
	closeDB     func() error
	server      *http.Server
	grpcServer  *grpc.Server
	grpcAddress string
}

// New 初始化数据库和 HTTP Server。
func New(cfg *config.Config, debug bool) (*Application, error) {
	db, err := dbs.Init(cfg.Db, debug)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database connection: %w", err)
	}
	grpcServer, err := newGRPCServer(cfg.GRPC, db)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return &Application{
		closeDB:     sqlDB.Close,
		server:      newServer(cfg, db, debug),
		grpcServer:  grpcServer,
		grpcAddress: cfg.GRPC.Listen,
	}, nil
}

// Run 启动应用，并在退出时关闭进程级资源。
func (a *Application) Run(ctx context.Context) error {
	defer a.closeDB()
	if a.grpcServer == nil {
		return runServer(ctx, a.server)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errorsCh := make(chan error, 2)
	go func() { errorsCh <- runServer(ctx, a.server) }()
	go func() { errorsCh <- runGRPCServer(ctx, a.grpcServer, a.grpcAddress) }()
	first := <-errorsCh
	cancel()
	return errors.Join(first, <-errorsCh)
}
