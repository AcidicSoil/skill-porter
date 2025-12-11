package conversion

import (
	"fmt"
	"strings"

	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

// BuildConvertCommand constructs the CLI arguments for skill-porter.
func BuildConvertCommand(inputPath string, target domain.ConversionTarget, outputDir string) ([]string, error) {
	if inputPath == "" {
		return nil, fmt.Errorf("input path is required")
	}

	var args []string
	t := strings.ToLower(string(target))

	// Normalize comparison logic
	switch strings.ToLower(string(target)) {
	case "gemini", "claude":
		args = append(args, "convert", inputPath, "--to", t)
		if outputDir != "" {
			args = append(args, "--output", outputDir)
		}
	case "auto":
		return nil, fmt.Errorf("cannot build command for 'Auto' target; must be resolved")
	default:
		return nil, fmt.Errorf("unsupported target: %s", target)
	}

	return args, nil
}
