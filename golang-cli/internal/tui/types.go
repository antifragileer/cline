package tui

// WelcomeAction represents a quick action on the welcome screen.
type WelcomeAction int

const (
	// ActionNewTask creates a new task.
	ActionNewTask WelcomeAction = iota
	// ActionContinueTask continues the most recent task.
	ActionContinueTask
	// ActionHistory shows task history.
	ActionHistory
	// ActionSettings opens settings.
	ActionSettings
	// ActionHelp shows help.
	ActionHelp
	// ActionQuit quits the application.
	ActionQuit
)
