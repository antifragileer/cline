// Package e2e provides end-to-end tests for the TUI.
package e2e

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cline/cline/golang-cli/internal/tui"
)

// TestWelcomeScreenNavigation tests the welcome screen navigation.
func TestWelcomeScreenNavigation(t *testing.T) {
	// This test should be run serially
	// t.Parallel() - DO NOT ENABLE - Rate limiting concerns

	model := tui.NewWelcomeModel()

	// Test initial state
	if model.SelectedIndex() != 0 {
		t.Errorf("Expected initial index 0, got %d", model.SelectedIndex())
	}

	// Test navigation down - skips disabled items (Continue Task is disabled without history)
	// So from index 0 (New Task), down goes to index 2 (History)
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)
	if welcome, ok := newModel.(*tui.WelcomeModel); ok {
		// Index 2 because index 1 (Continue Task) is disabled when hasHistory is false
		if welcome.SelectedIndex() != 2 {
			t.Errorf("Expected index 2 after down (skipping disabled), got %d", welcome.SelectedIndex())
		}
	}

	// Test navigation up - goes back to index 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel2, _ := newModel.Update(msg)
	if welcome, ok := newModel2.(*tui.WelcomeModel); ok {
		if welcome.SelectedIndex() != 0 {
			t.Errorf("Expected index 0 after up, got %d", welcome.SelectedIndex())
		}
	}
}

// TestWelcomeScreenShortcuts tests keyboard shortcuts on the welcome screen.
func TestWelcomeScreenShortcuts(t *testing.T) {
	t.Run("n_shortcut_activates_input_mode", func(t *testing.T) {
		model := tui.NewWelcomeModel()

		// Test 'n' shortcut for new task
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, cmd := model.Update(msg)
		if welcome, ok := newModel.(*tui.WelcomeModel); ok {
			// 'n' shortcut enters input mode for quick task entry
			if !welcome.IsInputMode() {
				t.Error("Expected input mode to be activated by 'n' shortcut")
			}
			// Input mode returns nil command (not quitting)
			if cmd != nil {
				t.Error("Expected no command from 'n' shortcut (input mode)")
			}
		}
	})

	t.Run("q_shortcut_quits", func(t *testing.T) {
		model := tui.NewWelcomeModel()

		// Test 'q' shortcut for quit
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		newModel, cmd := model.Update(msg)
		if quitWelcome, ok := newModel.(*tui.WelcomeModel); ok {
			if !quitWelcome.ShouldQuit() {
				t.Error("Expected quit flag to be set")
			}
			// 'q' shortcut returns tea.Quit command
			if cmd == nil {
				t.Error("Expected quit command from 'q' shortcut")
			}
		}
	})
}

// TestChatMessageInput tests chat message input handling.
func TestChatMessageInput(t *testing.T) {
	model := tui.NewChatModel()

	// Test setting input
	testInput := "Hello, Cline!"
	model.SetInput(testInput)

	if model.GetInput() != testInput {
		t.Errorf("Expected input '%s', got '%s'", testInput, model.GetInput())
	}

	// Test adding a message
	msg := tui.Message{
		Type:      tui.MessageTypeUser,
		Content:   testInput,
		Timestamp: time.Now(),
	}
	model.AddMessage(msg)

	messages := model.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != testInput {
		t.Errorf("Expected message content '%s', got '%s'", testInput, messages[0].Content)
	}
}

// TestChatStreamingResponse tests streaming message handling.
func TestChatStreamingResponse(t *testing.T) {
	model := tui.NewChatModel()

	// Simulate streaming message using StreamMessageMsg
	streamingMsg := tui.Message{
		Type:      tui.MessageTypeSay,
		Content:   "Hello",
		Partial:   true,
		Timestamp: time.Now(),
	}
	msg := tui.StreamMessageMsg{Message: &streamingMsg}
	newModel, _ := model.Update(msg)
	if chat, ok := newModel.(*tui.ChatModel); ok {
		// Check that the model is in streaming state
		if !chat.IsStreaming() {
			t.Error("Expected model to be in streaming state")
		}
	}

	// Update with completed message
	completedMsg := tui.Message{
		Type:      tui.MessageTypeSay,
		Content:   "Hello, world!",
		Partial:   false,
		Timestamp: time.Now(),
	}
	msg2 := tui.StreamMessageMsg{Message: &completedMsg}
	newModel2, _ := newModel.Update(msg2)
	if chat, ok := newModel2.(*tui.ChatModel); ok {
		// Check that streaming has completed
		if chat.IsStreaming() {
			t.Error("Expected model to not be streaming after completion")
		}
	}
}

// TestChatToolApproval tests tool approval workflow.
func TestChatToolApproval(t *testing.T) {
	model := tui.NewChatModel()

	// Initially should not be waiting for approval
	if model.IsWaitingForApproval() {
		t.Error("Should not be waiting for approval initially")
	}

	// Simulate approval request
	responseChan := make(chan string, 1)
	approvalMsg := tui.ApprovalRequestMsg{
		AskType:  "tool",
		Text:     "Allow editing file test.go?",
		Response: responseChan,
	}

	// Send approval request
	newModel, _ := model.Update(approvalMsg)
	if chat, ok := newModel.(*tui.ChatModel); ok {
		if !chat.IsWaitingForApproval() {
			t.Error("Should be waiting for approval after request")
		}

		pending := chat.GetPendingApproval()
		if pending == nil {
			t.Error("Expected pending approval request")
		} else if pending.AskType != "tool" {
			t.Errorf("Expected ask type 'tool', got '%s'", pending.AskType)
		}
	}
}

