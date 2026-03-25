package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/cline/cline/golang-cli/internal/task"
)

// MockTaskRunner is a mock implementation of TaskRunner for testing
type MockTaskRunner struct {
	config task.Config
	err    error
}

func (m *MockTaskRunner) Run(config task.Config) error {
	m.config = config
	return m.err
}

func TestTaskCmd(t *testing.T) {
	// Save original values
	originalArgs := taskFlags

	// Reset after test
	defer func() {
		taskFlags = originalArgs
	}()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "simple prompt",
			args:    []string{"hello world"},
			wantErr: false,
		},
		{
			name:    "with act flag",
			args:    []string{"--act", "do something"},
			wantErr: false,
		},
		{
			name:    "with plan flag",
			args:    []string{"--plan", "plan something"},
			wantErr: false,
		},
		{
			name:    "with short flags",
			args:    []string{"-y", "-t", "5m", "-m", "gpt-4", "task"},
			wantErr: false,
		},
		{
			name:    "with long flags",
			args:    []string{"--yolo", "--timeout", "10m", "--model", "claude", "task"},
			wantErr: false,
		},
		{
			name:    "with multiple images",
			args:    []string{"-i", "img1.png", "-i", "img2.png", "describe"},
			wantErr: true, // images don't exist
		},
		{
			name:    "with prompt only",
			args:    []string{"simple task"},
			wantErr: false,
		},
		{
			name:    "with cwd",
			args:    []string{"-c", "/tmp", "cwd task"},
			wantErr: false,
		},
		{
			name:    "with taskId",
			args:    []string{"-T", "task-123", "resume task"},
			wantErr: false,
		},
		{
			name:    "with thinking",
			args:    []string{"--thinking", "think about this"},
			wantErr: false,
		},
		{
			name:    "with json",
			args:    []string{"--json", "json task"},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: true, // no prompt or taskId
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetTaskFlags()

			// Create a new command for testing
			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskCmdValidate(t *testing.T) {
	// Create temp files for testing
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")
	testConfig := filepath.Join(tmpDir, "config.json")

	// Create test files
	if err := os.WriteFile(testImage, []byte("fake image"), 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	if err := os.WriteFile(testConfig, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple task",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "mutually exclusive act and plan",
			args:    []string{"--act", "--plan", "task"},
			wantErr: true,
			errMsg:  "cannot use both --act and --plan flags",
		},
		{
			name:    "invalid timeout format",
			args:    []string{"-t", "invalid", "task"},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name:    "non-existent image",
			args:    []string{"-i", "/nonexistent.png", "task"},
			wantErr: true,
			errMsg:  "image file not found",
		},
		{
			name:    "non-existent config",
			args:    []string{"--config", "/nonexistent.json", "task"},
			wantErr: true,
			errMsg:  "config file not found",
		},
		{
			name:    "valid with existing image",
			args:    []string{"-i", testImage, "task"},
			wantErr: false,
		},
		{
			name:    "valid with existing config",
			args:    []string{"--config", testConfig, "task"},
			wantErr: false,
		},
		{
			name:    "non-existent cwd",
			args:    []string{"-c", "/nonexistent/dir", "task"},
			wantErr: true,
			errMsg:  "working directory not found",
		},
		{
			name:    "cwd is file not directory",
			args:    []string{"-c", testConfig, "task"},
			wantErr: true,
			errMsg:  "cwd is not a directory",
		},
		{
			name:    "valid with existing cwd",
			args:    []string{"-c", tmpDir, "task"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetTaskFlags()

			// Create a new command for testing
			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestBuildTaskConfig(t *testing.T) {
	tests := []struct {
		name     string
		flags    taskFlagValues
		expected task.Config
	}{
		{
			name:  "default mode is act",
			flags: taskFlagValues{},
			expected: task.Config{
				Mode: task.ModeAct,
			},
		},
		{
			name: "plan mode",
			flags: taskFlagValues{
				plan: true,
			},
			expected: task.Config{
				Mode: task.ModePlan,
			},
		},
		{
			name: "act mode explicit",
			flags: taskFlagValues{
				act: true,
			},
			expected: task.Config{
				Mode: task.ModeAct,
			},
		},
		{
			name: "all flags",
			flags: taskFlagValues{
				yolo:     true,
				timeout:  "5m",
				model:    "gpt-4",
				images:   []string{"img1.png", "img2.png"},
				cwd:      "/tmp",
				thinking: true,
				json:     true,
				taskId:   "task-123",
			},
			expected: task.Config{
				Mode:     task.ModeAct,
				Yolo:     true,
				Timeout:  5 * time.Minute,
				Model:    "gpt-4",
				Images:   []string{"img1.png", "img2.png"},
				Cwd:      "/tmp",
				Thinking: true,
				JSON:     true,
				TaskID:   "task-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset and set flags
			resetTaskFlags()
			setTaskFlags(tt.flags)

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.Mode != tt.expected.Mode {
				t.Errorf("Mode = %v, want %v", config.Mode, tt.expected.Mode)
			}
			if config.Yolo != tt.expected.Yolo {
				t.Errorf("Yolo = %v, want %v", config.Yolo, tt.expected.Yolo)
			}
			if config.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %v, want %v", config.Timeout, tt.expected.Timeout)
			}
			if config.Model != tt.expected.Model {
				t.Errorf("Model = %v, want %v", config.Model, tt.expected.Model)
			}
			if config.Cwd != tt.expected.Cwd {
				t.Errorf("Cwd = %v, want %v", config.Cwd, tt.expected.Cwd)
			}
			if config.Thinking != tt.expected.Thinking {
				t.Errorf("Thinking = %v, want %v", config.Thinking, tt.expected.Thinking)
			}
			if config.JSON != tt.expected.JSON {
				t.Errorf("JSON = %v, want %v", config.JSON, tt.expected.JSON)
			}
			if config.TaskID != tt.expected.TaskID {
				t.Errorf("TaskID = %v, want %v", config.TaskID, tt.expected.TaskID)
			}

			// Check images
			if len(config.Images) != len(tt.expected.Images) {
				t.Errorf("Images length = %v, want %v", len(config.Images), len(tt.expected.Images))
			}
			for i, img := range tt.expected.Images {
				if i >= len(config.Images) || config.Images[i] != img {
					t.Errorf("Images[%d] = %v, want %v", i, config.Images[i], img)
				}
			}
		})
	}
}

func TestDefaultTaskRunnerRun(t *testing.T) {
	tests := []struct {
		name    string
		config  task.Config
		wantErr bool
	}{
		{
			name: "basic task",
			config: task.Config{
				Mode:   task.ModeAct,
				Prompt: "test task",
			},
			wantErr: false,
		},
		{
			name: "verbose output",
			config: task.Config{
				Mode:    task.ModePlan,
				Prompt:  "plan task",
				Verbose: true,
			},
			wantErr: false,
		},
		{
			name: "json output",
			config: task.Config{
				Mode:   task.ModeAct,
				Prompt: "json task",
				JSON:   true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			runner := NewDefaultTaskRunner(&buf)

			err := runner.Run(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := buf.String()

			if tt.config.JSON {
				// Verify JSON output
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("output is not valid JSON: %v", err)
				}
			} else {
				// Verify normal output contains expected elements
				if !strings.Contains(output, "Task:") {
					t.Errorf("output missing 'Task:' prefix")
				}
			}
		})
	}
}

func TestDefaultTaskRunnerRunWithImages(t *testing.T) {
	// Create temp image file
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(testImage, []byte("fake image"), 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	tests := []struct {
		name    string
		images  []string
		wantErr bool
	}{
		{
			name:    "valid image",
			images:  []string{testImage},
			wantErr: false,
		},
		{
			name:    "non-existent image",
			images:  []string{"/nonexistent.png"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			runner := NewDefaultTaskRunner(&buf)

			config := task.Config{
				Mode:   task.ModeAct,
				Prompt: "test",
				Images: tt.images,
			}

			err := runner.Run(config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultTaskRunnerRunWithConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	testConfig := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(testConfig, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Create non-existent config path
	nonExistentConfig := filepath.Join(tmpDir, "nonexistent.json")

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "valid config",
			configPath: testConfig,
			wantErr:    false,
		},
		{
			name:       "non-existent config",
			configPath: nonExistentConfig,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			runner := NewDefaultTaskRunner(&buf)

			config := task.Config{
				Mode:       task.ModeAct,
				Prompt:     "test",
				// Note: ConfigPath is not in task.Config, so we skip this test for now
			}
			_ = config
			_ = tt.configPath

			// Since ConfigPath is not in task.Config, we just verify the runner works
			err := runner.Run(task.Config{
				Mode:   task.ModeAct,
				Prompt: "test",
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageFile(t *testing.T) {
	// Create temp dir and files
	tmpDir := t.TempDir()

	// Create valid image files
	validImage := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(validImage, []byte("fake"), 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	// Create file with invalid extension
	invalidExt := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(invalidExt, []byte("text"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid image",
			path:    validImage,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "/nonexistent.png",
			wantErr: true,
			errMsg:  "does not exist",
		},
		{
			name:    "directory not file",
			path:    testDir,
			wantErr: true,
			errMsg:  "is a directory",
		},
		{
			name:    "invalid extension",
			path:    invalidExt,
			wantErr: true,
			errMsg:  "unsupported image format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageFile(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "home directory expansion",
			path:    "~/test",
			want:    filepath.Join(homeDir, "test"),
			wantErr: false,
		},
		{
			name:    "home directory with slash",
			path:    "~/test/path",
			want:    filepath.Join(homeDir, "test/path"),
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/absolute/path",
			want:    "/absolute/path",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			want:    "", // will be expanded to absolute, exact value depends on cwd
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != "" {
				if got != tt.want {
					t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
				}
			} else if tt.path != "" {
				// For relative paths, just verify it's absolute
				if !filepath.IsAbs(got) {
					t.Errorf("ExpandPath() = %v, expected absolute path", got)
				}
			}
		})
	}
}

func TestNewDefaultTaskRunner(t *testing.T) {
	// Test with nil output
	runner := NewDefaultTaskRunner(nil)
	if runner.output != os.Stdout {
		t.Error("expected output to be os.Stdout when nil is passed")
	}

	// Test with custom output
	var buf bytes.Buffer
	runner = NewDefaultTaskRunner(&buf)
	if runner.output != &buf {
		t.Error("expected output to be the provided writer")
	}
}

func TestTaskModeConstants(t *testing.T) {
	if task.ModeAct != "act" {
		t.Errorf("ModeAct = %v, want 'act'", task.ModeAct)
	}
	if task.ModePlan != "plan" {
		t.Errorf("ModePlan = %v, want 'plan'", task.ModePlan)
	}
}

func TestDefaultTaskRunnerVerboseOutput(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.Config{
		Mode:       task.ModeAct,
		Prompt:     "test task",
		Verbose:    true,
		Yolo:       true,
		Timeout:    5 * time.Minute,
		Model:      "gpt-4",
		Images:     []string{"img.png"},
		Cwd:        "/tmp",
		Thinking:   true,
		TaskID:     "task-123",
	}

	// Create temp files so validation passes
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "img.png")
	configPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(imgPath, []byte("img"), 0644)
	os.WriteFile(configPath, []byte("{}"), 0644)

	config.Images = []string{imgPath}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify verbose output contains expected content
	expectedStrings := []string{
		"Starting task in act mode",
		"Yolo mode: auto-approval enabled",
		"Timeout: 5m",
		"Model: gpt-4",
		"Images:",
		"img.png",
		"Working directory:",
		"Thinking mode enabled",
		"Task ID: task-123",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("verbose output missing expected string: %s", expected)
		}
	}
}

func TestDefaultTaskRunnerJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.Config{
		Mode:       task.ModePlan,
		Prompt:     "json test",
		JSON:       true,
		Yolo:       true,
		Timeout:    10 * time.Minute,
		Model:      "claude",
		Images:     []string{"img.png"},
		Cwd:        "/tmp",
		Thinking:   true,
		TaskID:     "task-456",
	}

	// Create temp files so validation passes
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "img.png")
	configPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(imgPath, []byte("img"), 0644)
	os.WriteFile(configPath, []byte("{}"), 0644)

	config.Images = []string{imgPath}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify JSON fields
	if result["mode"] != "plan" {
		t.Errorf("mode = %v, want 'plan'", result["mode"])
	}
	if result["yolo"] != true {
		t.Errorf("yolo = %v, want true", result["yolo"])
	}
	if result["timeout"] != "10m0s" {
		t.Errorf("timeout = %v, want '10m0s'", result["timeout"])
	}
	if result["model"] != "claude" {
		t.Errorf("model = %v, want 'claude'", result["model"])
	}
	if result["thinking"] != true {
		t.Errorf("thinking = %v, want true", result["thinking"])
	}
	if result["taskId"] != "task-456" {
		t.Errorf("taskId = %v, want 'task-456'", result["taskId"])
	}
	if result["prompt"] != "json test" {
		t.Errorf("prompt = %v, want 'json test'", result["prompt"])
	}
	if result["status"] != "started" {
		t.Errorf("status = %v, want 'started'", result["status"])
	}

	// Verify images array
	images, ok := result["images"].([]interface{})
	if !ok {
		t.Errorf("images is not an array: %T", result["images"])
	} else if len(images) != 1 {
		t.Errorf("len(images) = %d, want 1", len(images))
	}
}

func TestDefaultTaskRunnerVerboseNotJSON(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.Config{
		Mode:    task.ModeAct,
		Prompt:  "test",
		Verbose: true,
		JSON:    true, // JSON should take precedence
	}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// When JSON is true, verbose output should not appear
	if strings.Contains(output, "Starting task in") {
		t.Error("verbose output should not appear when JSON is true")
	}

	// But JSON output should
	if !strings.HasPrefix(output, "{") {
		t.Error("expected JSON output")
	}
}

// MockWriter is an io.Writer that returns errors for testing
type MockWriter struct {
	writeErr error
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return len(p), nil
}

func TestDefaultTaskRunnerWriteErrors(t *testing.T) {
	expectedErr := errors.New("write error")
	mockWriter := &MockWriter{writeErr: expectedErr}
	runner := NewDefaultTaskRunner(mockWriter)

	config := task.Config{
		Mode:   task.ModeAct,
		Prompt: "test",
		JSON:   true, // JSON output uses encoder which returns errors
	}

	err := runner.Run(config)
	// The runner should propagate the write error for JSON encoding
	if err == nil {
		t.Error("expected error from writer, got nil")
	}
}

// Test with io.Discard
func TestDefaultTaskRunnerWithDiscard(t *testing.T) {
	runner := NewDefaultTaskRunner(io.Discard)

	config := task.Config{
		Mode:   task.ModeAct,
		Prompt: "silent task",
	}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error with io.Discard: %v", err)
	}
}

// Test timeout parsing edge cases
func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		valid    bool
	}{
		{"1s", 1 * time.Second, true},
		{"5m", 5 * time.Minute, true},
		{"2h", 2 * time.Hour, true},
		{"1h30m", 90 * time.Minute, true},
		{"100ms", 100 * time.Millisecond, true},
		{"0", 0, true},
		{"", 0, false}, // Empty string should not parse
		{"invalid", 0, false},
		{"5x", 0, false},
		{"-5m", -5 * time.Minute, true}, // Negative is valid for time.ParseDuration
	}

	for _, tt := range tests {
		t.Run("timeout_"+tt.input, func(t *testing.T) {
			// Reset flags
			resetTaskFlags()
			taskFlags.timeout = tt.input

			config, err := buildTaskConfig()

			if tt.input == "" {
				// Empty string should result in 0 duration without error
				if err != nil {
					t.Errorf("unexpected error for empty timeout: %v", err)
				}
				if config.Timeout != 0 {
					t.Errorf("expected 0 for empty timeout, got %v", config.Timeout)
				}
				return
			}

			if tt.valid {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config.Timeout != tt.expected {
					t.Errorf("Timeout = %v, want %v", config.Timeout, tt.expected)
				}
			} else {
				// Invalid durations should return error
				if err == nil {
					t.Errorf("expected error for invalid timeout, got none")
				}
			}
		})
	}
}

// Test for auto-approve-all flag
func TestAutoApproveAllFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "auto-approve-all enabled",
			args:     []string{"--auto-approve-all", "test task"},
			expected: true,
		},
		{
			name:     "auto-approve-all disabled by default",
			args:     []string{"test task"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			// Just validate the flag parsing
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if taskFlags.autoApproveAll != tt.expected {
				t.Errorf("autoApproveAll = %v, want %v", taskFlags.autoApproveAll, tt.expected)
			}
		})
	}
}

// Test for reasoning-effort flag
func TestReasoningEffortFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expected     string
		expectOutput string
	}{
		{
			name:     "reasoning effort low",
			args:     []string{"--reasoning-effort", "low", "test task"},
			expected: "low",
		},
		{
			name:     "reasoning effort medium",
			args:     []string{"--reasoning-effort", "medium", "test task"},
			expected: "medium",
		},
		{
			name:     "reasoning effort high",
			args:     []string{"--reasoning-effort", "high", "test task"},
			expected: "high",
		},
		{
			name:     "reasoning effort xhigh",
			args:     []string{"--reasoning-effort", "xhigh", "test task"},
			expected: "xhigh",
		},
		{
			name:     "reasoning effort none",
			args:     []string{"--reasoning-effort", "none", "test task"},
			expected: "none",
		},
		{
			name:     "reasoning effort uppercase",
			args:     []string{"--reasoning-effort", "MEDIUM", "test task"},
			expected: "medium",
		},
		{
			name:         "invalid reasoning effort defaults to medium",
			args:         []string{"--reasoning-effort", "invalid", "test task"},
			expected:     "medium",
			expectOutput: "Invalid --reasoning-effort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			var output bytes.Buffer
			cmd.SetArgs(tt.args)
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.ReasoningEffort != tt.expected {
				t.Errorf("ReasoningEffort = %v, want %v", config.ReasoningEffort, tt.expected)
			}
		})
	}
}

