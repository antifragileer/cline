// Package task provides task execution functionality for the Cline CLI.
// This file implements the TaskRunner which executes tasks by communicating
// with the core extension via gRPC.
package task

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/host"
)

// OutputMode represents the output mode for task execution
type OutputMode int

const (
	// ModeTUI runs in interactive TUI mode
	ModeTUI OutputMode = iota
	// ModePlain runs in plain text mode
	ModePlain
	// ModeJSON runs in JSON output mode
	ModeJSON
)

// RunnerOptions provides options for creating a TaskRunner
type RunnerOptions struct {
	// ProtoClient is the gRPC client for communicating with core
	ProtoClient *host.ProtoClient

	// Output mode (TUI, Plain, or JSON)
	Mode OutputMode

	// Output writer for results
	Output io.Writer

	// Error writer for errors
	ErrorOutput io.Writer

	// Input reader for user input
	Input io.Reader

	// Auto-approve without prompting
	AutoApprove bool

	// Timeout for task execution
	Timeout time.Duration

	// Verbose output
	Verbose bool
}

// TaskRunner executes tasks by communicating with the core extension
type TaskRunner struct {
	protoClient *host.ProtoClient
	mode        OutputMode
	output      io.Writer
	errorOutput io.Writer
	input       io.Reader
	autoApprove bool
	timeout     time.Duration
	verbose     bool

	// Runtime state
	currentTaskID string
	completionCh  chan struct{}
	errorCh       chan error
	cancelFunc    context.CancelFunc
}

// NewTaskRunner creates a new TaskRunner with the given options
func NewTaskRunner(opts RunnerOptions) *TaskRunner {
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	errorOutput := opts.ErrorOutput
	if errorOutput == nil {
		errorOutput = os.Stderr
	}

	input := opts.Input
	if input == nil {
		input = os.Stdin
	}

	return &TaskRunner{
		protoClient: opts.ProtoClient,
		mode:        opts.Mode,
		output:      output,
		errorOutput: errorOutput,
		input:       input,
		autoApprove: opts.AutoApprove,
		timeout:     opts.Timeout,
		verbose:     opts.Verbose,
		completionCh: make(chan struct{}),
		errorCh:      make(chan error, 1),
	}
}

// Run executes a task with the given configuration
func (r *TaskRunner) Run(cfg TaskConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel
	defer cancel()

	// Apply timeout if specified
	if r.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		r.cancelFunc = cancel
		defer cancel()
	}

	// Validate gRPC client
	if r.protoClient == nil {
		return fmt.Errorf("gRPC client not initialized")
	}

	// Process images if provided
	imageDataUrls, err := r.processImages(cfg.Images)
	if err != nil {
		return fmt.Errorf("failed to process images: %w", err)
	}

	// Create or resume task
	var taskID string
	if cfg.TaskID != "" {
		// Resume existing task
		taskID = cfg.TaskID
		if r.verbose {
			fmt.Fprintf(r.output, "Resuming task: %s\n", taskID)
		}
	} else {
		// Create new task
		taskID, err = r.createTask(ctx, cfg.Prompt, imageDataUrls, cfg)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		if r.verbose {
			fmt.Fprintf(r.output, "Created task: %s\n", taskID)
		}
	}

	r.currentTaskID = taskID

	// Subscribe to state updates
	go r.subscribeToMessages(ctx)

	// Wait for completion or error
	select {
	case <-r.completionCh:
		return nil
	case err := <-r.errorCh:
		return err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("task timed out after %s", r.timeout)
		}
		return ctx.Err()
	}
}

// createTask creates a new task via gRPC
func (r *TaskRunner) createTask(ctx context.Context, prompt string, images []string, cfg TaskConfig) (string, error) {
	// Build settings from config
	settings := r.buildSettings(cfg)

	taskID, err := r.protoClient.NewTask(ctx, prompt, images, nil, settings)
	if err != nil {
		return "", err
	}

	return taskID, nil
}

// buildSettings builds the Settings proto from TaskConfig
func (r *TaskRunner) buildSettings(cfg TaskConfig) *cline.Settings {
	settings := &cline.Settings{}

	// Set mode
	mode := cline.PlanActMode_ACT
	if cfg.Mode == TaskModePlan {
		mode = cline.PlanActMode_PLAN
	}
	settings.Mode = &mode

	// Set model if specified (use ActMode for simplicity, could be enhanced to set both)
	if cfg.Model != "" {
		settings.ActModeApiModelId = &cfg.Model
	}

	// Set auto-approve flags via yolo mode
	yoloMode := cfg.Yolo
	settings.YoloModeToggled = &yoloMode

	return settings
}

