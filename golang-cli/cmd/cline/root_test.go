package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cline/cline/golang-cli/internal/task"
)

func TestExecute_VersionFlag(t *testing.T) {
	// Reset flags before test
	resetFlags()

	// Capture output using command's output buffers
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	rootCmd.SetOut(&stdoutBuf)
	rootCmd.SetErr(&stderrBuf)

	// Set args for version flag
	rootCmd.SetArgs([]string{"--version"})

	err := Execute()
	// Cobra returns nil for --version when using SetVersionTemplate
	if err != nil {
		t.Errorf("Execute() with --version returned error: %v", err)
	}

	output := stdoutBuf.String() + stderrBuf.String()

	if !strings.Contains(output, Version) {
		t.Errorf("Expected output to contain version %q, got: %s", Version, output)
	}
}

func TestExecute_HelpFlag(t *testing.T) {
	resetFlags()

	// Capture output using command's output buffers
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	rootCmd.SetOut(&stdoutBuf)
	rootCmd.SetErr(&stderrBuf)

	rootCmd.SetArgs([]string{"--help"})

	err := Execute()
	// Cobra returns nil for --help, it just prints help text
	if err != nil {
		t.Errorf("Execute() with --help returned error: %v", err)
	}

	output := stdoutBuf.String() + stderrBuf.String()

	expectedStrings := []string{
		"cline",
		"Usage:",
		"Flags:",
		"--act",
		"--plan",
		"--yolo",
		"--auto-approve-all",
		"--timeout",
		"--model",
		"--thinking",
		"--reasoning-effort",
		"--max-consecutive-mistakes",
		"--json",
		"--double-check-completion",
		"--auto-condense",
		"--hooks-dir",
		"--acp",
		"--kanban",
		"--taskId",
		"--continue",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help output to contain %q, got:\n%s", expected, output)
		}
	}
}

// TestExecute_TaskMode tests the Execute function in task mode
// Note: This is tested indirectly through TestRunRoot_TaskModeWithArgs
// because Cobra command state persistence makes multiple Execute() calls unreliable in tests
func TestExecute_TaskMode(t *testing.T) {
	// Skip this test - the functionality is covered by TestRunRoot_TaskModeWithArgs
	// which tests the same logic without Cobra state issues
	t.Skip("Covered by TestRunRoot_TaskModeWithArgs - avoids Cobra state persistence issues")
}

// TestExecute_InteractiveMode tests the Execute function in interactive mode
// Note: This is tested indirectly through TestRunRoot_InteractiveModeNoArgs
func TestExecute_InteractiveMode(t *testing.T) {
	// Skip this test - the functionality is covered by TestRunRoot_InteractiveModeNoArgs
	t.Skip("Covered by TestRunRoot_InteractiveModeNoArgs - avoids Cobra state persistence issues")
}

// TestExecute_VerboseFlag tests the verbose flag behavior
// Note: Verbose logging is tested in TestRunRoot_TaskModeWithArgs
func TestExecute_VerboseFlag(t *testing.T) {
	// Skip this test - verbose logging behavior is tested in TestRunRoot_TaskModeWithArgs
	t.Skip("Covered by TestRunRoot_TaskModeWithArgs - avoids Cobra state persistence issues")
}

func TestInitConfig_CustomConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	// Write test config
	content := "test_key: test_value\n"
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Reset viper state
	viper.Reset()
	cfgFile = configFile

	// Initialize config
	initConfig()

	// Verify config was read
	if viper.ConfigFileUsed() != configFile {
		t.Errorf("Expected config file %q, got %q", configFile, viper.ConfigFileUsed())
	}
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		level   slog.Level
	}{
		{
			name:    "non-verbose mode",
			verbose: false,
			level:   slog.LevelInfo,
		},
		{
			name:    "verbose mode",
			verbose: true,
			level:   slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verbose = tt.verbose
			initLogger()

			if logger == nil {
				t.Fatal("Logger should not be nil")
			}

			// Check that logger was set as default
			defaultLogger := slog.Default()
			if defaultLogger == nil {
				t.Fatal("Default logger should not be nil")
			}
		})
	}
}

