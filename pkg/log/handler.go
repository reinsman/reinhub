package log

import (
	"context"
	"log/slog"
)

// ContextHandler 装饰另一个 slog.Handler，将 context.Context 携带的属性加入每条日志。
type ContextHandler struct {
	next slog.Handler
}

func NewContextHandler(next slog.Handler) *ContextHandler {
	if next == nil {
		panic("nil slog handler")
	}
	return &ContextHandler{next: next}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}

	attrs := ContextAttrs(ctx)
	if len(attrs) == 0 {
		return h.next.Handle(ctx, record)
	}

	recordAttrs := make([]slog.Attr, 0, record.NumAttrs())
	recordKeys := make(map[string]struct{}, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		recordAttrs = append(recordAttrs, attr)
		if attr.Key != "" {
			recordKeys[attr.Key] = struct{}{}
		}
		return true
	})
	contextAttrs := attrs[:0]
	for _, attr := range attrs {
		if _, overridden := recordKeys[attr.Key]; !overridden || attr.Key == "" {
			contextAttrs = append(contextAttrs, attr)
		}
	}
	if len(contextAttrs) == 0 {
		return h.next.Handle(ctx, record)
	}

	enriched := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	enriched.AddAttrs(contextAttrs...)
	enriched.AddAttrs(recordAttrs...)
	return h.next.Handle(ctx, enriched)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{next: h.next.WithGroup(name)}
}
