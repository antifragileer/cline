// Package tui provides terminal UI components for user input handling.
package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInputModel(t *testing.T) {
	t.Run("creates single line input", func(t *testing.T) {
		m := NewInputModel(SingleLineInput, "Test Title")
		require.NotNil(t, m)
		assert.Equal(t, SingleLineInput, m.mode)
		assert.Equal(t, "Test Title", m.title)
		assert.NotNil(t, m.textInput)
		assert.NotNil(t, m.history)
		assert.NotNil(t, m.keyMap)
	})

	t.Run("creates multi line input", func(t *testing.T) {
		m := NewInputModel(MultiLineInput, "Test Title")
		require.NotNil(t, m)
		assert.Equal(t, MultiLineInput, m.mode)
		assert.Equal(t, "Test Title", m.title)
		assert.NotNil(t, m.textArea)
		assert.NotNil(t, m.history)
	})

	t.Run("single line input has focus", func(t *testing.T) {
		m := NewInputModel(SingleLineInput, "Test")
		assert.True(t, m.textInput.Focused())
	})

	t.Run("multi line input has focus", func(t *testing.T) {
		m := NewInputModel(MultiLineInput, "Test")
		assert.True(t, m.textArea.Focused())
	})
}

func TestNewSingleLineInput(t *testing.T) {
	m := NewSingleLineInput("Title", "Enter text here")
	require.NotNil(t, m)
	assert.Equal(t, SingleLineInput, m.mode)
	assert.Equal(t, "Title", m.title)
	assert.Equal(t, "Enter text here", m.placeholder)
	assert.Equal(t, "Enter text here", m.textInput.Placeholder)
}

func TestNewMultiLineInput(t *testing.T) {
	m := NewMultiLineInput("Title", "Enter multiline text")
	require.NotNil(t, m)
	assert.Equal(t, MultiLineInput, m.mode)
	assert.Equal(t, "Title", m.title)
	assert.Equal(t, "Enter multiline text", m.placeholder)
	assert.Equal(t, "Enter multiline text", m.textArea.Placeholder)
}

func TestInputModel_SetHistory(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	originalHistory := m.history

	newHistory := NewHistory(50)
	m.SetHistory(newHistory)

	assert.Equal(t, newHistory, m.history)
	assert.NotEqual(t, originalHistory, m.history)
}

func TestInputModel_GetHistory(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	history := m.GetHistory()
	assert.NotNil(t, history)
	assert.Equal(t, 0, history.Len())
}

func TestInputModel_AddToHistory(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")

	// Add valid entry
	m.AddToHistory("entry1")
	assert.Equal(t, 1, m.history.Len())
	assert.True(t, m.history.Contains("entry1"))

	// Don't add empty entry
	m.AddToHistory("")
	assert.Equal(t, 1, m.history.Len())

	// Don't add duplicate of last entry
	m.AddToHistory("entry1")
	assert.Equal(t, 1, m.history.Len())
}

func TestInputModel_SetWidth(t *testing.T) {
	t.Run("sets width for single line input", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.SetWidth(80)
		assert.Equal(t, 80, m.width)
		assert.Equal(t, 80, m.textInput.Width)
	})

	t.Run("sets width for multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.SetWidth(120)
		assert.Equal(t, 120, m.width)
		// textarea width is set via SetWidth method
	})
}

func TestInputModel_SetHeight(t *testing.T) {
	t.Run("sets height for multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.SetHeight(20)
		assert.Equal(t, 20, m.height)
	})

	t.Run("single line ignores height", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.SetHeight(10)
		assert.Equal(t, 10, m.height)
		// Single line input doesn't have height property
	})
}

func TestInputModel_GetValue(t *testing.T) {
	t.Run("gets value from single line input", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.SetValue("test value")
		assert.Equal(t, "test value", m.GetValue())
	})

	t.Run("gets value from multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.SetValue("line1\nline2")
		assert.Equal(t, "line1\nline2", m.GetValue())
	})
}

func TestInputModel_SetValue(t *testing.T) {
	t.Run("sets value for single line input", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.SetValue("new value")
		assert.Equal(t, "new value", m.textInput.Value())
	})

	t.Run("sets value for multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.SetValue("multiline\nvalue")
		assert.Equal(t, "multiline\nvalue", m.textArea.Value())
	})
}

