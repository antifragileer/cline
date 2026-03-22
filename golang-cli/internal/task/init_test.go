package task

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUI is a mock implementation of UIInterface for testing
type MockUI struct {
	initialized bool
	taskID      string
	prompt      string
	status      string
	err         error
	closed      bool
}

func (m *MockUI) Initialize(taskID string, prompt string) error {
	m.initialized = true
	m.taskID = taskID
	m.prompt = prompt
	return nil
}

func (m *MockUI) UpdateStatus(status string) error {
	m.status = status
	return nil
}

func (m *MockUI) ShowError(err error) error {
	m.err = err
	return nil
}

func (m *MockUI) Close() error {
	m.closed = true
	return nil
}

// createTestStorage creates a temporary storage context for testing
func createTestStorage(t *testing.T) *storage.StorageContext {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	require.NoError(t, err)
	return ctx
}

// createTestConfig creates a test layered configuration
func createTestConfig(t *testing.T) *config.LayeredConfig {
	cfg, err := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test-workspace",
		BaseDir:       t.TempDir(),
	})
	require.NoError(t, err)
	return cfg
}

func TestNewInitializer(t *testing.T) {
	t.Run("valid options", func(t *testing.T) {
		storage := createTestStorage(t)
		defer storage.Close()

		cfg := createTestConfig(t)
		var buf bytes.Buffer

		opts := InitOptions{
			Config:  cfg,
			Storage: storage,
			Output:  &buf,
		}

		init, err := NewInitializer(opts)
		require.NoError(t, err)
		assert.NotNil(t, init)
		assert.Equal(t, &buf, init.output)
	})

	t.Run("missing config", func(t *testing.T) {
		storage := createTestStorage(t)
		defer storage.Close()

		opts := InitOptions{
			Storage: storage,
		}

		_, err := NewInitializer(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("missing storage", func(t *testing.T) {
		cfg := createTestConfig(t)

		opts := InitOptions{
			Config: cfg,
		}

		_, err := NewInitializer(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage is required")
	})

	t.Run("nil output defaults to discard", func(t *testing.T) {
		storage := createTestStorage(t)
		defer storage.Close()

		cfg := createTestConfig(t)

		opts := InitOptions{
			Config:  cfg,
			Storage: storage,
			Output:  nil,
		}

		init, err := NewInitializer(opts)
		require.NoError(t, err)
		assert.Equal(t, io.Discard, init.output)
	})
}

func TestGenerateTaskID(t *testing.T) {
	storage := createTestStorage(t)
	defer storage.Close()

	cfg := createTestConfig(t)
	init, err := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("generate new UUID", func(t *testing.T) {
		id, err := init.generateTaskID("")
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.True(t, isValidUUID(id), "generated ID should be a valid UUID")
	})

	t.Run("use provided valid UUID", func(t *testing.T) {
		providedID := "550e8400-e29b-41d4-a716-446655440000"
		id, err := init.generateTaskID(providedID)
		require.NoError(t, err)
		assert.Equal(t, providedID, id)
	})

	t.Run("use provided custom ID", func(t *testing.T) {
		providedID := "my-custom-task-id"
		id, err := init.generateTaskID(providedID)
		require.NoError(t, err)
		assert.Equal(t, providedID, id)
	})
}

func TestGenerateUUID(t *testing.T) {
	t.Run("generates valid UUID v4", func(t *testing.T) {
		uuid, err := generateUUID()
		require.NoError(t, err)
		assert.NotEmpty(t, uuid)

		// Check format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		parts := strings.Split(uuid, "-")
		require.Len(t, parts, 5)
		assert.Len(t, parts[0], 8)
		assert.Len(t, parts[1], 4)
		assert.Len(t, parts[2], 4)
		assert.Len(t, parts[3], 4)
		assert.Len(t, parts[4], 12)

		// Check version (4)
		assert.True(t, strings.HasPrefix(parts[2], "4"))

		// Check variant (8, 9, a, or b)
		variant := parts[3][0]
		assert.Contains(t, "89ab", string(variant))
	})

	t.Run("generates unique UUIDs", func(t *testing.T) {
		uuid1, err := generateUUID()
		require.NoError(t, err)

		uuid2, err := generateUUID()
		require.NoError(t, err)

		assert.NotEqual(t, uuid1, uuid2)
	})
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UUID with dashes", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID without dashes", "550e8400e29b41d4a716446655440000", true},
		{"valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"empty string", "", false},
		{"too short", "550e8400", false},
		{"too long", "550e8400-e29b-41d4-a716-446655440000-extra", false},
		{"invalid characters", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"custom ID", "my-task-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetWorkingDirectory(t *testing.T) {
	storage := createTestStorage(t)
	defer storage.Close()

	cfg := createTestConfig(t)
	init, err := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("empty path uses current directory", func(t *testing.T) {
		cwd, err := init.setWorkingDirectory("")
		require.NoError(t, err)

		currentDir, err := os.Getwd()
		require.NoError(t, err)

		assert.Equal(t, currentDir, cwd)
	})

	t.Run("absolute path", func(t *testing.T) {
		tmpDir := t.TempDir()
		cwd, err := init.setWorkingDirectory(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, cwd)
	})

	t.Run("relative path", func(t *testing.T) {
		cwd, err := init.setWorkingDirectory(".")
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(cwd))
	})

	t.Run("home directory expansion", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		cwd, err := init.setWorkingDirectory("~")
		require.NoError(t, err)
		assert.Equal(t, homeDir, cwd)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := init.setWorkingDirectory("/nonexistent/path/that/does/not/exist")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not accessible")
	})

	t.Run("path is file not directory", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "testfile")
		err := os.WriteFile(tmpFile, []byte("test"), 0644)
		require.NoError(t, err)

		_, err = init.setWorkingDirectory(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}

func TestLoadAndValidateImages(t *testing.T) {
	storage := createTestStorage(t)
	defer storage.Close()

	cfg := createTestConfig(t)
	init, err := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("empty paths", func(t *testing.T) {
		images, err := init.loadAndValidateImages([]string{})
		require.NoError(t, err)
		assert.Nil(t, images)
	})

	t.Run("valid image", func(t *testing.T) {
		tmpDir := t.TempDir()
		imgPath := filepath.Join(tmpDir, "test.png")
		err := os.WriteFile(imgPath, []byte("fake png content"), 0644)
		require.NoError(t, err)

		images, err := init.loadAndValidateImages([]string{imgPath})
		require.NoError(t, err)
		require.Len(t, images, 1)

		assert.Equal(t, imgPath, images[0].Path)
		assert.Equal(t, "image/png", images[0].MimeType)
		assert.Equal(t, int64(16), images[0].Size)
		assert.NotEmpty(t, images[0].Content)

		// Verify base64 encoding
		decoded, err := base64.StdEncoding.DecodeString(images[0].Content)
		require.NoError(t, err)
		assert.Equal(t, []byte("fake png content"), decoded)
	})

	t.Run("multiple images", func(t *testing.T) {
		tmpDir := t.TempDir()

		img1 := filepath.Join(tmpDir, "test1.png")
		img2 := filepath.Join(tmpDir, "test2.jpg")
		err := os.WriteFile(img1, []byte("png"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(img2, []byte("jpg"), 0644)
		require.NoError(t, err)

		images, err := init.loadAndValidateImages([]string{img1, img2})
		require.NoError(t, err)
		require.Len(t, images, 2)

		assert.Equal(t, "image/png", images[0].MimeType)
		assert.Equal(t, "image/jpeg", images[1].MimeType)
	})

	t.Run("non-existent image", func(t *testing.T) {
		_, err := init.loadAndValidateImages([]string{"/nonexistent.png"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not accessible")
	})

	t.Run("unsupported format", func(t *testing.T) {
		tmpDir := t.TempDir()
		imgPath := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(imgPath, []byte("text"), 0644)
		require.NoError(t, err)

		_, err = init.loadAndValidateImages([]string{imgPath})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported image format")
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := init.loadAndValidateImages([]string{tmpDir})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is a directory")
	})
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".bmp", "image/bmp"},
		{".svg", "image/svg+xml"},
		{".txt", ""},
		{".pdf", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := getMimeType(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := storage.NewStorageContext(tmpDir, "test-workspace")
	require.NoError(t, err)
	defer storage.Close()

	// Create config with some defaults
	cfg, err := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test-workspace",
		BaseDir:       tmpDir,
	})
	require.NoError(t, err)

	init, err := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("sets defaults from config", func(t *testing.T) {
		taskCfg := TaskConfig{}

		err := init.loadConfiguration(&taskCfg)
		require.NoError(t, err)

		// Should have defaults set
		assert.NotEmpty(t, taskCfg.Mode)
		assert.NotEmpty(t, taskCfg.Model)
	})

	t.Run("preserves existing values", func(t *testing.T) {
		taskCfg := TaskConfig{
			Mode:  TaskModePlan,
			Model: "custom-model",
		}

		err := init.loadConfiguration(&taskCfg)
		require.NoError(t, err)

		assert.Equal(t, TaskModePlan, taskCfg.Mode)
		assert.Equal(t, "custom-model", taskCfg.Model)
	})

	t.Run("loads from config file", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "config.json")
		fileCfg := TaskConfig{
			Mode:     TaskModePlan,
			Model:    "file-model",
			Timeout:  5 * time.Minute,
			Yolo:     true,
			Thinking: true,
		}
		data, err := json.Marshal(fileCfg)
		require.NoError(t, err)
		err = os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		taskCfg := TaskConfig{
			ConfigPath: configFile,
		}

		err = init.loadConfiguration(&taskCfg)
		require.NoError(t, err)

		assert.Equal(t, TaskModePlan, taskCfg.Mode)
		assert.Equal(t, "file-model", taskCfg.Model)
		assert.Equal(t, 5*time.Minute, taskCfg.Timeout)
		assert.True(t, taskCfg.Yolo)
		assert.True(t, taskCfg.Thinking)
	})

	t.Run("CLI flags override config file", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "config.json")
		fileCfg := TaskConfig{
			Mode:  TaskModePlan,
			Model: "file-model",
		}
		data, err := json.Marshal(fileCfg)
		require.NoError(t, err)
		err = os.WriteFile(configFile, data, 0644)
		require.NoError(t, err)

		taskCfg := TaskConfig{
			ConfigPath: configFile,
			Mode:       TaskModeAct,
			Model:      "cli-model",
		}

		err = init.loadConfiguration(&taskCfg)
		require.NoError(t, err)

		// CLI values should be preserved
		assert.Equal(t, TaskModeAct, taskCfg.Mode)
		assert.Equal(t, "cli-model", taskCfg.Model)
	})

	t.Run("non-existent config file", func(t *testing.T) {
		taskCfg := TaskConfig{
			ConfigPath: "/nonexistent/config.json",
		}

		err := init.loadConfiguration(&taskCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid config file", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(configFile, []byte("not valid json"), 0644)
		require.NoError(t, err)

		taskCfg := TaskConfig{
			ConfigPath: configFile,
		}

		err = init.loadConfiguration(&taskCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

func TestPersistToHistory(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := storage.NewStorageContext(tmpDir, "test-workspace")
	require.NoError(t, err)
	defer storage.Close()

	cfg, err := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test-workspace",
		BaseDir:       tmpDir,
	})
	require.NoError(t, err)

	init, err := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("persists new task", func(t *testing.T) {
		taskCfg := TaskConfig{
			Prompt: "test prompt",
			Mode:   TaskModeAct,
			Cwd:    "/tmp",
			Model:  "test-model",
		}

		err := init.persistToHistory("task-123", taskCfg)
		require.NoError(t, err)

		// Verify it was saved
		data, ok := storage.GlobalState.Get("taskHistory")
		require.True(t, ok)

		historyJSON, err := json.Marshal(data)
		require.NoError(t, err)

		var history []TaskHistoryEntry
		err = json.Unmarshal(historyJSON, &history)
		require.NoError(t, err)

		require.Len(t, history, 1)
		assert.Equal(t, "task-123", history[0].TaskID)
		assert.Equal(t, "test prompt", history[0].Prompt)
		assert.Equal(t, TaskModeAct, history[0].Mode)
		assert.Equal(t, "/tmp", history[0].Cwd)
		assert.Equal(t, "test-model", history[0].Model)
		assert.Equal(t, "initialized", history[0].Status)
		assert.NotZero(t, history[0].CreatedAt)
	})

	t.Run("appends to existing history", func(t *testing.T) {
		// First entry already exists from previous test
		taskCfg := TaskConfig{
			Prompt: "second task",
			Mode:   TaskModePlan,
		}

		err := init.persistToHistory("task-456", taskCfg)
		require.NoError(t, err)

		data, ok := storage.GlobalState.Get("taskHistory")
		require.True(t, ok)

		historyJSON, err := json.Marshal(data)
		require.NoError(t, err)

		var history []TaskHistoryEntry
		err = json.Unmarshal(historyJSON, &history)
		require.NoError(t, err)

		require.Len(t, history, 2)
		assert.Equal(t, "task-456", history[1].TaskID)
	})

	t.Run("limits history to 100 entries", func(t *testing.T) {
		// Clear existing history
		err := storage.GlobalState.Set("taskHistory", []TaskHistoryEntry{})
		require.NoError(t, err)

		// Add 105 entries
		for i := 0; i < 105; i++ {
			taskCfg := TaskConfig{
				Prompt: fmt.Sprintf("task %d", i),
				Mode:   TaskModeAct,
			}
			err := init.persistToHistory(fmt.Sprintf("task-%d", i), taskCfg)
			require.NoError(t, err)
		}

		data, ok := storage.GlobalState.Get("taskHistory")
		require.True(t, ok)

		historyJSON, err := json.Marshal(data)
		require.NoError(t, err)

		var history []TaskHistoryEntry
		err = json.Unmarshal(historyJSON, &history)
		require.NoError(t, err)

		// Should only have last 100 entries
		assert.Len(t, history, 100)
		// First entry should be task-5 (oldest remaining)
		assert.Equal(t, "task-5", history[0].TaskID)
		// Last entry should be task-104 (most recent)
		assert.Equal(t, "task-104", history[99].TaskID)
	})
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := storage.NewStorageContext(tmpDir, "test-workspace")
	require.NoError(t, err)
	defer storage.Close()

	cfg, err := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test-workspace",
		BaseDir:       tmpDir,
	})
	require.NoError(t, err)

	t.Run("successful initialization without client", func(t *testing.T) {
		var buf bytes.Buffer
		mockUI := &MockUI{}

		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
			UI:      mockUI,
			Output:  &buf,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt:  "test task",
			Mode:    TaskModeAct,
			Verbose: true,
		}

		ctx := context.Background()
		resp, err := init.Init(ctx, taskCfg)
		require.NoError(t, err)

		// Verify response
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.TaskID)
		assert.Equal(t, "Task created locally", resp.Message)
		assert.NotZero(t, resp.Timestamp)

		// Verify UI was initialized
		assert.True(t, mockUI.initialized)
		assert.Equal(t, resp.TaskID, mockUI.taskID)
		assert.Equal(t, "test task", mockUI.prompt)

		// Verify verbose output
		output := buf.String()
		assert.Contains(t, output, "Task ID:")
		assert.Contains(t, output, "Working directory:")
		assert.Contains(t, output, "Configuration loaded")

		// Verify history was persisted
		data, ok := storage.GlobalState.Get("taskHistory")
		assert.True(t, ok)
		assert.NotNil(t, data)

		init.Close()
		assert.True(t, mockUI.closed)
	})

	t.Run("initialization with provided task ID", func(t *testing.T) {
		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt: "task with custom ID",
			TaskID: "my-custom-id",
		}

		ctx := context.Background()
		resp, err := init.Init(ctx, taskCfg)
		require.NoError(t, err)

		assert.Equal(t, "my-custom-id", resp.TaskID)
	})

	t.Run("initialization with images", func(t *testing.T) {
		var buf bytes.Buffer
		imgDir := t.TempDir()
		imgPath := filepath.Join(imgDir, "test.png")
		err := os.WriteFile(imgPath, []byte("fake image"), 0644)
		require.NoError(t, err)

		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
			Output:  &buf,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt:  "task with image",
			Images:  []string{imgPath},
			Verbose: true,
		}

		ctx := context.Background()
		resp, err := init.Init(ctx, taskCfg)
		require.NoError(t, err)

		assert.True(t, resp.Success)

		// Verify verbose output mentions images
		output := buf.String()
		assert.Contains(t, output, "Loaded 1 image(s)")
	})

	t.Run("initialization with custom working directory", func(t *testing.T) {
		workDir := t.TempDir()

		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt: "task with custom cwd",
			Cwd:    workDir,
		}

		ctx := context.Background()
		resp, err := init.Init(ctx, taskCfg)
		require.NoError(t, err)

		assert.True(t, resp.Success)
	})

	t.Run("failed image validation", func(t *testing.T) {
		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt: "task with bad image",
			Images: []string{"/nonexistent.png"},
		}

		ctx := context.Background()
		_, err = init.Init(ctx, taskCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load images")
	})

	t.Run("failed working directory", func(t *testing.T) {
		init, err := NewInitializer(InitOptions{
			Config:  cfg,
			Storage: storage,
		})
		require.NoError(t, err)

		taskCfg := TaskConfig{
			Prompt: "task with bad cwd",
			Cwd:    "/nonexistent/directory",
		}

		ctx := context.Background()
		_, err = init.Init(ctx, taskCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set working directory")
	})
}

