package logger

import (
	"context"
	"io"
	"log/slog"
	"runtime"
)

// JsonHandler is a [slog.JSONHandler] with the corrected source position of the caller.
type JsonHandler struct {
	h    *slog.JSONHandler
	opts slog.HandlerOptions
}

// NewJsonHandler creates a [JsonHandler] that writes to w, using the given options.
// If opts is nil, the default options are used.
func NewJsonHandler(w io.Writer, opts *slog.HandlerOptions) *JsonHandler {
	return &JsonHandler{
		h:    slog.NewJSONHandler(w, opts),
		opts: *opts,
	}
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *JsonHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// Handle formats its [slog.Record] argument as a JSON object on a single line.
func (h *JsonHandler) Handle(ctx context.Context, r slog.Record) error {
	// Skip 5 frames, as long as this method is not called directly by any package other than slog.
	// 0: Callers
	// 1: Handle
	// 2 & 3: slog.LogAttrs
	// 4: logger package methods

	if h.opts.AddSource && r.PC != 0 {
		var pcs [1]uintptr
		if runtime.Callers(5, pcs[:]) == 1 {
			r.PC = pcs[0]
		}
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
