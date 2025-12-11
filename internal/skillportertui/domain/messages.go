package domain

// SkillsDiscoveredMsg is sent when the discovery process completes
type SkillsDiscoveredMsg struct {
	Skills []SkillDir
}

// SkillConvertedMsg is sent when a single skill conversion completes successfully
type SkillConvertedMsg struct {
	SkillPath string // Using Path as ID
	Output    string
}

// ConversionErrorMsg is sent when a single skill conversion fails
type ConversionErrorMsg struct {
	SkillPath string // Using Path as ID
	Err       error
}
