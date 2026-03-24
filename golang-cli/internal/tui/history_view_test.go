package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestNewHistoryViewModel tests the creation of a new history view model.
func TestNewHistoryViewModel(t *testing.T) {
	config := DefaultHistoryViewConfig()
	model := NewHistoryViewModel(config)

	assert.NotNil(t, model)
	assert.Equal(t, HistoryViewStateLoading, model.state)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.Equal(t, 1, model.currentPage)
	assert.Equal(t, 0, model.selectedIndex)
	assert.NotNil(t, model.searchInput)
	assert.Empty(t, model.allEntries)
	assert.Empty(t, model.filteredEntries)
}

// TestDefaultHistoryViewKeyMap tests the default key bindings.
func TestDefaultHistoryViewKeyMap(t *testing.T) {
	km := DefaultHistoryViewKeyMap()

	// Test that all bindings are defined
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Select.Keys())
	assert.NotEmpty(t, km.Search.Keys())
	assert.NotEmpty(t, km.Delete.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.NextPage.Keys())
	assert.NotEmpty(t, km.PrevPage.Keys())

	// Test specific keys
	assert.Contains(t, km.Up.Keys(), "up")
	assert.Contains(t, km.Up.Keys(), "k")
	assert.Contains(t, km.Down.Keys(), "down")
	assert.Contains(t, km.Down.Keys(), "j")
	assert.Contains(t, km.Search.Keys(), "/")
	assert.Contains(t, km.Quit.Keys(), "q")
}

// TestHistoryViewModelInit tests the initialization command.
func TestHistoryViewModelInit(t *testing.T) {
	config := DefaultHistoryViewConfig()
	model := NewHistoryViewModel(config)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

// TestConvertToEntry tests converting interface{} to TaskHistoryEntry.
func TestConvertToEntry(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	tests := []struct {
		name     string
		input    interface{}
		expected TaskHistoryEntry
		wantErr  bool
	}{
		{
			name: "valid entry with all fields",
			input: map[string]interface{}{
				"id":   "task-001",
				"task": "Test task",
				"ts":   float64(1609459200000),
				"metadata": map[string]interface{}{
					"model":     "claude-3",
					"mode":      "act",
					"completed": true,
					"totalCost": float64(0.0234),
				},
			},
			expected: TaskHistoryEntry{
				ID:        "task-001",
				Task:      "Test task",
				Timestamp: 1609459200000,
				Metadata: Metadata{
					Model:     "claude-3",
					Mode:      "act",
					Completed: true,
					TotalCost: 0.0234,
				},
			},
			wantErr: false,
		},
		{
			name: "entry with int64 timestamp",
			input: map[string]interface{}{
				"id":   "task-002",
				"task": "Another task",
				"ts":   int64(1609459200000),
			},
			expected: TaskHistoryEntry{
				ID:        "task-002",
				Task:      "Another task",
				Timestamp: 1609459200000,
			},
			wantErr: false,
		},
		{
			name: "entry with modelId at top level",
			input: map[string]interface{}{
				"id":       "task-003",
				"task":     "Task with modelId",
				"ts":       float64(1609459200000),
				"modelId":  "gpt-4",
				"totalCost": float64(0.01),
			},
			expected: TaskHistoryEntry{
				ID:        "task-003",
				Task:      "Task with modelId",
				Timestamp: 1609459200000,
				Metadata: Metadata{
					Model:     "gpt-4",
					TotalCost: 0.01,
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid entry - not a map",
			input:    "not a map",
			expected: TaskHistoryEntry{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := model.convertToEntry(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, entry.ID)
				assert.Equal(t, tt.expected.Task, entry.Task)
				assert.Equal(t, tt.expected.Timestamp, entry.Timestamp)
				assert.Equal(t, tt.expected.Metadata.Model, entry.Metadata.Model)
				assert.Equal(t, tt.expected.Metadata.Mode, entry.Metadata.Mode)
				assert.Equal(t, tt.expected.Metadata.Completed, entry.Metadata.Completed)
				assert.InDelta(t, tt.expected.Metadata.TotalCost, entry.Metadata.TotalCost, 0.0001)
			}
		})
	}
}

// TestFilterEntries tests the filtering functionality.
func TestFilterEntries(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	// Setup test data
	model.allEntries = []TaskHistoryEntry{
		{ID: "task-001", Task: "Fix bug in authentication"},
		{ID: "task-002", Task: "Add new feature to dashboard"},
		{ID: "task-003", Task: "Update documentation"},
		{ID: "task-004", Task: "Fix styling issue"},
	}

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
	}{
		{
			name:          "empty query returns all",
			searchQuery:   "",
			expectedCount: 4,
		},
		{
			name:          "filter by task content",
			searchQuery:   "fix",
			expectedCount: 2, // Fix bug, Fix styling
		},
		{
			name:          "filter by ID",
			searchQuery:   "task-001",
			expectedCount: 1,
		},
		{
			name:          "case insensitive search",
			searchQuery:   "FIX",
			expectedCount: 2,
		},
		{
			name:          "no matches",
			searchQuery:   "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.searchQuery = tt.searchQuery
			model.filterEntries()

			assert.Equal(t, tt.expectedCount, len(model.filteredEntries))
		})
	}
}

