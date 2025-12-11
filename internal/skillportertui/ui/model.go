package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/config"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/discovery"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/logging"
)

type SessionState int

const (
	StateConfig SessionState = iota
	StateBrowsing
)

type Model struct {
	Config *config.AppConfig
	State  SessionState

	// Config View State
	Inputs     []textinput.Model
	FocusIndex int
	// For toggles that aren't text inputs
	// 0: Root (Text), 1: Output (Text), 2: Recursive (Bool), 3: Target (Enum), 4: Submit (Btn)

	// Browsing View State
	Skills       []domain.SkillDir
	Cursor       int
	SuccessCount int
	FailCount    int
	Err          error

	// Internal state
	width  int
	height int
	logger *logging.Logger
}

func NewModel(cfg *config.AppConfig, logger *logging.Logger) Model {
	m := Model{
		Config: cfg,
		Skills: []domain.SkillDir{},
		logger: logger,
		State:  StateConfig,
	}

	// Initialize Inputs
	m.Inputs = make([]textinput.Model, 2)

	// Root Path Input
	m.Inputs[0] = textinput.New()
	m.Inputs[0].Placeholder = "Current Directory (.)"
	m.Inputs[0].SetValue(cfg.ScanRoot)
	m.Inputs[0].Focus()
	m.Inputs[0].Width = 40
	m.Inputs[0].Prompt = "Scan Root: "

	// Output Path Input
	m.Inputs[1] = textinput.New()
	m.Inputs[1].Placeholder = "In-place (Leave empty)"
	m.Inputs[1].SetValue(cfg.OutBaseDir)
	m.Inputs[1].Width = 40
	m.Inputs[1].Prompt = "Output Dir: "

	// If Auto-Convert flag was set, maybe skip config? 
	// User requested interactive TUI, so we default to Config unless explicitly skipping.
	// We'll stick to Config mode start for now.

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func discoverSkillsCmd(root string, recursive bool) tea.Cmd {
	return func() tea.Msg {
		skills, err := discovery.DiscoverSkills(root, recursive)
		if err != nil {
			return domain.SkillsDiscoveredMsg{Skills: nil}
		}
		return domain.SkillsDiscoveredMsg{Skills: skills}
	}
}