// subscribeToMessages subscribes to ClineMessage updates from the core
func (r *TaskRunner) subscribeToMessages(ctx context.Context) {
	stream, err := r.protoClient.SubscribeToPartialMessage(ctx)
	if err != nil {
		r.errorCh <- fmt.Errorf("failed to subscribe to messages: %w", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Stream closed normally
				close(r.completionCh)
				return
			}
			r.errorCh <- fmt.Errorf("stream error: %w", err)
			return
		}

		// Process the message
		if err := r.processMessage(msg); err != nil {
			r.errorCh <- err
			return
		}
	}
}

// processMessage processes a ClineMessage from the core
func (r *TaskRunner) processMessage(msg *cline.ClineMessage) error {
	if msg == nil {
		return nil
	}

	// Route based on message type
	switch msg.Type {
	case cline.ClineMessageType_SAY:
		return r.handleSayMessage(msg)
	case cline.ClineMessageType_ASK:
		return r.handleAskMessage(msg)
	default:
		// Unknown message type, log but don't fail
		if r.verbose {
			fmt.Fprintf(r.errorOutput, "Unknown message type: %v\n", msg.Type)
		}
		return nil
	}
}

// handleSayMessage handles say messages from the core
func (r *TaskRunner) handleSayMessage(msg *cline.ClineMessage) error {
	switch msg.Say {
	case cline.ClineSay_TEXT:
		// Display assistant text
		return r.displayText(msg.Text, msg.Partial)

	case cline.ClineSay_ERROR:
		// Mark task as failed
		return fmt.Errorf("task error: %s", msg.Text)

	case cline.ClineSay_API_REQ_STARTED:
		// Show loading indicator
		if r.mode == ModePlain {
			fmt.Fprintln(r.output, "Processing...")
		}

	case cline.ClineSay_API_REQ_FINISHED:
		// Hide loading, show token usage if available
		if r.verbose && msg.Text != "" {
			fmt.Fprintf(r.output, "API request finished: %s\n", msg.Text)
		}

	case cline.ClineSay_COMPLETION_RESULT_SAY:
		// Task complete
		r.displayText(msg.Text, false)
		close(r.completionCh)
		return nil

	case cline.ClineSay_COMMAND_SAY:
		// Command output
		if r.mode == ModePlain {
			fmt.Fprintf(r.output, "$ %s\n", msg.Text)
		}

	case cline.ClineSay_BROWSER_ACTION:
		// Browser action output
		if r.verbose {
			fmt.Fprintf(r.output, "[Browser] %s\n", msg.Text)
		}

	default:
		// Unknown say type, display as-is in verbose mode
		if r.verbose {
			fmt.Fprintf(r.output, "[Say:%v] %s\n", msg.Say, msg.Text)
		}
	}

	return nil
}

// handleAskMessage handles ask messages from the core
func (r *TaskRunner) handleAskMessage(msg *cline.ClineMessage) error {
	switch msg.Ask {
	case cline.ClineAsk_COMMAND:
		// Command approval request
		approved := r.autoApprove
		if !approved {
			approved = r.requestApproval("command", msg.Text)
		}

		responseType := "rejected"
		if approved {
			responseType = "yesButtonClicked"
		}

		return r.protoClient.AskResponse(context.Background(), responseType, "", nil, nil)

	case cline.ClineAsk_TOOL:
		// Tool approval request
		approved := r.autoApprove
		if !approved {
			approved = r.requestApproval("tool", msg.Text)
		}

		responseType := "rejected"
		if approved {
			responseType = "yesButtonClicked"
		}

		return r.protoClient.AskResponse(context.Background(), responseType, "", nil, nil)

	case cline.ClineAsk_BROWSER_ACTION_LAUNCH:
		// Browser action approval
		approved := r.autoApprove
		if !approved {
			approved = r.requestApproval("browser", msg.Text)
		}

		responseType := "rejected"
		if approved {
			responseType = "yesButtonClicked"
		}

		return r.protoClient.AskResponse(context.Background(), responseType, "", nil, nil)

	case cline.ClineAsk_FOLLOWUP:
		// Display question, wait for user input
		response, err := r.getUserInput(msg.Text)
		if err != nil {
			return err
		}

		return r.protoClient.AskResponse(context.Background(), "messageResponse", response, nil, nil)

	case cline.ClineAsk_PLAN_MODE_RESPOND:
		// Plan mode response
		response, err := r.getUserInput(msg.Text)
		if err != nil {
			return err
		}

		return r.protoClient.AskResponse(context.Background(), "messageResponse", response, nil, nil)

	case cline.ClineAsk_COMPLETION_RESULT:
		// Task complete (ask variant)
		close(r.completionCh)
		return nil

	default:
		// Unknown ask type
		if r.verbose {
			fmt.Fprintf(r.errorOutput, "Unknown ask type: %v\n", msg.Ask)
		}
		// Auto-reject unknown asks
		return r.protoClient.AskResponse(context.Background(), "rejected", "", nil, nil)
	}
}