func TestInputModel_Focus(t *testing.T) {
	t.Run("focus single line input", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.textInput.Blur()
		assert.False(t, m.textInput.Focused())
		cmd := m.Focus()
		assert.NotNil(t, cmd)
		assert.True(t, m.textInput.Focused())
	})

	t.Run("focus multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.textArea.Blur()
		assert.False(t, m.textArea.Focused())
		cmd := m.Focus()
		assert.NotNil(t, cmd)
		assert.True(t, m.textArea.Focused())
	})
}

func TestInputModel_Blur(t *testing.T) {
	t.Run("blur single line input", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.Focus()
		assert.True(t, m.textInput.Focused())
		m.Blur()
		assert.False(t, m.textInput.Focused())
	})

	t.Run("blur multi line input", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		m.Focus()
		assert.True(t, m.textArea.Focused())
		m.Blur()
		assert.False(t, m.textArea.Focused())
	})
}

func TestInputModel_Reset(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	m.SetValue("some value")
	m.result = "result"
	m.submitted = true
	m.quitting = true
	m.err = assert.AnError

	m.Reset()

	assert.Equal(t, "", m.GetValue())
	assert.Equal(t, "", m.result)
	assert.False(t, m.submitted)
	assert.False(t, m.quitting)
	assert.Nil(t, m.err)
}

func TestInputModel_IsSubmitted(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	assert.False(t, m.IsSubmitted())

	m.submitted = true
	assert.True(t, m.IsSubmitted())
}

func TestInputModel_IsQuitting(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	assert.False(t, m.IsQuitting())

	m.quitting = true
	assert.True(t, m.IsQuitting())
}

func TestInputModel_Result(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	m.result = "test result"
	m.submitted = true

	result := m.Result()
	assert.Equal(t, "test result", result.Value)
	assert.True(t, result.Ok)
	assert.NotNil(t, result.History)
}

func TestInputModel_Error(t *testing.T) {
	m := NewSingleLineInput("Test", "placeholder")
	assert.Nil(t, m.Error())

	testErr := assert.AnError
	m.err = testErr
	assert.Equal(t, testErr, m.Error())
}

func TestInputModel_View(t *testing.T) {
	t.Run("returns empty when quitting", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.quitting = true
		assert.Equal(t, "", m.View())
	})

	t.Run("returns view with title", func(t *testing.T) {
		m := NewSingleLineInput("Test Title", "placeholder")
		view := m.View()
		assert.Contains(t, view, "Test Title")
	})

	t.Run("returns view with help text", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		view := m.View()
		assert.Contains(t, view, "enter")
		assert.Contains(t, view, "submit")
		assert.Contains(t, view, "ctrl+c")
		assert.Contains(t, view, "quit")
	})

	t.Run("returns view with error", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		m.err = assert.AnError
		view := m.View()
		assert.Contains(t, view, "Error:")
	})
}

func TestDefaultInputKeyMap(t *testing.T) {
	km := DefaultInputKeyMap()

	// Verify all expected bindings exist
	assert.NotNil(t, km.Submit)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Approve)
	assert.NotNil(t, km.Reject)
	assert.NotNil(t, km.Up)
	assert.NotNil(t, km.Down)
	assert.NotNil(t, km.History)

	// Verify specific keys
	assert.Contains(t, km.Submit.Keys(), "enter")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Approve.Keys(), "ctrl+a")
	assert.Contains(t, km.Approve.Keys(), "y")
	assert.Contains(t, km.Reject.Keys(), "ctrl+r")
	assert.Contains(t, km.Reject.Keys(), "n")
	assert.Contains(t, km.Up.Keys(), "up")
	assert.Contains(t, km.Down.Keys(), "down")
	assert.Contains(t, km.History.Keys(), "ctrl+p")
	assert.Contains(t, km.History.Keys(), "ctrl+n")
}

