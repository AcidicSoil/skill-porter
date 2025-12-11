package conversion

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// ExecuteCommand runs a command with arguments and captures stdout/stderr.
func ExecuteCommand(ctx context.Context, command string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	errOutput := stderr.String()

	if err != nil {
		// Combine output for debugging if needed, or just return error with stderr info
		return output, fmt.Errorf("command failed: %w\nStderr: %s", err, errOutput)
	}

	return output, nil
}
