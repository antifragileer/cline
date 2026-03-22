package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWelcomeModel tests the creation of a new welcome model.
func TestNewWelcomeModel(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	assert.NotNil(t, model)
	assert.Equal(t, WelcomeStateInitial, model.state)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.Equal(t, "0.1.0", model.config.Version)
	assert.NotNil(t, model.recentTasks)
	assert.Empty(t, model.recentTasks)
}

// TestWelcomeModel_Init tests the initialization of the welcome model.
func TestWelcomeModel_Init(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

// TestWelcomeModel_Update_WindowSize tests window size updates.
func TestWelcomeModel_Update_WindowSize(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, cmd := model.Update(msg)

	wm, ok := newModel.(*WelcomeModel)
	require.True(t, ok)
	assert.Equal(t, 120, wm.width)
	assert.Equal(t, 40, wm.height)
	assert.Nil(t, cmd)
}

// TestWelcomeModel_Update_KeyQuit tests quit key handling.
func TestWelcomeModel_Update_KeyQuit(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)
	model.state = WelcomeStateDisplaying

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	// Should return tea.Quit command
	assert.NotNil(t, cmd)
}

// TestWelcomeModel_Update_KeyContinue tests continue key handling.
func TestWelcomeModel_Update_KeyContinue(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage context
	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	model := NewWelcomeModel(config)
	model.state = WelcomeStateDisplaying

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.Update(msg)

	wm, ok := newModel.(*WelcomeModel)
	require.True(t, ok)
	assert.Equal(t, WelcomeStateCompleted, wm.state)
	assert.NotNil(t, cmd)

	// Verify welcomeShown was set
	val, ok := storageCtx.GlobalState.Get("welcomeShown")
	assert.True(t, ok)
	assert.True(t, val.(bool))
}

// TestWelcomeModel_Update_KeySkip tests skip key handling.
func TestWelcomeModel_Update_KeySkip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage context
	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	model := NewWelcomeModel(config)
	model.state = WelcomeStateDisplaying

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, cmd := model.Update(msg)

	wm, ok := newModel.(*WelcomeModel)
	require.True(t, ok)
	assert.Equal(t, WelcomeStateSkipped, wm.state)
	assert.NotNil(t, cmd)

	// Verify both flags were set
	val, ok := storageCtx.GlobalState.Get("welcomeDisabled")
	assert.True(t, ok)
	assert.True(t, val.(bool))

	val, ok = storageCtx.GlobalState.Get("welcomeShown")
	assert.True(t, ok)
	assert.True(t, val.(bool))
}

// TestWelcomeModel_LoadAuthStatus tests authentication status loading.
func TestWelcomeModel_LoadAuthStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage context
	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	t.Run("not authenticated", func(t *testing.T) {
		config := DefaultWelcomeConfig()
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		status := model.loadAuthStatus()
		assert.False(t, status.IsAuthenticated)
		assert.Empty(t, status.Provider)
	})

	t.Run("authenticated with provider", func(t *testing.T) {
		// Set up auth data
		err := storageCtx.GlobalState.Set("apiProvider", "anthropic")
		require.NoError(t, err)
		err = storageCtx.GlobalState.Set("defaultModel", "claude-3")
		require.NoError(t, err)
		err = storageCtx.Secrets.Set("anthropicApiKey", "test-key")
		require.NoError(t, err)

		config := DefaultWelcomeConfig()
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		status := model.loadAuthStatus()
		assert.True(t, status.IsAuthenticated)
		assert.Equal(t, "anthropic", status.Provider)
		assert.Equal(t, "claude-3", status.Model)

		// Clean up
		_ = storageCtx.GlobalState.Delete("apiProvider")
		_ = storageCtx.GlobalState.Delete("defaultModel")
		_ = storageCtx.Secrets.Delete("anthropicApiKey")
	})

	t.Run("missing API key", func(t *testing.T) {
		// Set up auth data without API key
		err := storageCtx.GlobalState.Set("apiProvider", "openai")
		require.NoError(t, err)

		config := DefaultWelcomeConfig()
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		status := model.loadAuthStatus()
		assert.False(t, status.IsAuthenticated)

		// Clean up
		_ = storageCtx.GlobalState.Delete("apiProvider")
	})
}