func TestInputModel_Init(t *testing.T) {
	t.Run("single line returns blink command", func(t *testing.T) {
		m := NewSingleLineInput("Test", "placeholder")
		cmd := m.Init()
		assert.NotNil(t, cmd)
	})

	t.Run("multi line returns blink command", func(t *testing.T) {
		m := NewMultiLineInput("Test", "placeholder")
		cmd := m.Init()
		assert.NotNil(t, cmd)
	})
}

func TestInputResult(t *testing.T) {
	result := InputResult{
		Value:   "test value",
		Ok:      true,
		History: []string{"history1", "history2"},
	}

	assert.Equal(t, "test value", result.Value)
	assert.True(t, result.Ok)
	assert.Equal(t, 2, len(result.History))
}

func TestNewPromptModel(t *testing.T) {
	t.Run("creates yes/no prompt", func(t *testing.T) {
		m := NewPromptModel(YesNoPrompt, "Title", "Message")
		require.NotNil(t, m)
		assert.Equal(t, YesNoPrompt, m.promptType)
		assert.Equal(t, "Title", m.title)
		assert.Equal(t, "Message", m.message)
		assert.Equal(t, 0, m.selected)
	})

	t.Run("creates approval prompt", func(t *testing.T) {
		m := NewPromptModel(ApprovalPrompt, "Title", "Message")
		require.NotNil(t, m)
		assert.Equal(t, ApprovalPrompt, m.promptType)
	})

	t.Run("creates choice prompt", func(t *testing.T) {
		m := NewPromptModel(ChoicePrompt, "Title", "Message")
		require.NotNil(t, m)
		assert.Equal(t, ChoicePrompt, m.promptType)
	})
}

func TestNewYesNoPrompt(t *testing.T) {
	m := NewYesNoPrompt("Confirm", "Are you sure?")
	require.NotNil(t, m)
	assert.Equal(t, YesNoPrompt, m.promptType)
	assert.Equal(t, "Confirm", m.title)
	assert.Equal(t, "Are you sure?", m.message)
}

func TestNewApprovalPrompt(t *testing.T) {
	m := NewApprovalPrompt("Approval", "Approve changes?", "Detailed info")
	require.NotNil(t, m)
	assert.Equal(t, ApprovalPrompt, m.promptType)
	assert.Equal(t, "Detailed info", m.details)
}

func TestNewChoicePrompt(t *testing.T) {
	choices := []string{"Option 1", "Option 2", "Option 3"}
	m := NewChoicePrompt("Select", "Choose one:", choices)
	require.NotNil(t, m)
	assert.Equal(t, ChoicePrompt, m.promptType)
	assert.Equal(t, choices, m.choices)
	assert.Equal(t, 3, len(m.choices))
}

func TestPromptModel_SetWidth(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	m.SetWidth(80)
	assert.Equal(t, 80, m.width)
}

func TestPromptModel_SetHeight(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	m.SetHeight(20)
	assert.Equal(t, 20, m.height)
}

func TestPromptModel_Result(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	m.result = true
	m.submitted = true

	result := m.Result()
	assert.True(t, result.Approved)
	assert.True(t, result.Ok)

	m.result = false
	result = m.Result()
	assert.False(t, result.Approved)
}

func TestPromptModel_IsApproved(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	assert.False(t, m.IsApproved())

	m.result = true
	assert.False(t, m.IsApproved()) // Not submitted yet

	m.submitted = true
	assert.True(t, m.IsApproved())
}

func TestPromptModel_IsRejected(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	assert.False(t, m.IsRejected())

	m.result = false
	assert.False(t, m.IsRejected()) // Not submitted yet

	m.submitted = true
	assert.True(t, m.IsRejected())
}

func TestPromptModel_IsSubmitted(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	assert.False(t, m.IsSubmitted())

	m.submitted = true
	assert.True(t, m.IsSubmitted())
}

func TestPromptModel_IsQuitting(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	assert.False(t, m.IsQuitting())

	m.quitting = true
	assert.True(t, m.IsQuitting())
}

