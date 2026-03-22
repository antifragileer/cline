package tui

import (
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// TUIKeyMap defines the key bindings for the TUI.
type TUIKeyMap struct {
	// Quit exits the application.
	Quit []string
}

// DefaultTUIKeyMap returns the default key bindings.
func DefaultTUIKeyMap() TUIKeyMap {
	return TUIKeyMap{
		Quit: []string{"q", "ctrl+c", "esc"},
	}
}

// Message types for TUI events

// InitMsg is sent when the TUI is initialized.
type InitMsg struct{}

// ErrorMsg wraps an error for transmission through the TUI.
type ErrorMsg struct {
	Err error
}

// ContentMsg updates the content display.
type ContentMsg struct {
	Content string
}

// ShutdownMsg triggers graceful shutdown.
type ShutdownMsg struct{}

// Update implements the bubbletea.Model interface.
// It handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Window resize events (SIGWINCH)
	case tea.WindowSizeMsg:
		m.dimensions.Width = msg.Width
		m.dimensions.Height = msg.Height
		if !m.ready {
			m.ready = true
		}

	// Custom window size message for testing
	case WindowSizeMsg:
		m.dimensions.Width = msg.Width
		m.dimensions.Height = msg.Height

	// Keyboard events
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	// Initialization message
	case InitMsg:
		m.ready = true
		return m, nil

	// Error message
	case ErrorMsg:
		m.err = msg.Err
		return m, nil

	// Content update message
	case ContentMsg:
		m.content = msg.Content
		return m, nil

	// Shutdown message
	case ShutdownMsg:
		return m, tea.Quit
	}

	return m, nil
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle quit keys
	if m.isQuitKey(key) {
		m.Shutdown()
		return m, tea.Quit
	}

	return m, nil
}

// isQuitKey checks if the key is a quit key.
func (m Model) isQuitKey(key string) bool {
	keys := DefaultTUIKeyMap().Quit
	for _, k := range keys {
		if key == k {
			return true
		}
	}
	return false
}

// SetupSignalHandling sets up a signal handler for graceful shutdown.
// It listens for SIGINT and SIGTERM signals.
func SetupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	return sigChan
}

// SignalCmd creates a command that listens for system signals.
func SignalCmd(sigChan chan os.Signal) tea.Cmd {
	return func() tea.Msg {
		sig := <-sigChan
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			return ShutdownMsg{}
		case syscall.SIGWINCH:
			// Window resize is handled automatically by bubbletea
			return nil
		default:
			return nil
		}
	}
}

// BatchCmd creates a command that sends a batch of messages.
func BatchCmd(msgs ...tea.Msg) tea.Cmd {
	return func() tea.Msg {
		// Return the first message, subsequent messages can be sent via tea.Batch
		if len(msgs) > 0 {
			return msgs[0]
		}
		return nil
	}
}

// UpdateContent creates a command to update the content.
func UpdateContent(content string) tea.Cmd {
	return func() tea.Msg {
		return ContentMsg{Content: content}
	}
}

// ShutdownCmd creates a command to shut down the TUI.
func ShutdownCmd() tea.Cmd {
	return func() tea.Msg {
		return ShutdownMsg{}
	}
}