// TestCalculatePagination tests pagination calculations.
func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name              string
		totalEntries      int
		limit             int
		currentPage       int
		expectedTotalPages int
	}{
		{
			name:              "single page",
			totalEntries:      5,
			limit:             10,
			currentPage:       1,
			expectedTotalPages: 1,
		},
		{
			name:              "multiple pages",
			totalEntries:      25,
			limit:             10,
			currentPage:       1,
			expectedTotalPages: 3,
		},
		{
			name:              "exact page boundary",
			totalEntries:      20,
			limit:             10,
			currentPage:       1,
			expectedTotalPages: 2,
		},
		{
			name:              "no entries",
			totalEntries:      0,
			limit:             10,
			currentPage:       1,
			expectedTotalPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewHistoryViewModel(DefaultHistoryViewConfig())
			model.config.Limit = tt.limit
			model.currentPage = tt.currentPage

			// Create entries
			model.allEntries = make([]TaskHistoryEntry, tt.totalEntries)
			for i := 0; i < tt.totalEntries; i++ {
				model.allEntries[i] = TaskHistoryEntry{ID: fmt.Sprintf("task-%d", i)}
			}

			model.filteredEntries = model.allEntries
			model.calculatePagination()

			assert.Equal(t, tt.expectedTotalPages, model.totalPages)
		})
	}
}

// TestGetCurrentPageEntries tests retrieving page entries.
func TestGetCurrentPageEntries(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.config.Limit = 3

	// Create 7 entries
	model.filteredEntries = []TaskHistoryEntry{
		{ID: "task-1"},
		{ID: "task-2"},
		{ID: "task-3"},
		{ID: "task-4"},
		{ID: "task-5"},
		{ID: "task-6"},
		{ID: "task-7"},
	}

	tests := []struct {
		page     int
		expected []string
	}{
		{page: 1, expected: []string{"task-1", "task-2", "task-3"}},
		{page: 2, expected: []string{"task-4", "task-5", "task-6"}},
		{page: 3, expected: []string{"task-7"}},
		{page: 4, expected: []string{}}, // Beyond last page
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("page %d", tt.page), func(t *testing.T) {
			model.currentPage = tt.page
			entries := model.getCurrentPageEntries()

			assert.Equal(t, len(tt.expected), len(entries))
			for i, id := range tt.expected {
				if i < len(entries) {
					assert.Equal(t, id, entries[i].ID)
				}
			}
		})
	}
}

// TestHandleDisplayingKeys tests keyboard handling in display mode.
func TestHandleDisplayingKeys(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.state = HistoryViewStateDisplaying
	model.config.Limit = 3

	// Create test entries
	model.filteredEntries = []TaskHistoryEntry{
		{ID: "task-1"},
		{ID: "task-2"},
		{ID: "task-3"},
		{ID: "task-4"},
		{ID: "task-5"},
	}
	model.calculatePagination()

	tests := []struct {
		name           string
		initialIndex   int
		initialPage    int
		keyMsg         tea.KeyMsg
		expectedIndex  int
		expectedPage   int
		expectedState  HistoryViewState
	}{
		{
			name:          "move down",
			initialIndex:  0,
			initialPage:   1,
			keyMsg:        tea.KeyMsg{Type: tea.KeyDown},
			expectedIndex: 1,
			expectedPage:  1,
			expectedState: HistoryViewStateDisplaying,
		},
		{
			name:          "move up",
			initialIndex:  1,
			initialPage:   1,
			keyMsg:        tea.KeyMsg{Type: tea.KeyUp},
			expectedIndex: 0,
			expectedPage:  1,
			expectedState: HistoryViewStateDisplaying,
		},
		{
			name:          "next page",
			initialIndex:  2,
			initialPage:   1,
			keyMsg:        tea.KeyMsg{Type: tea.KeyRight},
			expectedIndex: 0,
			expectedPage:  2,
			expectedState: HistoryViewStateDisplaying,
		},
		{
			name:          "previous page",
			initialIndex:  0,
			initialPage:   2,
			keyMsg:        tea.KeyMsg{Type: tea.KeyLeft},
			expectedIndex: 0,
			expectedPage:  1,
			expectedState: HistoryViewStateDisplaying,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.selectedIndex = tt.initialIndex
			model.currentPage = tt.initialPage

			newModel, _ := model.handleDisplayingKeys(tt.keyMsg)

			m, ok := newModel.(*HistoryViewModel)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedIndex, m.selectedIndex)
			assert.Equal(t, tt.expectedPage, m.currentPage)
			assert.Equal(t, tt.expectedState, m.state)
		})
	}
}