func TestPromptModel_SelectedChoice(t *testing.T) {
	choices := []string{"A", "B", "C"}
	m := NewChoicePrompt("Test", "Select", choices)

	assert.Equal(t, 0, m.SelectedChoice())

	m.SetSelectedChoice(1)
	assert.Equal(t, 1, m.SelectedChoice())

	// Out of bounds should not change
	m.SetSelectedChoice(10)
	assert.Equal(t, 1, m.SelectedChoice())

	m.SetSelectedChoice(-1)
	assert.Equal(t, 1, m.SelectedChoice())
}

func TestPromptModel_View(t *testing.T) {
	t.Run("returns empty when quitting without submit", func(t *testing.T) {
		m := NewYesNoPrompt("Test", "Message")
		m.quitting = true
		m.submitted = false
		assert.Equal(t, "", m.View())
	})

	t.Run("shows title and message", func(t *testing.T) {
		m := NewYesNoPrompt("Confirm", "Are you sure?")
		view := m.View()
		assert.Contains(t, view, "Confirm")
		assert.Contains(t, view, "Are you sure?")
	})

	t.Run("shows details when toggled", func(t *testing.T) {
		m := NewApprovalPrompt("Test", "Message", "Extra details")
		m.showDetails = true
		view := m.View()
		assert.Contains(t, view, "Extra details")
	})

	t.Run("shows buttons for yes/no prompt", func(t *testing.T) {
		m := NewYesNoPrompt("Test", "Message")
		view := m.View()
		assert.Contains(t, view, "[Y]es")
		assert.Contains(t, view, "[N]o")
	})

	t.Run("shows choices for choice prompt", func(t *testing.T) {
		m := NewChoicePrompt("Test", "Select", []string{"A", "B"})
		view := m.View()
		assert.Contains(t, view, "A")
		assert.Contains(t, view, "B")
	})

	t.Run("shows help text", func(t *testing.T) {
		m := NewYesNoPrompt("Test", "Message")
		view := m.View()
		assert.Contains(t, view, "ctrl+a")
		assert.Contains(t, view, "ctrl+r")
	})
}

func TestPromptResult(t *testing.T) {
	result := PromptResult{
		Approved: true,
		Ok:       true,
	}

	assert.True(t, result.Approved)
	assert.True(t, result.Ok)
}

func TestPromptModel_Init(t *testing.T) {
	m := NewYesNoPrompt("Test", "Message")
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestNewHistory(t *testing.T) {
	h := NewHistory(100)
	require.NotNil(t, h)
	assert.Equal(t, 100, h.maxSize)
	assert.Equal(t, 0, h.Len())
	assert.Equal(t, -1, h.Position())
	assert.NotNil(t, h.entries)
}

func TestNewHistoryWithFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("creates new history with file path", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "test-history.json")
		h, err := NewHistoryWithFile(50, filePath)
		require.NoError(t, err)
		assert.Equal(t, 50, h.MaxSize())
		assert.Equal(t, filePath, h.GetFilePath())
	})

	t.Run("loads existing history from file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing-history.json")

		// Create file with existing data
		existingData := []string{"entry1", "entry2", "entry3"}
		data, _ := marshalJSON(existingData)
		err := os.WriteFile(filePath, data, 0600)
		require.NoError(t, err)

		h, err := NewHistoryWithFile(50, filePath)
		require.NoError(t, err)
		assert.Equal(t, 3, h.Len())
		assert.True(t, h.Contains("entry1"))
		assert.True(t, h.Contains("entry2"))
		assert.True(t, h.Contains("entry3"))
	})
}

