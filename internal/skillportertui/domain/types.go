package domain

// ConversionStatus represents the state of a skill conversion
type ConversionStatus string

const (
	StatusPending ConversionStatus = "Pending"
	StatusRunning ConversionStatus = "Running"
	StatusSuccess ConversionStatus = "Success"
	StatusFailed  ConversionStatus = "Failed"
)

// ConversionTarget represents the target platform for conversion
type ConversionTarget string

const (
	TargetGemini ConversionTarget = "Gemini"
	TargetClaude ConversionTarget = "Claude"
	TargetAuto   ConversionTarget = "Auto"
)

// SkillDir represents a discovered skill directory and its conversion state
type SkillDir struct {
	Name            string
	Path            string
	CurrentPlatform string           // e.g. "Claude", "Gemini", "Universal"
	Status          ConversionStatus
	Target          ConversionTarget
	OutputPath      string
	ErrorLog        string
}

// Summary holds the counts of skills in various states
type Summary struct {
	Total   int
	Success int
	Failed  int
	Pending int
}
