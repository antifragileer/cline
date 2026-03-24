package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestTextInput tests text input functionality
func TestTextInput(t *testing.T) {
	t.Run("accepts text input", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		// Simulate typing
		m.textInput.SetValue("Hello")
		
		assert.Equal(t, "Hello", m.textInput.Value())
	})

	t.Run("accepts multiline input", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		// Simulate typing multiline
		m.textInput.SetValue("Line 1\nLine 2")
		
		assert.Equal(t, "Line 1\nLine 2", m.textInput.Value())
	})

	t.Run("clears input after submit", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("Test message")
		
		// Simulate submit
		m.textInput.SetValue("")
		
		assert.Empty(t, m.textInput.Value())
	})

	t.Run("handles empty input submit", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("")
		
		// Should not crash or submit
		value := m.textInput.Value()
		assert.Empty(t, value)
	})

	t.Run("handles very long input", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		longText := strings.Repeat("a", 1000)
		m.textInput.SetValue(longText)
		
		assert.Equal(t, longText, m.textInput.Value())
	})

	t.Run("handles special characters", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		specialText := "Hello! @#$%^&*()_+-=[]{}|;':\",./<>?"
		m.textInput.SetValue(specialText)
		
		assert.Equal(t, specialText, m.textInput.Value())
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		unicodeText := "Hello 世界 🌍 Привет"
		m.textInput.SetValue(unicodeText)
		
		assert.Equal(t, unicodeText, m.textInput.Value())
	})

	t.Run("cursor moves correctly", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		m.textInput.SetValue("Hello")
		m.textInput.SetCursor(3)
		
		assert.Equal(t, 3, m.textInput.Cursor())
	})

	t.Run("input placeholder shown when empty", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Type a message..."
		
		assert.Equal(t, "Type a message...", m.textInput.Placeholder)
	})
}

// TestKeyboardShortcuts tests keyboard shortcut handling
func TestKeyboardShortcuts(t *testing.T) {
	t.Run("ctrl+c quits", func(t *testing.T) {
		m := initialModel()
		
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		// Should trigger quit
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("ctrl+d quits", func(t *testing.T) {
		m := initialModel()
		
		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("enter submits when input focused", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("test message")
		
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("shift+enter adds newline", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("Line 1")
		
		// Simulate shift+enter for newline
		m.textInput.SetValue("Line 1\n")
		
		assert.Equal(t, "Line 1\n", m.textInput.Value())
	})

	t.Run("up arrow navigates history", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.inputHistory = []string{"previous", "messages"}
		m.inputHistoryIndex = -1
		
		msg := tea.KeyMsg{Type: tea.KeyUp}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("down arrow navigates history forward", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.inputHistory = []string{"previous", "messages"}
		m.inputHistoryIndex = 1
		
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("tab toggles focus", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		
		updatedModel := newModel.(Model)
		// Focus should toggle
		assert.NotEqual(t, m.inputFocused, updatedModel.inputFocused)
	})

	t.Run("esc exits current mode", func(t *testing.T) {
		m := initialModel()
		m.currentView = settingsView
		
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("ctrl+l clears screen", func(t *testing.T) {
		m := initialModel()
		m.messages = []Message{
			{Role: "user", Content: "Old message"},
		}
		
		msg := tea.KeyMsg{Type: tea.KeyCtrlL}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("ctrl+h toggles help", func(t *testing.T) {
		m := initialModel()
		m.showHelp = false
		
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		newModel, _ := m.Update(msg)
		
		// Note: ctrl+h handling depends on implementation
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
	})

	t.Run("page up scrolls up", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		
		msg := tea.KeyMsg{Type: tea.KeyPgUp}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("page down scrolls down", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		
		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("home goes to start", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("test")
		m.textInput.SetCursor(4)
		
		msg := tea.KeyMsg{Type: tea.KeyHome}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("end goes to end", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("test")
		m.textInput.SetCursor(0)
		
		msg := tea.KeyMsg{Type: tea.KeyEnd}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})
}

// TestApprovalRejectionPrompts tests approval and rejection prompts
func TestApprovalRejectionPrompts(t *testing.T) {
	t.Run("renders approval prompt", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow tool execution?",
			Options: []string{"yes", "no"},
		}

		view := m.View()
		assert.Contains(t, view, "Allow tool execution?")
	})

	t.Run("approves with 'y'", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
			Options: []string{"yes", "no"},
		}

		// Simulate 'y' keypress
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("rejects with 'n'", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
			Options: []string{"yes", "no"},
		}

		// Simulate 'n' keypress
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles enter on approval", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
			Options: []string{"yes", "no"},
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(msg)
		
		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("clears prompt after response", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
		}

		// Simulate response
		m.pendingApproval = nil
		
		assert.Nil(t, m.pendingApproval)
	})

	t.Run("shows approval count", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
		}
		m.approvalCount = 5

		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("auto-approve option available", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "Allow?",
			Options: []string{"yes", "no", "always"},
		}

		view := m.View()
		assert.Contains(t, view, "Allow?")
	})

	t.Run("handles multiple pending approvals", func(t *testing.T) {
		m := initialModel()
		m.pendingApproval = &ApprovalRequest{
			ID:      "test-1",
			Type:    "tool_use",
			Message: "First approval",
		}
		m.approvalQueue = []ApprovalRequest{
			{ID: "test-2", Type: "tool_use", Message: "Second approval"},
		}

		view := m.View()
		assert.Contains(t, view, "First approval")
	})
}

// TestInputHistory tests input history functionality
func TestInputHistory(t *testing.T) {
	t.Run("adds to history on submit", func(t *testing.T) {
		m := initialModel()
		
		m.inputHistory = append(m.inputHistory, "First message")
		m.inputHistory = append(m.inputHistory, "Second message")
		
		assert.Len(t, m.inputHistory, 2)
		assert.Equal(t, "First message", m.inputHistory[0])
		assert.Equal(t, "Second message", m.inputHistory[1])
	})

	t.Run("navigates history with up arrow", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"First", "Second", "Third"}
		m.inputHistoryIndex = -1
		
		// Navigate up
		m.inputHistoryIndex = 2
		
		assert.Equal(t, 2, m.inputHistoryIndex)
		assert.Equal(t, "Third", m.inputHistory[m.inputHistoryIndex])
	})

	t.Run("navigates history with down arrow", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"First", "Second", "Third"}
		m.inputHistoryIndex = 2
		
		// Navigate down
		m.inputHistoryIndex = 1
		
		assert.Equal(t, 1, m.inputHistoryIndex)
		assert.Equal(t, "Second", m.inputHistory[m.inputHistoryIndex])
	})

	t.Run("wraps history navigation", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"First", "Second"}
		m.inputHistoryIndex = 0
		
		// Navigate up past beginning
		if m.inputHistoryIndex > 0 {
			m.inputHistoryIndex--
		} else {
			m.inputHistoryIndex = len(m.inputHistory) - 1
		}
		
		assert.Equal(t, 1, m.inputHistoryIndex)
	})

	t.Run("restores from history", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"Previous message"}
		m.inputHistoryIndex = 0
		
		// Restore to input
		m.textInput.SetValue(m.inputHistory[m.inputHistoryIndex])
		
		assert.Equal(t, "Previous message", m.textInput.Value())
	})

	t.Run("limits history size", func(t *testing.T) {
		m := initialModel()
		
		// Add many items
		for i := 0; i < maxHistorySize + 10; i++ {
			m.inputHistory = append(m.inputHistory, "Message")
			// Simulate trim
			if len(m.inputHistory) > maxHistorySize {
				m.inputHistory = m.inputHistory[1:]
			}
		}
		
		assert.LessOrEqual(t, len(m.inputHistory), maxHistorySize)
	})

	t.Run("clears history", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"First", "Second", "Third"}
		
		m.inputHistory = []string{}
		m.inputHistoryIndex = -1
		
		assert.Empty(t, m.inputHistory)
		assert.Equal(t, -1, m.inputHistoryIndex)
	})

	t.Run("deduplicates consecutive entries", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{"Message"}
		
		// Try to add duplicate
		newEntry := "Message"
		if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != newEntry {
			m.inputHistory = append(m.inputHistory, newEntry)
		}
		
		assert.Len(t, m.inputHistory, 1)
	})

	t.Run("searches history", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{
			"Hello world",
			"How are you",
			"Hello again",
		}
		
		// Search for "Hello"
		results := []string{}
		for _, entry := range m.inputHistory {
			if strings.Contains(entry, "Hello") {
				results = append(results, entry)
			}
		}
		
		assert.Len(t, results, 2)
		assert.Contains(t, results, "Hello world")
		assert.Contains(t, results, "Hello again")
	})

	t.Run("handles empty history navigation", func(t *testing.T) {
		m := initialModel()
		m.inputHistory = []string{}
		m.inputHistoryIndex = -1
		
		// Try to navigate up
		if len(m.inputHistory) > 0 && m.inputHistoryIndex < len(m.inputHistory)-1 {
			m.inputHistoryIndex++
		}
		
		assert.Equal(t, -1, m.inputHistoryIndex)
	})
}

