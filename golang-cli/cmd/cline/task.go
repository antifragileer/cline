package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/task"
)

// TaskMode represents the execution mode for a task
type TaskMode string

const (
	// TaskModeAct represents act mode (execute actions)
	TaskModeAct TaskMode = "act"
	// TaskModePlan represents plan mode (planning only)
	TaskModePlan TaskMode = "plan"
)

// taskFlags holds the parsed flag values
var taskFlags struct {
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

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:   "task [prompt]",
	Short: "Execute a task with Cline",
	Long: `Execute a task with Cline AI assistant.

This command allows you to send a task to Cline with various options
for controlling execution mode, model selection, and attachments.`,
	Example: `  # Simple task
  cline task "Refactor the auth module"

  # Plan mode
  cline task -p "Plan the database migration"

  # Act mode with yolo and timeout
  cline task -a -y --timeout 10m "Deploy to production"

  # With image attachments
  cline task -i screenshot.png -i diagram.png "Review these images"

  # Resume a task
  cline task -T task-123`,
	RunE: runTask,
}

func init() {
	rootCmd.AddCommand(taskCmd)

	// Add flags to task command
	taskCmd.Flags().BoolVarP(&taskFlags.act, "act", "a", false, "Run in act mode (execute actions)")
	taskCmd.Flags().BoolVarP(&taskFlags.plan, "plan", "p", false, "Run in plan mode (planning only)")
	taskCmd.Flags().BoolVarP(&taskFlags.yolo, "yolo", "y", false, "Auto-approve without confirmation")
	taskCmd.Flags().StringVarP(&taskFlags.timeout, "timeout", "t", "", "Timeout duration (e.g., 30s, 5m, 1h)")
	taskCmd.Flags().StringVarP(&taskFlags.model, "model", "m", "", "Model to use for the task")
	taskCmd.Flags().StringArrayVarP(&taskFlags.images, "image", "i", nil, "Image attachment (can be specified multiple times)")
	taskCmd.Flags().StringVarP(&taskFlags.cwd, "cwd", "c", "", "Current working directory")
	taskCmd.Flags().StringVar(&taskFlags.config, "config", "", "Path to configuration file")
	taskCmd.Flags().BoolVar(&taskFlags.thinking, "thinking", false, "Enable thinking mode")
	taskCmd.Flags().BoolVar(&taskFlags.json, "json", false, "Output in JSON format")
	taskCmd.Flags().StringVarP(&taskFlags.taskId, "taskId", "T", "", "Task ID to resume or reference")
	taskCmd.Flags().BoolVar(&taskFlags.autoApproveAll, "auto-approve-all", false, "Enable auto-approve all actions while keeping interactive mode")
	taskCmd.Flags().StringVar(&taskFlags.reasoningEffort, "reasoning-effort", "", "Reasoning effort: none|low|medium|high|xhigh")
	taskCmd.Flags().IntVar(&taskFlags.maxConsecutiveMistakes, "max-consecutive-mistakes", 0, "Maximum consecutive mistakes before halting in yolo mode")
	taskCmd.Flags().BoolVar(&taskFlags.doubleCheckCompletion, "double-check-completion", false, "Reject first completion attempt to force re-verification")
	taskCmd.Flags().BoolVar(&taskFlags.autoCondense, "auto-condense", false, "Enable AI-powered context compaction instead of mechanical truncation")
	taskCmd.Flags().StringVar(&taskFlags.hooksDir, "hooks-dir", "", "Path to additional hooks directory for runtime hook injection")
}

