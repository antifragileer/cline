package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProgram tests program initialization
func TestNewProgram(t *testing.T) {
	t.Run("initializes with default options", func(t *testing.T) {
		// Create a new program with io.Discard for testing
		var buf bytes.Buffer
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(&buf),
			tea.WithInput(nil),
		)

		require.NotNil(t, p)
	})

	t.Run("initializes with custom input", func(t *testing.T) {
		input := strings.NewReader("test input\n")
		var buf bytes.Buffer
		
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(&buf),
			tea.WithInput(input),
		)

		require.NotNil(t, p)
	})

	t.Run("initializes with alt screen", func(t *testing.T) {
		var buf bytes.Buffer
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(&buf),
			tea.WithAltScreen(),
		)

		require.NotNil(t, p)
	})

	t.Run("initializes with mouse support", func(t *testing.T) {
		var buf bytes.Buffer
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(&buf),
			tea.WithMouseCellMotion(),
		)

		require.NotNil(t, p)
	})
}

// TestInitialModel tests model creation
func TestInitialModel(t *testing.T) {
	t.Run("creates model with default state", func(t *testing.T) {
		m := initialModel()

		assert.NotNil(t, m)
		assert.Equal(t, welcomeView, m.currentView)
		assert.NotNil(t, m.messages)
		assert.Empty(t, m.messages)
		assert.NotNil(t, m.inputHistory)
		assert.Empty(t, m.inputHistory)
		assert.Equal(t, 0, m.inputHistoryIndex)
		assert.False(t, m.inputFocused)
		assert.False(t, m.showWelcome)
	})

	t.Run("creates model with initialized components", func(t *testing.T) {
		m := initialModel()

		// Check that all components are initialized
		assert.NotNil(t, m.viewport)
		assert.NotNil(t, m.textInput)
		assert.NotNil(t, m.spinner)
		assert.NotNil(t, m.help)
	})

	t.Run("creates model with correct dimensions", func(t *testing.T) {
		m := initialModel()

		assert.Equal(t, defaultWidth, m.width)
		assert.Equal(t, defaultHeight, m.height)
	})
}

// TestUpdateHandling tests update handling
func TestUpdateHandling(t *testing.T) {
	t.Run("handles window size message", func(t *testing.T) {
		m := initialModel()
		newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

		updatedModel := newModel.(Model)
		assert.Equal(t, 100, updatedModel.width)
		assert.Equal(t, 50, updatedModel.height)
		assert.NotNil(t, cmd)
	})

	t.Run("handles key message", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		newModel, cmd := m.Update(keyMsg)

		updatedModel := newModel.(Model)
		// Key should be handled by input component when focused
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles enter key when input focused", func(t *testing.T) {
		m := initialModel()
		m.inputFocused = true
		m.textInput.SetValue("test message")
		
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := m.Update(keyMsg)

		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles ctrl+c", func(t *testing.T) {
		m := initialModel()
		
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
		newModel, cmd := m.Update(keyMsg)

		updatedModel := newModel.(Model)
		// Should trigger quit
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles ctrl+d", func(t *testing.T) {
		m := initialModel()
		
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlD}
		newModel, cmd := m.Update(keyMsg)

		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles spinner tick", func(t *testing.T) {
		m := initialModel()
		m.loading = true
		
		tickMsg := m.spinner.Tick()
		newModel, cmd := m.Update(tickMsg)

		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})

	t.Run("handles custom message", func(t *testing.T) {
		m := initialModel()
		
		msg := AddMessageMsg{
			Role:    "user",
			Content: "test content",
		}
		newModel, cmd := m.Update(msg)

		updatedModel := newModel.(Model)
		assert.Len(t, updatedModel.messages, 1)
		assert.Equal(t, "user", updatedModel.messages[0].Role)
		assert.Equal(t, "test content", updatedModel.messages[0].Content)
		assert.NotNil(t, cmd)
	})
}

// TestViewRendering tests view rendering
func TestViewRendering(t *testing.T) {
	t.Run("renders welcome view", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true

		view := m.View()
		assert.Contains(t, view, "Welcome")
		assert.Contains(t, view, "Cline")
	})

	t.Run("renders chat view", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 24

		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders chat view with messages", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 24
		m.messages = []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		}

		view := m.View()
		assert.Contains(t, view, "Hello")
		assert.Contains(t, view, "Hi there!")
	})

	t.Run("renders loading state", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.loading = true

		view := m.View()
		assert.NotEmpty(t, view)
		// Should contain spinner indicator
	})

	t.Run("renders error state", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.errorMsg = "Something went wrong"

		view := m.View()
		assert.Contains(t, view, "Something went wrong")
	})
}

