package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkills(t *testing.T) {
	// Create temp hierarchy
	tmpDir := t.TempDir()

	// 1. Claude Skill
	claudeDir := filepath.Join(tmpDir, "claude-skill")
	os.Mkdir(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "SKILL.md"), []byte{}, 0644)

	// 2. Gemini Skill
	geminiDir := filepath.Join(tmpDir, "gemini-skill")
	os.Mkdir(geminiDir, 0755)
	os.WriteFile(filepath.Join(geminiDir, "gemini-extension.json"), []byte{}, 0644)

	// 3. Universal Skill
	univDir := filepath.Join(tmpDir, "univ-skill")
	os.Mkdir(univDir, 0755)
	os.WriteFile(filepath.Join(univDir, "SKILL.md"), []byte{}, 0644)
	os.WriteFile(filepath.Join(univDir, "gemini-extension.json"), []byte{}, 0644)

	// 4. Nested Skill
	nestedRoot := filepath.Join(tmpDir, "category")
	os.Mkdir(nestedRoot, 0755)
	nestedSkill := filepath.Join(nestedRoot, "nested-skill")
	os.Mkdir(nestedSkill, 0755)
	os.WriteFile(filepath.Join(nestedSkill, "SKILL.md"), []byte{}, 0644)

	// 5. Empty Dir (should be ignored)
	os.Mkdir(filepath.Join(tmpDir, "empty"), 0755)

	// Test Recursive
	skills, err := DiscoverSkills(tmpDir, true)
	if err != nil {
		t.Fatalf("DiscoverSkills recursive failed: %v", err)
	}
	
	if len(skills) != 4 {
		t.Errorf("Expected 4 skills (recursive), got %d", len(skills))
		for _, s := range skills {
			t.Logf("Found: %s (%s)", s.Name, s.CurrentPlatform)
		}
	}

	// Test Non-Recursive
	skills, err = DiscoverSkills(tmpDir, false)
	if err != nil {
		t.Fatalf("DiscoverSkills non-recursive failed: %v", err)
	}

	// Should find claude, gemini, univ. Should NOT find nested.
	// Note: WalkDir non-recursive logic usually finds roots immediate children.
	// "claude-skill", "gemini-skill", "univ-skill" are immediate children.
	// "category" is immediate, but not a skill. "category/nested-skill" is grandchild.
	// So expected count 3.
	if len(skills) != 3 {
		t.Errorf("Expected 3 skills (non-recursive), got %d", len(skills))
	}

	// Check Platform Detection
	for _, s := range skills {
		if s.Name == "univ-skill" && s.CurrentPlatform != "Universal" {
			t.Errorf("Expected Universal platform, got %s", s.CurrentPlatform)
		}
	}
}
