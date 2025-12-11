package ui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/config"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/conversion"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

// Focus indices
const (
	inputRoot = iota
	inputOutput
	toggleRecursive
	toggleTarget
	btnSubmit
	fieldCount // 5
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.State == StateConfig {
		return m.updateConfig(msg)
	}

	return m.updateBrowsing(msg)
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.Inputs))

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Handle Submit on Enter if on Button
			if s == "enter" && m.FocusIndex == btnSubmit {
				// Apply Config
				m.Config.ScanRoot = m.Inputs[inputRoot].Value()
				if m.Config.ScanRoot == "" {
					m.Config.ScanRoot = "."
				}
				
				m.Config.OutBaseDir = m.Inputs[inputOutput].Value()
				// Create Output Dir if set
				if m.Config.OutBaseDir != "" {
					_ = os.MkdirAll(m.Config.OutBaseDir, 0755)
				}

				// Transition
				m.State = StateBrowsing
				return m, discoverSkillsCmd(m.Config.ScanRoot, m.Config.RecursiveMode)
			}

			// Handle Toggle Switching on Enter/Space
			if (s == "enter" || s == " ") && m.FocusIndex == toggleRecursive {
				m.Config.RecursiveMode = !m.Config.RecursiveMode
				return m, nil
			}
			if (s == "enter" || s == " ") && m.FocusIndex == toggleTarget {
				switch m.Config.DefaultTarget {
				case domain.TargetAuto:
					m.Config.DefaultTarget = domain.TargetGemini
				case domain.TargetGemini:
					m.Config.DefaultTarget = domain.TargetClaude
				case domain.TargetClaude:
					m.Config.DefaultTarget = domain.TargetAuto
				}
				return m, nil
			}

			// Navigation
			if s == "up" || s == "shift+tab" {
				m.FocusIndex--
			} else {
				m.FocusIndex++
			}

			// Cycle
			if m.FocusIndex > fieldCount-1 {
				m.FocusIndex = 0
			} else if m.FocusIndex < 0 {
				m.FocusIndex = fieldCount - 1
			}

			// Update Text Input Focus
			for i := 0; i <= 1; i++ {
				if i == m.FocusIndex {
					cmds[i] = m.Inputs[i].Focus()
					m.Inputs[i].TextStyle = selectedItemStyle
				} else {
					m.Inputs[i].Blur()
					m.Inputs[i].TextStyle = lipgloss.NewStyle() // Reset
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Update inputs only if they are focused
	for i := range m.Inputs {
		m.Inputs[i], cmds[i] = m.Inputs[i].Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Skills)-1 {
				m.Cursor++
			}
		case "c":
			if len(m.Skills) > 0 {
				idx := m.Cursor
				skill := &m.Skills[idx]
				if skill.Status == domain.StatusPending || skill.Status == domain.StatusFailed {
					m.Skills[idx].Status = domain.StatusRunning
					cmd = convertSkillCmd(skill, m.Config, domain.TargetAuto)
				}
			}
		case "g":
			if len(m.Skills) > 0 {
				idx := m.Cursor
				m.Skills[idx].Status = domain.StatusRunning
				cmd = convertSkillCmd(&m.Skills[idx], m.Config, domain.TargetGemini)
			}
		case "a":
			if len(m.Skills) > 0 {
				idx := m.Cursor
				m.Skills[idx].Status = domain.StatusRunning
				cmd = convertSkillCmd(&m.Skills[idx], m.Config, domain.TargetClaude)
			}
		case "r":
			m.Skills = []domain.SkillDir{}
			m.Cursor = 0
			m.SuccessCount = 0
			m.FailCount = 0
			// Need to re-trigger discovery based on CURRENT config (which might have been edited in Setup)
			cmd = discoverSkillsCmd(m.Config.ScanRoot, m.Config.RecursiveMode)
		
		case "A": // Auto-Convert All Pending
			var cmds []tea.Cmd
			for i := range m.Skills {
				if m.Skills[i].Status == domain.StatusPending {
					m.Skills[i].Status = domain.StatusRunning
					cmds = append(cmds, convertSkillCmd(&m.Skills[i], m.Config, domain.TargetAuto))
				}
			}
			if len(cmds) > 0 {
				cmd = tea.Batch(cmds...)
			}
		case "backspace", "delete":
			// Allow going back to Config?
			// Sure, why not. "b" or "esc" or "backspace"
			// Let's use "esc" to go back to Config
		case "esc":
			m.State = StateConfig
			return m, nil
		}
	
	case domain.SkillsDiscoveredMsg:
		m.Skills = msg.Skills
		m.Cursor = 0
		m.SuccessCount = 0
		m.FailCount = 0

	case domain.SkillConvertedMsg:
		for i := range m.Skills {
			if m.Skills[i].Path == msg.SkillPath {
				m.Skills[i].Status = domain.StatusSuccess
				m.Skills[i].OutputPath = msg.Output
				m.SuccessCount++
				break
			}
		}

	case domain.ConversionErrorMsg:
		for i := range m.Skills {
			if m.Skills[i].Path == msg.SkillPath {
				m.Skills[i].Status = domain.StatusFailed
				m.Skills[i].ErrorLog = msg.Err.Error()
				m.FailCount++
				break
			}
		}
	}

	return m, cmd
}

func convertSkillCmd(skill *domain.SkillDir, cfg *config.AppConfig, override domain.ConversionTarget) tea.Cmd {
	s := *skill
	return func() tea.Msg {
		target := override
		
		if target == domain.TargetAuto {
			target = s.Target
		}
		
		if target == domain.TargetAuto {
			target = cfg.DefaultTarget
		}
		
		if target == domain.TargetAuto {
			if s.CurrentPlatform == "Claude" {
				target = domain.TargetGemini
			} else if s.CurrentPlatform == "Gemini" {
				target = domain.TargetClaude
			} else {
				target = domain.TargetGemini
			}
		}

		outDir := cfg.OutBaseDir
		if outDir != "" {
			outDir = filepath.Join(outDir, s.Name)
		}

		args, err := conversion.BuildConvertCommand(s.Path, target, outDir)
		if err != nil {
			return domain.ConversionErrorMsg{SkillPath: s.Path, Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		output, err := conversion.ExecuteCommand(ctx, "skill-porter", args)
		if err != nil {
			return domain.ConversionErrorMsg{SkillPath: s.Path, Err: err}
		}

		return domain.SkillConvertedMsg{SkillPath: s.Path, Output: output}
	}
}