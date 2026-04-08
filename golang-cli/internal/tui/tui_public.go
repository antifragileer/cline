package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// SupportsTUI returns true if the current environment supports TUI.
// This checks if stdout is a terminal.
func SupportsTUI() bool {
	// For now, always return true
	// In a real implementation, this would check if stdout is a TTY
	return true
}

// WelcomeScreen displays the welcome screen and returns the selected action.
//
// Parameters:
//   - hasConfig: whether the user has valid configuration
//
// Returns:
//   - WelcomeAction: the selected action
//   - string: the task prompt (if ActionNewTask with input)
//   - error: any error that occurred
func WelcomeScreen(hasConfig bool) (WelcomeAction, string, error) {
	model := NewWelcomeModel()

	// Set configuration status
	if !hasConfig {
		// Could show a message or disable certain actions
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return ActionQuit, "", fmt.Errorf("welcome screen error: %w", err)
	}

	wm, ok := m.(*WelcomeModel)
	if !ok {
		return ActionQuit, "", fmt.Errorf("unexpected model type")
	}

	return wm.GetSelected(), wm.GetInputValue(), nil
}

// ChatScreen displays the chat interface for interactive task execution.
//
// Parameters:
//   - taskID: the task ID to resume (empty for new task)
//   - onSend: callback when user sends a message
//   - onInterrupt: callback when user interrupts
//
// Returns:
//   - string: the final message content
//   - error: any error that occurred
func ChatScreen(taskID string, onSend func(string) error, onInterrupt func() error) (string, error) {
	model := NewChatModel()

	if taskID != "" {
		model.SetTaskID(taskID)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	// Store callbacks in model for later use
	// This is a simplified version - in reality you'd need a more complex
	// integration with the task runner

	_, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("chat screen error: %w", err)
	}

	return "", nil
}