func TestRunRoot_TaskModeWithArgs(t *testing.T) {
	resetFlags()
	initLogger()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRoot(rootCmd, []string{"test prompt"})

	if err != nil {
		t.Errorf("runRoot() with args returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Task: test prompt") {
		t.Errorf("Expected task mode output, got: %s", output)
	}
}

func TestRunRoot_InteractiveModeNoArgs(t *testing.T) {
	resetFlags()
	initLogger()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRoot(rootCmd, []string{})

	if err != nil {
		t.Errorf("runRoot() with no args returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Interactive mode starting") {
		t.Errorf("Expected interactive mode output, got: %s", output)
	}
}

func TestRootCommandStructure(t *testing.T) {
	// Verify command structure
	if rootCmd.Use != "cline [prompt]" {
		t.Errorf("Expected Use to be 'cline [prompt]', got %q", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Expected Short description to be non-empty")
	}

	if rootCmd.Long == "" {
		t.Error("Expected Long description to be non-empty")
	}

	// Verify global flags exist
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected --config flag to exist")
	}
	if configFlag.Shorthand != "" {
		t.Errorf("Expected config flag to have no shorthand, got %q", configFlag.Shorthand)
	}

	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected --verbose flag to exist")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("Expected verbose flag shorthand to be 'v', got %q", verboseFlag.Shorthand)
	}

	// Verify all new flags exist
	flags := []struct {
		name      string
		shorthand string
	}{
		{"act", "a"},
		{"plan", "p"},
		{"yolo", "y"},
		{"auto-approve-all", ""},
		{"timeout", "t"},
		{"model", "m"},
		{"thinking", ""},
		{"reasoning-effort", ""},
		{"max-consecutive-mistakes", ""},
		{"json", ""},
		{"double-check-completion", ""},
		{"auto-condense", ""},
		{"hooks-dir", ""},
		{"acp", ""},
		{"kanban", ""},
		{"taskId", "T"},
		{"continue", ""},
	}

	for _, flag := range flags {
		f := rootCmd.Flags().Lookup(flag.name)
		if f == nil {
			t.Errorf("Expected --%s flag to exist", flag.name)
			continue
		}
		if flag.shorthand != "" && f.Shorthand != flag.shorthand {
			t.Errorf("Expected %s flag shorthand to be %q, got %q", flag.name, flag.shorthand, f.Shorthand)
		}
	}
}

// Helper function to reset flags between tests
func resetFlags() {
	cfgFile = ""
	verbose = false
	actFlag = false
	planFlag = false
	yoloFlag = false
	autoApproveAllFlag = false
	timeoutFlag = ""
	modelFlag = ""
	thinkingFlag = ""
	reasoningEffortFlag = ""
	maxConsecutiveMistakesFlag = ""
	doubleCheckCompletionFlag = false
	autoCondenseFlag = false
	jsonFlag = false
	hooksDirFlag = ""
	cwdFlag = ""
	acpFlag = false
	kanbanFlag = false
	taskIdFlag = ""
	continueFlag = false
	viper.Reset()
	rootCmd.SetArgs([]string{})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
}

// Test runInteractiveMode directly
func TestRunInteractiveMode(t *testing.T) {
	// Initialize logger for the test
	verbose = false
	initLogger()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &RootOptions{}
	err := runInteractiveMode(opts)
	if err != nil {
		t.Errorf("runInteractiveMode() returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	expected := "Interactive mode starting"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got: %s", expected, output)
	}
}

// Test command with environment variables
func TestInitConfig_EnvironmentVariables(t *testing.T) {
	viper.Reset()

	// Set an environment variable
	os.Setenv("CLINE_TEST_KEY", "test_value")
	defer os.Unsetenv("CLINE_TEST_KEY")

	viper.SetEnvPrefix("CLINE")
	viper.AutomaticEnv()

	// Read the environment variable
	value := viper.GetString("TEST_KEY")
	if value != "test_value" {
		t.Errorf("Expected TEST_KEY to be 'test_value', got %q", value)
	}
}

// Test the Execute function returns nil on successful execution
func TestExecute_Success(t *testing.T) {
	resetFlags()

	// Capture stdout to suppress output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use version flag which always succeeds
	rootCmd.SetArgs([]string{"--version"})

	err := Execute()
	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Drain the pipe
	var buf bytes.Buffer
	buf.ReadFrom(r)
	_ = buf.String()
}

// Benchmark the Execute function
func BenchmarkExecute_Version(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetFlags()
		rootCmd.SetArgs([]string{"--version"})

		// Suppress output during benchmark
		oldStdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)

		_ = Execute()

		os.Stdout = oldStdout
	}
}

