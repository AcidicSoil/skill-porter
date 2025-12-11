package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

type AppConfig struct {
	ScanRoot        string
	RecursiveMode   bool
	DefaultTarget   domain.ConversionTarget
	OutBaseDir      string
	AutoConvertMode bool
	Debug           bool
}

func Load(args []string) (*AppConfig, error) {
	cfg := &AppConfig{}

	defaultRoot, _ := os.Getwd()
	defaultTarget := string(domain.TargetAuto)

	fs := flag.NewFlagSet("skill-porter-tui", flag.ContinueOnError)
	
	var rootFlag string
	fs.StringVar(&rootFlag, "root", "", "Root directory to scan for skills (default: current directory)")
	fs.BoolVar(&cfg.RecursiveMode, "recursive", true, "Scan recursively")
	targetStr := fs.String("target", defaultTarget, "Default conversion target (Gemini, Claude, Auto)")
	var outFlag string
	fs.StringVar(&outFlag, "out", "", "Base directory for output (default: in-place)")
	fs.BoolVar(&cfg.AutoConvertMode, "auto", false, "Enable auto-convert mode")
	fs.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if rootFlag != "" {
		cfg.ScanRoot = rootFlag
	} else if envRoot := os.Getenv("SKILL_PORTER_ROOT"); envRoot != "" {
		cfg.ScanRoot = envRoot
	} else {
		cfg.ScanRoot = defaultRoot
	}

	if outFlag != "" {
		cfg.OutBaseDir = outFlag
	} else if envOut := os.Getenv("SKILL_PORTER_OUT"); envOut != "" {
		cfg.OutBaseDir = envOut
	}

	switch strings.ToLower(*targetStr) {
	case "gemini":
		cfg.DefaultTarget = domain.TargetGemini
	case "claude":
		cfg.DefaultTarget = domain.TargetClaude
	case "auto":
		cfg.DefaultTarget = domain.TargetAuto
	default:
		return nil, fmt.Errorf("invalid target: %s", *targetStr)
	}

	if info, err := os.Stat(cfg.ScanRoot); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid scan root: %s", cfg.ScanRoot)
	}

	// Validate/Create Out Path
	if cfg.OutBaseDir != "" {
		if err := os.MkdirAll(cfg.OutBaseDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create output directory: %v", err)
		}

		if info, err := os.Stat(cfg.OutBaseDir); err != nil {
			return nil, fmt.Errorf("output directory error: %s", cfg.OutBaseDir)
		} else if !info.IsDir() {
			return nil, fmt.Errorf("output path is not a directory: %s", cfg.OutBaseDir)
		}
		absOut, err := filepath.Abs(cfg.OutBaseDir)
		if err == nil {
			cfg.OutBaseDir = absOut
		}
	}
	
	absRoot, err := filepath.Abs(cfg.ScanRoot)
	if err == nil {
		cfg.ScanRoot = absRoot
	}

	return cfg, nil
}