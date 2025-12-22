package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLoggerJSONMode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("json", buf)
	logger.Info("hello", map[string]any{"k": "v"})

	if !strings.Contains(buf.String(), `"hello"`) || !strings.Contains(buf.String(), `"k":"v"`) {
		t.Fatalf("expected json output, got: %s", buf.String())
	}
}

func TestContextRoundTrip(t *testing.T) {
	logger := NewLogger("text", &bytes.Buffer{})
	ctx := WithLogger(context.Background(), logger)
	if FromContext(ctx) != logger {
		t.Fatalf("expected logger from context")
	}
}
