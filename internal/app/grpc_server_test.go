package app

import (
	"context"
	"net"
	"testing"
	"time"

	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"github.com/reinsman/reinhub/internal/config"
	"github.com/reinsman/reinhub/internal/domain/hub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestHubFrameworkDoesNotGrantAuthenticationOrExecute(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer(grpc.UnaryInterceptor(authorizeUnary), grpc.StreamInterceptor(authorizeStream))
	hub.New(nil).Register(server)
	t.Cleanup(server.Stop)
	go func() { _ = server.Serve(listener) }()

	// insecure 仅用于内存 transport；实际 app 启动必须加载 TLS 证书。
	connection, err := grpc.NewClient("passthrough:///hub-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = hubv1.NewAuthServiceClient(connection).StartLogin(ctx, &hubv1.StartLoginRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("login = %v", err)
	}
	_, err = hubv1.NewRunServiceClient(connection).ClaimRuns(ctx, &hubv1.ClaimRunsRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("claim = %v", err)
	}
	stream, err := hubv1.NewDaemonServiceClient(connection).Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = stream.Recv()
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("stream = %v", err)
	}
	if len(server.GetServiceInfo()) != 4 {
		t.Fatal("expected all four Hub services")
	}
}

func TestHubRequiresTLSWhenEnabled(t *testing.T) {
	server, err := newGRPCServer(config.GRPCConfig{Listen: "127.0.0.1:0"}, nil)
	if err == nil || server != nil {
		t.Fatal("Hub must reject missing TLS credentials")
	}
}

func TestRunGRPCServerStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runGRPCServer(ctx, grpc.NewServer(), "127.0.0.1:0") }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("gRPC server did not stop")
	}
}