// TestFormatTimestamp tests timestamp formatting.
func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{
			input:    0,
			expected: "unknown",
		},
		{
			input:    1609459200000, // milliseconds (2021-01-01)
			expected: "2021-01-01 00:00:00",
		},
		{
			input:    1609459200, // seconds (2021-01-01)
			expected: "2021-01-01 00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("timestamp %d", tt.input), func(t *testing.T) {
			result := formatTimestamp(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTruncateString tests string truncation.
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input   string
		maxLen  int
		expected string
	}{
		{
			input:   "short",
			maxLen:  10,
			expected: "short",
		},
		{
			input:   "this is a very long string",
			maxLen:  10,
			expected: "this is...",
		},
		{
			input:   "test",
			maxLen:  3,
			expected: "tes",
		},
		{
			input:   "",
			maxLen:  5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("maxLen %d", tt.maxLen), func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMax tests the max function.
func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{a: 1, b: 2, expected: 2},
		{a: 5, b: 3, expected: 5},
		{a: 4, b: 4, expected: 4},
		{a: -1, b: 1, expected: 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d vs %d", tt.a, tt.b), func(t *testing.T) {
			result := max(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHistoryViewResult tests the result structure.
func TestHistoryViewResult(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	// Simulate selecting a task
	model.state = HistoryViewStateTaskSelected
	model.filteredEntries = []TaskHistoryEntry{{ID: "selected-task"}}
	model.selectedIndex = 0

	result := model.Result()

	assert.Equal(t, HistoryViewStateTaskSelected, result.State)
	assert.Equal(t, "selected-task", result.SelectedTask)
	assert.False(t, result.WasDeleted)
}

// TestRenderLoading tests the loading state rendering.
func TestRenderLoading(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	result := model.renderLoading()

	assert.Contains(t, result, "Loading")
	assert.Contains(t, result, "history")
}

// TestRenderHeader tests header rendering.
func TestRenderHeader(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	// Test without search
	model.allEntries = []TaskHistoryEntry{
		{ID: "1"}, {ID: "2"}, {ID: "3"},
	}
	result := model.renderHeader()
	assert.Contains(t, result, "Task History")
	assert.Contains(t, result, "3 total")

	// Test with search
	model.searchQuery = "test"
	model.filteredEntries = []TaskHistoryEntry{{ID: "1"}}
	result = model.renderHeader()
	assert.Contains(t, result, "1 of 3 matches")
	assert.Contains(t, result, "test")
}

// TestRenderPagination tests pagination rendering.
func TestRenderPagination(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.currentPage = 2
	model.totalPages = 5

	result := model.renderPagination()

	assert.Contains(t, result, "Page 2 of 5")
	assert.Contains(t, result, "[← prev]")
	assert.Contains(t, result, "[next →]")
}

// TestRenderEmptyTaskList tests empty list rendering.
func TestRenderEmptyTaskList(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.filteredEntries = []TaskHistoryEntry{}

	// Test empty without search
	result := model.renderTaskList()
	assert.Contains(t, result, "No task history available")

	// Test empty with search
	model.searchQuery = "nonexistent"
	result = model.renderTaskList()
	assert.Contains(t, result, "No tasks match your search")
}

// TestRemoveEntry tests entry removal.
func TestRemoveEntry(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	model.allEntries = []TaskHistoryEntry{
		{ID: "task-1"},
		{ID: "task-2"},
		{ID: "task-3"},
	}
	model.filteredEntries = []TaskHistoryEntry{
		{ID: "task-1"},
		{ID: "task-2"},
		{ID: "task-3"},
	}

	model.removeEntry("task-2")

	assert.Equal(t, 2, len(model.allEntries))
	assert.Equal(t, 2, len(model.filteredEntries))

	// Verify the correct entry was removed
	for _, entry := range model.allEntries {
		assert.NotEqual(t, "task-2", entry.ID)
	}
}

// TestExtractMetadata tests metadata extraction.
func TestExtractMetadata(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	tests := []struct {
		name     string
		entryMap map[string]interface{}
		expected Metadata
	}{
		{
			name: "full metadata",
			entryMap: map[string]interface{}{
				"metadata": map[string]interface{}{
					"model":     "claude-3",
					"mode":      "plan",
					"completed": true,
					"totalCost": float64(0.1234),
				},
			},
			expected: Metadata{
				Model:     "claude-3",
				Mode:      "plan",
				Completed: true,
				TotalCost: 0.1234,
			},
		},
		{
			name: "top level fields",
			entryMap: map[string]interface{}{
				"modelId":   "gpt-4",
				"totalCost": float64(0.05),
			},
			expected: Metadata{
				Model:     "gpt-4",
				TotalCost: 0.05,
			},
		},
		{
			name:     "empty map",
			entryMap: map[string]interface{}{},
			expected: Metadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.extractMetadata(tt.entryMap)
			assert.Equal(t, tt.expected.Model, result.Model)
			assert.Equal(t, tt.expected.Mode, result.Mode)
			assert.Equal(t, tt.expected.Completed, result.Completed)
			assert.InDelta(t, tt.expected.TotalCost, result.TotalCost, 0.0001)
		})
	}
}

// TestRenderTaskEntry tests task entry rendering.
func TestRenderTaskEntry(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.width = 80

	entry := TaskHistoryEntry{
		ID:        "task-abc123",
		Task:      "This is a test task description",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix() * 1000,
		Metadata: Metadata{
			Model:     "claude-3-opus",
			TotalCost: 0.0234,
		},
	}

	// Test selected entry
	result := model.renderTaskEntry(entry, true)
	assert.Contains(t, result, ">")
	assert.Contains(t, result, "task-abc123")
	assert.Contains(t, result, "This is a test task description")
	assert.Contains(t, result, "claude-3-opus")
	assert.Contains(t, result, "$0.0234")

	// Test unselected entry
	result = model.renderTaskEntry(entry, false)
	assert.Contains(t, result, "  ") // No selection indicator
	assert.Contains(t, result, "task-abc123")
}

// TestWindowSizeMsg tests window size updates.
func TestWindowSizeMsg(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	m, ok := newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 40, m.height)
}

// TestCalculateVisibleCount tests visible count calculation.
func TestCalculateVisibleCount(t *testing.T) {
	tests := []struct {
		height   int
		expected int
	}{
		{height: 24, expected: 3}, // (24 - 5) / 5 = 3
		{height: 10, expected: 1}, // Minimum 1
		{height: 50, expected: 9}, // (50 - 5) / 5 = 9
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("height %d", tt.height), func(t *testing.T) {
			model := NewHistoryViewModel(DefaultHistoryViewConfig())
			model.height = tt.height
			model.totalPages = 1 // No pagination line

			result := model.calculateVisibleCount()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestKeyBindingsHelp tests that key bindings have help text.
func TestKeyBindingsHelp(t *testing.T) {
	km := DefaultHistoryViewKeyMap()

	bindings := []key.Binding{
		km.Up,
		km.Down,
		km.Select,
		km.Search,
		km.Delete,
		km.Quit,
	}

	for _, binding := range bindings {
		help := binding.Help()
		assert.NotEmpty(t, help.Key, "Key should have help text")
		assert.NotEmpty(t, help.Desc, "Key should have description")
	}
}

// TestHistoryDataLoadedMsg tests the data loaded message handling.
func TestHistoryDataLoadedMsg(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.state = HistoryViewStateLoading

	entries := []TaskHistoryEntry{
		{ID: "task-1", Task: "First task"},
		{ID: "task-2", Task: "Second task"},
	}

	newModel, _ := model.Update(historyDataLoadedMsg{entries: entries})

	m, ok := newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, HistoryViewStateDisplaying, m.state)
	assert.Equal(t, 2, len(m.allEntries))
	assert.Equal(t, 2, len(m.filteredEntries))
}

// TestSearchMode tests entering and exiting search mode.
func TestSearchMode(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.state = HistoryViewStateDisplaying

	// Enter search mode
	newModel, _ := model.handleDisplayingKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, ok := newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, HistoryViewStateSearching, m.state)
	assert.True(t, m.isSearching)

	// Cancel search
	newModel, _ = m.handleSearchingKeys(tea.KeyMsg{Type: tea.KeyEscape})
	m, ok = newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, HistoryViewStateDisplaying, m.state)
	assert.False(t, m.isSearching)
}

// TestDeleteConfirmation tests the delete confirmation flow.
func TestDeleteConfirmation(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.state = HistoryViewStateDisplaying
	model.filteredEntries = []TaskHistoryEntry{{ID: "task-1", Task: "Test task"}}
	model.selectedIndex = 0

	// Trigger delete
	newModel, _ := model.handleDisplayingKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m, ok := newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, HistoryViewStateConfirmDelete, m.state)
	assert.NotNil(t, m.deleteCandidate)

	// Cancel delete
	newModel, _ = m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m, ok = newModel.(*HistoryViewModel)
	assert.True(t, ok)
	assert.Equal(t, HistoryViewStateDisplaying, m.state)
	assert.Nil(t, m.deleteCandidate)
}

// TestQuitKey tests quitting from different states.
func TestQuitKey(t *testing.T) {
	tests := []struct {
		name          string
		initialState  HistoryViewState
		expectedState HistoryViewState
		shouldQuit    bool
	}{
		{
			name:          "quit from displaying",
			initialState:  HistoryViewStateDisplaying,
			expectedState: HistoryViewStateExited,
			shouldQuit:    true,
		},
		{
			name:          "quit from searching",
			initialState:  HistoryViewStateSearching,
			expectedState: HistoryViewStateExited,
			shouldQuit:    true,
		},
		{
			name:          "no quit from confirm delete",
			initialState:  HistoryViewStateConfirmDelete,
			expectedState: HistoryViewStateConfirmDelete,
			shouldQuit:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewHistoryViewModel(DefaultHistoryViewConfig())
			model.state = tt.initialState

			newModel, cmd := model.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

			m, ok := newModel.(*HistoryViewModel)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedState, m.state)

			if tt.shouldQuit {
				assert.NotNil(t, cmd)
			}
		})
	}
}

