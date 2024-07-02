package server

import (
	"context"
	"log/slog"
)

type ctxSlogKey string

const slogFields ctxSlogKey = "slog_fields"

type contextHandler struct {
	slog.Handler
}

// Extract log fields in context and add them to record.
func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

// AppendCtx adds log attr to context.
func AppendCtx(parent context.Context, attr slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	if attrs, ok := parent.Value(slogFields).([]slog.Attr); ok {
		attrs = append(attrs, attr)
		return context.WithValue(parent, slogFields, attrs)
	}

	s := []slog.Attr{}
	s = append(s, attr)
	return context.WithValue(parent, slogFields, s)
}

func NewCtxLogger(h slog.Handler) *slog.Logger {
	handler := &contextHandler{h}
	return slog.New(handler)
}
