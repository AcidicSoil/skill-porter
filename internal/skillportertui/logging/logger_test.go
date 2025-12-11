package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.Info("test message", map[string]string{"key": "value"})

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO level, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected message, got: %s", output)
	}
	if !strings.Contains(output, "\"key\":\"value\"") {
		t.Errorf("Expected data, got: %s", output)
	}

	// Verify JSON validity
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}
}
