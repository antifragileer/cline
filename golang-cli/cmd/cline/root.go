package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/task"
)

const (
	// Version is the CLI version
	Version = "0.1.0"

	// DefaultConfigName is the default configuration file name
	DefaultConfigName = "config"

	// DefaultConfigType is the default configuration file type
	DefaultConfigType = "yaml"

	// ConfigDirName is the name of the configuration directory
	ConfigDirName = "cline"
)

// Valid reasoning effort values
var validReasoningEfforts = []string{"none", "low", "medium", "high", "xhigh"}

// Root command flags
var (
	// Global flags
	cfgFile string
	verbose bool

	// Mode flags
	actFlag  bool
	planFlag bool

	// Auto-approval flags
	yoloFlag           bool
	autoApproveAllFlag bool

	// Execution flags
	timeoutFlag             string
	modelFlag               string
	thinkingFlag            string // Can be boolean or token count
	reasoningEffortFlag     string
	maxConsecutiveMistakesFlag string
	doubleCheckCompletionFlag  bool
	autoCondenseFlag          bool

	// Output flags
	jsonFlag bool

	// Path flags
	hooksDirFlag string
	cwdFlag      string

	// Special mode flags
	acpFlag    bool
	kanbanFlag bool

	// Task management flags
	taskIdFlag    string
	continueFlag  bool

	// Logger instance
	logger *slog.Logger

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "cline [prompt]",
		Short: "Cline CLI - AI-powered coding assistant",
		Long: `Cline is an AI-powered coding assistant that helps you write, edit, and understand code.

With Cline, you can:
  • Ask questions about your codebase
  • Generate new code files and functions
  • Refactor and improve existing code
  • Debug and fix issues

Usage:
  cline "your task or question here"    Execute a single task with the given prompt
  cline                                  Start interactive mode`,
		Example: `  # Execute a single task
  cline "refactor the auth module to use JWT"

  # Start interactive mode
  cline

  # Use act mode with yolo
  cline -a -y "deploy to production"

  # Plan mode with specific model
  cline -p -m claude-sonnet-4-6 "plan the database migration"

  # Resume a task
  cline -T task-123

  # Continue most recent task
  cline --continue

  # Run in ACP mode
  cline --acp

  # Use a custom configuration file
  cline --config /path/to/config.yaml "explain this code"`,
		RunE: runRoot,
		Args: cobra.ArbitraryArgs,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%s/%s.%s)", ConfigDirName, DefaultConfigName, DefaultConfigType))
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Disable flag parsing for the root command to allow arbitrary prompt arguments
	rootCmd.DisableFlagParsing = false

	// Mode flags (mutually exclusive)
	rootCmd.Flags().BoolVarP(&actFlag, "act", "a", false, "Run in act mode (execute actions)")
	rootCmd.Flags().BoolVarP(&planFlag, "plan", "p", false, "Run in plan mode (planning only)")

	// Auto-approval flags
	rootCmd.Flags().BoolVarP(&yoloFlag, "yolo", "y", false, "Enable yes/yolo mode (auto-approve actions)")
	rootCmd.Flags().BoolVar(&autoApproveAllFlag, "auto-approve-all", false, "Enable auto-approve all actions while keeping interactive mode")

	// Execution flags
	rootCmd.Flags().StringVarP(&timeoutFlag, "timeout", "t", "", "Optional timeout in seconds (e.g., 30, 300, 3600)")
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Model to use for the task")
	rootCmd.Flags().StringVar(&thinkingFlag, "thinking", "", "Enable extended thinking (default: 1024 tokens, or specify token count)")
	rootCmd.Flags().StringVar(&reasoningEffortFlag, "reasoning-effort", "", "Reasoning effort: none|low|medium|high|xhigh")
	rootCmd.Flags().StringVar(&maxConsecutiveMistakesFlag, "max-consecutive-mistakes", "", "Maximum consecutive mistakes before halting in yolo mode")
	rootCmd.Flags().BoolVar(&doubleCheckCompletionFlag, "double-check-completion", false, "Reject first completion attempt to force re-verification")
	rootCmd.Flags().BoolVar(&autoCondenseFlag, "auto-condense", false, "Enable AI-powered context compaction instead of mechanical truncation")

	// Output flags
	rootCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output messages as JSON instead of styled text")

	// Path flags
	rootCmd.Flags().StringVar(&hooksDirFlag, "hooks-dir", "", "Path to additional hooks directory for runtime hook injection")
	rootCmd.Flags().StringVarP(&cwdFlag, "cwd", "c", "", "Working directory for the task")

	// Special mode flags
	rootCmd.Flags().BoolVar(&acpFlag, "acp", false, "Run in ACP (Agent Client Protocol) mode for editor integration")
	rootCmd.Flags().BoolVar(&kanbanFlag, "kanban", false, "Run npx kanban@latest --agent cline")

	// Task management flags
	rootCmd.Flags().StringVarP(&taskIdFlag, "taskId", "T", "", "Resume an existing task by ID")
	rootCmd.Flags().BoolVar(&continueFlag, "continue", false, "Resume the most recent task from the current working directory")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Version flag handling
	rootCmd.SetVersionTemplate(fmt.Sprintf("Cline CLI version %s\n", Version))
	rootCmd.Version = Version
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cline" (without extension).
		configDir := filepath.Join(home, "."+ConfigDirName)
		viper.AddConfigPath(configDir)
		viper.SetConfigName(DefaultConfigName)
		viper.SetConfigType(DefaultConfigType)
	}

	viper.SetEnvPrefix("CLINE")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
	}
}

