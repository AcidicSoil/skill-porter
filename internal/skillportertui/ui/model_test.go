package ui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/config"
	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

func TestUpdate_Conversion(t *testing.T) {
	// Setup
	cfg := &config.AppConfig{}
	skills := []domain.SkillDir{
		{Name: "Skill1", Path: "/tmp/s1", Status: domain.StatusPending},
	}
	m := Model{
		Config: cfg,
		Skills: skills,
		Cursor: 0,
	}

	// 1. Trigger Conversion ('c')
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	newModel := newM.(Model)

	if newModel.Skills[0].Status != domain.StatusRunning {
		t.Errorf("Expected status Running, got %s", newModel.Skills[0].Status)
	}
	if cmd == nil {
		t.Error("Expected cmd to be returned, got nil")
	}

	// 2. Handle Success Message
	successMsg := domain.SkillConvertedMsg{
		SkillPath: "/tmp/s1",
		Output:    "Success Output",
	}
	newM, _ = newModel.Update(successMsg)
	newModel = newM.(Model)

	if newModel.Skills[0].Status != domain.StatusSuccess {
		t.Errorf("Expected status Success, got %s", newModel.Skills[0].Status)
	}
	if newModel.SuccessCount != 1 {
		t.Errorf("Expected success count 1, got %d", newModel.SuccessCount)
	}

	// 3. Handle Error Message
	// Reset model
	m.Skills[0].Status = domain.StatusPending
	m.SuccessCount = 0
	
	errorMsg := domain.ConversionErrorMsg{
		SkillPath: "/tmp/s1",
		Err:       errors.New("fail"),
	}
	newM, _ = m.Update(errorMsg)
	newModel = newM.(Model)

	if newModel.Skills[0].Status != domain.StatusFailed {
		t.Errorf("Expected status Failed, got %s", newModel.Skills[0].Status)
	}
	if newModel.FailCount != 1 {
		t.Errorf("Expected fail count 1, got %d", newModel.FailCount)
	}
}