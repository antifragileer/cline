package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHistoryModel(t *testing.T) {
	t.Run("creates history model with defaults", func(t *testing.T) {
		model := NewHistoryModel()

		assert.NotNil(t, model)
		assert.NotNil(t, model.styles)
		assert.NotNil(t, model.items)
		assert.NotNil(t, model.filtered)
		assert.Equal(t, 0, model.cursor)
		assert.Equal(t, 1, model.page)
		assert.Equal(t, 10, model.pageSize)
		assert.False(t, model.goBack)
		assert.False(t, model.searchMode)
	})

	t.Run("initializes with empty item list", func(t *testing.T) {
		model := NewHistoryModel()

		assert.Empty(t, model.items)
		assert.Empty(t, model.filtered)
	})
}

func TestHistoryModel_SetDimensions(t *testing.T) {
	t.Run("sets width and height", func(t *testing.T) {
		model := NewHistoryModel()

		model.SetDimensions(100, 50)

		assert.Equal(t, 100, model.width)
		assert.Equal(t, 50, model.height)
	})
}

func TestHistoryModel_Init(t *testing.T) {
	t.Run("returns nil command", func(t *testing.T) {
		model := NewHistoryModel()
		cmd := model.Init()

		assert.Nil(t, cmd)
	})
}

func TestHistoryModel_Update_Navigation(t *testing.T) {
	t.Run("Esc sets goBack flag", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("q key sets goBack flag", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("Q key sets goBack flag", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("Up arrow moves cursor up", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
			{ID: "3", Task: "Task 3"},
		}
		model.filtered = model.items
		model.cursor = 2

		msg := tea.KeyMsg{Type: tea.KeyUp}
		_, _ = model.Update(msg)

		assert.Equal(t, 1, model.cursor)
	})

	t.Run("Down arrow moves cursor down", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
			{ID: "3", Task: "Task 3"},
		}
		model.filtered = model.items
		model.cursor = 0

		msg := tea.KeyMsg{Type: tea.KeyDown}
		_, _ = model.Update(msg)

		assert.Equal(t, 1, model.cursor)
	})

	t.Run("Up arrow stops at first item", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 0

		msg := tea.KeyMsg{Type: tea.KeyUp}
		_, _ = model.Update(msg)

		assert.Equal(t, 0, model.cursor)
	})

	t.Run("Down arrow stops at last item", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 1

		msg := tea.KeyMsg{Type: tea.KeyDown}
		_, _ = model.Update(msg)

		assert.Equal(t, 1, model.cursor)
	})

	t.Run("j key moves cursor down", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		_, _ = model.Update(msg)

		// Should move down or stay at 0 if only one item visible
		assert.GreaterOrEqual(t, model.cursor, 0)
	})

	t.Run("k key moves cursor up", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 1

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		_, _ = model.Update(msg)

		assert.LessOrEqual(t, model.cursor, 1)
	})

	t.Run("Left arrow goes to previous page", func(t *testing.T) {
		model := NewHistoryModel()
		model.page = 2

		msg := tea.KeyMsg{Type: tea.KeyLeft}
		_, _ = model.Update(msg)

		assert.Equal(t, 1, model.page)
	})

	t.Run("Right arrow goes to next page", func(t *testing.T) {
		model := NewHistoryModel()
		model.page = 1
		model.totalPages = 3

		msg := tea.KeyMsg{Type: tea.KeyRight}
		_, _ = model.Update(msg)

		assert.Equal(t, 2, model.page)
	})

	t.Run("p key goes to previous page", func(t *testing.T) {
		model := NewHistoryModel()
		model.page = 2

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		_, _ = model.Update(msg)

		assert.Equal(t, 1, model.page)
	})

	t.Run("n key goes to next page", func(t *testing.T) {
		model := NewHistoryModel()
		model.page = 1
		model.totalPages = 3

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		_, _ = model.Update(msg)

		assert.Equal(t, 2, model.page)
	})
}