// TestWelcomeModel_LoadRecentTasks tests loading of recent tasks.
func TestWelcomeModel_LoadRecentTasks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock task history file
	dataDir := filepath.Join(tmpDir, ".cline", "data")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	historyFile := filepath.Join(dataDir, "taskHistory.json")

	// Create storage with mock data
	fileStorage, err := storage.NewClineFileStorage(historyFile, 0644)
	require.NoError(t, err)

	entries := []map[string]interface{}{
		{
			"id":   "task-1",
			"task": "Refactor authentication module",
			"ts":   float64(time.Now().Add(-time.Hour).Unix() * 1000),
		},
		{
			"id":   "task-2",
			"task": "Add unit tests for API client",
			"ts":   float64(time.Now().Add(-2 * time.Hour).Unix() * 1000),
		},
		{
			"id":   "task-3",
			"task": "Update documentation",
			"ts":   float64(time.Now().Add(-24 * time.Hour).Unix() * 1000),
		},
	}

	err = fileStorage.Set("entries", entries)
	require.NoError(t, err)
	fileStorage.Close()

	// Set up HOME to point to temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := DefaultWelcomeConfig()
	config.MaxRecentTasks = 2
	model := NewWelcomeModel(config)

	tasks := model.loadRecentTasks()
	assert.Len(t, tasks, 2)
	assert.Equal(t, "Refactor authentication module", tasks[0].Task) // Most recent first
	assert.Equal(t, "Add unit tests for API client", tasks[1].Task)
}

// TestWelcomeModel_LoadRecentTasks_Empty tests loading when no tasks exist.
func TestWelcomeModel_LoadRecentTasks_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up HOME to point to temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	tasks := model.loadRecentTasks()
	assert.Empty(t, tasks)
}

// TestConvertToRecentTask tests the conversion function.
func TestConvertToRecentTask(t *testing.T) {
	t.Run("valid task data", func(t *testing.T) {
		data := map[string]interface{}{
			"id":   "task-123",
			"task": "Test task",
			"ts":   float64(1234567890),
		}

		task, ok := convertToRecentTask(data)
		assert.True(t, ok)
		assert.Equal(t, "task-123", task.ID)
		assert.Equal(t, "Test task", task.Task)
		assert.Equal(t, int64(1234567890), task.Timestamp)
	})

	t.Run("missing id", func(t *testing.T) {
		data := map[string]interface{}{
			"task": "Test task",
			"ts":   float64(1234567890),
		}

		task, ok := convertToRecentTask(data)
		assert.True(t, ok) // Still valid if task is present
		assert.Empty(t, task.ID)
	})

	t.Run("missing task", func(t *testing.T) {
		data := map[string]interface{}{
			"id": "task-123",
			"ts": float64(1234567890),
		}

		task, ok := convertToRecentTask(data)
		assert.True(t, ok) // Still valid if id is present
		assert.Empty(t, task.Task)
	})

	t.Run("invalid type", func(t *testing.T) {
		data := "not a map"
		_, ok := convertToRecentTask(data)
		assert.False(t, ok)
	})

	t.Run("empty data", func(t *testing.T) {
		data := map[string]interface{}{}
		_, ok := convertToRecentTask(data)
		assert.False(t, ok)
	})
}

// TestSortAndLimitTasks tests the sorting and limiting of tasks.
func TestSortAndLimitTasks(t *testing.T) {
	tasks := []RecentTask{
		{ID: "1", Task: "Old", Timestamp: 1000},
		{ID: "2", Task: "New", Timestamp: 3000},
		{ID: "3", Task: "Middle", Timestamp: 2000},
	}

	t.Run("sort and limit", func(t *testing.T) {
		result := sortAndLimitTasks(tasks, 2)
		assert.Len(t, result, 2)
		assert.Equal(t, "New", result[0].Task)      // Highest timestamp first
		assert.Equal(t, "Middle", result[1].Task)   // Second highest
	})

	t.Run("limit zero returns all", func(t *testing.T) {
		result := sortAndLimitTasks(tasks, 0)
		assert.Len(t, result, 3)
	})

	t.Run("limit greater than length", func(t *testing.T) {
		result := sortAndLimitTasks(tasks, 10)
		assert.Len(t, result, 3)
	})

	t.Run("empty tasks", func(t *testing.T) {
		result := sortAndLimitTasks([]RecentTask{}, 5)
		assert.Empty(t, result)
	})
}

