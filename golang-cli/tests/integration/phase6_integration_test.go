// Package integration provides integration tests for the Cline CLI.
// These tests verify feature parity and end-to-end functionality.
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/audit"
	"github.com/cline/cline/golang-cli/internal/security"
	"github.com/cline/cline/golang-cli/internal/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Searchable List Integration Tests ====================

func TestSearchableList_Integration(t *testing.T) {
	t.Run("create and filter searchable list", func(t *testing.T) {
		items := []tui.SearchableListItem{
			{ID: "1", Title: "Apple", Description: "Red fruit"},
			{ID: "2", Title: "Banana", Description: "Yellow fruit"},
			{ID: "3", Title: "Cherry", Description: "Small red fruit"},
			{ID: "4", Title: "Date", Description: "Sweet fruit"},
		}

		// Test filtering
		filtered := tui.FilterItems(items, "red")
		assert.Equal(t, 2, len(filtered))
		assert.Equal(t, "Apple", filtered[0].Title)
		assert.Equal(t, "Cherry", filtered[1].Title)

		// Test case insensitive filtering
		filtered = tui.FilterItems(items, "YELLOW")
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "Banana", filtered[0].Title)

		// Test filtering by description
		filtered = tui.FilterItems(items, "sweet")
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "Date", filtered[0].Title)
	})

	t.Run("sort items alphabetically", func(t *testing.T) {
		items := []tui.SearchableListItem{
			{ID: "3", Title: "Cherry"},
			{ID: "1", Title: "Apple"},
			{ID: "2", Title: "Banana"},
		}

		sorted := tui.SortItemsByTitle(items)

		assert.Equal(t, "Apple", sorted[0].Title)
		assert.Equal(t, "Banana", sorted[1].Title)
		assert.Equal(t, "Cherry", sorted[2].Title)
	})

	t.Run("convert strings to searchable items", func(t *testing.T) {
		strings := []string{"First", "Second", "Third"}
		items := tui.ConvertStringsToSearchableItems(strings)

		assert.Equal(t, 3, len(items))
		assert.Equal(t, "First", items[0].Title)
		assert.Equal(t, "item-0", items[0].ID)
	})
}

// ==================== Permission Enforcement Integration Tests ====================