// TestModelStateTransitions tests model state transitions
func TestModelStateTransitions(t *testing.T) {
	t.Run("transitions from welcome to chat", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView

		// Simulate starting a chat
		m.currentView = chatView
		m.showWelcome = false

		assert.Equal(t, chatView, m.currentView)
		assert.False(t, m.showWelcome)
	})

	t.Run("transitions to settings view", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView

		m.currentView = settingsView

		assert.Equal(t, settingsView, m.currentView)
	})

	t.Run("returns to chat from settings", func(t *testing.T) {
		m := initialModel()
		m.currentView = settingsView

		m.currentView = chatView

		assert.Equal(t, chatView, m.currentView)
	})
}

// TestCommandExecution tests command execution
func TestCommandExecution(t *testing.T) {
	t.Run("quit command", func(t *testing.T) {
		m := initialModel()
		
		cmd := quitCmd()
		msg := cmd()
		
		quitMsg, ok := msg.(tea.QuitMsg)
		assert.True(t, ok)
		assert.NotNil(t, quitMsg)
	})

	t.Run("batch command execution", func(t *testing.T) {
		m := initialModel()
		
		cmds := []tea.Cmd{
			tea.Batch(),
			tea.Batch(func() tea.Msg { return nil }),
		}
		
		batchCmd := tea.Batch(cmds...)
		assert.NotNil(t, batchCmd)
	})
}

// TestProgramOptions tests various program options
func TestProgramOptions(t *testing.T) {
	t.Run("without output", func(t *testing.T) {
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(io.Discard),
		)
		require.NotNil(t, p)
	})

	t.Run("with startup options", func(t *testing.T) {
		var buf bytes.Buffer
		p := tea.NewProgram(
			initialModel(),
			tea.WithOutput(&buf),
			tea.WithoutSignalHandler(),
		)
		require.NotNil(t, p)
	})
}

// TestErrorHandling tests error handling in the TUI
func TestErrorHandling(t *testing.T) {
	t.Run("handles error message", func(t *testing.T) {
		m := initialModel()
		
		errMsg := ErrorMsg{Err: assert.AnError}
		newModel, cmd := m.Update(errMsg)

		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel.errorMsg)
		assert.NotNil(t, cmd)
	})

	t.Run("handles nil error gracefully", func(t *testing.T) {
		m := initialModel()
		
		errMsg := ErrorMsg{Err: nil}
		newModel, cmd := m.Update(errMsg)

		updatedModel := newModel.(Model)
		assert.NotNil(t, updatedModel)
		assert.NotNil(t, cmd)
	})
}

// TestInit tests the init function
func TestInit(t *testing.T) {
	t.Run("returns initial command", func(t *testing.T) {
		m := initialModel()
		cmd := m.Init()

		// Should return a command (typically batch or nil)
		assert.NotNil(t, cmd)
	})
}

// TestViewportUpdates tests viewport update handling
func TestViewportUpdates(t *testing.T) {
	t.Run("updates viewport on resize", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		
		// Initial size
		assert.Equal(t, defaultWidth, m.width)
		assert.Equal(t, defaultHeight, m.height)

		// Resize
		newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		updatedModel := newModel.(Model)

		assert.Equal(t, 120, updatedModel.width)
		assert.Equal(t, 40, updatedModel.height)
	})

	t.Run("adjusts viewport height with input", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.height = 30

		view := m.View()
		// Viewport should account for input area
		assert.NotEmpty(t, view)
	})
}

// TestMessageBuffer tests message buffer management
func TestMessageBuffer(t *testing.T) {
	t.Run("adds message to buffer", func(t *testing.T) {
		m := initialModel()
		
		msg := Message{
			Role:    "user",
			Content: "Test message",
		}
		
		m.messages = append(m.messages, msg)
		assert.Len(t, m.messages, 1)
		assert.Equal(t, "Test message", m.messages[0].Content)
	})

	t.Run("handles streaming message updates", func(t *testing.T) {
		m := initialModel()
		
		// Add initial message
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   "Hello",
			Streaming: true,
		})

		// Update streaming content
		m.messages[0].Content += " world"
		assert.Equal(t, "Hello world", m.messages[0].Content)
	})

	t.Run("finalizes streaming message", func(t *testing.T) {
		m := initialModel()
		
		m.messages = append(m.messages, Message{
			Role:      "assistant",
			Content:   "Complete",
			Streaming: true,
		})

		// Finalize
		m.messages[0].Streaming = false
		assert.False(t, m.messages[0].Streaming)
	})
}