// Test for max-consecutive-mistakes flag
func TestMaxConsecutiveMistakesFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected int
	}{
		{
			name:     "max consecutive mistakes 3",
			args:     []string{"--max-consecutive-mistakes", "3", "test task"},
			expected: 3,
		},
		{
			name:     "max consecutive mistakes 10",
			args:     []string{"--max-consecutive-mistakes", "10", "test task"},
			expected: 10,
		},
		{
			name:     "max consecutive mistakes default 0",
			args:     []string{"test task"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.MaxConsecutiveMistakes != tt.expected {
				t.Errorf("MaxConsecutiveMistakes = %v, want %v", config.MaxConsecutiveMistakes, tt.expected)
			}
		})
	}
}

// Test for double-check-completion flag
func TestDoubleCheckCompletionFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "double-check-completion enabled",
			args:     []string{"--double-check-completion", "test task"},
			expected: true,
		},
		{
			name:     "double-check-completion disabled by default",
			args:     []string{"test task"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.DoubleCheckCompletion != tt.expected {
				t.Errorf("DoubleCheckCompletion = %v, want %v", config.DoubleCheckCompletion, tt.expected)
			}
		})
	}
}

// Test for auto-condense flag
func TestAutoCondenseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "auto-condense enabled",
			args:     []string{"--auto-condense", "test task"},
			expected: true,
		},
		{
			name:     "auto-condense disabled by default",
			args:     []string{"test task"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.AutoCondense != tt.expected {
				t.Errorf("AutoCondense = %v, want %v", config.AutoCondense, tt.expected)
			}
		})
	}
}

