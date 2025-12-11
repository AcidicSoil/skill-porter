package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

// DiscoverSkills walks the directory tree rooted at root and returns a list of discovered skills.
// It identifies skills by the presence of SKILL.md (Claude) or gemini-extension.json (Gemini).
func DiscoverSkills(root string, recursive bool) ([]domain.SkillDir, error) {
	var skills []domain.SkillDir
	seen := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		// Handle recursion depth
		if !recursive {
			rel, err := filepath.Rel(root, path)
			if err == nil && rel != "." {
				// If relative path contains a separator, it's deeper than immediate children
				if strings.Contains(rel, string(os.PathSeparator)) {
					return filepath.SkipDir
				}
			}
		}

		// Check for skill markers
		isClaude := fileExists(filepath.Join(path, "SKILL.md"))
		isGemini := fileExists(filepath.Join(path, "gemini-extension.json"))
		
		platform := ""
		if isClaude && isGemini {
			platform = "Universal"
		} else if isClaude {
			platform = "Claude"
		} else if isGemini {
			platform = "Gemini"
		}

		if platform != "" {
			// Deduplicate (unlikely needed with WalkDir logic but safe)
			if seen[path] {
				return filepath.SkipDir
			}
			seen[path] = true

			skills = append(skills, domain.SkillDir{
				Name:            filepath.Base(path),
				Path:            path,
				CurrentPlatform: platform,
				Status:          domain.StatusPending,
				Target:          domain.TargetAuto, // Default
			})
			
			// Assume skills are not nested within other skills
			return filepath.SkipDir
		}

		return nil
	})

	return skills, err
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