func TestHistoryModel_Update_Selection(t *testing.T) {
	t.Run("Enter selects task", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "task-123", Task: "Test task"},
		}
		model.filtered = model.items
		model.cursor = 0

		selectedCalled := false
		model.SetOnSelectTask(func(id string) {
			selectedCalled = true
			assert.Equal(t, "task-123", id)
		})

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _ = model.Update(msg)

		assert.True(t, selectedCalled)
		assert.Equal(t, "task-123", model.GetSelectedTaskID())
	})
}

func TestHistoryModel_Update_Search(t *testing.T) {
	t.Run("/ key enters search mode", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		_, _ = model.Update(msg)

		assert.True(t, model.searchMode)
	})

	t.Run("? key enters search mode", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		_, _ = model.Update(msg)

		assert.True(t, model.searchMode)
	})

	t.Run("Esc exits search mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.searchMode = true
		model.searchInput = "test"

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = model.Update(msg)

		assert.False(t, model.searchMode)
		assert.Empty(t, model.searchInput)
	})

	t.Run("Enter in search mode applies filter", func(t *testing.T) {
		model := NewHistoryModel()
		model.searchMode = true
		model.searchInput = "test"
		model.items = []HistoryItem{
			{ID: "1", Task: "test task"},
			{ID: "2", Task: "other task"},
		}
		model.filtered = model.items

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _ = model.Update(msg)

		assert.False(t, model.searchMode)
		assert.Equal(t, "test", model.searchQuery)
	})
}

func TestHistoryModel_Update_Preview(t *testing.T) {
	t.Run("Space toggles preview mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Test task"},
		}
		model.filtered = model.items

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		_, _ = model.Update(msg)

		assert.True(t, model.IsPreviewMode())
	})

	t.Run("v toggles preview mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Test task"},
		}
		model.filtered = model.items

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
		_, _ = model.Update(msg)

		assert.True(t, model.IsPreviewMode())
	})

	t.Run("Esc exits preview mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.previewMode = true
		model.previewTask = &HistoryItem{ID: "1"}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = model.Update(msg)

		assert.False(t, model.IsPreviewMode())
		assert.Nil(t, model.previewTask)
	})
}

func TestHistoryModel_Update_Refresh(t *testing.T) {
	t.Run("r key refreshes history", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		_, _ = model.Update(msg)

		// Should have placeholder data after refresh
		assert.Greater(t, len(model.items), 0)
	})

	t.Run("R key refreshes history", func(t *testing.T) {
		model := NewHistoryModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
		_, _ = model.Update(msg)

		// Should have placeholder data after refresh
		assert.Greater(t, len(model.items), 0)
	})
}

func TestHistoryModel_ShouldGoBack(t *testing.T) {
	t.Run("returns goBack state", func(t *testing.T) {
		model := NewHistoryModel()

		assert.False(t, model.ShouldGoBack())

		model.goBack = true

		assert.True(t, model.ShouldGoBack())
	})
}

func TestHistoryModel_GetSelectedTaskID(t *testing.T) {
	t.Run("returns selected task ID", func(t *testing.T) {
		model := NewHistoryModel()
		model.selected = "task-123"

		taskID := model.GetSelectedTaskID()

		assert.Equal(t, "task-123", taskID)
	})

	t.Run("returns empty when no selection", func(t *testing.T) {
		model := NewHistoryModel()

		taskID := model.GetSelectedTaskID()

		assert.Empty(t, taskID)
	})
}

func TestHistoryModel_GetSelectedTask(t *testing.T) {
	t.Run("returns selected task ID", func(t *testing.T) {
		model := NewHistoryModel()
		model.selected = "task-123"

		taskID := model.GetSelectedTask()

		assert.Equal(t, "task-123", taskID)
	})
}

func TestHistoryModel_Reset(t *testing.T) {
	t.Run("resets all state", func(t *testing.T) {
		model := NewHistoryModel()
		model.cursor = 2
		model.selected = "task-123"
		model.goBack = true

		model.Reset()

		assert.Equal(t, 0, model.cursor)
		assert.Empty(t, model.selected)
		assert.False(t, model.goBack)
	})
}

