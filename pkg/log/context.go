// Package log 提供请求级结构化日志辅助能力。
package log

import (
	"context"
	"log/slog"
)

type contextAttrsKey struct{}

// WithAttrs 返回带有日志属性的子 Context；使用 slog.InfoContext、
// slog.ErrorContext 等 API 记录日志时会自动附加这些属性。
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	merged := mergeAttrs(ContextAttrs(ctx), attrs)
	return context.WithValue(ctx, contextAttrsKey{}, merged)
}

// With 返回附加 slog 键值对的子 Context。
func With(ctx context.Context, args ...any) context.Context {
	if len(args)%2 != 0 {
		panic("log fields must be key-value pairs")
	}

	attrs := make([]slog.Attr, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			panic("log field key must be a string")
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return WithAttrs(ctx, attrs...)
}

// ContextAttrs 返回 Context 中结构化日志属性的副本。
func ContextAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, _ := ctx.Value(contextAttrsKey{}).([]slog.Attr)
	return append([]slog.Attr(nil), attrs...)
}

func mergeAttrs(base, overrides []slog.Attr) []slog.Attr {
	merged := append([]slog.Attr(nil), base...)
	indexes := make(map[string]int, len(merged))
	for i, attr := range merged {
		if attr.Key != "" {
			indexes[attr.Key] = i
		}
	}
	for _, attr := range overrides {
		if index, ok := indexes[attr.Key]; ok && attr.Key != "" {
			merged[index] = attr
			continue
		}
		if attr.Key != "" {
			indexes[attr.Key] = len(merged)
		}
		merged = append(merged, attr)
	}
	return merged
}