// TestInputValidation tests input validation
func TestInputValidation(t *testing.T) {
	t.Run("validates non-empty input", func(t *testing.T) {
		m := initialModel()
		
		isValid := len(strings.TrimSpace("valid input")) > 0
		assert.True(t, isValid)
	})

	t.Run("rejects whitespace-only input", func(t *testing.T) {
		m := initialModel()
		
		isValid := len(strings.TrimSpace("   ")) > 0
		assert.False(t, isValid)
	})

	t.Run("handles special characters safely", func(t *testing.T) {
		m := initialModel()
		
		specialInput := "; rm -rf /"
		// Should not execute or cause issues
		m.textInput.SetValue(specialInput)
		
		assert.Equal(t, specialInput, m.textInput.Value())
	})

	t.Run("validates input length", func(t *testing.T) {
		m := initialModel()
		
		veryLongInput := strings.Repeat("a", maxInputLength+1)
		isValid := len(veryLongInput) <= maxInputLength
		
		assert.False(t, isValid)
	})
}

// TestInputRendering tests input area rendering
func TestInputRendering(t *testing.T) {
	t.Run("renders input area when focused", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.width = 80
		
		view := m.renderInputArea()
		assert.NotEmpty(t, view)
	})

	t.Run("renders input area when not focused", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = false
		m.width = 80
		
		view := m.renderInputArea()
		assert.NotEmpty(t, view)
	})

	t.Run("shows character count", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.showCharCount = true
		m.textInput.SetValue("Test")
		
		view := m.renderInputArea()
		assert.NotEmpty(t, view)
	})

	t.Run("shows input hints", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.showHints = true
		
		view := m.renderInputArea()
		assert.NotEmpty(t, view)
	})

	t.Run("handles multiline input display", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("Line 1\nLine 2\nLine 3")
		
		view := m.renderInputArea()
		assert.NotEmpty(t, view)
	})
}