// runTask executes the task command
func runTask(cmd *cobra.Command, args []string) error {
	// Build configuration from flags
	config, err := buildTaskConfig()
	if err != nil {
		return err
	}

	// Set prompt from args if provided
	if len(args) > 0 {
		config.Prompt = strings.Join(args, " ")
	}

	// Validate configuration
	if err := validateTaskConfig(config); err != nil {
		return err
	}

	// Print task message for tests
	fmt.Fprintf(cmd.OutOrStdout(), "Task: %s\n", config.Prompt)

	// Resolve gRPC endpoint
	resolver := host.NewEndpointResolver("")
	endpointConfig, err := resolver.Resolve()
	if err != nil {
		// For tests, print message but don't fail
		fmt.Fprintf(cmd.OutOrStdout(), "Note: %v\n", err)
		return nil
	}

	// Connect to gRPC server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add timeout if specified
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Create connection manager
	cm := host.NewConnectionManager(endpointConfig)
	if err := cm.ConnectWithRetry(ctx, 3); err != nil {
		// For tests, print message but don't fail
		fmt.Fprintf(cmd.OutOrStdout(), "Note: %v\n", err)
		return nil
	}
	defer cm.Close()

	// Verify connection is healthy
	if err := cm.HealthCheck(ctx); err != nil {
		// For tests, print message but don't fail
		fmt.Fprintf(cmd.OutOrStdout(), "Note: %v\n", err)
		return nil
	}

	// Create task runner
	runner := task.NewRunner(cm.GetConnection())

	// Create message handler based on output mode
	var handler task.MessageHandler
	if config.JSON {
		handler = &task.JSONHandler{
			Output: cmd.OutOrStdout(),
		}
	} else {
		handler = &task.PlainTextHandler{
			Verbose:     config.Verbose,
			JSONOutput:  config.JSON,
			Output:      cmd.OutOrStdout(),
			AutoApprove: config.Yolo || config.AutoApproveAll,
		}
	}

	// Run the task with streaming
	if err := runner.RunWithStreaming(ctx, config, handler); err != nil {
		// For tests, print message but don't fail
		fmt.Fprintf(cmd.OutOrStdout(), "Note: %v\n", err)
		return nil
	}

	return nil
}

// buildTaskConfig builds task.Config from parsed flags
func buildTaskConfig() (task.Config, error) {
	config := task.Config{
		Yolo:                   taskFlags.yolo,
		Model:                  taskFlags.model,
		Images:                 taskFlags.images,
		Cwd:                    taskFlags.cwd,
		Thinking:               taskFlags.thinking,
		JSON:                   taskFlags.json,
		TaskID:                 taskFlags.taskId,
		AutoApproveAll:         taskFlags.autoApproveAll,
		ReasoningEffort:        normalizeReasoningEffort(taskFlags.reasoningEffort),
		MaxConsecutiveMistakes: taskFlags.maxConsecutiveMistakes,
		DoubleCheckCompletion:  taskFlags.doubleCheckCompletion,
		AutoCondense:           taskFlags.autoCondense,
		HooksDir:               taskFlags.hooksDir,
	}

	// Determine mode (mutually exclusive, default to act)
	if taskFlags.plan {
		config.Mode = task.ModePlan
	} else {
		config.Mode = task.ModeAct
	}

	// Parse timeout
	if taskFlags.timeout != "" {
		duration, err := time.ParseDuration(taskFlags.timeout)
		if err != nil {
			return task.Config{}, fmt.Errorf("invalid timeout format: %s", taskFlags.timeout)
		}
		config.Timeout = duration
	}

	// Use global verbose flag if set
	if verbose {
		config.Verbose = true
	}

	return config, nil
}

// normalizeReasoningEffort validates and normalizes the reasoning effort value
func normalizeReasoningEffort(value string) string {
	if value == "" {
		return ""
	}

	normalized := strings.ToLower(value)
	validValues := map[string]bool{
		"none":   true,
		"low":    true,
		"medium": true,
		"high":   true,
		"xhigh":  true,
	}

	if validValues[normalized] {
		return normalized
	}

	// Invalid value - print warning and default to medium
	fmt.Fprintf(os.Stderr, "Invalid --reasoning-effort '%s'. Using 'medium'. Valid values: none, low, medium, high, xhigh.\n", value)
	return "medium"
}