// displayText displays text output based on mode
func (r *TaskRunner) displayText(text string, partial bool) error {
	switch r.mode {
	case ModeJSON:
		// Output as JSON
		fmt.Fprintf(r.output, `{"type":"text","partial":%t,"content":%q}`+"\n", partial, text)
	case ModePlain:
		// Plain text output
		if partial {
			// For partial messages, just print (no newline until complete)
			fmt.Fprint(r.output, text)
		} else {
			// Complete message
			fmt.Fprintln(r.output, text)
		}
	default:
		// TUI mode - would be handled by TUI component
		if r.verbose {
			fmt.Fprintln(r.output, text)
		}
	}
	return nil
}

// requestApproval prompts the user for approval
func (r *TaskRunner) requestApproval(requestType, details string) bool {
	if r.mode == ModePlain {
		// Plain mode approval prompt
		fmt.Fprintf(r.output, "\nCline wants to execute %s:\n", requestType)
		fmt.Fprintf(r.output, "%s\n", details)
		fmt.Fprint(r.output, "Approve? (y/n/a=always): ")

		var response string
		fmt.Fscanln(r.input, &response)

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			return true
		case "a", "always":
			r.autoApprove = true
			return true
		default:
			return false
		}
	}

	// For TUI mode, this would be handled by the TUI
	// Default to auto-approve if no interactive input available
	return r.autoApprove
}

// getUserInput prompts for and reads user input
func (r *TaskRunner) getUserInput(prompt string) (string, error) {
	if r.mode == ModePlain {
		// Plain mode input
		fmt.Fprintf(r.output, "\n%s\n", prompt)
		fmt.Fprint(r.output, "> ")

		var response string
		_, err := fmt.Fscanln(r.input, &response)
		if err != nil {
			return "", err
		}

		return response, nil
	}

	// For TUI mode, this would be handled by the TUI
	return "", fmt.Errorf("user input not available in non-interactive mode")
}

// processImages converts image file paths to base64 data URLs
func (r *TaskRunner) processImages(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	result := make([]string, 0, len(paths))

	for _, path := range paths {
		dataUrl, err := r.loadImageAsDataURL(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", path, err)
		}
		result = append(result, dataUrl)
	}

	return result, nil
}

// loadImageAsDataURL loads an image file and returns it as a base64 data URL
func (r *TaskRunner) loadImageAsDataURL(path string) (string, error) {
	// Expand path if needed
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

	// Check file exists
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("file not accessible: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(absPath))
	mimeType := getMimeTypeFromExt(ext)
	if mimeType == "" {
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(content)

	// Build data URL
	dataUrl := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	return dataUrl, nil
}

// getMimeTypeFromExt returns the MIME type for a file extension
func getMimeTypeFromExt(ext string) string {
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
	case ".svg":
		return "image/svg+xml"
	default:
		return ""
	}
}

// Cancel cancels the currently running task
func (r *TaskRunner) Cancel() error {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	if r.protoClient != nil {
		return r.protoClient.CancelTask(context.Background())
	}

	return nil
}

// GetCurrentTaskID returns the current task ID
func (r *TaskRunner) GetCurrentTaskID() string {
	return r.currentTaskID
}

// RunTask is a convenience function to run a task with default options
func RunTask(protoClient *host.ProtoClient, cfg TaskConfig) error {
	opts := RunnerOptions{
		ProtoClient: protoClient,
		Mode:        ModePlain,
		AutoApprove: cfg.Yolo,
		Timeout:     cfg.Timeout,
		Verbose:     cfg.Verbose,
	}

	runner := NewTaskRunner(opts)
	return runner.Run(cfg)
}

// RunTaskWithMode runs a task with a specific output mode
func RunTaskWithMode(protoClient *host.ProtoClient, cfg TaskConfig, mode OutputMode) error {
	opts := RunnerOptions{
		ProtoClient: protoClient,
		Mode:        mode,
		AutoApprove: cfg.Yolo,
		Timeout:     cfg.Timeout,
		Verbose:     cfg.Verbose,
	}

	runner := NewTaskRunner(opts)
	return runner.Run(cfg)
}
