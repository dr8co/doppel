package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
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

	mu *sync.Mutex

	builderPool sync.Pool
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

	writer := w
	// Configure color output based on the writer
	if f, ok := w.(*os.File); ok {
		if f == os.Stderr {
			writer = colorable.NewColorableStderr()
		} else if f != os.Stdout {
			writer = colorable.NewColorable(f)
		}
		color.Output = writer
	} else {
		// Disable colors for non-file writers
		color.NoColor = true
	}

	return &PrettyHandler{
		opts:   *opts,
		writer: writer,
		colors: colors,
		mu:     &sync.Mutex{},
		builderPool: sync.Pool{
			New: func() interface{} {
				builder := &strings.Builder{}
				// Pre-allocate 1024 bytes
				builder.Grow(1024)
				return builder
			},
		},
	}
}

// clone creates a copy of the handler with the same options and writer
func (h *PrettyHandler) clone() *PrettyHandler {
	return &PrettyHandler{
		opts:        h.opts,
		writer:      h.writer,
		attrs:       slices.Clip(h.attrs),
		groups:      slices.Clip(h.groups),
		colors:      h.colors,
		mu:          h.mu,
		builderPool: h.builderPool,
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
	buf := h.builderPool.Get().(*strings.Builder)
	defer func() {
		buf.Reset()
		h.builderPool.Put(buf)
	}()

	h.formatRecord(buf, r)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.writer.Write([]byte(buf.String()))
	return err
}

// formatRecord formats a log record into the provided buffer
func (h *PrettyHandler) formatRecord(buf *strings.Builder, r slog.Record) {
	// Timestamp
	buf.WriteString(h.colors.timestamp.Sprint(r.Time.Format("15:04:05.000")))
	buf.WriteString(" ")

	// Level with color and padding
	levelColor := h.getLevelColor(r.Level)
	levelStr := h.formatLevelString(r.Level)
	buf.WriteString(levelColor.Sprint(levelStr))
	buf.WriteString(" ")

	// Source information (if enabled)
	if h.opts.AddSource && r.PC != 0 {
		frame := getFrame(r.PC)
		buf.WriteString(h.colors.source.Sprintf("[%s:%d]", frame.File, frame.Line))
		buf.WriteString(" ")
	}

	// Message
	buf.WriteString(h.colors.message.Sprint(r.Message))

	// Attributes
	if r.NumAttrs() > 0 || len(h.attrs) > 0 {
		buf.WriteString(" ")
		buf.WriteString(h.colors.bracket.Sprint("{\n  "))

		first := true

		// Handler-level attributes
		for _, attr := range h.attrs {
			if !first {
				buf.WriteString(", ")
			}
			h.formatAttr(buf, attr)
			first = false
		}

		// Record attributes
		r.Attrs(func(attr slog.Attr) bool {
			if !first {
				buf.WriteString(", ")
			}
			h.formatAttr(buf, attr)
			first = false
			return true
		})

		buf.WriteString(h.colors.bracket.Sprint("\n}"))
	}

	buf.WriteString("\n")
}

// WithAttrs returns a new handler with the given attributes
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

// WithGroup returns a new handler with the given group
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
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
		// Pad to 5 characters for consistency
		s := strings.ToUpper(level.String())
		if len(s) < 5 {
			s += strings.Repeat(" ", 5-len(s))
		}
		return s
	}
}

// formatAttr formats a single attribute with colors
func (h *PrettyHandler) formatAttr(buf *strings.Builder, attr slog.Attr) {
	buf.WriteString(h.colors.attrKey.Sprint(attr.Key))
	buf.WriteString(h.colors.attrValue.Sprint("="))

	// Format value based on type
	switch attr.Value.Kind() {
	case slog.KindString:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.Quote(attr.Value.String())))
	case slog.KindInt64:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.FormatInt(attr.Value.Int64(), 10)))
	case slog.KindUint64:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.FormatUint(attr.Value.Uint64(), 10)))
	case slog.KindFloat64:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.FormatFloat(attr.Value.Float64(), 'f', -1, 64)))
	case slog.KindBool:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.FormatBool(attr.Value.Bool())))
	case slog.KindDuration:
		buf.WriteString(h.colors.attrValue.Sprint(attr.Value.Duration().String()))
	case slog.KindTime:
		buf.WriteString(h.colors.attrValue.Sprint(attr.Value.Time().Format(time.RFC3339)))
	default:
		buf.WriteString(h.colors.attrValue.Sprint(strconv.Quote(attr.Value.String())))
	}
}

// Frame cache for better performance
var (
	frameCache        sync.Map
	frameCacheSize    uint32
	maxFrameCacheSize = uint32(10000)
)

// getFrame extracts frame information from PC with proper skip calculation
func getFrame(pc uintptr) runtime.Frame {
	// Check cache first
	if cached, ok := frameCache.Load(pc); ok {
		return cached.(runtime.Frame)
	}

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

	// Cache the result
	if frameCacheSize < maxFrameCacheSize {
		frameCache.Store(pc, frame)
		frameCacheSize++
	} else {
		// If the cache is full, clear it
		frameCache.Clear()
		frameCacheSize = 0
		frameCache.Store(pc, frame)
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