func TestTaskModeConstants(t *testing.T) {
	assert.Equal(t, TaskMode("act"), TaskModeAct)
	assert.Equal(t, TaskMode("plan"), TaskModePlan)
}

func TestMustMarshalJSON(t *testing.T) {
	t.Run("valid object", func(t *testing.T) {
		obj := map[string]string{"key": "value"}
		data := mustMarshalJSON(obj)
		assert.Equal(t, `{"key":"value"}`, string(data))
	})

	t.Run("complex object", func(t *testing.T) {
		obj := TaskConfig{
			Prompt: "test",
			Mode:   TaskModeAct,
		}
		data := mustMarshalJSON(obj)
		assert.Contains(t, string(data), `"prompt":"test"`)
		assert.Contains(t, string(data), `"mode":"act"`)
	})
}

func TestGetVersion(t *testing.T) {
	version := getVersion()
	assert.NotEmpty(t, version)
	// Should match semantic versioning format or be a placeholder
	assert.True(t, version == "0.1.0" || len(version) > 0)
}

func TestCreateTaskStream(t *testing.T) {
	// This is a placeholder function that should return an error
	// since the actual proto implementation is not available
	ctx := context.Background()
	// We can't easily create a real grpc.ClientConn in tests without a server
	// So we'll just verify the function signature exists and returns the expected error type
	// In a real implementation, this would be tested with a mock connection
	_ = ctx
	// The function is expected to return an error about proto implementation
	// We can't test this without a real gRPC connection
}