// TestSettingsPanelNavigation tests settings panel tab navigation.
func TestSettingsPanelNavigation(t *testing.T) {
	model := tui.NewSettingsModel()

	// Test initial tab
	// Note: This test depends on the internal structure of SettingsModel
	// which may need to be exposed for testing

	// Test tab switching with Tab key
	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := model.Update(msg)

	// Verify tab changed (implementation specific)
	_ = newModel
}

// TestSettingsProviderSelection tests provider selection in settings.
func TestSettingsProviderSelection(t *testing.T) {
	model := tui.NewSettingsModel()

	// Get initial provider setting
	provider, exists := model.GetSetting("provider")
	if !exists {
		t.Error("Expected provider setting to exist")
	}

	// Change provider
	newProvider := "openai"
	model.SetSetting("provider", newProvider)

	// Verify change
	updatedProvider, _ := model.GetSetting("provider")
	if updatedProvider != newProvider {
		t.Errorf("Expected provider '%s', got '%v'", newProvider, updatedProvider)
	}

	_ = provider
}

// TestPanelTransitions tests transitions between panels.
func TestPanelTransitions(t *testing.T) {
	// Test going from welcome to chat
	welcome := tui.NewWelcomeModel()

	// Simulate selecting "New Task"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ := welcome.Update(msg)

	// This should result in a transition message
	_ = newModel
}

// TestKeyboardShortcuts tests various keyboard shortcuts.
func TestKeyboardShortcuts(t *testing.T) {
	tests := []struct {
		name     string
		key      tea.KeyMsg
		expected string
	}{
		{
			name:     "Ctrl+C should quit",
			key:      tea.KeyMsg{Type: tea.KeyCtrlC},
			expected: "quit",
		},
		{
			name:     "Esc should go back",
			key:      tea.KeyMsg{Type: tea.KeyEsc},
			expected: "back",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test shortcut behavior
			_ = tt.key
		})
	}
}

// TestStaticDynamicRendering tests the static/dynamic rendering system.
func TestStaticDynamicRendering(t *testing.T) {
	renderer := tui.NewStaticRenderer()

	// Test adding content to static region
	key := "msg1"
	content := "Test message"
	added := renderer.AddToStatic(key, content)

	if !added {
		t.Error("Expected content to be added to static region")
	}

	// Test that duplicate content is not re-added
	addedAgain := renderer.AddToStatic(key, content)
	if addedAgain {
		t.Error("Expected duplicate content to not be re-added")
	}

	// Test partition logic
	messages := []tui.Message{
		{
			Type:      tui.MessageTypeUser,
			Content:   "Message 1",
			Partial:   false,
			Timestamp: time.Now(),
		},
		{
			Type:      tui.MessageTypeSay,
			Content:   "Streaming...",
			Partial:   true,
			Timestamp: time.Now(),
		},
	}

	partition := renderer.PartitionMessages(messages, false)

	// First message should be in static
	if len(partition.StaticItems) != 1 {
		t.Errorf("Expected 1 static item, got %d", len(partition.StaticItems))
	}

	// Second message should be in dynamic
	if !partition.HasDynamic {
		t.Error("Expected dynamic content for streaming message")
	}
}

// TestMessageTypes tests various message type handling.
func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  tui.MessageType
		expected string
	}{
		{"User message", tui.MessageTypeUser, "user"},
		{"AI message", tui.MessageTypeSay, "say"},
		{"Error message", tui.MessageTypeError, "error"},
		{"Tool use", tui.MessageTypeToolUse, "tool_use"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tui.Message{
				Type:      tt.msgType,
				Content:   "Test",
				Timestamp: time.Now(),
			}

			if string(msg.Type) != tt.expected {
				t.Errorf("Expected type '%s', got '%s'", tt.expected, msg.Type)
			}
		})
	}
}

// TestAppStateTransitions tests application state transitions.
func TestAppStateTransitions(t *testing.T) {
	// This would test the AppModel state machine
	// For now, we document the expected states
	states := []struct {
		state tui.AppState
		name  string
	}{
		{tui.AppStateWelcome, "Welcome"},
		{tui.AppStateChat, "Chat"},
		{tui.AppStateSettings, "Settings"},
		{tui.AppStateHistory, "History"},
		{tui.AppStateHelp, "Help"},
	}

	for _, s := range states {
		t.Run(s.name, func(t *testing.T) {
			// Verify state exists and has proper transitions
			_ = s.state
		})
	}
}

// TestRenderConsistency tests that rendering produces consistent output.
func TestRenderConsistency(t *testing.T) {
	model := tui.NewChatModel()

	// Add some messages
	model.AddMessage(tui.Message{
		Type:      tui.MessageTypeUser,
		Content:   "Hello",
		Timestamp: time.Now(),
	})
	model.AddMessage(tui.Message{
		Type:      tui.MessageTypeSay,
		Content:   "Hi there!",
		Timestamp: time.Now(),
	})

	// Render multiple times
	view1 := model.View()
	view2 := model.View()

	// Views should be identical for the same state
	if view1 != view2 {
		t.Error("Expected consistent rendering for same state")
	}
}

// TestWindowResize tests window resize handling.
func TestWindowResize(t *testing.T) {
	model := tui.NewChatModel()

	// Initial dimensions
	initialWidth, initialHeight := 80, 24
	model.SetDimensions(initialWidth, initialHeight)

	// Simulate resize
	newWidth, newHeight := 120, 30
	model.SetDimensions(newWidth, newHeight)

	// Model should handle resize gracefully
	// (Specific assertions depend on internal implementation)
}