// Ensure rootCmd is properly initialized with cobra.Command
func TestRootCmdType(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	// Verify it's a proper cobra command
	var _ *cobra.Command = rootCmd
}

// Test multiple prompts (all args are joined as a single prompt)
func TestExecute_MultipleArgs(t *testing.T) {
	resetFlags()
	initLogger()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with runRoot directly - all args are joined as a single prompt
	err := runRoot(rootCmd, []string{"first prompt", "second prompt", "third prompt"})
	if err != nil {
		t.Errorf("runRoot() with multiple args returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// All args are joined as a single prompt
	if !strings.Contains(output, "first prompt") {
		t.Errorf("Expected output to contain first prompt, got: %s", output)
	}
	if !strings.Contains(output, "second prompt") {
		t.Errorf("Expected output to contain second prompt, got: %s", output)
	}
	if !strings.Contains(output, "third prompt") {
		t.Errorf("Expected output to contain third prompt, got: %s", output)
	}
}

// Test config flag with non-existent file
func TestInitConfig_NonExistentFile(t *testing.T) {
	viper.Reset()
	cfgFile = "/nonexistent/path/config.yaml"

	// Should not panic, just won't read a config
	initConfig()

	// When a custom config file is specified, viper stores the path even if file doesn't exist
	// This is expected behavior - the error would occur on ReadInConfig, not SetConfigFile
	if viper.ConfigFileUsed() != cfgFile {
		t.Errorf("Expected config file to be %q, got %q", cfgFile, viper.ConfigFileUsed())
	}
}

// ==================== NEW FLAG TESTS ====================

func TestValidateRootOptions_ActAndPlanMutuallyExclusive(t *testing.T) {
	resetFlags()
	actFlag = true
	planFlag = true

	cmd := &cobra.Command{}
	_, err := validateRootOptions(cmd, []string{})

	if err == nil {
		t.Error("Expected error when both --act and --plan are set")
	}

	if !strings.Contains(err.Error(), "cannot use both --act and --plan") {
		t.Errorf("Expected error message about mutually exclusive flags, got: %v", err)
	}
}

func TestValidateRootOptions_ReasoningEffort(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		wantErr       bool
		expectedValue string
	}{
		{"valid low", "low", false, "low"},
		{"valid medium", "medium", false, "medium"},
		{"valid high", "high", false, "high"},
		{"valid xhigh", "xhigh", false, "xhigh"},
		{"valid none", "none", false, "none"},
		{"valid uppercase", "HIGH", false, "high"},
		{"valid mixed case", "Medium", false, "medium"},
		{"invalid value", "invalid", true, ""},
		{"empty", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			reasoningEffortFlag = tt.value

			cmd := &cobra.Command{}
			opts, err := validateRootOptions(cmd, []string{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for value %q, got nil", tt.value)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for value %q: %v", tt.value, err)
				return
			}

			if opts.ReasoningEffort != tt.expectedValue {
				t.Errorf("Expected value %q, got %q", tt.expectedValue, opts.ReasoningEffort)
			}
		})
	}
}

func TestValidateRootOptions_Timeout(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErr     bool
		expectedDur time.Duration
	}{
		{"seconds integer", "30", false, 30 * time.Second},
		{"duration seconds", "30s", false, 30 * time.Second},
		{"duration minutes", "5m", false, 5 * time.Minute},
		{"duration hours", "1h", false, 1 * time.Hour},
		{"complex duration", "1h30m", false, 90 * time.Minute},
		{"invalid", "invalid", true, 0},
		{"empty", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			timeoutFlag = tt.value

			cmd := &cobra.Command{}
			opts, err := validateRootOptions(cmd, []string{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for value %q, got nil", tt.value)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for value %q: %v", tt.value, err)
				return
			}

			if opts.Timeout != tt.expectedDur {
				t.Errorf("Expected duration %v, got %v", tt.expectedDur, opts.Timeout)
			}
		})
	}
}

