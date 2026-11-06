package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"github.com/reinsman/reinhub/internal/config"
	"github.com/reinsman/reinhub/internal/domain/hub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func newGRPCServer(cfg config.GRPCConfig, db *gorm.DB) (*grpc.Server, error) {
	if cfg.Listen == "" {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load Hub TLS certificate: %w", err)
	}
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12,
		})),
		grpc.UnaryInterceptor(authorizeUnary),
		grpc.StreamInterceptor(authorizeStream),
	)
	hub.New(db).Register(server)
	return server, nil
}

func authorizeUnary(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	switch info.FullMethod {
	case hubv1.AuthService_StartLogin_FullMethodName, hubv1.AuthService_GetLoginStatus_FullMethodName:
		return handler(ctx, request)
	default:
		// TODO: 验证连接凭据、有效期、撤销和 daemon 身份，再交给领域做资源授权。
		return nil, status.Error(codes.Unauthenticated, "Hub authentication is not implemented")
	}
}

func authorizeStream(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	// TODO: 认证并绑定流身份；Hello.daemon_id 不是可信身份凭据。
	return status.Error(codes.Unauthenticated, "Hub stream authentication is not implemented")
}

func runGRPCServer(ctx context.Context, server *grpc.Server, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(ctx)
	stopped := make(chan struct{})
	defer func() { cancel(); <-stopped }()
	go func() {
		defer close(stopped)
		<-ctx.Done()
		graceful := make(chan struct{})
		go func() { server.GracefulStop(); close(graceful) }()
		select {
		case <-graceful:
		case <-time.After(10 * time.Second):
			server.Stop()
			<-graceful
		}
	}()
	err = server.Serve(listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
}
