package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Logger provides structured logging with optional JSON formatting.
type Logger struct {
	writer   io.Writer
	jsonMode bool
}

// NewLogger builds a new Logger writing to the given writer.
func NewLogger(format string, w io.Writer) *Logger {
	return &Logger{
		writer:   w,
		jsonMode: strings.EqualFold(format, "json"),
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string, fields map[string]any) {
	l.log("info", msg, fields)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields map[string]any) {
	l.log("warn", msg, fields)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields map[string]any) {
	l.log("error", msg, fields)
}

// Debug logs debug messages when using text output.
func (l *Logger) Debug(msg string, fields map[string]any) {
	// Keep debug noise out of JSON unless explicitly added in the future.
	if l.jsonMode {
		return
	}
	l.log("debug", msg, fields)
}

func (l *Logger) log(level, msg string, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["level"] = level
	fields["msg"] = msg
	fields["ts"] = time.Now().Format(time.RFC3339)

	if l.jsonMode {
		encoder := json.NewEncoder(l.writer)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(fields)
		return
	}

	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf("[%s] %s", strings.ToUpper(level), msg))
	for k, v := range fields {
		if k == "level" || k == "msg" || k == "ts" {
			continue
		}
		builder.WriteString(fmt.Sprintf(" %s=%v", k, v))
	}
	fmt.Fprintln(l.writer, builder.String())
}

// Context key type prevents collisions.
type ctxKey struct{}

// WithLogger stores a logger inside a context.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger if present or a fallback that writes nowhere.
func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return NewLogger("text", io.Discard)
	}
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok && l != nil {
		return l
	}
	return NewLogger("text", io.Discard)
}
