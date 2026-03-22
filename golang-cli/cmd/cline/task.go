package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// TaskMode represents the execution mode for a task
type TaskMode string

const (
	// TaskModeAct represents act mode (execute actions)
	TaskModeAct TaskMode = "act"
	// TaskModePlan represents plan mode (planning only)
	TaskModePlan TaskMode = "plan"
)

// TaskConfig holds all configuration options for a task
type TaskConfig struct {
	// Mode specifies whether to run in act or plan mode
	Mode TaskMode `json:"mode"`

	// Yolo enables auto-approval without confirmation
	Yolo bool `json:"yolo"`

	// Timeout is the maximum duration for task execution
	Timeout time.Duration `json:"timeout"`

	// Model specifies the model to use for the task
	Model string `json:"model"`

	// Images is a list of image file paths to attach
	Images []string `json:"images"`

	// Verbose enables verbose output
	Verbose bool `json:"verbose"`

	// Cwd is the current working directory for the task
	Cwd string `json:"cwd"`

	// ConfigPath is the path to a custom configuration file
	ConfigPath string `json:"configPath"`

	// Thinking enables thinking mode
	Thinking bool `json:"thinking"`

	// JSON enables JSON output format
	JSON bool `json:"json"`

	// TaskID specifies a task ID to resume or reference
	TaskID string `json:"taskId"`

	// Prompt is the task prompt/message
	Prompt string `json:"prompt"`
}

// TaskRunner defines the interface for executing tasks
type TaskRunner interface {
	Run(config TaskConfig) error
}

// DefaultTaskRunner is the default implementation of TaskRunner
type DefaultTaskRunner struct {
	output io.Writer
}

// NewDefaultTaskRunner creates a new DefaultTaskRunner
func NewDefaultTaskRunner(output io.Writer) *DefaultTaskRunner {
	if output == nil {
		output = os.Stdout
	}
	return &DefaultTaskRunner{output: output}
}

// Run executes a task with the given configuration
func (r *DefaultTaskRunner) Run(config TaskConfig) error {
	if config.Verbose && !config.JSON {
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
			fmt.Fprintf(r.output, "Images: %s\n", strings.Join(config.Images, ", "))
		}
		if config.Cwd != "" {
			fmt.Fprintf(r.output, "Working directory: %s\n", config.Cwd)
		}
		if config.ConfigPath != "" {
			fmt.Fprintf(r.output, "Config file: %s\n", config.ConfigPath)
		}
		if config.Thinking {
			fmt.Fprintln(r.output, "Thinking mode enabled")
		}
		if config.TaskID != "" {
			fmt.Fprintf(r.output, "Task ID: %s\n", config.TaskID)
		}
	}

	// Validate images exist and are readable
	for _, img := range config.Images {
		if _, err := os.Stat(img); err != nil {
			return fmt.Errorf("image file not accessible: %s: %w", img, err)
		}
	}

	// Validate config file exists if specified
	if config.ConfigPath != "" {
		if _, err := os.Stat(config.ConfigPath); err != nil {
			return fmt.Errorf("config file not found: %s: %w", config.ConfigPath, err)
		}
	}

	// Output JSON if requested
	if config.JSON {
		output := map[string]interface{}{
			"mode":       config.Mode,
			"yolo":       config.Yolo,
			"timeout":    config.Timeout.String(),
			"model":      config.Model,
			"images":     config.Images,
			"cwd":        config.Cwd,
			"configPath": config.ConfigPath,
			"thinking":   config.Thinking,
			"taskId":     config.TaskID,
			"prompt":     config.Prompt,
			"status":     "started",
		}
		encoder := json.NewEncoder(r.output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Normal output
	fmt.Fprintf(r.output, "Task: %s\n", config.Prompt)
	fmt.Fprintln(r.output, "Status: Started successfully")

	return nil
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

// taskFlags holds the parsed flag values
var taskFlags struct {
	act      bool
	plan     bool
	yolo     bool
	timeout  string
	model    string
	images   []string
	cwd      string
	config   string
	thinking bool
	json     bool
	taskId   string
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

	// Create runner and execute
	runner := NewDefaultTaskRunner(cmd.OutOrStdout())
	return runner.Run(config)
}

// buildTaskConfig builds TaskConfig from parsed flags
func buildTaskConfig() (TaskConfig, error) {
	config := TaskConfig{
		Yolo:       taskFlags.yolo,
		Model:      taskFlags.model,
		Images:     taskFlags.images,
		Cwd:        taskFlags.cwd,
		ConfigPath: taskFlags.config,
		Thinking:   taskFlags.thinking,
		JSON:       taskFlags.json,
		TaskID:     taskFlags.taskId,
	}

	// Determine mode (mutually exclusive, default to act)
	if taskFlags.plan {
		config.Mode = TaskModePlan
	} else {
		config.Mode = TaskModeAct
	}

	// Parse timeout
	if taskFlags.timeout != "" {
		duration, err := time.ParseDuration(taskFlags.timeout)
		if err != nil {
			return TaskConfig{}, fmt.Errorf("invalid timeout format: %s", taskFlags.timeout)
		}
		config.Timeout = duration
	}

	// Use global verbose flag if set
	if verbose {
		config.Verbose = true
	}

	return config, nil
}

// validateTaskConfig validates the task configuration
func validateTaskConfig(config TaskConfig) error {
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
	if config.ConfigPath != "" {
		if _, err := os.Stat(config.ConfigPath); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("config file not found: %s", config.ConfigPath))
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
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// ValidateImageFile validates that an image file exists and is readable
func ValidateImageFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("image file does not exist: %s", path)
		}
		return fmt.Errorf("cannot access image file: %s: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("image path is a directory, not a file: %s", path)
	}

	// Check file extension for common image types
	ext := strings.ToLower(filepath.Ext(path))
	validExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
		".svg":  true,
	}

	if !validExts[ext] {
		return fmt.Errorf("unsupported image format: %s (supported: png, jpg, jpeg, gif, webp, bmp, svg)", ext)
	}

	return nil
}

// ExpandPath expands a path that may contain ~ (home directory)
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}