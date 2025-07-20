package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// PrettyHandler implements slog.Handler for human-friendly terminal output
type PrettyHandler struct {
	// Options for the handler
	opts slog.HandlerOptions

	// Writer where logs will be written
	writer io.Writer

	// Attributes to include in every log record
	attrs []slog.Attr

	// Groups for log records
	groups []string

	// Colors for different log elements
	colors *prettyColors
}

// prettyColors holds color functions for different elements
type prettyColors struct {
	timestamp *color.Color
	debug     *color.Color
	info      *color.Color
	warn      *color.Color
	error     *color.Color
	source    *color.Color
	message   *color.Color
	attrKey   *color.Color
	attrValue *color.Color
	bracket   *color.Color
}

// NewPrettyHandler creates a new pretty handler
func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	colors := &prettyColors{
		timestamp: color.New(color.FgHiBlack),
		debug:     color.New(color.FgMagenta, color.Bold),
		info:      color.New(color.FgGreen, color.Bold),
		warn:      color.New(color.FgYellow, color.Bold),
		error:     color.New(color.FgRed, color.Bold),
		source:    color.New(color.FgCyan),
		message:   color.New(color.Bold),
		attrKey:   color.New(color.FgCyan),
		attrValue: color.New(color.FgHiBlack),
		bracket:   color.New(color.FgHiBlack),
	}

	// Configure color output based on the writer
	if f, ok := w.(*os.File); ok {
		color.Output = f
	} else {
		// Disable colors for non-file writers
		color.NoColor = true
	}

	return &PrettyHandler{
		opts:   *opts,
		writer: w,
		colors: colors,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle formats and outputs a log record
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var buf strings.Builder

	// Timestamp
	_, _ = h.colors.timestamp.Fprint(&buf, r.Time.Format("15:04:05.000"))
	buf.WriteString(" ")

	// Level with color and padding
	levelColor := h.getLevelColor(r.Level)
	levelStr := h.formatLevelString(r.Level)
	_, _ = levelColor.Fprint(&buf, levelStr)
	buf.WriteString(" ")

	// Source information (if enabled)
	if h.opts.AddSource && r.PC != 0 {
		frame := getFrame(r.PC)
		_, _ = h.colors.source.Fprintf(&buf, "[%s:%d]", frame.File, frame.Line)
		buf.WriteString(" ")
	}

	// Message
	_, _ = h.colors.message.Fprint(&buf, r.Message)

	// Attributes
	if r.NumAttrs() > 0 || len(h.attrs) > 0 {
		buf.WriteString(" ")
		_, _ = h.colors.bracket.Fprint(&buf, "{\n  ")

		first := true

		// Handler-level attributes
		for _, attr := range h.attrs {
			if !first {
				buf.WriteString(", ")
			}
			h.formatAttr(&buf, attr)
			first = false
		}

		// Record attributes
		r.Attrs(func(attr slog.Attr) bool {
			if !first {
				buf.WriteString(", ")
			}
			h.formatAttr(&buf, attr)
			first = false
			return true
		})

		_, _ = h.colors.bracket.Fprint(&buf, "\n}")
	}

	buf.WriteString("\n")

	_, err := h.writer.Write([]byte(buf.String()))
	return err
}

// WithAttrs returns a new handler with the given attributes
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	return &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  newAttrs,
		groups: h.groups,
		colors: h.colors,
	}
}

// WithGroup returns a new handler with the given group
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, 0, len(h.groups)+1)
	newGroups = append(newGroups, h.groups...)
	newGroups = append(newGroups, name)

	return &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  h.attrs,
		groups: newGroups,
		colors: h.colors,
	}
}

// getLevelColor returns the appropriate color for a log level
func (h *PrettyHandler) getLevelColor(level slog.Level) *color.Color {
	switch level {
	case slog.LevelDebug:
		return h.colors.debug
	case slog.LevelInfo:
		return h.colors.info
	case slog.LevelWarn:
		return h.colors.warn
	case slog.LevelError:
		return h.colors.error
	default:
		return h.colors.info
	}
}

// formatLevelString returns a formatted level string with proper padding
func (h *PrettyHandler) formatLevelString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO "
	case slog.LevelWarn:
		return "WARN "
	case slog.LevelError:
		return "ERROR"
	default:
		return strings.ToUpper(level.String())
	}
}

// formatAttr formats a single attribute with colors
func (h *PrettyHandler) formatAttr(buf *strings.Builder, attr slog.Attr) {
	_, _ = h.colors.attrKey.Fprint(buf, attr.Key)
	_, _ = h.colors.attrValue.Fprint(buf, "=")

	// Format value based on type
	switch attr.Value.Kind() {
	case slog.KindString:
		_, _ = h.colors.attrValue.Fprint(buf, strconv.Quote(attr.Value.String()))
	case slog.KindInt64:
		_, _ = h.colors.attrValue.Fprint(buf, strconv.FormatInt(attr.Value.Int64(), 10))
	case slog.KindFloat64:
		_, _ = h.colors.attrValue.Fprint(buf, strconv.FormatFloat(attr.Value.Float64(), 'f', -1, 64))
	case slog.KindBool:
		_, _ = h.colors.attrValue.Fprint(buf, strconv.FormatBool(attr.Value.Bool()))
	case slog.KindDuration:
		_, _ = h.colors.attrValue.Fprint(buf, attr.Value.Duration().String())
	case slog.KindTime:
		_, _ = h.colors.attrValue.Fprint(buf, attr.Value.Time().Format(time.RFC3339))
	default:
		_, _ = h.colors.attrValue.Fprint(buf, strconv.Quote(attr.Value.String()))
	}
}

// getFrame extracts frame information from PC with proper skip calculation
func getFrame(pc uintptr) runtime.Frame {
	// Get the full stack trace
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()

	// If we got a frame from the logger package, try to find the real caller
	if isLoggerFrame(frame) || isSlogFrame(frame) {
		// Try to get the next frame in the chain
		var pcs [10]uintptr
		n := runtime.Callers(2, pcs[:]) // Skip more frames

		if n > 0 {
			frames = runtime.CallersFrames(pcs[:n])
			for {
				var more bool
				frame, more = frames.Next()
				if !more {
					break
				}
				// Skip the frames from the logger package and slog package
				if !isLoggerFrame(frame) && !isSlogFrame(frame) {
					break
				}
			}
		}
	}

	// Simplify the file path - show only the filename
	if idx := strings.LastIndex(frame.File, "/"); idx >= 0 {
		frame.File = frame.File[idx+1:]
	}

	return frame
}

// isLoggerFrame checks if the frame belongs to our logger package
func isLoggerFrame(frame runtime.Frame) bool {
	return strings.Contains(frame.Function, "github.com/dr8co/doppel/internal/logger.") ||
		strings.Contains(frame.File, "/internal/logger/logger")
}

// isSlogFrame checks if the frame belongs to the slog package
func isSlogFrame(frame runtime.Frame) bool {
	return strings.Contains(frame.Function, "log/slog.") ||
		strings.Contains(frame.File, "log/slog")
}