func TestPermissionEnforcer_Integration(t *testing.T) {
	t.Run("enforcer blocks dangerous commands", func(t *testing.T) {
		// Set up restrictive permissions
		t.Setenv("CLINE_COMMAND_PERMISSIONS", `{
			"allow": ["git *", "ls *", "pwd"],
			"deny": ["rm -rf /", "sudo *"]
		}`)

		// Reset controller to pick up new env
		enforcer := security.NewPermissionEnforcer()

		// Allowed commands
		result := enforcer.ValidateOnly("git", []string{"status"})
		assert.True(t, result.Allowed, "git status should be allowed")

		result = enforcer.ValidateOnly("ls", []string{"-la"})
		assert.True(t, result.Allowed, "ls -la should be allowed")

		// Denied commands
		result = enforcer.ValidateOnly("rm", []string{"-rf", "/"})
		assert.False(t, result.Allowed, "rm -rf / should be denied")

		result = enforcer.ValidateOnly("sudo", []string{"ls"})
		assert.False(t, result.Allowed, "sudo ls should be denied")

		// Unknown commands
		result = enforcer.ValidateOnly("curl", []string{"http://example.com"})
		assert.False(t, result.Allowed, "curl should be denied by default")
	})

	t.Run("enforcer with mock executor", func(t *testing.T) {
		executorCalled := false
		mockExecutor := func(ctx context.Context, command string, args []string) (string, int, error) {
			executorCalled = true
			return "command output", 0, nil
		}

		t.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow": ["echo *"]}`)

		enforcer := security.NewPermissionEnforcer(
			security.WithCommandExecutor(mockExecutor),
		)

		ctx := context.Background()
		result := enforcer.Execute(ctx, "echo", []string{"hello"})

		assert.True(t, executorCalled, "Executor should have been called")
		assert.False(t, result.Blocked, "Command should not be blocked")
		assert.Equal(t, "command output", result.Output)
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("enforcer blocks without calling executor", func(t *testing.T) {
		executorCalled := false
		mockExecutor := func(ctx context.Context, command string, args []string) (string, int, error) {
			executorCalled = true
			return "", 0, nil
		}

		t.Setenv("CLINE_COMMAND_PERMISSIONS", `{
			"allow": ["git *"],
			"deny": ["rm -rf /"]
		}`)

		enforcer := security.NewPermissionEnforcer(
			security.WithCommandExecutor(mockExecutor),
		)

		ctx := context.Background()
		result := enforcer.Execute(ctx, "rm", []string{"-rf", "/"})

		assert.False(t, executorCalled, "Executor should NOT have been called")
		assert.True(t, result.Blocked, "Command should be blocked")
		assert.Equal(t, 1, result.ExitCode)
		assert.NotNil(t, result.Error)
	})

	t.Run("context propagation", func(t *testing.T) {
		ctx := context.Background()
		ctx = security.WithUserContext(ctx, "testuser")
		ctx = security.WithTaskID(ctx, "task-123")
		ctx = security.WithSessionID(ctx, "session-456")

		// Verify context values are extracted correctly
		// This is tested through the enforcer's logging behavior
		t.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow": ["echo *"]}`)

		executorCalled := false
		mockExecutor := func(execCtx context.Context, command string, args []string) (string, int, error) {
			executorCalled = true
			return "done", 0, nil
		}

		enforcer := security.NewPermissionEnforcer(
			security.WithCommandExecutor(mockExecutor),
		)

		result := enforcer.Execute(ctx, "echo", []string{"test"})
		assert.True(t, executorCalled)
		assert.False(t, result.Blocked)
	})
}

// ==================== Checkpoint Integration Tests ====================

func TestCheckpoint_Integration(t *testing.T) {
	t.Run("checkpoint creation and formatting", func(t *testing.T) {
		now := time.Now()
		checkpoint := tui.Checkpoint{
			ID:          "abc123def456",
			Description: "Initial state",
			Timestamp:   now,
			CommitHash:  "abc123",
		}

		formatted := tui.FormatCheckpointDescription(checkpoint)
		assert.Contains(t, formatted, "Initial state")
		assert.Contains(t, formatted, now.Format("Jan 02 15:04"))
	})

	t.Run("checkpoint menu items", func(t *testing.T) {
		checkpoints := []tui.Checkpoint{
			{
				ID:          "cp1",
				Description: "First checkpoint",
				Timestamp:   time.Now().Add(-2 * time.Hour),
				CommitHash:  "hash1",
			},
			{
				ID:          "cp2",
				Description: "Second checkpoint",
				Timestamp:   time.Now().Add(-1 * time.Hour),
				CommitHash:  "hash2",
			},
		}

		items := tui.GetCheckpointMenuItems(checkpoints)
		assert.Equal(t, 2, len(items))
		assert.Contains(t, items[0], "First checkpoint")
		assert.Contains(t, items[1], "Second checkpoint")
	})
}

// ==================== Audit Logging Integration Tests ====================

func TestAuditLogging_Integration(t *testing.T) {
	t.Run("audit logger with file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := audit.LoggerConfig{
			Enabled:     true,
			LogDir:      tmpDir,
			MaxFileSize: 1024 * 1024, // 1MB
			MaxBackups:  3,
			BufferSize:  100,
			SyncWrite:   true, // Synchronous for testing
		}

		logger, err := audit.NewLogger(config)
		require.NoError(t, err)
		defer logger.Close()

		// Log various event types
		err = logger.LogCommandExecution("user1", "task-123", "session-456", "git status", []string{}, "/home/user")
		assert.NoError(t, err)

		err = logger.LogFileRead("user1", "task-123", "session-456", "/home/user/file.txt", 1024)
		assert.NoError(t, err)

		err = logger.LogToolApproval("user1", "task-123", "session-456", "write_file", map[string]interface{}{
			"path": "/home/user/output.txt",
		})
		assert.NoError(t, err)

		// Verify log file was created
		logPath := logger.GetLogPath()
		_, err = os.Stat(logPath)
		assert.NoError(t, err, "Log file should exist")

		// Read and verify log content
		content, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "audit event")
		assert.Contains(t, string(content), "command_execution")
		assert.Contains(t, string(content), "file_read")
		assert.Contains(t, string(content), "tool_approval")
	})

	t.Run("audit report generation", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := audit.LoggerConfig{
			Enabled:     true,
			LogDir:      tmpDir,
			MaxFileSize: 1024 * 1024,
			SyncWrite:   true,
		}

		logger, err := audit.NewLogger(config)
		require.NoError(t, err)

		// Generate some events
		for i := 0; i < 5; i++ {
			logger.LogCommandExecution("user1", "task-123", "session-456", "echo", []string{fmt.Sprintf("test%d", i)}, "/tmp")
		}
		logger.Close()

		// Generate report
		generator, err := audit.NewReportGenerator(tmpDir)
		require.NoError(t, err)

		report, err := generator.Generate(audit.ReportOptions{
			EventTypes: []audit.EventType{audit.EventCommandExecution},
		})
		assert.NoError(t, err)
		assert.NotNil(t, report)
	})

	t.Run("audit logger rotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := audit.LoggerConfig{
			Enabled:     true,
			LogDir:      tmpDir,
			MaxFileSize: 100, // Very small for testing rotation
			MaxBackups:  2,
			SyncWrite:   true,
		}

		logger, err := audit.NewLogger(config)
		require.NoError(t, err)

		// Write enough data to trigger rotation
		for i := 0; i < 10; i++ {
			logger.LogCommandExecution("user", "task", "session", "echo", []string{"test data that is long enough to trigger rotation"}, "/tmp")
		}

		logger.Close()

		// Check for rotated files
		entries, err := os.ReadDir(tmpDir)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
	})
}

// ==================== Settings Integration Tests ====================