// initLogger initializes the global logger
func initLogger() {
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// RootOptions holds all parsed root command options
type RootOptions struct {
	// Mode
	Act  bool
	Plan bool

	// Auto-approval
	Yolo           bool
	AutoApproveAll bool

	// Execution
	Timeout             time.Duration
	Model               string
	Thinking            *int // nil = not set, 0 = enabled with default, >0 = specific token count
	ReasoningEffort     string
	MaxConsecutiveMistakes *int
	DoubleCheckCompletion bool
	AutoCondense          bool

	// Output
	JSON bool

	// Paths
	HooksDir string
	Cwd      string
	Config   string

	// Special modes
	Acp    bool
	Kanban bool

	// Task management
	TaskID   string
	Continue bool

	// Prompt
	Prompt string
}

// validateRootOptions validates flag combinations and values
func validateRootOptions(cmd *cobra.Command, args []string) (*RootOptions, error) {
	opts := &RootOptions{
		Act:                   actFlag,
		Plan:                  planFlag,
		Yolo:                  yoloFlag,
		AutoApproveAll:        autoApproveAllFlag,
		Model:                 modelFlag,
		ReasoningEffort:       reasoningEffortFlag,
		DoubleCheckCompletion: doubleCheckCompletionFlag,
		AutoCondense:          autoCondenseFlag,
		JSON:                  jsonFlag,
		HooksDir:              hooksDirFlag,
		Cwd:                   cwdFlag,
		Config:                cfgFile,
		Acp:                   acpFlag,
		Kanban:                kanbanFlag,
		TaskID:                taskIdFlag,
		Continue:              continueFlag,
	}

	// Validate mutually exclusive flags
	if opts.Act && opts.Plan {
		return nil, fmt.Errorf("cannot use both --act and --plan flags")
	}

	// Validate reasoning effort
	if opts.ReasoningEffort != "" {
		found := false
		for _, valid := range validReasoningEfforts {
			if strings.EqualFold(opts.ReasoningEffort, valid) {
				found = true
				opts.ReasoningEffort = valid // Normalize to lowercase
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid --reasoning-effort '%s'. Valid values: %s", opts.ReasoningEffort, strings.Join(validReasoningEfforts, ", "))
		}
	}

	// Parse timeout
	if timeoutFlag != "" {
		// Try parsing as integer seconds first
		if seconds, err := strconv.Atoi(timeoutFlag); err == nil {
			opts.Timeout = time.Duration(seconds) * time.Second
		} else {
			// Try parsing as duration string
			duration, err := time.ParseDuration(timeoutFlag)
			if err != nil {
				return nil, fmt.Errorf("invalid --timeout format '%s': expected seconds (e.g., '30') or duration (e.g., '30s', '5m', '1h')", timeoutFlag)
			}
			opts.Timeout = duration
		}
	}

	// Parse thinking flag
	if cmd.Flags().Changed("thinking") {
		if thinkingFlag == "" {
			// --thinking without value
			defaultTokens := 1024
			opts.Thinking = &defaultTokens
		} else {
			// --thinking with token count
			tokens, err := strconv.Atoi(thinkingFlag)
			if err != nil || tokens < 0 {
				return nil, fmt.Errorf("invalid --thinking value '%s': expected non-negative integer", thinkingFlag)
			}
			opts.Thinking = &tokens
		}
	}

	// Parse max consecutive mistakes
	if maxConsecutiveMistakesFlag != "" {
		count, err := strconv.Atoi(maxConsecutiveMistakesFlag)
		if err != nil || count < 1 {
			return nil, fmt.Errorf("invalid --max-consecutive-mistakes value '%s': expected integer >= 1", maxConsecutiveMistakesFlag)
		}
		opts.MaxConsecutiveMistakes = &count
	}

	// Validate taskId and continue are mutually exclusive
	if opts.TaskID != "" && opts.Continue {
		return nil, fmt.Errorf("cannot use both --taskId and --continue flags")
	}

	// Validate kanban doesn't take a prompt
	if opts.Kanban && len(args) > 0 {
		return nil, fmt.Errorf("use --kanban without a prompt")
	}

	// Validate continue doesn't take a prompt
	if opts.Continue && len(args) > 0 {
		return nil, fmt.Errorf("use --continue without a prompt")
	}

	// Validate continue doesn't work with piped input
	if opts.Continue {
		// Check if stdin is piped
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			return nil, fmt.Errorf("use --continue without piped input")
		}
	}

	// Set prompt from args
	if len(args) > 0 {
		opts.Prompt = strings.Join(args, " ")
	}

	return opts, nil
}

// runRoot executes the root command logic
func runRoot(cmd *cobra.Command, args []string) error {
	// Validate and parse options
	opts, err := validateRootOptions(cmd, args)
	if err != nil {
		return err
	}

	// Handle kanban mode
	if opts.Kanban {
		return runKanbanMode()
	}

	// Handle ACP mode
	if opts.Acp {
		return runAcpMode(opts)
	}

	// Handle continue mode
	if opts.Continue {
		return runContinueMode(opts)
	}

	// Handle task resumption
	if opts.TaskID != "" {
		return runResumeTask(opts)
	}

	// Handle task with prompt
	if opts.Prompt != "" {
		return runTaskWithPrompt(opts)
	}

	// No prompt provided, start interactive mode
	return runInteractiveMode(opts)
}

// runKanbanMode runs the kanban alias mode
func runKanbanMode() error {
	logger.Info("running kanban mode")
	fmt.Println("Kanban mode: would run 'npx kanban@latest --agent cline'")
	// TODO: Implement actual kanban execution
	return nil
}

// runAcpMode runs in ACP (Agent Client Protocol) mode
func runAcpMode(opts *RootOptions) error {
	logger.Info("running ACP mode", "cwd", opts.Cwd, "hooksDir", opts.HooksDir)
	fmt.Println("ACP mode: Agent Client Protocol integration")
	// TODO: Implement ACP mode
	return nil
}

// runContinueMode resumes the most recent task
func runContinueMode(opts *RootOptions) error {
	logger.Info("continuing most recent task")
	fmt.Println("Continue mode: resuming most recent task")
	// TODO: Implement continue logic - find most recent task and resume
	return nil
}

// runResumeTask resumes a specific task by ID
func runResumeTask(opts *RootOptions) error {
	logger.Info("resuming task", "taskId", opts.TaskID)
	fmt.Printf("Resume task: %s\n", opts.TaskID)
	// TODO: Implement task resumption logic
	return nil
}

// runTaskWithPrompt runs a task with the given prompt
func runTaskWithPrompt(opts *RootOptions) error {
	logger.Info("running task",
		"prompt", opts.Prompt,
		"act", opts.Act,
		"plan", opts.Plan,
		"yolo", opts.Yolo,
		"model", opts.Model,
	)

	// For now, always use gRPC mode
	return runTaskWithGRPC(opts)
}

// runTaskWithGRPC runs a task using gRPC connection to the core extension
func runTaskWithGRPC(opts *RootOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add timeout if specified
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Resolve gRPC endpoint
	resolver := host.NewEndpointResolver("")
	endpointConfig, err := resolver.Resolve()
	if err != nil {
		return fmt.Errorf("failed to resolve gRPC endpoint: %w", err)
	}

	// Create connection manager
	cm := host.NewConnectionManager(endpointConfig)
	if err := cm.ConnectWithRetry(ctx, 3); err != nil {
		return fmt.Errorf("failed to connect to Cline core extension at %s: %w", endpointConfig.Address, err)
	}
	defer cm.Close()

	// Verify connection is healthy
	if err := cm.HealthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Connected to Cline core extension at %s\n", endpointConfig.Address)
	}

	// Create task runner
	runner := task.NewRunner(cm.GetConnection())

	// Build task config
	config := task.Config{
		Mode:                    getTaskModeAsTaskMode(opts),
		Yolo:                    opts.Yolo,
		Timeout:                 opts.Timeout,
		Model:                   opts.Model,
		Verbose:                 verbose,
		Cwd:                     opts.Cwd,
		Thinking:                opts.Thinking != nil,
		JSON:                    opts.JSON,
		TaskID:                  opts.TaskID,
		Prompt:                  opts.Prompt,
		AutoApproveAll:          opts.AutoApproveAll,
		ReasoningEffort:         opts.ReasoningEffort,
		DoubleCheckCompletion:   opts.DoubleCheckCompletion,
		AutoCondense:            opts.AutoCondense,
		HooksDir:                opts.HooksDir,
	}

	// Add thinking budget if specified
	if opts.Thinking != nil {
		config.ThinkingBudget = *opts.Thinking
	}

	// Add max consecutive mistakes if specified
	if opts.MaxConsecutiveMistakes != nil {
		config.MaxConsecutiveMistakes = *opts.MaxConsecutiveMistakes
	}

	// Create message handler based on output mode
	var handler task.MessageHandler
	if opts.JSON {
		handler = &task.JSONHandler{
			Output: os.Stdout,
		}
	} else {
		handler = &task.PlainTextHandler{
			Verbose:     verbose,
			JSONOutput:  opts.JSON,
			Output:      os.Stdout,
			AutoApprove: opts.Yolo || opts.AutoApproveAll,
		}
	}

	// Run the task with streaming
	if err := runner.RunWithStreaming(ctx, config, handler); err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}