func TestValidateRootOptions_Thinking(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		changed       bool
		wantErr       bool
		expectedVal   *int
		expectDefault bool
	}{
		{"flag not set", "", false, false, nil, false},
		{"flag set no value", "", true, false, nil, true},
		{"valid tokens", "2048", true, false, intPtr(2048), false},
		{"valid zero", "0", true, false, intPtr(0), false},
		{"invalid negative", "-1", true, true, nil, false},
		{"invalid string", "abc", true, true, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			thinkingFlag = tt.value

			cmd := &cobra.Command{}
			cmd.Flags().StringVar(&thinkingFlag, "thinking", "", "")
			if tt.changed {
				cmd.Flags().Set("thinking", tt.value)
			}

			opts, err := validateRootOptions(cmd, []string{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for value %q, got nil", tt.value)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for value %q: %v", tt.value, err)
				return
			}

			if tt.expectDefault {
				if opts.Thinking == nil || *opts.Thinking != 1024 {
					t.Errorf("Expected default 1024, got %v", opts.Thinking)
				}
				return
			}

			if tt.expectedVal == nil {
				if opts.Thinking != nil {
					t.Errorf("Expected nil, got %v", *opts.Thinking)
				}
				return
			}

			if opts.Thinking == nil {
				t.Errorf("Expected %d, got nil", *tt.expectedVal)
				return
			}

			if *opts.Thinking != *tt.expectedVal {
				t.Errorf("Expected %d, got %d", *tt.expectedVal, *opts.Thinking)
			}
		})
	}
}

func TestValidateRootOptions_MaxConsecutiveMistakes(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErr     bool
		expectedVal *int
	}{
		{"valid", "5", false, intPtr(5)},
		{"valid one", "1", false, intPtr(1)},
		{"invalid zero", "0", true, nil},
		{"invalid negative", "-1", true, nil},
		{"invalid string", "abc", true, nil},
		{"empty", "", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			maxConsecutiveMistakesFlag = tt.value

			cmd := &cobra.Command{}
			opts, err := validateRootOptions(cmd, []string{})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for value %q, got nil", tt.value)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for value %q: %v", tt.value, err)
				return
			}

			if tt.expectedVal == nil && opts.MaxConsecutiveMistakes != nil {
				t.Errorf("Expected nil, got %v", *opts.MaxConsecutiveMistakes)
				return
			}

			if tt.expectedVal != nil {
				if opts.MaxConsecutiveMistakes == nil {
					t.Errorf("Expected %d, got nil", *tt.expectedVal)
					return
				}
				if *opts.MaxConsecutiveMistakes != *tt.expectedVal {
					t.Errorf("Expected %d, got %d", *tt.expectedVal, *opts.MaxConsecutiveMistakes)
				}
			}
		})
	}
}

func TestValidateRootOptions_TaskIdAndContinueMutuallyExclusive(t *testing.T) {
	resetFlags()
	taskIdFlag = "task-123"
	continueFlag = true

	cmd := &cobra.Command{}
	_, err := validateRootOptions(cmd, []string{})

	if err == nil {
		t.Error("Expected error when both --taskId and --continue are set")
	}

	if !strings.Contains(err.Error(), "cannot use both --taskId and --continue") {
		t.Errorf("Expected error message about mutually exclusive flags, got: %v", err)
	}
}

func TestValidateRootOptions_KanbanWithPrompt(t *testing.T) {
	resetFlags()
	kanbanFlag = true

	cmd := &cobra.Command{}
	_, err := validateRootOptions(cmd, []string{"some prompt"})

	if err == nil {
		t.Error("Expected error when --kanban is used with a prompt")
	}

	if !strings.Contains(err.Error(), "use --kanban without a prompt") {
		t.Errorf("Expected error message about kanban not taking prompt, got: %v", err)
	}
}

func TestValidateRootOptions_ContinueWithPrompt(t *testing.T) {
	resetFlags()
	continueFlag = true

	cmd := &cobra.Command{}
	_, err := validateRootOptions(cmd, []string{"some prompt"})

	if err == nil {
		t.Error("Expected error when --continue is used with a prompt")
	}

	if !strings.Contains(err.Error(), "use --continue without a prompt") {
		t.Errorf("Expected error message about continue not taking prompt, got: %v", err)
	}
}