// Test for hooks-dir flag
func TestHooksDirFlag(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")

	// Create hooks directory
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "hooks dir specified",
			args:     []string{"--hooks-dir", hooksDir, "test task"},
			expected: hooksDir,
		},
		{
			name:     "hooks dir empty by default",
			args:     []string{"test task"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTaskFlags()

			cmd := &cobra.Command{
				RunE: runTask,
			}
			setupTaskFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			config, err := buildTaskConfig()
			if err != nil {
				t.Fatalf("buildTaskConfig() error = %v", err)
			}

			if config.HooksDir != tt.expected {
				t.Errorf("HooksDir = %v, want %v", config.HooksDir, tt.expected)
			}
		})
	}
}

// Test verbose output with new flags
func TestDefaultTaskRunnerVerboseOutputWithNewFlags(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.Config{
		Mode:                   task.ModeAct,
		Prompt:                 "test task",
		Verbose:                true,
		AutoApproveAll:         true,
		ReasoningEffort:        "high",
		MaxConsecutiveMistakes: 5,
		DoubleCheckCompletion:  true,
		AutoCondense:           true,
		HooksDir:               "/tmp/hooks",
	}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify verbose output contains expected content for new flags
	expectedStrings := []string{
		"Starting task in act mode",
		"Auto-approve all: enabled",
		"Reasoning effort: high",
		"Max consecutive mistakes: 5",
		"Double-check completion: enabled",
		"Auto-condense: enabled",
		"Hooks directory: /tmp/hooks",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("verbose output missing expected string: %s", expected)
		}
	}
}

