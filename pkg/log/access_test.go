package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reinsman/reinhub/pkg/httpx"

	"github.com/gin-gonic/gin"
)

func TestAccessLogPropagatesAndCollectsFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(httpx.RequestIDHeaderEnv, httpx.DefaultRequestIDHeader)

	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(NewContextHandler(slog.NewJSONHandler(&output, nil))))
	t.Cleanup(func() { slog.SetDefault(previous) })

	engine := gin.New()
	engine.Use(AccessLog())
	engine.GET("/users/:id", func(c *gin.Context) {
		ctx := With(c.Request.Context(), "uid", 42)
		c.Request = c.Request.WithContext(ctx)
		slog.InfoContext(ctx, "inside service")
		AddAccessFields(ctx, "domain", "legacy")
		AddAccessFields(ctx, "domain", "user")
		SetAccessResponse(ctx, 0, "ok", "")
		c.String(http.StatusOK, "done")
	})

	request := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	request.Header.Set(httpx.DefaultRequestIDHeader, "request-1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	entries := decodeEntries(t, output.Bytes())
	if len(entries) != 2 {
		t.Fatalf("got %d log entries, want 2", len(entries))
	}

	serviceLog := entries[0]
	if serviceLog["request_id"] != "request-1" || serviceLog["uid"] != float64(42) {
		t.Fatalf("service context fields = %v", serviceLog)
	}
	if _, exists := serviceLog["trace_id"]; exists {
		t.Fatalf("service log unexpectedly contains trace_id: %v", serviceLog)
	}
	if _, exists := serviceLog["trace"]; exists {
		t.Fatalf("service log unexpectedly contains trace group: %v", serviceLog)
	}

	accessLog := entries[1]
	if accessLog["level"] != "INFO" || accessLog["domain"] != "user" {
		t.Fatalf("access log fields = %v", accessLog)
	}
	if accessLog["request_id"] != "request-1" {
		t.Fatalf("root request_id = %v, want request-1", accessLog["request_id"])
	}

	requestGroup := accessLog["request"].(map[string]any)
	if requestGroup["method"] != http.MethodGet ||
		requestGroup["path"] != "/users/42" ||
		requestGroup["body_size"] != float64(4) {
		t.Fatalf("request group = %v", requestGroup)
	}
	responseGroup := accessLog["response"].(map[string]any)
	if responseGroup["status"] != float64(http.StatusOK) ||
		responseGroup["code"] != float64(0) ||
		responseGroup["msg"] != "ok" {
		t.Fatalf("response group = %v", responseGroup)
	}
	if _, exists := accessLog["trace"]; exists {
		t.Fatal("access log must not include tracing fields")
	}
}

func TestAccessLogLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(httpx.RequestIDHeaderEnv, httpx.DefaultRequestIDHeader)

	tests := []struct {
		name       string
		httpStatus int
		code       int
		wantLevel  string
	}{
		{name: "redirect", httpStatus: http.StatusMultipleChoices, wantLevel: "WARN"},
		{name: "http error", httpStatus: http.StatusBadRequest, wantLevel: "ERROR"},
		{name: "business error", httpStatus: http.StatusOK, code: 1001, wantLevel: "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := slog.Default()
			slog.SetDefault(slog.New(NewContextHandler(slog.NewJSONHandler(&output, nil))))
			defer slog.SetDefault(previous)

			engine := gin.New()
			engine.Use(AccessLog())
			engine.GET("/", func(c *gin.Context) {
				if tt.code != 0 {
					SetAccessResponse(c.Request.Context(), tt.code, "failed", "detail")
				}
				c.Status(tt.httpStatus)
			})
			engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			entries := decodeEntries(t, output.Bytes())
			if len(entries) != 1 {
				t.Fatalf("got %d log entries, want 1", len(entries))
			}
			if entries[0]["level"] != tt.wantLevel {
				t.Fatalf("level = %v, want %s", entries[0]["level"], tt.wantLevel)
			}
		})
	}
}

func decodeEntries(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var entries []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		var entry map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("unmarshal log entry: %v", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan logs: %v", err)
	}
	return entries
}
