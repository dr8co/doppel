package logger

import (
	"context"
	"io"
	"log/slog"
)

// JsonHandler is a [slog.JSONHandler] with corrected source position of the caller.
type JsonHandler struct {
	h *slog.JSONHandler
}

// NewJsonHandler creates a [JsonHandler] that writes to w, using the given options.
// If opts is nil, the default options are used.
func NewJsonHandler(w io.Writer, opts *slog.HandlerOptions) *JsonHandler {
	return &JsonHandler{
		h: slog.NewJSONHandler(w, opts),
	}
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *JsonHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// Handle formats its [slog.Record] argument as a JSON object on a single line.
func (h *JsonHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.PC != 0 {
		frame := getFrame(r.PC)
		r.PC = frame.PC
	}

	return h.h.Handle(ctx, r)
}

// WithAttrs returns a new [JsonHandler] whose attributes consists of h's attributes followed by attrs.
func (h *JsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.h.WithAttrs(attrs)
}

// WithGroup returns a new [JsonHandler] with the given group appended to h's groups.
func (h *JsonHandler) WithGroup(name string) slog.Handler {
	return h.h.WithGroup(name)
}
