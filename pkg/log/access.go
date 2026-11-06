package log

import (
	"context"
	"log/slog"
	"time"

	"github.com/reinsman/reinhub/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type requestLogKey struct{}

// RequestLog 保存生成最终访问日志所需的请求级状态。
// 所有更新必须在 Handler 链返回前完成。
type RequestLog struct {
	Begin time.Time

	code        int
	message     string
	err         string
	hasResponse bool
	fields      map[string]any
}

// InitAccessLog 在 Context 中安装新的访问日志收集器。
func InitAccessLog(ctx context.Context, begin time.Time) (context.Context, *RequestLog) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestLog := &RequestLog{
		Begin:  begin,
		fields: make(map[string]any),
	}
	return context.WithValue(ctx, requestLogKey{}, requestLog), requestLog
}

// GetAccessLog 返回 Context 中的访问日志收集器；未安装时返回 false。
func GetAccessLog(ctx context.Context) (*RequestLog, bool) {
	if ctx == nil {
		return nil, false
	}
	requestLog, ok := ctx.Value(requestLogKey{}).(*RequestLog)
	return requestLog, ok
}

// AddAccessAttrs 向最终访问日志添加字段；同名字段以后写入的值为准。
func AddAccessAttrs(ctx context.Context, attrs ...slog.Attr) {
	requestLog, ok := GetAccessLog(ctx)
	if !ok {
		return
	}

	for _, attr := range attrs {
		requestLog.fields[attr.Key] = attr.Value.Any()
	}
}

// AddAccessFields 是 AddAccessAttrs 的键值对调用形式。
func AddAccessFields(ctx context.Context, fields ...any) {
	if len(fields)%2 != 0 {
		panic("access log fields must be key-value pairs")
	}
	for i := 0; i < len(fields); i += 2 {
		if _, ok := fields[i].(string); !ok {
			panic("access log field key must be a string")
		}
	}

	requestLog, ok := GetAccessLog(ctx)
	if !ok {
		return
	}

	for i := 0; i < len(fields); i += 2 {
		key := fields[i].(string)
		requestLog.fields[key] = fields[i+1]
	}
}

// SetAccessResponse 记录需要写入访问日志的应用响应信息。
func SetAccessResponse(ctx context.Context, code int, message, err string) {
	requestLog, ok := GetAccessLog(ctx)
	if !ok {
		return
	}
	requestLog.code = code
	requestLog.message = message
	requestLog.err = err
	requestLog.hasResponse = true
}

// AccessLog 在后续 Gin Handler 全部执行完成后记录一次请求访问日志。
// 同时通过 httpx 和 slog 的 Context 字段继续传递请求 ID。
func AccessLog() gin.HandlerFunc {
	requestIDHeader := httpx.RequestIDHeader()
	return func(c *gin.Context) {
		begin := time.Now()
		ctx, requestLog := InitAccessLog(c.Request.Context(), begin)

		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = httpx.GenerateRequestID()
		}
		ctx = httpx.WithRequestID(ctx, requestID)
		ctx = WithAttrs(ctx, slog.String("request_id", requestID))
		c.Request = c.Request.WithContext(ctx)
		path := c.Request.URL.Path

		c.Next()

		level := slog.LevelInfo
		if c.Writer.Status() >= 400 || requestLog.hasResponse && requestLog.code != 0 {
			level = slog.LevelError
		} else if c.Writer.Status() >= 300 {
			level = slog.LevelWarn
		}

		requestGroup := slog.Group(
			"request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("body_size", c.Writer.Size()),
		)
		responseAttrs := []slog.Attr{slog.Int("status", c.Writer.Status())}
		if requestLog.hasResponse {
			responseAttrs = append(responseAttrs,
				slog.Int("code", requestLog.code),
				slog.String("msg", requestLog.message),
				slog.String("error", requestLog.err),
			)
		}

		attrs := make([]slog.Attr, 0, 4+len(requestLog.fields))
		attrs = append(attrs, slog.Int("cost", int(time.Since(begin).Milliseconds())))
		for key, value := range requestLog.fields {
			attrs = append(attrs, slog.Any(key, value))
		}
		attrs = append(attrs, requestGroup, slog.GroupAttrs("response", responseAttrs...))

		slog.LogAttrs(c.Request.Context(), level, "handle request", attrs...)
	}
}