func BenchmarkGenerateTaskID(b *testing.B) {
	storage, _ := storage.NewStorageContext(b.TempDir(), "test")
	defer storage.Close()

	cfg, _ := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test",
		BaseDir:       b.TempDir(),
	})

	init, _ := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = init.generateTaskID("")
	}
}

func BenchmarkLoadImage(b *testing.B) {
	tmpDir := b.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")
	os.WriteFile(imgPath, []byte("fake image content for benchmarking"), 0644)

	storage, _ := storage.NewStorageContext(tmpDir, "test")
	defer storage.Close()

	cfg, _ := config.NewLayeredConfig(config.ConfigOptions{
		WorkspaceHash: "test",
		BaseDir:       tmpDir,
	})

	init, _ := NewInitializer(InitOptions{
		Config:  cfg,
		Storage: storage,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = init.loadImage(imgPath)
	}
}

// TestNewTaskRequestResponse verifies the request/response structures
func TestNewTaskRequestResponse(t *testing.T) {
	t.Run("NewTaskRequest serialization", func(t *testing.T) {
		req := NewTaskRequest{
			TaskID:    "task-123",
			Prompt:    "test prompt",
			Mode:      TaskModeAct,
			Model:     "claude-3",
			Timeout:   120,
			Yolo:      true,
			Thinking:  false,
			Cwd:       "/tmp",
			Timestamp: 1234567890,
			Metadata: map[string]string{
				"key": "value",
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded NewTaskRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, req.TaskID, decoded.TaskID)
		assert.Equal(t, req.Prompt, decoded.Prompt)
		assert.Equal(t, req.Mode, decoded.Mode)
		assert.Equal(t, req.Model, decoded.Model)
		assert.Equal(t, req.Timeout, decoded.Timeout)
		assert.Equal(t, req.Yolo, decoded.Yolo)
		assert.Equal(t, req.Thinking, decoded.Thinking)
		assert.Equal(t, req.Cwd, decoded.Cwd)
		assert.Equal(t, req.Timestamp, decoded.Timestamp)
		assert.Equal(t, req.Metadata, decoded.Metadata)
	})

	t.Run("NewTaskResponse serialization", func(t *testing.T) {
		resp := NewTaskResponse{
			Success:   true,
			TaskID:    "task-456",
			Message:   "Task created",
			Timestamp: 1234567890,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded NewTaskResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.Success, decoded.Success)
		assert.Equal(t, resp.TaskID, decoded.TaskID)
		assert.Equal(t, resp.Message, decoded.Message)
		assert.Equal(t, resp.Timestamp, decoded.Timestamp)
	})
}

// TestImageData verifies the ImageData structure
func TestImageData(t *testing.T) {
	img := ImageData{
		Path:     "/path/to/image.png",
		Content:  "base64encodedcontent",
		MimeType: "image/png",
		Size:     1024,
	}

	assert.Equal(t, "/path/to/image.png", img.Path)
	assert.Equal(t, "base64encodedcontent", img.Content)
	assert.Equal(t, "image/png", img.MimeType)
	assert.Equal(t, int64(1024), img.Size)
}

// TestTaskHistoryEntry verifies the TaskHistoryEntry structure
func TestTaskHistoryEntry(t *testing.T) {
	entry := TaskHistoryEntry{
		TaskID:    "task-789",
		Prompt:    "history test",
		Mode:      TaskModePlan,
		Cwd:       "/workspace",
		Model:     "gpt-4",
		CreatedAt: 1234567890,
		Status:    "completed",
	}

	assert.Equal(t, "task-789", entry.TaskID)
	assert.Equal(t, "history test", entry.Prompt)
	assert.Equal(t, TaskModePlan, entry.Mode)
	assert.Equal(t, "/workspace", entry.Cwd)
	assert.Equal(t, "gpt-4", entry.Model)
	assert.Equal(t, int64(1234567890), entry.CreatedAt)
	assert.Equal(t, "completed", entry.Status)
}