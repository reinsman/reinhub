package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/reinsman/reinhub/internal/config"
	"google.golang.org/grpc"
)

// Application 持有应用运行所需的进程级资源。
type Application struct {
	closeStorage func() error
	server       *http.Server
	grpcServer   *grpc.Server
	grpcAddress  string
}

// New 组装当前 Hub 的文件存储、HTTP API 和 gRPC/TLS 服务。
func New(cfg *config.Config) (*Application, error) {
	return newHubApplication(cfg)
}

// Run 启动应用，并在退出时关闭进程级资源。
func (a *Application) Run(ctx context.Context) error {
	defer a.closeStorage()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errorsCh := make(chan error, 2)
	go func() { errorsCh <- runServer(ctx, a.server) }()
	go func() { errorsCh <- runGRPCServer(ctx, a.grpcServer, a.grpcAddress) }()
	first := <-errorsCh
	cancel()
	return errors.Join(first, <-errorsCh)
}