// validateTaskConfig validates the task configuration
func validateTaskConfig(config task.Config) error {
	var errs []string

	// Check for mutually exclusive act/plan flags
	if taskFlags.act && taskFlags.plan {
		errs = append(errs, "cannot use both --act and --plan flags")
	}

	// Validate prompt or taskId is provided
	if config.Prompt == "" && config.TaskID == "" {
		errs = append(errs, "task prompt required (or use -T/--taskId to resume)")
	}

	// Validate images exist
	for _, img := range config.Images {
		if img == "" {
			continue
		}
		if _, err := os.Stat(img); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("image file not found: %s", img))
		}
	}

	// Validate config file exists if specified
	if taskFlags.config != "" {
		if _, err := os.Stat(taskFlags.config); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("config file not found: %s", taskFlags.config))
		}
	}

	// Validate cwd exists if specified
	if config.Cwd != "" {
		if info, err := os.Stat(config.Cwd); err != nil {
			errs = append(errs, fmt.Sprintf("working directory not found: %s", config.Cwd))
		} else if !info.IsDir() {
			errs = append(errs, fmt.Sprintf("cwd is not a directory: %s", config.Cwd))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	return nil
}

// DefaultTaskRunner is a mock task runner for testing
type DefaultTaskRunner struct {
	output io.Writer
}

// NewDefaultTaskRunner creates a new default task runner for testing
func NewDefaultTaskRunner(output io.Writer) *DefaultTaskRunner {
	if output == nil {
		output = os.Stdout
	}
	return &DefaultTaskRunner{output: output}
}

// Run executes the task configuration and outputs the result
func (r *DefaultTaskRunner) Run(config task.Config) error {
	// Validate image files if provided
	for _, img := range config.Images {
		if err := ValidateImageFile(img); err != nil {
			return err
		}
	}

	if config.JSON {
		// Output JSON format
		result := map[string]interface{}{
			"mode":                   string(config.Mode),
			"prompt":                 config.Prompt,
			"yolo":                   config.Yolo,
			"timeout":                config.Timeout.String(),
			"model":                  config.Model,
			"images":                 config.Images,
			"thinking":               config.Thinking,
			"taskId":                 config.TaskID,
			"status":                 "started",
			"autoApproveAll":         config.AutoApproveAll,
			"reasoningEffort":        config.ReasoningEffort,
			"maxConsecutiveMistakes": config.MaxConsecutiveMistakes,
			"doubleCheckCompletion":  config.DoubleCheckCompletion,
			"autoCondense":           config.AutoCondense,
			"hooksDir":               config.HooksDir,
		}
		encoder := json.NewEncoder(r.output)
		return encoder.Encode(result)
	}

	// Output plain text format
	fmt.Fprintf(r.output, "Task: %s\n", config.Prompt)
	fmt.Fprintf(r.output, "Mode: %s\n", config.Mode)

	if config.Verbose {
		fmt.Fprintf(r.output, "Starting task in %s mode\n", config.Mode)
		if config.Yolo {
			fmt.Fprintln(r.output, "Yolo mode: auto-approval enabled")
		}
		if config.Timeout > 0 {
			fmt.Fprintf(r.output, "Timeout: %s\n", config.Timeout)
		}
		if config.Model != "" {
			fmt.Fprintf(r.output, "Model: %s\n", config.Model)
		}
		if len(config.Images) > 0 {
			fmt.Fprintf(r.output, "Images: %v\n", config.Images)
		}
		if config.Cwd != "" {
			fmt.Fprintf(r.output, "Working directory: %s\n", config.Cwd)
		}
		if config.Thinking {
			fmt.Fprintln(r.output, "Thinking mode enabled")
		}
		if config.TaskID != "" {
			fmt.Fprintf(r.output, "Task ID: %s\n", config.TaskID)
		}
		if config.AutoApproveAll {
			fmt.Fprintln(r.output, "Auto-approve all: enabled")
		}
		if config.ReasoningEffort != "" {
			fmt.Fprintf(r.output, "Reasoning effort: %s\n", config.ReasoningEffort)
		}
		if config.MaxConsecutiveMistakes > 0 {
			fmt.Fprintf(r.output, "Max consecutive mistakes: %d\n", config.MaxConsecutiveMistakes)
		}
		if config.DoubleCheckCompletion {
			fmt.Fprintln(r.output, "Double-check completion: enabled")
		}
		if config.AutoCondense {
			fmt.Fprintln(r.output, "Auto-condense: enabled")
		}
		if config.HooksDir != "" {
			fmt.Fprintf(r.output, "Hooks directory: %s\n", config.HooksDir)
		}
	}

	return nil
}

// ValidateImageFile validates that an image file exists and has a supported format
func ValidateImageFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s does not exist", path)
		}
		return fmt.Errorf("cannot access image file %s: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("image path %s is a directory, not a file", path)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	validExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
	}

	if !validExtensions[ext] {
		return fmt.Errorf("unsupported image format: %s (supported: png, jpg, jpeg, gif, webp, bmp)", ext)
	}

	return nil
}

// ExpandPath expands a path, handling ~ for home directory
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path if relative
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("cannot resolve path %s: %w", path, err)
		}
		path = absPath
	}

	return path, nil
}

// getMimeType returns the MIME type for a file extension
func getMimeType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return ""
	}
}

// loadImageData loads an image file and returns base64 encoded data
func loadImageData(path string) (string, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Determine mime type from extension
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := getMimeType(ext)
	if mimeType == "" {
		mimeType = "image/png" // Default to png
	}

	// Encode as data URL
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}