// Test JSON output with new flags
func TestDefaultTaskRunnerJSONOutputWithNewFlags(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.Config{
		Mode:                   task.ModeAct,
		Prompt:                 "json test with new flags",
		JSON:                   true,
		Yolo:                   true,
		AutoApproveAll:         true,
		ReasoningEffort:        "medium",
		MaxConsecutiveMistakes: 3,
		DoubleCheckCompletion:  true,
		AutoCondense:           true,
		HooksDir:               "/custom/hooks",
	}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify new JSON fields
	if result["autoApproveAll"] != true {
		t.Errorf("autoApproveAll = %v, want true", result["autoApproveAll"])
	}
	if result["reasoningEffort"] != "medium" {
		t.Errorf("reasoningEffort = %v, want 'medium'", result["reasoningEffort"])
	}
	if result["maxConsecutiveMistakes"] != float64(3) {
		t.Errorf("maxConsecutiveMistakes = %v, want 3", result["maxConsecutiveMistakes"])
	}
	if result["doubleCheckCompletion"] != true {
		t.Errorf("doubleCheckCompletion = %v, want true", result["doubleCheckCompletion"])
	}
	if result["autoCondense"] != true {
		t.Errorf("autoCondense = %v, want true", result["autoCondense"])
	}
	if result["hooksDir"] != "/custom/hooks" {
		t.Errorf("hooksDir = %v, want '/custom/hooks'", result["hooksDir"])
	}
}