// getTaskModeAsTaskMode converts RootOptions mode to task.Mode
func getTaskModeAsTaskMode(opts *RootOptions) task.Mode {
	if opts.Plan {
		return task.ModePlan
	}
	return task.ModeAct
}

// isTTY checks if stdout is a terminal
func isTTY() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// runInteractiveMode starts the interactive TUI
func runInteractiveMode(opts *RootOptions) error {
	logger.Info("starting interactive mode",
		"act", opts.Act,
		"plan", opts.Plan,
		"yolo", opts.Yolo,
		"model", opts.Model,
	)

	// Apply CLI flags even in interactive mode
	// This ensures flags like --yolo affect the initial TUI state
	fmt.Println("Interactive mode starting with options:")
	if opts.Act {
		fmt.Println("  - Act mode")
	}
	if opts.Plan {
		fmt.Println("  - Plan mode")
	}
	if opts.Yolo {
		fmt.Println("  - Yolo mode enabled")
	}
	if opts.Model != "" {
		fmt.Printf("  - Model: %s\n", opts.Model)
	}
	if opts.AutoApproveAll {
		fmt.Println("  - Auto-approve all enabled")
	}

	// TODO: Implement actual interactive TUI
	fmt.Println("\nInteractive mode not yet fully implemented.")

	return nil
}

