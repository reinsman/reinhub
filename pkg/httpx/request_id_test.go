package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDHeaderDefault(t *testing.T) {
	t.Setenv(RequestIDHeaderEnv, "")
	if header := RequestIDHeader(); header != DefaultRequestIDHeader {
		t.Fatalf("RequestIDHeader() = %q, want %q", header, DefaultRequestIDHeader)
	}
}

func TestHttpRequestPropagatesRequestID(t *testing.T) {
	t.Setenv(RequestIDHeaderEnv, "x-trace-id")
	var receivedRequestID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		receivedRequestID = request.Header.Get("x-trace-id")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx := WithRequestID(context.Background(), "request-123")
	if err := Get(ctx, server.URL, nil); err != nil {
		t.Fatal(err)
	}
	if receivedRequestID != "request-123" {
		t.Fatalf("downstream request ID = %q, want %q", receivedRequestID, "request-123")
	}
}