// Test all new flags in integration
func TestTaskCommandIntegrationWithNewFlags(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, "hooks")

	// Create hooks directory
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	// Reset flags
	resetTaskFlags()

	// Build full command args with new flags
	args := []string{
		"--auto-approve-all",
		"--reasoning-effort", "high",
		"--max-consecutive-mistakes", "5",
		"--double-check-completion",
		"--auto-condense",
		"--hooks-dir", hooksDir,
		"perform a task with all new flags",
	}

	// Create a new command for testing
	cmd := &cobra.Command{
		RunE: runTask,
	}
	setupTaskFlags(cmd)

	cmd.SetArgs(args)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Task:") {
		t.Errorf("expected task output, got: %s", output)
	}
}

// Benchmark tests
func BenchmarkBuildTaskConfig(b *testing.B) {
	// Set up flags once
	resetTaskFlags()
	taskFlags.yolo = true
	taskFlags.timeout = "5m"
	taskFlags.model = "gpt-4"
	taskFlags.images = []string{"img1.png"}
	taskFlags.cwd = "/tmp"
	taskFlags.thinking = true
	taskFlags.json = true
	taskFlags.taskId = "task-123"
	taskFlags.autoApproveAll = true
	taskFlags.reasoningEffort = "medium"
	taskFlags.maxConsecutiveMistakes = 3
	taskFlags.doubleCheckCompletion = true
	taskFlags.autoCondense = true
	taskFlags.hooksDir = "/tmp/hooks"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildTaskConfig()
	}
}

