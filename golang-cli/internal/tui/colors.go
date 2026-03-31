// Package tui provides terminal UI components for the Cline CLI.
package tui

// Color constants for the CLI
// Using hex values for consistent rendering across terminals
// These match the TypeScript CLI color scheme

const (
	// Primary brand color - light purple-blue
	PrimaryBlue = "#B1B9F9"

	// Selection/highlight color - cyan blue
	SelectionBlue = "#00D9FF"

	// Plan mode color - golden yellow
	PlanYellow = "#FFB000"

	// Standard colors
	White   = "#FFFFFF"
	Gray    = "#808080"
	DimGray = "#505050"

	// Status colors
	SuccessGreen = "#00FF00"
	ErrorRed     = "#FF4444"
	WarningAmber = "#FFB000"

	// Background colors
	DarkBackground = "#1a1a1a"
)

// GetModeColor returns the appropriate color for the current mode
func GetModeColor(mode string) string {
	if mode == "plan" {
		return PlanYellow
	}
	return PrimaryBlue
}

// GetModeSelectionColor returns the selection color for the current mode
func GetModeSelectionColor(mode string) string {
	if mode == "plan" {
		return PlanYellow
	}
	return SelectionBlue
}