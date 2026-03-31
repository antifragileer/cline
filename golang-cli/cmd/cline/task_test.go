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
	config task.TaskConfig
	err    error
}

func (m *MockTaskRunner) Run(config task.TaskConfig) error {
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
		expected task.TaskConfig
	}{
		{
			name:  "default mode is act",
			flags: taskFlagValues{},
			expected: task.TaskConfig{
				Mode: task.TaskModeAct,
			},
		},
		{
			name: "plan mode",
			flags: taskFlagValues{
				plan: true,
			},
			expected: task.TaskConfig{
				Mode: task.TaskModePlan,
			},
		},
		{
			name: "act mode explicit",
			flags: taskFlagValues{
				act: true,
			},
			expected: task.TaskConfig{
				Mode: task.TaskModeAct,
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
			expected: task.TaskConfig{
				Mode:     task.TaskModeAct,
				Yolo:     true,
				Timeout:  5 * time.Minute,
				Model:    "gpt-4",
				Images:   []string{"img1.png", "img2.png"},
				Cwd:      "/tmp",
				Thinking: true,
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
		config  task.TaskConfig
		wantErr bool
	}{
		{
			name: "basic task",
			config: task.TaskConfig{
				Mode:   task.TaskModeAct,
				Prompt: "test task",
			},
			wantErr: false,
		},
		{
			name: "verbose output",
			config: task.TaskConfig{
				Mode:    task.TaskModePlan,
				Prompt:  "plan task",
				Verbose: true,
			},
			wantErr: false,
		},
		{
			name: "json output",
			config: task.TaskConfig{
				Mode:   task.TaskModeAct,
				Prompt: "json task",
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

			if taskFlags.json {
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

			config := task.TaskConfig{
				Mode:   task.TaskModeAct,
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
	// This test verifies the DefaultTaskRunner works with various configurations
	tests := []struct {
		name    string
		config  task.TaskConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: task.TaskConfig{
				Mode:   task.TaskModeAct,
				Prompt: "test",
			},
			wantErr: false,
		},
		{
			name: "empty prompt",
			config: task.TaskConfig{
				Mode:   task.TaskModeAct,
				Prompt: "",
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
	if task.TaskModeAct != "act" {
		t.Errorf("TaskModeAct = %v, want 'act'", task.TaskModeAct)
	}
	if task.TaskModePlan != "plan" {
		t.Errorf("TaskModePlan = %v, want 'plan'", task.TaskModePlan)
	}
}

func TestDefaultTaskRunnerVerboseOutput(t *testing.T) {
	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.TaskConfig{
		Mode:     task.TaskModeAct,
		Prompt:   "test task",
		Verbose:  true,
		Yolo:     true,
		Timeout:  5 * time.Minute,
		Model:    "gpt-4",
		Images:   []string{"img.png"},
		Cwd:      "/tmp",
		Thinking: true,
		TaskID:   "task-123",
	}

	// Create temp files so validation passes
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "img.png")
	os.WriteFile(imgPath, []byte("img"), 0644)

	config.Images = []string{imgPath}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify verbose output contains expected content
	expectedStrings := []string{
		"Task: test task",
		"Mode: act",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("verbose output missing expected string: %s", expected)
		}
	}
}

func TestDefaultTaskRunnerJSONOutput(t *testing.T) {
	// Set JSON flag
	taskFlags.json = true
	defer func() { taskFlags.json = false }()

	var buf bytes.Buffer
	runner := NewDefaultTaskRunner(&buf)

	config := task.TaskConfig{
		Mode:     task.TaskModePlan,
		Prompt:   "json test",
		Yolo:     true,
		Timeout:  10 * time.Minute,
		Model:    "claude",
		Images:   []string{"img.png"},
		Cwd:      "/tmp",
		Thinking: true,
		TaskID:   "task-456",
	}

	// Create temp files so validation passes
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "img.png")
	os.WriteFile(imgPath, []byte("img"), 0644)

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

	// Set JSON flag to trigger encoder error
	taskFlags.json = true
	defer func() { taskFlags.json = false }()

	config := task.TaskConfig{
		Mode:   task.TaskModeAct,
		Prompt: "test",
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

	config := task.TaskConfig{
		Mode:   task.TaskModeAct,
		Prompt: "silent task",
	}

	err := runner.Run(config)
	if err != nil {
		t.Errorf("unexpected error with io.Discard: %v", err)
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