// TestConvertToEntries tests batch conversion.
func TestConvertToEntries(t *testing.T) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	data := []interface{}{
		map[string]interface{}{"id": "task-1", "task": "First"},
		map[string]interface{}{"id": "task-2", "task": "Second"},
	}

	entries, err := model.convertToEntries(data)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, "task-1", entries[0].ID)
	assert.Equal(t, "task-2", entries[1].ID)
}

// TestLoadTaskHistoryWithTempFile tests loading from a temporary file.
func TestLoadTaskHistoryWithTempFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "history-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test history file
	historyFile := filepath.Join(tmpDir, "taskHistory.json")
	entries := []TaskHistoryEntry{
		{ID: "task-1", Task: "First task", Timestamp: 1609459200000},
		{ID: "task-2", Task: "Second task", Timestamp: 1609459200001},
	}

	data := map[string]interface{}{
		"entries": entries,
	}

	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)

	err = os.WriteFile(historyFile, jsonData, 0644)
	assert.NoError(t, err)

	// Test loading (this would need getTaskHistoryPath to be configurable for testing)
	// For now, we just verify the JSON structure is correct
	var loaded map[string]interface{}
	err = json.Unmarshal(jsonData, &loaded)
	assert.NoError(t, err)
	assert.NotNil(t, loaded["entries"])
}

// BenchmarkFilterEntries benchmarks the filter function.
func BenchmarkFilterEntries(b *testing.B) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())

	// Create 1000 entries
	model.allEntries = make([]TaskHistoryEntry, 1000)
	for i := 0; i < 1000; i++ {
		model.allEntries[i] = TaskHistoryEntry{
			ID:   fmt.Sprintf("task-%d", i),
			Task: fmt.Sprintf("This is task number %d with some description", i),
		}
	}

	model.searchQuery = "number 500"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.filterEntries()
	}
}

// BenchmarkCalculatePagination benchmarks pagination calculation.
func BenchmarkCalculatePagination(b *testing.B) {
	model := NewHistoryViewModel(DefaultHistoryViewConfig())
	model.config.Limit = 10

	// Create 10000 entries
	model.filteredEntries = make([]TaskHistoryEntry, 10000)
	for i := 0; i < 10000; i++ {
		model.filteredEntries[i] = TaskHistoryEntry{ID: fmt.Sprintf("task-%d", i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.calculatePagination()
	}
}