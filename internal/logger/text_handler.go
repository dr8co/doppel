package logger

import (
	"context"
	"io"
	"log/slog"
)

// TextHandler is a [slog.TextHandler] with corrected source position of the caller.
type TextHandler struct {
	h *slog.TextHandler
}

// NewTextHandler creates a [TextHandler] that writes to w, using the given options.
// If opts is nil, the default options are used.
func NewTextHandler(w io.Writer, opts *slog.HandlerOptions) *TextHandler {
	return &TextHandler{
		h: slog.NewTextHandler(w, opts),
	}
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *TextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// Handle formats its [slog.Record] argument as a single line of space-separated key=value items.
func (h *TextHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.PC != 0 {
		frame := getFrame(r.PC)
		r.PC = frame.PC
	}

	return h.h.Handle(ctx, r)
}

// WithAttrs returns a new [TextHandler] whose attributes consists of h's attributes followed by attrs.
func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.h.WithAttrs(attrs)
}

// WithGroup returns a new [TextHandler] with the given group appended to h's groups.
func (h *TextHandler) WithGroup(name string) slog.Handler {
	return h.h.WithGroup(name)
}
