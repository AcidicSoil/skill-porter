package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/config"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/logging"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/ui"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logging error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	
	// Pass debug flag
	logger := logging.New(f, cfg.Debug)
	logger.Info("Application started", cfg)

	model := ui.NewModel(cfg, logger)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(ui.Model); ok {
		if m.FailCount > 0 {
			os.Exit(1)
		}
	} else {
		logger.Error("Could not cast final model", nil)
	}
}