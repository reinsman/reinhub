package app

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"github.com/reinsman/reinhub/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

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

func TestHubIsLoopbackOnlyAndUsesSeparateTLSLeaf(t *testing.T) {
	for _, address := range []string{":9010", "0.0.0.0:9010", "localhost:9010"} {
		cfg := &config.Config{Listen: address, GRPC: config.GRPCConfig{Listen: "127.0.0.1:9011"}, Directory: t.TempDir()}
		if _, err := New(cfg); err == nil {
			t.Fatalf("unsafe Hub bind accepted: %s", address)
		}
	}
	dir := t.TempDir()
	cfg := &config.Config{Listen: "127.0.0.1:0", GRPC: config.GRPCConfig{Listen: "127.0.0.1:0"}, Directory: dir}
	application, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer application.closeStorage()
	if _, err := New(cfg); err == nil {
		t.Fatal("two processes may own Hub state")
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	root := x509.NewCertPool()
	if !root.AppendCertsFromPEM(caPEM) {
		t.Fatal("invalid CA")
	}
	data, err := os.ReadFile(filepath.Join(dir, "server.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if leaf.IsCA {
		t.Fatal("server leaf must not be a CA")
	}
	if _, err = leaf.Verify(x509.VerifyOptions{DNSName: "localhost", Roots: root}); err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1 << 20)
	go func() { _ = application.grpcServer.Serve(listener) }()
	t.Cleanup(application.grpcServer.Stop)
	connection, err := grpc.NewClient("passthrough:///localhost",
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(root, "localhost")),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = hubv1.NewRuntimeServiceClient(connection).RegisterRuntime(ctx, &hubv1.RegisterRuntimeRequest{
		Runtime: &hubv1.Runtime{Id: "runtime", DaemonId: "daemon", WorkspaceId: "workspace", Backend: hubv1.BackendKind_BACKEND_KIND_NATIVE},
	})
	if err != nil {
		t.Fatalf("Hub registration without mode switches or login: %v", err)
	}
	_, err = hubv1.NewAuthServiceClient(connection).StartLogin(ctx, &hubv1.StartLoginRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("login must remain unimplemented: %v", err)
	}
}

func TestHubDataDirectoryUsesHome(t *testing.T) {
	for _, directory := range []string{"", "~/.reinhub"} {
		t.Run(directory, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			cfg := &config.Config{
				Listen: "127.0.0.1:0", GRPC: config.GRPCConfig{Listen: "127.0.0.1:0"},
				Directory: directory,
			}
			application, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer application.closeStorage()
			for _, name := range []string{"ca.pem", "server.pem", "server-key.pem", "hub.lock"} {
				if _, err := os.Stat(filepath.Join(home, ".reinhub", name)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