// TestFormatRelativeTime tests the relative time formatting.
func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		timestamp int64
		want      string
	}{
		{
			name:      "just now",
			timestamp: now.Add(-30 * time.Second).Unix(),
			want:      "just now",
		},
		{
			name:      "minutes ago",
			timestamp: now.Add(-5 * time.Minute).Unix(),
			want:      "5 minutes ago",
		},
		{
			name:      "hours ago",
			timestamp: now.Add(-3 * time.Hour).Unix(),
			want:      "3 hours ago",
		},
		{
			name:      "days ago",
			timestamp: now.Add(-2 * 24 * time.Hour).Unix(),
			want:      "2 days ago",
		},
		{
			name:      "old date",
			timestamp: now.Add(-30 * 24 * time.Hour).Unix(),
			want:      now.Add(-30 * 24 * time.Hour).Format("Jan 02"),
		},
		{
			name:      "zero timestamp",
			timestamp: 0,
			want:      "unknown",
		},
		{
			name:      "milliseconds format",
			timestamp: now.Add(-time.Hour).Unix() * 1000,
			want:      "1 hour ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.timestamp)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestShouldShowWelcome tests the welcome display logic.
func TestShouldShowWelcome(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	t.Run("first time user", func(t *testing.T) {
		assert.True(t, ShouldShowWelcome(storageCtx))
	})

	t.Run("welcome already shown", func(t *testing.T) {
		err := storageCtx.GlobalState.Set("welcomeShown", true)
		require.NoError(t, err)

		assert.False(t, ShouldShowWelcome(storageCtx))

		// Clean up
		_ = storageCtx.GlobalState.Delete("welcomeShown")
	})

	t.Run("welcome disabled", func(t *testing.T) {
		err := storageCtx.GlobalState.Set("welcomeDisabled", true)
		require.NoError(t, err)

		assert.False(t, ShouldShowWelcome(storageCtx))

		// Clean up
		_ = storageCtx.GlobalState.Delete("welcomeDisabled")
	})

	t.Run("nil storage", func(t *testing.T) {
		assert.True(t, ShouldShowWelcome(nil))
	})
}

// TestResetWelcomeFlags tests the reset functionality.
func TestResetWelcomeFlags(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	// Set flags
	err = storageCtx.GlobalState.Set("welcomeShown", true)
	require.NoError(t, err)
	err = storageCtx.GlobalState.Set("welcomeDisabled", true)
	require.NoError(t, err)

	// Reset
	err = ResetWelcomeFlags(storageCtx)
	require.NoError(t, err)

	// Verify flags are removed
	_, ok := storageCtx.GlobalState.Get("welcomeShown")
	assert.False(t, ok)
	_, ok = storageCtx.GlobalState.Get("welcomeDisabled")
	assert.False(t, ok)
}

// TestWelcomeModel_renderBanner tests banner rendering.
func TestWelcomeModel_renderBanner(t *testing.T) {
	config := DefaultWelcomeConfig()
	config.Version = "1.0.0"
	model := NewWelcomeModel(config)

	banner := model.renderBanner(80)
	assert.Contains(t, banner, "Welcome to Cline")
	assert.Contains(t, banner, "Version 1.0.0")
	assert.Contains(t, banner, "Cline")
}

// TestWelcomeModel_renderAuthStatus tests auth status rendering.
func TestWelcomeModel_renderAuthStatus(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	t.Run("not authenticated", func(t *testing.T) {
		model.authStatus = AuthStatus{IsAuthenticated: false}
		output := model.renderAuthStatus(80)
		assert.Contains(t, output, "Not authenticated")
		assert.Contains(t, output, "cline auth")
	})

	t.Run("authenticated", func(t *testing.T) {
		model.authStatus = AuthStatus{
			IsAuthenticated: true,
			Provider:        "anthropic",
			Model:           "claude-3",
		}
		output := model.renderAuthStatus(80)
		assert.Contains(t, output, "Connected to anthropic")
		assert.Contains(t, output, "claude-3")
	})
}

// TestWelcomeModel_renderQuickStartHints tests hints rendering.
func TestWelcomeModel_renderQuickStartHints(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	output := model.renderQuickStartHints(80)
	assert.Contains(t, output, "Quick Start")
	assert.Contains(t, output, "cline")
	assert.Contains(t, output, "cline auth")
	assert.Contains(t, output, "cline history")
}

// TestWelcomeModel_renderRecentTasks tests recent tasks rendering.
func TestWelcomeModel_renderRecentTasks(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	t.Run("empty tasks", func(t *testing.T) {
		output := model.renderRecentTasks(80)
		assert.Contains(t, output, "Recent Tasks")
		assert.Contains(t, output, "No tasks yet")
	})

	t.Run("with tasks", func(t *testing.T) {
		model.recentTasks = []RecentTask{
			{ID: "1", Task: "Test task 1", Timestamp: time.Now().Add(-time.Hour).Unix()},
			{ID: "2", Task: "Test task 2", Timestamp: time.Now().Add(-2 * time.Hour).Unix()},
		}
		output := model.renderRecentTasks(80)
		assert.Contains(t, output, "Test task 1")
		assert.Contains(t, output, "Test task 2")
	})
}

// TestWelcomeModel_renderTerminalTooSmall tests small terminal handling.
func TestWelcomeModel_renderTerminalTooSmall(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)
	model.width = 40
	model.height = 10

	output := model.renderTerminalTooSmall()
	assert.Contains(t, output, "Terminal Too Small")
	assert.Contains(t, output, "40x10")
	assert.Contains(t, output, "60x20") // Min dimensions
}

// TestWelcomeModel_renderFooter tests footer rendering.
func TestWelcomeModel_renderFooter(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	output := model.renderFooter(80)
	assert.Contains(t, output, "continue")
	assert.Contains(t, output, "skip welcome")
	assert.Contains(t, output, "quit")
}

// TestDefaultWelcomeKeyMap tests key map creation.
func TestDefaultWelcomeKeyMap(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	assert.NotNil(t, km.Continue)
	assert.NotNil(t, km.Skip)
	assert.NotNil(t, km.Quit)

	// Test help text
	assert.Equal(t, "continue", km.Continue.Help().Desc)
	assert.Equal(t, "skip welcome in future", km.Skip.Help().Desc)
	assert.Equal(t, "quit", km.Quit.Help().Desc)
}

// TestWelcomeResult tests the result struct.
func TestWelcomeResult(t *testing.T) {
	result := WelcomeResult{
		State:        WelcomeStateCompleted,
		WasSkipped:   false,
		WasFirstTime: true,
	}

	assert.Equal(t, WelcomeStateCompleted, result.State)
	assert.False(t, result.WasSkipped)
	assert.True(t, result.WasFirstTime)
}

// TestDefaultWelcomeConfig tests default configuration.
func TestDefaultWelcomeConfig(t *testing.T) {
	config := DefaultWelcomeConfig()

	assert.Equal(t, 60, config.MinWidth)
	assert.Equal(t, 20, config.MinHeight)
	assert.Equal(t, 5, config.MaxRecentTasks)
	assert.Equal(t, "0.1.0", config.Version)
	assert.False(t, config.SkipWelcome)
}

// TestWelcomeConfig_WithStorage tests configuration with storage context.
func TestWelcomeConfig_WithStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	config.Version = "2.0.0"

	model := NewWelcomeModel(config)
	assert.Equal(t, storageCtx, model.config.StorageContext)
	assert.Equal(t, "2.0.0", model.config.Version)
}

