package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// PrettyHandler implements slog.Handler for human-friendly terminal output
type PrettyHandler struct {
	// Options for the handler
	opts slog.HandlerOptions

	// Writer where logs will be written
	renderer *lipgloss.Renderer

	// Attributes to include in every log record
	attrs []slog.Attr

	// Groups for log records
	groups []string

	// Styles for different log elements
	styles *prettyStyles

	// Mutex to protect concurrent writes
	mu *sync.Mutex

	// Pool for reusing string builders to reduce allocations
	builderPool *sync.Pool
}

// prettyStyles holds color functions for different elements
type prettyStyles struct {
	timestamp lipgloss.Style
	debug     lipgloss.Style
	info      lipgloss.Style
	warn      lipgloss.Style
	error     lipgloss.Style
	source    lipgloss.Style
	message   lipgloss.Style
	attrKey   lipgloss.Style
	attrValue lipgloss.Style
	bracket   lipgloss.Style
}

// NewPrettyHandler creates a new pretty handler
func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	renderer := lipgloss.NewRenderer(w)

	styles := &prettyStyles{
		timestamp: renderer.NewStyle().Foreground(lipgloss.Color("#888888")),
		debug:     renderer.NewStyle().Foreground(lipgloss.Color("#ff00ff")).Bold(true),
		info:      renderer.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true),
		warn:      renderer.NewStyle().Foreground(lipgloss.Color("#ffff00")).Bold(true),
		error:     renderer.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true),
		source:    renderer.NewStyle().Foreground(lipgloss.Color("#00ffff")),
		message:   renderer.NewStyle().Bold(true),
		attrKey:   renderer.NewStyle().Foreground(lipgloss.Color("#00ffff")),
		attrValue: renderer.NewStyle().Foreground(lipgloss.Color("#888888")),
		bracket:   renderer.NewStyle().Foreground(lipgloss.Color("#666666")),
	}

	return &PrettyHandler{
		opts:     *opts,
		renderer: renderer,
		styles:   styles,
		mu:       &sync.Mutex{},
		builderPool: &sync.Pool{
			New: func() any {
				builder := &strings.Builder{}
				// Pre-allocate 1024 bytes
				builder.Grow(1024)
				return builder
			},
		},
	}
}

// clone creates a copy of the handler with the same options and renderer
func (h *PrettyHandler) clone() *PrettyHandler {
	return &PrettyHandler{
		opts:        h.opts,
		renderer:    h.renderer,
		attrs:       slices.Clip(h.attrs),
		groups:      slices.Clip(h.groups),
		styles:      h.styles,
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

	_, err := h.renderer.Output().Write([]byte(buf.String()))
	return err
}

// formatRecord formats a log record into the provided buffer
func (h *PrettyHandler) formatRecord(buf *strings.Builder, r slog.Record) {
	// Timestamp
	buf.WriteString(h.styles.timestamp.Render(r.Time.Format("15:04:05.000")))
	buf.WriteString(" ")

	// Level with color and padding
	levelStyle := h.getLevelStyle(r.Level)
	levelStr := h.formatLevelString(r.Level)
	buf.WriteString(levelStyle.Render(levelStr))
	buf.WriteString(" ")

	// Source information (if enabled)
	if h.opts.AddSource && r.PC != 0 {
		frame := getFrame(r.PC)
		buf.WriteString(h.styles.source.Render(fmt.Sprintf("[%s:%d]", frame.File, frame.Line)))
		buf.WriteString(" ")
	}

	// Message
	buf.WriteString(h.styles.message.Render(r.Message))

	// Attributes
	if r.NumAttrs() > 0 || len(h.attrs) > 0 {
		buf.WriteString(" ")
		buf.WriteString(h.styles.bracket.Render("{\n  "))

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

		buf.WriteString(h.styles.bracket.Render("\n}"))
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
func (h *PrettyHandler) getLevelStyle(level slog.Level) lipgloss.Style {
	switch level {
	case slog.LevelDebug:
		return h.styles.debug
	case slog.LevelInfo:
		return h.styles.info
	case slog.LevelWarn:
		return h.styles.warn
	case slog.LevelError:
		return h.styles.error
	default:
		return h.renderer.NewStyle().Foreground(lipgloss.Color("240"))
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
	buf.WriteString(h.styles.attrKey.Render(attr.Key))
	buf.WriteString(h.styles.attrValue.Render("="))

	// Format value based on type
	switch attr.Value.Kind() {
	case slog.KindString:
		buf.WriteString(h.styles.attrValue.Render(strconv.Quote(attr.Value.String())))
	case slog.KindInt64:
		buf.WriteString(h.styles.attrValue.Render(strconv.FormatInt(attr.Value.Int64(), 10)))
	case slog.KindUint64:
		buf.WriteString(h.styles.attrValue.Render(strconv.FormatUint(attr.Value.Uint64(), 10)))
	case slog.KindFloat64:
		buf.WriteString(h.styles.attrValue.Render(strconv.FormatFloat(attr.Value.Float64(), 'f', -1, 64)))
	case slog.KindBool:
		buf.WriteString(h.styles.attrValue.Render(strconv.FormatBool(attr.Value.Bool())))
	case slog.KindDuration:
		buf.WriteString(h.styles.attrValue.Render(attr.Value.Duration().String()))
	case slog.KindTime:
		buf.WriteString(h.styles.attrValue.Render(attr.Value.Time().Format(time.RFC3339)))
	default:
		buf.WriteString(h.styles.attrValue.Render(strconv.Quote(attr.Value.String())))
	}
}

// Frame cache for better performance
var (
	frameCache        sync.Map
	frameCacheSize    atomic.Uint32
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
	if frameCacheSize.Load() < maxFrameCacheSize {
		frameCache.Store(pc, frame)
		frameCacheSize.Add(1)
	} else {
		// If the cache is full, clear it
		frameCache.Clear()
		frameCacheSize.Store(0)
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
