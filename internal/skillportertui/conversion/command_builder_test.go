package conversion

import (
	"testing"

	"github.com/jduncan-rva/skill-porter/internal/skillportertui/domain"
)

func TestBuildConvertCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		target    domain.ConversionTarget
		output    string
		wantArgs  []string
		wantErr   bool
	}{
		{
			name:     "Gemini Conversion",
			input:    "./skill",
			target:   domain.TargetGemini,
			output:   "./out",
			wantArgs: []string{"convert", "./skill", "--to", "gemini", "--output", "./out"},
			wantErr:  false,
		},
		{
			name:     "Claude Conversion No Output",
			input:    "/abs/skill",
			target:   domain.TargetClaude,
			output:   "",
			wantArgs: []string{"convert", "/abs/skill", "--to", "claude"},
			wantErr:  false,
		},
		{
			name:    "Auto Target Error",
			input:   "./skill",
			target:  domain.TargetAuto,
			wantErr: true,
		},
		{
			name:    "Missing Input",
			input:   "",
			target:  domain.TargetGemini,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildConvertCommand(tt.input, tt.target, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildConvertCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.wantArgs) {
					t.Errorf("Args length mismatch: got %d, want %d", len(got), len(tt.wantArgs))
				}
				for i, v := range got {
					if v != tt.wantArgs[i] {
						t.Errorf("Arg[%d] mismatch: got %s, want %s", i, v, tt.wantArgs[i])
					}
				}
			}
		})
	}
}