func TestHistory_Add(t *testing.T) {
	h := NewHistory(5)

	t.Run("adds entries", func(t *testing.T) {
		h.Add("entry1")
		assert.Equal(t, 1, h.Len())
		assert.True(t, h.Contains("entry1"))
	})

	t.Run("does not add empty entries", func(t *testing.T) {
		initialLen := h.Len()
		h.Add("")
		assert.Equal(t, initialLen, h.Len())
	})

	t.Run("does not add duplicate of last entry", func(t *testing.T) {
		h.Add("unique")
		h.Add("unique")
		// Count occurrences
		count := 0
		for i := 0; i < h.Len(); i++ {
			entry, _ := h.Get(i)
			if entry == "unique" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("removes oldest when exceeding max size", func(t *testing.T) {
		h.Clear()
		h.Add("1")
		h.Add("2")
		h.Add("3")
		h.Add("4")
		h.Add("5")
		h.Add("6") // This should push out "1"

		assert.Equal(t, 5, h.Len())
		assert.False(t, h.Contains("1"))
		assert.True(t, h.Contains("6"))
	})

	t.Run("resets position after add", func(t *testing.T) {
		h.Clear()
		h.Add("a")
		h.Add("b")
		h.Add("c")

		// Navigate to previous
		h.Previous()
		h.Previous()
		assert.Equal(t, 1, h.Position())

		// Add should reset position
		h.Add("d")
		assert.Equal(t, -1, h.Position())
	})
}

func TestHistory_Previous(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	t.Run("returns previous entries", func(t *testing.T) {
		assert.Equal(t, "third", h.Previous())
		assert.Equal(t, "second", h.Previous())
		assert.Equal(t, "first", h.Previous())
	})

	t.Run("stops at oldest entry", func(t *testing.T) {
		// Already at first from previous test
		assert.Equal(t, "first", h.Previous())
		assert.Equal(t, "first", h.Previous())
	})

	t.Run("returns empty for empty history", func(t *testing.T) {
		emptyH := NewHistory(10)
		assert.Equal(t, "", emptyH.Previous())
	})
}

func TestHistory_Next(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	t.Run("navigates forward through history", func(t *testing.T) {
		// Go back first
		h.Previous()
		h.Previous()
		h.Previous()

		// Now go forward
		assert.Equal(t, "second", h.Next())
		assert.Equal(t, "third", h.Next())
	})

	t.Run("returns empty at end and resets position", func(t *testing.T) {
		// Already at third
		assert.Equal(t, "", h.Next())
		assert.Equal(t, -1, h.Position())
	})

	t.Run("returns empty for empty history", func(t *testing.T) {
		emptyH := NewHistory(10)
		assert.Equal(t, "", emptyH.Next())
	})
}

func TestHistory_Get(t *testing.T) {
	h := NewHistory(10)
	h.Add("a")
	h.Add("b")
	h.Add("c")

	t.Run("gets entry by index", func(t *testing.T) {
		entry, ok := h.Get(0)
		assert.True(t, ok)
		assert.Equal(t, "a", entry)

		entry, ok = h.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "b", entry)

		entry, ok = h.Get(2)
		assert.True(t, ok)
		assert.Equal(t, "c", entry)
	})

	t.Run("returns false for invalid index", func(t *testing.T) {
		_, ok := h.Get(-1)
		assert.False(t, ok)

		_, ok = h.Get(100)
		assert.False(t, ok)
	})
}

func TestHistory_GetAll(t *testing.T) {
	h := NewHistory(10)
	h.Add("x")
	h.Add("y")
	h.Add("z")

	entries := h.GetAll()
	assert.Equal(t, 3, len(entries))
	assert.Equal(t, "x", entries[0])
	assert.Equal(t, "y", entries[1])
	assert.Equal(t, "z", entries[2])

	// Verify it's a copy
	entries[0] = "modified"
	entry, _ := h.Get(0)
	assert.Equal(t, "x", entry)
}

func TestHistory_Len(t *testing.T) {
	h := NewHistory(10)
	assert.Equal(t, 0, h.Len())

	h.Add("1")
	assert.Equal(t, 1, h.Len())

	h.Add("2")
	h.Add("3")
	assert.Equal(t, 3, h.Len())
}

func TestHistory_SetMaxSize(t *testing.T) {
	h := NewHistory(10)
	h.Add("1")
	h.Add("2")
	h.Add("3")
	h.Add("4")
	h.Add("5")

	t.Run("reduces max size and removes oldest", func(t *testing.T) {
		h.SetMaxSize(3)
		assert.Equal(t, 3, h.MaxSize())
		assert.Equal(t, 3, h.Len())
		// Oldest (1, 2) should be removed
		assert.False(t, h.Contains("1"))
		assert.False(t, h.Contains("2"))
		assert.True(t, h.Contains("3"))
		assert.True(t, h.Contains("4"))
		assert.True(t, h.Contains("5"))
	})

	t.Run("increases max size", func(t *testing.T) {
		h.SetMaxSize(10)
		assert.Equal(t, 10, h.MaxSize())
		// Existing entries preserved
		assert.Equal(t, 3, h.Len())
	})
}

func TestHistory_Clear(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("clears entries", func(t *testing.T) {
		h := NewHistory(10)
		h.Add("a")
		h.Add("b")

		h.Clear()
		assert.Equal(t, 0, h.Len())
		assert.Equal(t, -1, h.Position())
	})

	t.Run("removes file when clearing", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "clear-test.json")
		h, _ := NewHistoryWithFile(10, filePath)
		h.Add("test")
		h.Save()

		_, err := os.Stat(filePath)
		assert.NoError(t, err)

		h.Clear()

		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestHistory_Search(t *testing.T) {
	h := NewHistory(10)
	h.Add("hello world")
	h.Add("goodbye world")
	h.Add("hello there")
	h.Add("something else")

	t.Run("finds matching entries", func(t *testing.T) {
		matches := h.Search("hello")
		assert.Equal(t, 2, len(matches))
		assert.Contains(t, matches, "hello world")
		assert.Contains(t, matches, "hello there")
	})

	t.Run("returns all for empty query", func(t *testing.T) {
		matches := h.Search("")
		assert.Equal(t, 4, len(matches))
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		matches := h.Search("xyz")
		assert.Equal(t, 0, len(matches))
	})
}

func TestHistory_Contains(t *testing.T) {
	h := NewHistory(10)
	h.Add("test")

	assert.True(t, h.Contains("test"))
	assert.False(t, h.Contains("not present"))
}

func TestHistory_Remove(t *testing.T) {
	h := NewHistory(10)
	h.Add("a")
	h.Add("b")
	h.Add("c")

	t.Run("removes entry at index", func(t *testing.T) {
		ok := h.Remove(1)
		assert.True(t, ok)
		assert.Equal(t, 2, h.Len())
		assert.False(t, h.Contains("b"))
	})

	t.Run("returns false for invalid index", func(t *testing.T) {
		ok := h.Remove(100)
		assert.False(t, ok)

		ok = h.Remove(-1)
		assert.False(t, ok)
	})
}

func TestHistory_SaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("saves and loads history", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "save-load.json")

		h1 := NewHistory(10)
		h1.SetFilePath(filePath)
		h1.Add("entry1")
		h1.Add("entry2")

		err := h1.Save()
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Load in new history
		h2 := NewHistory(10)
		h2.SetFilePath(filePath)
		err = h2.Load()
		require.NoError(t, err)

		assert.Equal(t, 2, h2.Len())
		assert.True(t, h2.Contains("entry1"))
		assert.True(t, h2.Contains("entry2"))
	})

	t.Run("returns error for no file path", func(t *testing.T) {
		h := NewHistory(10)
		h.Add("test")

		err := h.Save()
		assert.NoError(t, err) // No error, just no-op

		err = h.Load()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no file path configured")
	})

	t.Run("handles non-existent file gracefully", func(t *testing.T) {
		h := NewHistory(10)
		h.SetFilePath(filepath.Join(tempDir, "non-existent.json"))

		err := h.Load()
		assert.NoError(t, err)
		assert.Equal(t, 0, h.Len())
	})
}

func TestHistory_Position(t *testing.T) {
	h := NewHistory(10)
	assert.Equal(t, -1, h.Position())

	h.Add("a")
	h.Add("b")

	h.Previous()
	assert.Equal(t, 0, h.Position())

	h.Previous()
	assert.Equal(t, 1, h.Position())

	h.ResetPosition()
	assert.Equal(t, -1, h.Position())
}

func TestHistory_GetFilePath(t *testing.T) {
	h := NewHistory(10)
	assert.Equal(t, "", h.GetFilePath())

	h.SetFilePath("/some/path")
	assert.Equal(t, "/some/path", h.GetFilePath())
}

// Helper function to marshal JSON
func marshalJSON(v interface{}) ([]byte, error) {
	// Simple JSON marshaling for test data
	// In real code, use encoding/json
	return []byte(`["entry1","entry2","entry3"]`), nil
}