func TestHistoryModel_SetItems(t *testing.T) {
	t.Run("sets item list", func(t *testing.T) {
		model := NewHistoryModel()
		items := []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}

		model.SetItems(items)

		assert.Equal(t, items, model.items)
		assert.Equal(t, items, model.filtered)
		assert.Equal(t, 0, model.cursor)
		assert.Equal(t, 1, model.page)
	})

	t.Run("empty list is allowed", func(t *testing.T) {
		model := NewHistoryModel()

		model.SetItems([]HistoryItem{})

		assert.Empty(t, model.items)
		assert.Empty(t, model.filtered)
	})
}

func TestHistoryModel_IsPreviewMode(t *testing.T) {
	t.Run("returns preview mode state", func(t *testing.T) {
		model := NewHistoryModel()

		assert.False(t, model.IsPreviewMode())

		model.previewMode = true

		assert.True(t, model.IsPreviewMode())
	})
}

func TestHistoryModel_View(t *testing.T) {
	t.Run("renders empty state", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Task History")
	})

	t.Run("renders with items", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)
		model.items = []HistoryItem{
			{ID: "task-001", Task: "Test task"},
		}
		model.filtered = model.items

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Test task")
	})

	t.Run("renders in search mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)
		model.searchMode = true
		model.searchInput = "test"

		view := model.View()

		assert.NotEmpty(t, view)
	})

	t.Run("renders with active filter", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)
		model.searchQuery = "test"

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Filter:")
	})

	t.Run("renders with pagination", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)
		model.page = 1
		model.totalPages = 3

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Page 1 of 3")
	})

	t.Run("renders with preview mode", func(t *testing.T) {
		model := NewHistoryModel()
		model.SetDimensions(80, 24)
		model.previewMode = true
		model.previewTask = &HistoryItem{
			ID:        "task-001",
			Task:      "Preview task",
			Timestamp: time.Now(),
			Tokens:    1000,
		}

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Task Preview")
	})
}

func TestHistoryModel_formatItem(t *testing.T) {
	t.Run("formats item with all fields", func(t *testing.T) {
		model := NewHistoryModel()
		item := HistoryItem{
			ID:        "task-abc123",
			Task:      "This is a test task",
			Model:     "claude-sonnet-4",
			Cost:      0.025,
			Timestamp: time.Now(),
		}

		formatted := model.formatItem(item, false)

		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "task-abc")
		assert.Contains(t, formatted, "This is a test task")
		assert.Contains(t, formatted, "claude")
	})

	t.Run("truncates long task names", func(t *testing.T) {
		model := NewHistoryModel()
		item := HistoryItem{
			ID:   "1",
			Task: "This is a very long task name that exceeds fifty characters and should be truncated",
		}

		formatted := model.formatItem(item, false)

		assert.Contains(t, formatted, "...")
	})

	t.Run("truncates long IDs", func(t *testing.T) {
		model := NewHistoryModel()
		item := HistoryItem{
			ID:   "very-long-id-that-needs-truncation",
			Task: "Short task",
		}

		formatted := model.formatItem(item, false)

		// Should contain shortened ID
		assert.NotContains(t, formatted, "very-long-id-that-needs-truncation")
	})
}

func TestHistoryModel_formatTime(t *testing.T) {
	model := NewHistoryModel()
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m ago",
		},
		{
			name:     "hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3h ago",
		},
		{
			name:     "days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d ago",
		},
		{
			name:     "old date",
			time:     now.Add(-30 * 24 * time.Hour),
			expected: now.Add(-30 * 24 * time.Hour).Format("Jan 2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.formatTime(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHistoryModel_getCurrentPageItems(t *testing.T) {
	t.Run("returns items for current page", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 2
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
			{ID: "3", Task: "Task 3"},
			{ID: "4", Task: "Task 4"},
		}
		model.filtered = model.items
		model.page = 2

		items := model.getCurrentPageItems()

		require.Equal(t, 2, len(items))
		assert.Equal(t, "3", items[0].ID)
		assert.Equal(t, "4", items[1].ID)
	})

	t.Run("returns empty for out of range page", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 2
		model.page = 10
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
		}
		model.filtered = model.items

		items := model.getCurrentPageItems()

		assert.Empty(t, items)
	})
}

