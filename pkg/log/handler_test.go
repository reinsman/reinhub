package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestContextHandlerAddsContextAttrs(t *testing.T) {
	var output bytes.Buffer
	handler := NewContextHandler(slog.NewJSONHandler(&output, nil))
	logger := slog.New(handler).With("component", "user")
	ctx := With(context.Background(), "request_id", "request-1", "uid", 42)

	logger.InfoContext(ctx, "query user", "operation", "get")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["request_id"] != "request-1" {
		t.Fatalf("request_id = %v, want request-1", entry["request_id"])
	}
	if entry["uid"] != float64(42) {
		t.Fatalf("uid = %v, want 42", entry["uid"])
	}
	if entry["component"] != "user" || entry["operation"] != "get" {
		t.Fatalf("logger and record attrs were not preserved: %v", entry)
	}
}

func TestContextHandlerExplicitAttrTakesPrecedence(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&output, nil)))
	ctx := With(context.Background(), "request_id", "from-context")

	logger.InfoContext(ctx, "override", "request_id", "from-record")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["request_id"] != "from-record" {
		t.Fatalf("request_id = %v, want from-record", entry["request_id"])
	}
}
