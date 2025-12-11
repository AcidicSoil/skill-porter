package conversion

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecuteCommand_Success(t *testing.T) {
	ctx := context.Background()
	// Use 'echo' as a portable command
	out, err := ExecuteCommand(ctx, "echo", []string{"hello"})
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("Expected 'hello' in output, got: %q", out)
	}
}

func TestExecuteCommand_Failure(t *testing.T) {
	ctx := context.Background()
	// Use 'false' or a command that exits non-zero (ls non-existent)
	// 'ls /non/existent/path' usually fails
	_, err := ExecuteCommand(ctx, "ls", []string{"/non/existent/path/999"})
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecuteCommand_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// sleep 1 second should fail
	_, err := ExecuteCommand(ctx, "sleep", []string{"1"})
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
