package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("238")).
			PaddingRight(2)

	detailsStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	itemStyle = lipgloss.NewStyle().PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170"))

	statusPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	statusSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusFailStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingTop(1)
	
	// Config Styles
	configTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)
	
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)
	
	activeButtonStyle = buttonStyle.
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#F25D94")).
			MarginRight(2)
)

func (m Model) View() string {
	if m.State == StateConfig {
		return m.viewConfig()
	}
	return m.viewBrowsing()
}

func (m Model) viewConfig() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Skill Porter Setup") + "\n\n")
	b.WriteString(configTitleStyle.Render("Configuration") + "\n\n")

	// 1. Root Input
	b.WriteString(m.Inputs[0].View() + "\n\n")

	// 2. Output Input
	b.WriteString(m.Inputs[1].View() + "\n\n")

	// 3. Recursive Toggle
	recCheck := "[ ]"
	if m.Config.RecursiveMode {
		recCheck = "[x]"
	}
	recStyle := blurredStyle
	if m.FocusIndex == toggleRecursive {
		recStyle = focusedStyle
	}
	b.WriteString(recStyle.Render(fmt.Sprintf("%s Recursive Scan", recCheck)) + "\n\n")

	// 4. Target Select
targetStyle := blurredStyle
	if m.FocusIndex == toggleTarget {
		targetStyle = focusedStyle
	}
	b.WriteString(targetStyle.Render(fmt.Sprintf("Default Target: < %s >", m.Config.DefaultTarget)) + "\n\n")

	// 5. Submit Button
	btn := buttonStyle.Render("Start Scanning")
	if m.FocusIndex == btnSubmit {
		btn = activeButtonStyle.Render("Start Scanning")
	}
	b.WriteString("\n" + btn + "\n")
	
b.WriteString(footerStyle.Render("\nTab/Shift+Tab to navigate • Enter/Space to toggle/submit • Ctrl+C to quit"))

	return lipgloss.NewStyle().Margin(1, 2).Render(b.String())
}

func (m Model) viewBrowsing() string {
	title := titleStyle.Render("Skill Porter Dashboard") + "\n\n"

	if len(m.Skills) == 0 {
		return title + "Scanning or No skills found...\nPress 'r' to rescan or 'esc' to configure."
	}

	// Render List
	var listBuilder strings.Builder
	for i, skill := range m.Skills {
		cursor := " "
		style := itemStyle

		if m.Cursor == i {
			cursor = ">"
			style = selectedItemStyle
		}

		status := string(skill.Status)
		var statusStr string
		switch skill.Status {
		case domain.StatusRunning:
			statusStr = statusRunningStyle.Render(status)
		case domain.StatusSuccess:
			statusStr = statusSuccessStyle.Render(status)
		case domain.StatusFailed:
			statusStr = statusFailStyle.Render(status)
		default:
			statusStr = statusPendingStyle.Render(status)
		}

		row := fmt.Sprintf("%s %s [%s]", cursor, skill.Name, statusStr)
		listBuilder.WriteString(style.Render(row) + "\n")
	}
	listView := listStyle.Render(listBuilder.String())

	// Render Details
	var detailsBuilder strings.Builder
	if m.Cursor >= 0 && m.Cursor < len(m.Skills) {
		selected := m.Skills[m.Cursor]
		detailsBuilder.WriteString(fmt.Sprintf("Name: %s\n", selected.Name))
		detailsBuilder.WriteString(fmt.Sprintf("Path: %s\n", selected.Path))
		detailsBuilder.WriteString(fmt.Sprintf("Platform: %s\n", selected.CurrentPlatform))
		detailsBuilder.WriteString(fmt.Sprintf("Target: %s\n", selected.Target))
		detailsBuilder.WriteString(fmt.Sprintf("Status: %s\n", selected.Status))
		
		if selected.OutputPath != "" {
			detailsBuilder.WriteString("\nOutput:\n")
			detailsBuilder.WriteString(selected.OutputPath)
		}
		if selected.ErrorLog != "" {
			detailsBuilder.WriteString("\nError:\n")
			detailsBuilder.WriteString(selected.ErrorLog)
		}
	}
	detailsView := detailsStyle.Render(detailsBuilder.String())

	// Render Footer
	total := len(m.Skills)

pending := total - m.SuccessCount - m.FailCount
	summary := fmt.Sprintf("Total: %d | Success: %d | Failed: %d | Pending: %d", 
		total, m.SuccessCount, m.FailCount, pending)
	
	help := "\nKeys: ↑/↓: Navigate • c: Convert • g/a: Force Target • A: All • Esc: Config • q: Quit"
	footerView := footerStyle.Render(summary + help)

	// Layout
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailsView)
	
	return lipgloss.JoinVertical(lipgloss.Left, title, mainView, footerView)
}
