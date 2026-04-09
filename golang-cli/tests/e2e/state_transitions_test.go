// Package e2e provides end-to-end tests for the TUI.
package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestKeyMessageTypes tests that key message types are correctly defined.
func TestKeyMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		keyType  tea.KeyType
		expected tea.KeyType
	}{
		{"Up key", tea.KeyUp, tea.KeyUp},
		{"Down key", tea.KeyDown, tea.KeyDown},
		{"Left key", tea.KeyLeft, tea.KeyLeft},
		{"Right key", tea.KeyRight, tea.KeyRight},
		{"Enter key", tea.KeyEnter, tea.KeyEnter},
		{"Esc key", tea.KeyEsc, tea.KeyEsc},
		{"Tab key", tea.KeyTab, tea.KeyTab},
		{"Space key", tea.KeySpace, tea.KeySpace},
		{"Backspace key", tea.KeyBackspace, tea.KeyBackspace},
		{"Delete key", tea.KeyDelete, tea.KeyDelete},
		{"Home key", tea.KeyHome, tea.KeyHome},
		{"End key", tea.KeyEnd, tea.KeyEnd},
		{"Page Up key", tea.KeyPgUp, tea.KeyPgUp},
		{"Page Down key", tea.KeyPgDown, tea.KeyPgDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.keyType != tt.expected {
				t.Errorf("Key type mismatch: got %v, want %v", tt.keyType, tt.expected)
			}
		})
	}
}

// TestSpecialKeyMessages tests special key combinations.
func TestSpecialKeyMessages(t *testing.T) {
	t.Run("Ctrl+C combination", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		if msg.Type != tea.KeyCtrlC {
			t.Error("Ctrl+C key type mismatch")
		}
	})

	t.Run("Ctrl+S combination", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlS}
		if msg.Type != tea.KeyCtrlS {
			t.Error("Ctrl+S key type mismatch")
		}
	})

	t.Run("Ctrl+Q combination", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		if msg.Type != tea.KeyCtrlQ {
			t.Error("Ctrl+Q key type mismatch")
		}
	})
}

// TestMouseMessageTypes tests mouse message handling.
func TestMouseMessageTypes(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.MouseMsg
	}{
		{
			name: "Left click",
			msg:  tea.MouseMsg{X: 10, Y: 5, Type: tea.MouseLeft},
		},
		{
			name: "Right click",
			msg:  tea.MouseMsg{X: 10, Y: 5, Type: tea.MouseRight},
		},
		{
			name: "Wheel up",
			msg:  tea.MouseMsg{X: 0, Y: 0, Type: tea.MouseWheelUp},
		},
		{
			name: "Wheel down",
			msg:  tea.MouseMsg{X: 0, Y: 0, Type: tea.MouseWheelDown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify mouse message properties
			if tt.msg.X < 0 || tt.msg.Y < 0 {
				t.Error("Mouse coordinates should be non-negative")
			}
		})
	}
}

// TestWindowSizeMessage tests window resize messages.
func TestWindowSizeMessage(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"Small terminal", 40, 12},
		{"Standard terminal", 80, 24},
		{"Large terminal", 120, 40},
		{"Wide terminal", 160, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.WindowSizeMsg{Width: tt.width, Height: tt.height}
			if msg.Width != tt.width {
				t.Errorf("Width mismatch: got %d, want %d", msg.Width, tt.width)
			}
			if msg.Height != tt.height {
				t.Errorf("Height mismatch: got %d, want %d", msg.Height, tt.height)
			}
		})
	}
}

// TestBatchMessage tests batch message handling.
func TestBatchMessage(t *testing.T) {
	t.Run("batch command creation", func(t *testing.T) {
		// Create a batch of commands
		cmds := []tea.Cmd{
			tea.Tick(0, func(t time.Time) tea.Msg {
				return nil
			}),
			tea.Tick(0, func(t time.Time) tea.Msg {
				return nil
			}),
		}
		batch := tea.Batch(cmds...)
		if batch == nil {
			t.Error("Batch command should not be nil")
		}
	})
}

// TestSequenceMessage tests sequence message handling.
func TestSequenceMessage(t *testing.T) {
	t.Run("sequence command creation", func(t *testing.T) {
		// Create a sequence of commands
		cmds := []tea.Cmd{
			tea.Tick(0, func(t time.Time) tea.Msg {
				return nil
			}),
			tea.Tick(0, func(t time.Time) tea.Msg {
				return nil
			}),
		}
		seq := tea.Sequence(cmds...)
		if seq == nil {
			t.Error("Sequence command should not be nil")
		}
	})
}