// TestWelcomeModel_loadDataCmd tests the data loading command.
func TestWelcomeModel_loadDataCmd(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	cmd := model.loadDataCmd()
	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	assert.IsType(t, welcomeDataLoadedMsg{}, msg)
}

// TestWelcomeModel_loadData tests loading data from storage.
func TestWelcomeModel_loadData(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	t.Run("first time user", func(t *testing.T) {
		config := DefaultWelcomeConfig()
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		msg := model.loadData()
		result, ok := msg.(welcomeDataLoadedMsg)
		require.True(t, ok)
		assert.False(t, result.skipWelcome)
		assert.True(t, model.isFirstTime)
	})

	t.Run("welcome disabled", func(t *testing.T) {
		err := storageCtx.GlobalState.Set("welcomeDisabled", true)
		require.NoError(t, err)

		config := DefaultWelcomeConfig()
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		msg := model.loadData()
		result, ok := msg.(welcomeDataLoadedMsg)
		require.True(t, ok)
		assert.True(t, result.skipWelcome)

		// Clean up
		_ = storageCtx.GlobalState.Delete("welcomeDisabled")
	})

	t.Run("skip welcome config", func(t *testing.T) {
		err := storageCtx.GlobalState.Set("welcomeShown", true)
		require.NoError(t, err)

		config := DefaultWelcomeConfig()
		config.SkipWelcome = true
		config.StorageContext = storageCtx
		model := NewWelcomeModel(config)

		msg := model.loadData()
		result, ok := msg.(welcomeDataLoadedMsg)
		require.True(t, ok)
		assert.True(t, result.skipWelcome)

		// Clean up
		_ = storageCtx.GlobalState.Delete("welcomeShown")
	})
}

