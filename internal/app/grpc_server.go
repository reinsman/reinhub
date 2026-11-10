package app

import (
	"context"
	"errors"
	"net"
	"time"

	"google.golang.org/grpc"
)

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