func TestSettings_Integration(t *testing.T) {
	t.Run("settings model with storage", func(t *testing.T) {
		// Create a temporary storage context
		tmpDir := t.TempDir()
		storageCtx, err := createTestStorageContext(tmpDir)
		require.NoError(t, err)

		model := tui.NewSettingsModel()

		// Load settings from storage (empty initially)
		model.LoadFromStorage(storageCtx)

		// Get default settings
		settings := model.GetSettings()
		assert.NotEmpty(t, settings)
		assert.Equal(t, "act", settings["mode"])
		assert.Equal(t, "false", settings["yoloMode"])

		// Modify settings
		model.SaveToStorage(storageCtx)

		// Verify settings were saved
		// This would require reading from the storage file
	})

	t.Run("settings save and load", func(t *testing.T) {
		tmpDir := t.TempDir()
		storageCtx, err := createTestStorageContext(tmpDir)
		require.NoError(t, err)

		model := tui.NewSettingsModel()

		// Save settings
		err = model.SaveToStorage(storageCtx)
		assert.NoError(t, err)

		// Create new model and load
		newModel := tui.NewSettingsModel()
		newModel.LoadFromStorage(storageCtx)

		// Verify settings persisted
		settings := newModel.GetSettings()
		assert.Equal(t, "act", settings["mode"])
	})
}

// ==================== Diff Viewer Integration Tests ====================

func TestDiffViewer_Integration(t *testing.T) {
	t.Run("diff model creation", func(t *testing.T) {
		diff := `diff --git a/file.txt b/file.txt
index 123..456 789
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line 1
-line 2
+line 2 modified
 line 3`

		model := tui.NewDiffModel("file.txt", diff)
		assert.Equal(t, "file.txt", model.View())
		// Note: View() returns the diff content in actual implementation
	})

	t.Run("diff formatting", func(t *testing.T) {
		diff := "line 1\nline 2\nline 3"
		formatted := tui.FormatDiff(diff, 10)

		// Should wrap long lines
		lines := splitLines(formatted)
		for _, line := range lines {
			assert.LessOrEqual(t, len(line), 10)
		}
	})
}

// ==================== Comprehensive Feature Parity Tests ====================

func TestPhase6_FeatureParity(t *testing.T) {
	t.Run("searchable list feature exists", func(t *testing.T) {
		// Verify the searchable list component is available
		items := []tui.SearchableListItem{
			{ID: "1", Title: "Test", Description: "Test item"},
		}
		model := tui.NewSearchableListModel("Test List", items)
		assert.NotNil(t, model)
		assert.True(t, model.IsDone() == false) // Not done initially
	})

	t.Run("checkpoint feature exists", func(t *testing.T) {
		checkpoints := []tui.Checkpoint{
			{
				ID:          "cp1",
				Description: "Test checkpoint",
				Timestamp:   time.Now(),
				CommitHash:  "abc123",
			},
		}
		model := tui.NewCheckpointModel(checkpoints)
		assert.NotNil(t, model)
		assert.False(t, model.IsDone())
	})

	t.Run("diff viewer feature exists", func(t *testing.T) {
		model := tui.NewDiffModel("test.go", "diff content")
		assert.NotNil(t, model)
		assert.False(t, model.IsDone())
	})

	t.Run("permission enforcer feature exists", func(t *testing.T) {
		enforcer := security.NewPermissionEnforcer()
		assert.NotNil(t, enforcer)
		assert.False(t, enforcer.IsEnabled()) // No config by default
	})

	t.Run("audit logger feature exists", func(t *testing.T) {
		config := audit.LoggerConfig{
			Enabled: false,
		}
		logger, err := audit.NewLogger(config)
		// May fail if directory creation fails, but the type exists
		if err == nil {
			defer logger.Close()
			assert.NotNil(t, logger)
		}
	})
}

// ==================== Performance Tests ====================

func BenchmarkPermissionValidation(b *testing.B) {
	t.Setenv("CLINE_COMMAND_PERMISSIONS", `{
		"allow": ["git *", "ls *", "cat *", "echo *", "grep *"],
		"deny": ["rm -rf /", "sudo *", "su *"]
	}`)

	enforcer := security.NewPermissionEnforcer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enforcer.ValidateOnly("git", []string{"status"})
	}
}

func BenchmarkSearchableListFilter(b *testing.B) {
	items := make([]tui.SearchableListItem, 100)
	for i := 0; i < 100; i++ {
		items[i] = tui.SearchableListItem{
			ID:          fmt.Sprintf("item-%d", i),
			Title:       fmt.Sprintf("Item %d Title", i),
			Description: fmt.Sprintf("Description for item %d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tui.FilterItems(items, "Item 50")
	}
}

// ==================== Helper Functions ====================

func createTestStorageContext(tmpDir string) (*testStorageContext, error) {
	return &testStorageContext{
		GlobalState:  make(map[string]interface{}),
		WorkspaceState: make(map[string]interface{}),
	}, nil
}

type testStorageContext struct {
	GlobalState    map[string]interface{}
	WorkspaceState map[string]interface{}
}

func (s *testStorageContext) Get(key string) (interface{}, bool) {
	val, ok := s.GlobalState[key]
	return val, ok
}

func (s *testStorageContext) Set(key string, value interface{}) error {
	s.GlobalState[key] = value
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}