func TestValidateRootOptions_AllFlags(t *testing.T) {
	resetFlags()
	actFlag = true
	yoloFlag = true
	autoApproveAllFlag = true
	timeoutFlag = "30m"
	modelFlag = "claude-sonnet-4-6"
	thinkingFlag = "2048"
	reasoningEffortFlag = "high"
	maxConsecutiveMistakesFlag = "5"
	doubleCheckCompletionFlag = true
	autoCondenseFlag = true
	jsonFlag = true
	hooksDirFlag = "/path/to/hooks"
	cwdFlag = "/path/to/cwd"
	acpFlag = false
	kanbanFlag = false
	taskIdFlag = ""
	continueFlag = false

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&thinkingFlag, "thinking", "", "")
	cmd.Flags().Set("thinking", "2048")

	opts, err := validateRootOptions(cmd, []string{"test prompt"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify all flags were parsed correctly
	if !opts.Act {
		t.Error("Expected Act to be true")
	}
	if !opts.Yolo {
		t.Error("Expected Yolo to be true")
	}
	if !opts.AutoApproveAll {
		t.Error("Expected AutoApproveAll to be true")
	}
	if opts.Timeout != 30*time.Minute {
		t.Errorf("Expected Timeout to be 30m, got %v", opts.Timeout)
	}
	if opts.Model != "claude-sonnet-4-6" {
		t.Errorf("Expected Model to be claude-sonnet-4-6, got %s", opts.Model)
	}
	if opts.Thinking == nil || *opts.Thinking != 2048 {
		t.Errorf("Expected Thinking to be 2048, got %v", opts.Thinking)
	}
	if opts.ReasoningEffort != "high" {
		t.Errorf("Expected ReasoningEffort to be high, got %s", opts.ReasoningEffort)
	}
	if opts.MaxConsecutiveMistakes == nil || *opts.MaxConsecutiveMistakes != 5 {
		t.Errorf("Expected MaxConsecutiveMistakes to be 5, got %v", opts.MaxConsecutiveMistakes)
	}
	if !opts.DoubleCheckCompletion {
		t.Error("Expected DoubleCheckCompletion to be true")
	}
	if !opts.AutoCondense {
		t.Error("Expected AutoCondense to be true")
	}
	if !opts.JSON {
		t.Error("Expected JSON to be true")
	}
	if opts.HooksDir != "/path/to/hooks" {
		t.Errorf("Expected HooksDir to be /path/to/hooks, got %s", opts.HooksDir)
	}
	if opts.Cwd != "/path/to/cwd" {
		t.Errorf("Expected Cwd to be /path/to/cwd, got %s", opts.Cwd)
	}
	if opts.Prompt != "test prompt" {
		t.Errorf("Expected Prompt to be 'test prompt', got %s", opts.Prompt)
	}
}

func TestGetTaskMode(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RootOptions
		expected task.TaskMode
	}{
		{"act mode", &RootOptions{Act: true, Plan: false}, task.TaskModeAct},
		{"plan mode", &RootOptions{Act: false, Plan: true}, task.TaskModePlan},
		{"default to act", &RootOptions{Act: false, Plan: false}, task.TaskModeAct},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTaskMode(tt.opts)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRunKanbanMode(t *testing.T) {
	verbose = false
	initLogger()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runKanbanMode()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runKanbanMode() returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Kanban mode") {
		t.Errorf("Expected output to contain 'Kanban mode', got: %s", output)
	}
}

func TestRunAcpMode(t *testing.T) {
	// Skip this test - it's an integration test that blocks waiting for stdin input
	// The ACP server requires actual stdin input to proceed, causing test timeout
	t.Skip("Integration test - requires stdin input, skip to avoid timeout")
}

func TestRunContinueMode(t *testing.T) {
	verbose = false
	initLogger()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &RootOptions{}
	err := runContinueMode(opts)

	w.Close()
	os.Stdout = oldStdout

	// gRPC connection errors are expected in test environment
	// The test should verify the function attempts to run, not that it succeeds
	if err != nil && !strings.Contains(err.Error(), "gRPC connection") {
		t.Errorf("runContinueMode() returned unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Continue mode") {
		t.Errorf("Expected output to contain 'Continue mode', got: %s", output)
	}
}

func TestRunResumeTask(t *testing.T) {
	verbose = false
	initLogger()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := &RootOptions{
		TaskID: "task-123",
	}
	err := runResumeTask(opts)

	w.Close()
	os.Stdout = oldStdout

	// gRPC connection errors are expected in test environment
	// The test should verify the function attempts to run, not that it succeeds
	if err != nil && !strings.Contains(err.Error(), "gRPC connection") {
		t.Errorf("runResumeTask() returned unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "task-123") {
		t.Errorf("Expected output to contain task ID, got: %s", output)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
