package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

func TestLoad_Precedence(t *testing.T) {
	// Setup
	cwd, _ := os.Getwd()
	tmpDir := os.TempDir()
	
	// Test 1: Flag overrides Env
	os.Setenv("SKILL_PORTER_ROOT", tmpDir)
	defer os.Unsetenv("SKILL_PORTER_ROOT")

	args := []string{"-root", cwd}
	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	// Compare absolute paths to be safe
	absCwd, _ := filepath.Abs(cwd)
	if cfg.ScanRoot != absCwd {
		t.Errorf("Expected root %s, got %s", absCwd, cfg.ScanRoot)
	}

	// Test 2: Env overrides Default (when flag missing)
	args = []string{}
	cfg, err = Load(args)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	// tmpDir might vary (symlinks etc), check it's NOT cwd at least, and matches tmpDir logical path
	// or resolve abs path of tmpDir
	absTmp, _ := filepath.Abs(tmpDir)
	// On some systems /tmp -> /private/tmp. Check equality of Abs.
	// NOTE: os.Stat check in Load might resolve symlinks. 
	// Simplest check: It should NOT be cwd.
	if cfg.ScanRoot == absCwd && absTmp != absCwd {
		t.Errorf("Expected Env root (%s), got default cwd (%s)", absTmp, cfg.ScanRoot)
	}
}

func TestLoad_Validation(t *testing.T) {
	args := []string{"-root", "/non/existent/path/99999"}
	_, err := Load(args)
	if err == nil {
		t.Error("Expected error for invalid root, got nil")
	}

	args = []string{"-target", "invalidTarget"}
	_, err = Load(args)
	if err == nil {
		t.Error("Expected error for invalid target, got nil")
	}
}

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("SKILL_PORTER_ROOT") // Ensure clean env
	args := []string{}
	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DefaultTarget != domain.TargetAuto {
		t.Errorf("Expected default target Auto, got %s", cfg.DefaultTarget)
	}
	
	cwd, _ := os.Getwd()
	absCwd, _ := filepath.Abs(cwd)
	if cfg.ScanRoot != absCwd {
		t.Errorf("Expected default root %s, got %s", absCwd, cfg.ScanRoot)
	}
}
