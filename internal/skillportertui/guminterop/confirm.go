package guminterop

import (
	"os/exec"
)

// IsGumAvailable checks if the 'gum' binary is in the PATH.
func IsGumAvailable() bool {
	_, err := exec.LookPath("gum")
	return err == nil
}

// Confirm uses gum to prompt for confirmation. Returns true if confirmed or gum missing.
func Confirm(msg string) bool {
	if !IsGumAvailable() {
		return true // Fallback to allowed if gum is not present
	}
	// gum confirm exits 0 for Yes, 1 for No
	cmd := exec.Command("gum", "confirm", msg)
	err := cmd.Run()
	return err == nil
}