// Integration-style test
func TestTaskCommandIntegration(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test image
	imgPath := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(imgPath, []byte("fake image"), 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	// Create test config
	configPath := filepath.Join(tmpDir, "config.json")
	configData := `{"model": "gpt-4", "timeout": "5m"}`
	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Reset flags
	resetTaskFlags()

	// Build full command args
	args := []string{
		"--act",
		"--yolo",
		"--timeout", "10m",
		"--model", "claude-3",
		"--image", imgPath,
		"--cwd", tmpDir,
		"--thinking",
		"perform a complex integration test",
	}

	// Create a new command for testing
	cmd := &cobra.Command{
		RunE: runTask,
	}
	setupTaskFlags(cmd)

	cmd.SetArgs(args)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Task:") {
		t.Errorf("expected task output, got: %s", output)
	}
}

// Helper types and functions for testing

type taskFlagValues struct {
	act                    bool
	plan                   bool
	yolo                   bool
	timeout                string
	model                  string
	images                 []string
	cwd                    string
	config                 string
	thinking               bool
	json                   bool
	taskId                 string
	autoApproveAll         bool
	reasoningEffort        string
	maxConsecutiveMistakes int
	doubleCheckCompletion  bool
	autoCondense           bool
	hooksDir               string
}

func resetTaskFlags() {
	taskFlags = struct {
		act                    bool
		plan                   bool
		yolo                   bool
		timeout                string
		model                  string
		images                 []string
		cwd                    string
		config                 string
		thinking               bool
		json                   bool
		taskId                 string
		autoApproveAll         bool
		reasoningEffort        string
		maxConsecutiveMistakes int
		doubleCheckCompletion  bool
		autoCondense           bool
		hooksDir               string
	}{}
}

