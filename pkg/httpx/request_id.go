package httpx

import (
	"context"
	"os"
	"strings"

	"github.com/reinsman/reinhub/pkg/strutil"
)

const (
	RequestIDHeaderEnv     = "REQUEST-ID-HEADER"
	DefaultRequestIDHeader = "x-request-id"
)

type requestIDContextKey struct{}

// RequestIDHeader 返回用于传递请求 ID 的 HTTP 请求头名称。
func RequestIDHeader() string {
	header := strings.TrimSpace(os.Getenv(RequestIDHeaderEnv))
	if header == "" {
		return DefaultRequestIDHeader
	}
	return header
}

// WithRequestID 将请求 ID 保存到 Context，供下游调用继续传递。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestID 返回 Context 中保存的请求 ID。
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func GenerateRequestID() string {
	return strutil.Uuid()
}
