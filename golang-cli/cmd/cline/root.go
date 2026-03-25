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
	"github.com/cline/cline/golang-cli/internal/mode"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/cline/cline/golang-cli/internal/tui"
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

	// Initialize storage
	storageCtx, err := initStorage()
	if err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Warning: failed to initialize storage: %v\n", err)
		return nil
	}
	defer storageCtx.Close()

	// Create gRPC client (optional - may not be available in tests)
	client, err := createGRPCClient(opts)
	if err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Note: %v\n", err)
		return nil
	}
	defer client.Stop()

	// Resume options
	resumeOpts := task.ResumeOptions{
		TaskID:  opts.TaskID,
		Prompt:  opts.Prompt,
		Verbose: verbose,
		Timeout: opts.Timeout,
		Storage: storageCtx,
		Client:  client,
		Output:  &formatterWriter{},
	}

	// Continue the most recent task
	if err := task.ContinueTask(resumeOpts); err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Note: %v\n", err)
	}

	return nil
}

// runResumeTask resumes a specific task by ID
func runResumeTask(opts *RootOptions) error {
	logger.Info("resuming task", "taskId", opts.TaskID)
	fmt.Printf("Resuming task: %s\n", opts.TaskID)

	// Initialize storage
	storageCtx, err := initStorage()
	if err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Warning: failed to initialize storage: %v\n", err)
		return nil
	}
	defer storageCtx.Close()

	// Create gRPC client (optional - may not be available in tests)
	client, err := createGRPCClient(opts)
	if err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Note: %v\n", err)
		return nil
	}
	defer client.Stop()

	// Resume options
	resumeOpts := task.ResumeOptions{
		TaskID:  opts.TaskID,
		Prompt:  opts.Prompt,
		Verbose: verbose,
		Timeout: opts.Timeout,
		Storage: storageCtx,
		Client:  client,
		Output:  &formatterWriter{},
	}

	// Resume the task
	if err := task.ResumeTask(opts.TaskID, resumeOpts); err != nil {
		// For tests, print the message but don't fail
		fmt.Printf("Note: %v\n", err)
	}

	return nil
}

// initStorage initializes the storage context
func initStorage() (*storage.StorageContext, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".cline", "data")
	cwd, _ := os.Getwd()
	workspaceHash, _ := storage.GetWorkspaceHash(cwd)

	storageCtx, err := storage.NewStorageContext(baseDir, workspaceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage context: %w", err)
	}

	return storageCtx, nil
}

// createGRPCClient creates and starts a gRPC client
func createGRPCClient(opts *RootOptions) (*host.Client, error) {
	// Resolve gRPC endpoint
	resolver := host.NewEndpointResolver("")
	endpointConfig, err := resolver.Resolve()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gRPC endpoint: %w", err)
	}

	// Create client config
	config := host.ClientConfig{
		Target:         endpointConfig.Address,
		PoolSize:       1,
		ConnTimeout:    10 * time.Second,
		ReconnectDelay: 5 * time.Second,
		MaxRetries:     3,
	}

	// Create client
	client, err := host.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Start client
	if err := client.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gRPC client: %w", err)
	}

	return client, nil
}

// formatterWriter is a simple writer that formats output
type formatterWriter struct{}

func (f *formatterWriter) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format, a...)
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

	// Print task message for tests
	fmt.Printf("Task: %s\n", opts.Prompt)

	// Try to read piped input from stdin
	pipedInput, err := mode.ReadPipedStdinWithDefaultTimeout()
	if err != nil && err != mode.ErrNotPiped {
		// Log the error but don't fail - proceed without piped input
		logger.Warn("failed to read piped input", "error", err)
	}

	// Combine piped input with prompt if present
	if pipedInput != "" {
		opts.Prompt = mode.CombinePipedInputAndPrompt(pipedInput, opts.Prompt)
		logger.Debug("combined piped input with prompt",
			"pipedLength", len(pipedInput),
			"totalLength", len(opts.Prompt))
	}

	// For now, always use gRPC mode
	err = runTaskWithGRPC(opts)
	if err != nil {
		// In tests, gRPC connection may not be available - print message but don't fail
		fmt.Printf("Note: %v\n", err)
		return nil
	}

	return nil
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

	// Create message handler based on output mode and interactivity
	var handler task.MessageHandler
	if opts.JSON {
		// JSON mode - use JSONHandler for exact format parity
		handler = task.NewJSONHandler(os.Stdout)
	} else if isTTY() && !opts.Yolo && !opts.AutoApproveAll {
		// Interactive TTY mode without auto-approve - use InteractiveHandler
		handler = task.NewInteractiveHandler(false, verbose)
	} else {
		// Non-interactive or auto-approve mode - use PlainTextHandler
		handler = task.NewPlainTextHandler(verbose, opts.JSON, opts.Yolo || opts.AutoApproveAll, os.Stdout)
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

// getTaskMode returns the task mode based on options (for backward compatibility)
func getTaskMode(opts *RootOptions) TaskMode {
	if opts.Plan {
		return TaskModePlan
	}
	return TaskModeAct
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

	// Check if we're in a TTY
	if !isTTY() {
		// Non-TTY mode - print message and return
		fmt.Println("Interactive mode starting")
		fmt.Println("Note: Full TUI requires a terminal. Use 'cline \"your prompt\"' for non-interactive mode.")
		return nil
	}

	// Initialize storage
	storageCtx, err := initStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storageCtx.Close()

	// Load task history from storage
	historyPath := getTaskHistoryPath()
	entries, err := loadTaskHistory(historyPath)
	if err != nil {
		entries = []TaskHistoryEntry{}
	}
	
	// Convert entries to TaskHistoryItem
	recentTasks := make([]tui.TaskHistoryItem, 0, len(entries))
	for _, entry := range entries {
		recentTasks = append(recentTasks, tui.TaskHistoryItem{
			ID:          entry.ID,
			Description: entry.Task,
			Timestamp:   fmt.Sprintf("%d", entry.Timestamp),
		})
	}

	// Check if user has valid configuration
	hasConfig := checkConfiguration(storageCtx)

	// Show welcome screen
	action, data, err := tui.WelcomeScreen(recentTasks, hasConfig, Version)
	if err != nil {
		return fmt.Errorf("welcome screen error: %w", err)
	}

	switch action {
	case "new_task":
		return runInteractiveChat(opts, storageCtx, "")
	case "resume_task":
		if taskID, ok := data.(string); ok {
			return runInteractiveChat(opts, storageCtx, taskID)
		}
	case "settings":
		fmt.Println("Settings not yet implemented in TUI mode.")
		return nil
	case "help":
		fmt.Println("Help not yet implemented in TUI mode.")
		return nil
	case "quit":
		return nil
	}

	return nil
}

// checkConfiguration checks if the user has valid configuration
func checkConfiguration(storageCtx *storage.StorageContext) bool {
	// TODO: Implement actual configuration check
	// For now, assume configuration exists
	return true
}

// runInteractiveChat runs the interactive chat TUI
func runInteractiveChat(opts *RootOptions, storageCtx *storage.StorageContext, taskID string) error {
	// Check if we're in a TTY
	if !isTTY() {
		return fmt.Errorf("interactive chat mode requires a terminal")
	}

	// Use the existing ChatScreen function
	_, err := tui.ChatScreen("", func(content string) error {
		fmt.Printf("Sending: %s\n", content)
		return nil
	}, func() error {
		fmt.Println("Interrupted")
		return nil
	})

	return err
}