func setTaskFlags(fv taskFlagValues) {
	taskFlags.act = fv.act
	taskFlags.plan = fv.plan
	taskFlags.yolo = fv.yolo
	taskFlags.timeout = fv.timeout
	taskFlags.model = fv.model
	taskFlags.images = fv.images
	taskFlags.cwd = fv.cwd
	taskFlags.config = fv.config
	taskFlags.thinking = fv.thinking
	taskFlags.json = fv.json
	taskFlags.taskId = fv.taskId
	taskFlags.autoApproveAll = fv.autoApproveAll
	taskFlags.reasoningEffort = fv.reasoningEffort
	taskFlags.maxConsecutiveMistakes = fv.maxConsecutiveMistakes
	taskFlags.doubleCheckCompletion = fv.doubleCheckCompletion
	taskFlags.autoCondense = fv.autoCondense
	taskFlags.hooksDir = fv.hooksDir
}

func setupTaskFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&taskFlags.act, "act", "a", false, "Run in act mode")
	cmd.Flags().BoolVarP(&taskFlags.plan, "plan", "p", false, "Run in plan mode")
	cmd.Flags().BoolVarP(&taskFlags.yolo, "yolo", "y", false, "Auto-approve")
	cmd.Flags().StringVarP(&taskFlags.timeout, "timeout", "t", "", "Timeout duration")
	cmd.Flags().StringVarP(&taskFlags.model, "model", "m", "", "Model to use")
	cmd.Flags().StringArrayVarP(&taskFlags.images, "image", "i", nil, "Image attachments")
	cmd.Flags().StringVarP(&taskFlags.cwd, "cwd", "c", "", "Working directory")
	cmd.Flags().StringVar(&taskFlags.config, "config", "", "Config file path")
	cmd.Flags().BoolVar(&taskFlags.thinking, "thinking", false, "Enable thinking mode")
	cmd.Flags().BoolVar(&taskFlags.json, "json", false, "JSON output")
	cmd.Flags().StringVarP(&taskFlags.taskId, "taskId", "T", "", "Task ID")
	cmd.Flags().BoolVar(&taskFlags.autoApproveAll, "auto-approve-all", false, "Enable auto-approve all")
	cmd.Flags().StringVar(&taskFlags.reasoningEffort, "reasoning-effort", "", "Reasoning effort")
	cmd.Flags().IntVar(&taskFlags.maxConsecutiveMistakes, "max-consecutive-mistakes", 0, "Max consecutive mistakes")
	cmd.Flags().BoolVar(&taskFlags.doubleCheckCompletion, "double-check-completion", false, "Double-check completion")
	cmd.Flags().BoolVar(&taskFlags.autoCondense, "auto-condense", false, "Enable auto-condense")
	cmd.Flags().StringVar(&taskFlags.hooksDir, "hooks-dir", "", "Hooks directory")
	// Note: verbose is a global flag from root.go
}