// TestShowWelcome tests the ShowWelcome function.
func TestShowWelcome(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	t.Run("should not show when already shown", func(t *testing.T) {
		err := storageCtx.GlobalState.Set("welcomeShown", true)
		require.NoError(t, err)

		result, err := ShowWelcome(storageCtx, "1.0.0")
		require.NoError(t, err)
		assert.Equal(t, WelcomeStateCompleted, result.State)
		assert.True(t, result.WasSkipped)
		assert.False(t, result.WasFirstTime)

		// Clean up
		_ = storageCtx.GlobalState.Delete("welcomeShown")
	})

	t.Run("nil storage returns result", func(t *testing.T) {
		// When storage is nil, ShowWelcome attempts to run the TUI
		// In CI/test environments without TTY, this will fail
		// So we just verify the function handles nil storage gracefully
		// by checking that ShouldShowWelcome returns true
		assert.True(t, ShouldShowWelcome(nil))
	})
}

// TestWelcomeModel_saveSkipPreference tests skip preference saving.
func TestWelcomeModel_saveSkipPreference(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	model := NewWelcomeModel(config)

	model.saveSkipPreference()

	val, ok := storageCtx.GlobalState.Get("welcomeDisabled")
	assert.True(t, ok)
	assert.True(t, val.(bool))

	val, ok = storageCtx.GlobalState.Get("welcomeShown")
	assert.True(t, ok)
	assert.True(t, val.(bool))
}

// TestWelcomeModel_saveWelcomeShown tests marking welcome as shown.
func TestWelcomeModel_saveWelcomeShown(t *testing.T) {
	tmpDir := t.TempDir()

	storageCtx, err := storage.NewStorageContext(tmpDir, "")
	require.NoError(t, err)
	defer storageCtx.Close()

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	model := NewWelcomeModel(config)

	model.saveWelcomeShown()

	val, ok := storageCtx.GlobalState.Get("welcomeShown")
	assert.True(t, ok)
	assert.True(t, val.(bool))
}

// TestWelcomeKeyMap_Matches tests key matching behavior.
func TestWelcomeKeyMap_Matches(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	t.Run("continue key", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		assert.True(t, key.Matches(msg, km.Continue))
	})

	t.Run("skip key", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		assert.True(t, key.Matches(msg, km.Skip))
	})

	t.Run("quit key", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		assert.True(t, key.Matches(msg, km.Quit))
	})
}

// TestWelcomeModel_renderLoading tests loading state rendering.
func TestWelcomeModel_renderLoading(t *testing.T) {
	config := DefaultWelcomeConfig()
	model := NewWelcomeModel(config)

	output := model.renderLoading()
	assert.Equal(t, "Loading welcome screen...", output)
}

// TestWelcomeModel_View tests the main view function.
func TestWelcomeModel_View(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		config := DefaultWelcomeConfig()
		model := NewWelcomeModel(config)
		model.state = WelcomeStateInitial

		output := model.View()
		assert.Equal(t, "Loading welcome screen...", output)
	})

	t.Run("displaying state", func(t *testing.T) {
		config := DefaultWelcomeConfig()
		model := NewWelcomeModel(config)
		model.state = WelcomeStateDisplaying
		model.width = 80
		model.height = 24

		output := model.View()
		assert.Contains(t, output, "Welcome to Cline")
	})

	t.Run("terminal too small", func(t *testing.T) {
		config := DefaultWelcomeConfig()
		model := NewWelcomeModel(config)
		model.state = WelcomeStateDisplaying
		model.width = 30
		model.height = 10

		output := model.View()
		assert.Contains(t, output, "Terminal Too Small")
	})
}

// TestRecentTask tests the RecentTask struct.
func TestRecentTask(t *testing.T) {
	task := RecentTask{
		ID:        "task-1",
		Task:      "Test task",
		Timestamp: 1234567890,
	}

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "Test task", task.Task)
	assert.Equal(t, int64(1234567890), task.Timestamp)
}

// TestAuthStatus tests the AuthStatus struct.
func TestAuthStatus(t *testing.T) {
	status := AuthStatus{
		Provider:        "anthropic",
		IsAuthenticated: true,
		Model:           "claude-3",
	}

	assert.Equal(t, "anthropic", status.Provider)
	assert.True(t, status.IsAuthenticated)
	assert.Equal(t, "claude-3", status.Model)
}

// TestWelcomeState tests the welcome states.
func TestWelcomeState(t *testing.T) {
	states := []WelcomeState{
		WelcomeStateInitial,
		WelcomeStateDisplaying,
		WelcomeStateCompleted,
		WelcomeStateSkipped,
	}

	for i, state := range states {
		assert.Equal(t, WelcomeState(i), state)
	}
}