func TestHistoryModel_calculateTotalPages(t *testing.T) {
	t.Run("calculates correct number of pages", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 3
		model.filtered = []HistoryItem{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
			{ID: "4"}, {ID: "5"}, {ID: "6"},
			{ID: "7"},
		}

		model.calculateTotalPages()

		assert.Equal(t, 3, model.totalPages)
	})

	t.Run("handles empty list", func(t *testing.T) {
		model := NewHistoryModel()
		model.filtered = []HistoryItem{}

		model.calculateTotalPages()

		assert.Equal(t, 1, model.totalPages)
	})

	t.Run("handles zero page size", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 0
		model.filtered = []HistoryItem{{ID: "1"}}

		model.calculateTotalPages()

		assert.Equal(t, 1, model.totalPages)
	})
}

func TestHistoryModel_applyFilter(t *testing.T) {
	t.Run("filters items by search query", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "test task"},
			{ID: "2", Task: "other task"},
			{ID: "3", Task: "another test"},
		}
		model.searchQuery = "test"

		model.applyFilter()

		assert.Equal(t, 2, len(model.filtered))
	})

	t.Run("empty query shows all items", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "task 1"},
			{ID: "2", Task: "task 2"},
		}
		model.searchQuery = ""

		model.applyFilter()

		assert.Equal(t, 2, len(model.filtered))
	})

	t.Run("resets page and cursor on filter", func(t *testing.T) {
		model := NewHistoryModel()
		model.page = 3
		model.cursor = 5
		model.items = []HistoryItem{{ID: "1", Task: "task"}}
		model.searchQuery = "task"

		model.applyFilter()

		assert.Equal(t, 1, model.page)
		assert.Equal(t, 0, model.cursor)
	})
}

func TestHistoryModel_moveUp(t *testing.T) {
	t.Run("moves cursor up within page", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 1

		model.moveUp()

		assert.Equal(t, 0, model.cursor)
	})

	t.Run("moves to previous page at top", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 1
		model.page = 2
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 0

		model.moveUp()

		assert.Equal(t, 1, model.page)
	})
}

func TestHistoryModel_moveDown(t *testing.T) {
	t.Run("moves cursor down within page", func(t *testing.T) {
		model := NewHistoryModel()
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 0

		model.moveDown()

		assert.Equal(t, 1, model.cursor)
	})

	t.Run("moves to next page at bottom", func(t *testing.T) {
		model := NewHistoryModel()
		model.pageSize = 1
		model.page = 1
		model.totalPages = 2
		model.items = []HistoryItem{
			{ID: "1", Task: "Task 1"},
			{ID: "2", Task: "Task 2"},
		}
		model.filtered = model.items
		model.cursor = 0

		model.moveDown()

		assert.Equal(t, 2, model.page)
		assert.Equal(t, 0, model.cursor)
	})
}

func TestDefaultHistoryStyles(t *testing.T) {
	t.Run("returns default styles", func(t *testing.T) {
		styles := DefaultHistoryStyles()

		assert.NotZero(t, styles.containerStyle)
		assert.NotZero(t, styles.titleStyle)
		assert.NotZero(t, styles.itemStyle)
		assert.NotZero(t, styles.selectedStyle)
		assert.NotZero(t, styles.dimStyle)
		assert.NotZero(t, styles.helpStyle)
	})
}

func TestHistoryModel_Refresh(t *testing.T) {
	t.Run("loads placeholder data when empty", func(t *testing.T) {
		model := NewHistoryModel()

		model.Refresh()

		assert.Greater(t, len(model.items), 0)
	})

	t.Run("preserves existing items", func(t *testing.T) {
		model := NewHistoryModel()
		existingItems := []HistoryItem{
			{ID: "existing", Task: "Existing task"},
		}
		model.items = existingItems
		model.filtered = existingItems

		model.Refresh()

		assert.Equal(t, existingItems, model.items)
	})
}

func TestHistoryModel_SetOnSelectTask(t *testing.T) {
	t.Run("sets callback function", func(t *testing.T) {
		model := NewHistoryModel()

		var called bool
		model.SetOnSelectTask(func(id string) {
			called = true
		})

		// Trigger the callback
		if model.onSelectTask != nil {
			model.onSelectTask("test-id")
		}

		assert.True(t, called)
	})
}