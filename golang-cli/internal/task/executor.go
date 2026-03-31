// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
)

// Executor handles the full task execution lifecycle
type Executor struct {
	taskClient cline.TaskServiceClient
	stateClient cline.StateServiceClient
	uiClient   cline.UiServiceClient
	conn       *grpc.ClientConn

	// Execution state
	taskID       string
	mode         TaskMode
	isRunning    bool
	isCancelled  bool
	mu           sync.RWMutex
	completionCh chan struct{}
	errorCh      chan error

	// Cancellation
	cancelFunc context.CancelFunc
	sigChan    chan os.Signal
}

// NewExecutor creates a new task executor
func NewExecutor(conn *grpc.ClientConn) *Executor {
	return &Executor{
		taskClient:   cline.NewTaskServiceClient(conn),
		stateClient:  cline.NewStateServiceClient(conn),
		uiClient:     cline.NewUiServiceClient(conn),
		conn:         conn,
		completionCh: make(chan struct{}),
		errorCh:      make(chan error, 1),
		sigChan:      make(chan os.Signal, 1),
	}
}

// Execute runs a task with full lifecycle management
func (e *Executor) Execute(ctx context.Context, config TaskConfig, handler MessageHandler) error {
	// Validate configuration
	if err := e.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Set up cancellation context
	ctx, e.cancelFunc = context.WithCancel(ctx)
	defer e.cancelFunc()

	// Set up signal handling for graceful shutdown
	e.setupSignalHandling()
	defer e.cleanupSignalHandling()

	// Mark as running
	e.mu.Lock()
	e.isRunning = true
	e.mode = config.Mode
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.isRunning = false
		e.mu.Unlock()
		close(e.completionCh)
	}()

	// Prepare images
	imageData, err := e.prepareImages(config.Images)
	if err != nil {
		return fmt.Errorf("failed to prepare images: %w", err)
	}

	// Create or resume task
	taskID, err := e.initializeTask(ctx, config, imageData)
	if err != nil {
		return fmt.Errorf("failed to initialize task: %w", err)
	}

	e.taskID = taskID

	if config.Verbose {
		handler.OnInfo(fmt.Sprintf("Task initialized: %s", taskID))
	}

	// Execute the task with state streaming
	return e.executeWithStreaming(ctx, config, handler)
}

// validateConfig validates the task configuration
func (e *Executor) validateConfig(config TaskConfig) error {
	if config.Prompt == "" && config.TaskID == "" {
		return fmt.Errorf("task prompt required (or use TaskID to resume)")
	}

	// Validate mode
	if config.Mode != TaskModeAct && config.Mode != TaskModePlan {
		return fmt.Errorf("invalid mode: %s (must be 'act' or 'plan')", config.Mode)
	}

	// Validate images exist
	for _, img := range config.Images {
		if img == "" {
			continue
		}
		if _, err := os.Stat(img); os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", img)
		}
	}

	return nil
}

// prepareImages loads and encodes image files
func (e *Executor) prepareImages(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	imageData := make([]string, 0, len(paths))
	for _, path := range paths {
		data, err := loadImageData(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", path, err)
		}
		imageData = append(imageData, data)
	}

	return imageData, nil
}

// initializeTask creates a new task or resumes an existing one
func (e *Executor) initializeTask(ctx context.Context, config TaskConfig, imageData []string) (string, error) {
	if config.TaskID != "" {
		// Resume existing task
		return e.resumeTask(ctx, config.TaskID, config.Prompt, imageData)
	}
	// Create new task
	return e.createTask(ctx, config, imageData)
}

// createTask creates a new task via gRPC
func (e *Executor) createTask(ctx context.Context, config TaskConfig, imageData []string) (string, error) {
	settings := buildSettingsFromConfig(config)

	req := &cline.NewTaskRequest{
		Metadata:     &cline.Metadata{},
		Text:         config.Prompt,
		Images:       imageData,
		TaskSettings: settings,
	}

	resp, err := e.taskClient.NewTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create new task: %w", err)
	}

	return resp.GetValue(), nil
}

// resumeTask resumes an existing task via gRPC
func (e *Executor) resumeTask(ctx context.Context, taskID string, prompt string, imageData []string) (string, error) {
	// First, show the task to load it
	_, err := e.taskClient.ShowTaskWithId(ctx, &cline.StringRequest{Value: taskID})
	if err != nil {
		return "", fmt.Errorf("failed to show task: %w", err)
	}

	// If there's a prompt to send, send it as an ask response
	if prompt != "" {
		_, err := e.taskClient.AskResponse(ctx, &cline.AskResponseRequest{
			Metadata:     &cline.Metadata{},
			ResponseType: "messageResponse",
			Text:         prompt,
			Images:       imageData,
		})
		if err != nil {
			return "", fmt.Errorf("failed to send prompt to resumed task: %w", err)
		}
	}

	return taskID, nil
}

// executeWithStreaming executes the task with state streaming
func (e *Executor) executeWithStreaming(ctx context.Context, config TaskConfig, handler MessageHandler) error {
	// Subscribe to state updates
	stream, err := e.stateClient.SubscribeToState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to state: %w", err)
	}

	// Process messages in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- e.processStateStream(ctx, stream, config, handler)
	}()

	// Wait for completion, error, or cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-e.completionCh:
		return nil
	}
}

// processStateStream processes messages from the state stream
func (e *Executor) processStateStream(ctx context.Context, stream cline.StateService_SubscribeToStateClient, config TaskConfig, handler MessageHandler) error {
	processedMessages := make(map[int64]bool)
	var mu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		state, err := stream.Recv()
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Canceled {
				return nil // Context was cancelled
			}
			return fmt.Errorf("error receiving state: %w", err)
		}

		// Process state messages
		if err := e.processState(state, processedMessages, &mu, config, handler); err != nil {
			return err
		}
	}
}

// processState processes a single state update
func (e *Executor) processState(state *cline.State, processedMessages map[int64]bool, mu *sync.Mutex, config TaskConfig, handler MessageHandler) error {
	// Extract messages from state JSON
	messages, err := extractMessagesFromState(state.StateJson)
	if err != nil {
		if config.Verbose {
			handler.OnInfo(fmt.Sprintf("Failed to parse state: %v", err))
		}
		return nil
	}

	// Process each message
	for _, msg := range messages {
		ts := msg.Ts

		mu.Lock()
		if processedMessages[ts] {
			mu.Unlock()
			continue
		}
		mu.Unlock()

		if err := e.processMessage(msg, config, handler); err != nil {
			return err
		}

		mu.Lock()
		processedMessages[ts] = true
		mu.Unlock()

		// Check for completion or cancellation
		if e.checkCompletionOrCancellation(msg) {
			return nil
		}
	}

	return nil
}

// checkCompletionOrCancellation checks if the task should complete or was cancelled
func (e *Executor) checkCompletionOrCancellation(msg *ClineMessage) bool {
	e.mu.RLock()
	cancelled := e.isCancelled
	e.mu.RUnlock()

	if cancelled {
		return true
	}

	// Check for completion
	if msg.Type == "say" && msg.Say == "completion_result" {
		return true
	}
	if msg.Type == "ask" && msg.Ask == "completion_result" {
		return true
	}

	return false
}

// processMessage processes a single message
func (e *Executor) processMessage(msg *ClineMessage, config TaskConfig, handler MessageHandler) error {
	switch msg.Type {
	case "say":
		return e.processSayMessage(msg, config, handler)
	case "ask":
		return e.processAskMessage(msg, config, handler)
	}
	return nil
}

// processSayMessage processes a SAY message
func (e *Executor) processSayMessage(msg *ClineMessage, config TaskConfig, handler MessageHandler) error {
	// Skip partial messages in non-verbose mode
	if msg.Partial && !config.Verbose {
		return nil
	}

	handler.OnSay(string(msg.Say), msg.Text, msg.Partial)
	return nil
}

// processAskMessage processes an ASK message (requires user response)
func (e *Executor) processAskMessage(msg *ClineMessage, config TaskConfig, handler MessageHandler) error {
	// Check for auto-approval
	if shouldAutoApprove(string(msg.Ask), config) {
		return e.sendAutoApproval(msg)
	}

	// Get user response through handler
	response, err := handler.OnAsk(string(msg.Ask), msg.Text)
	if err != nil {
		return err
	}

	// Send response back to core
	return e.sendAskResponse(response, "", nil, nil)
}

// sendAutoApproval sends an auto-approval response
func (e *Executor) sendAutoApproval(msg *ClineMessage) error {
	return e.sendAskResponse("yesButtonClicked", "", nil, nil)
}

// sendAskResponse sends a response to an ask message
func (e *Executor) sendAskResponse(responseType, text string, images, files []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &cline.AskResponseRequest{
		Metadata:     &cline.Metadata{},
		ResponseType: responseType,
		Text:         text,
		Images:       images,
		Files:        files,
	}

	_, err := e.taskClient.AskResponse(ctx, req)
	return err
}

// setupSignalHandling sets up signal handling for graceful shutdown
func (e *Executor) setupSignalHandling() {
	signal.Notify(e.sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-e.sigChan:
			fmt.Fprintf(os.Stderr, "\nReceived signal %v, cancelling task...\n", sig)
			e.Cancel()
		case <-e.completionCh:
			// Task completed normally
		}
	}()
}

// cleanupSignalHandling cleans up signal handling
func (e *Executor) cleanupSignalHandling() {
	signal.Stop(e.sigChan)
	close(e.sigChan)
}

// Cancel cancels the current task execution
func (e *Executor) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isCancelled || !e.isRunning {
		return
	}

	e.isCancelled = true

	if e.cancelFunc != nil {
		e.cancelFunc()
	}
}

// IsRunning returns whether the executor is currently running a task
func (e *Executor) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// IsCancelled returns whether the task was cancelled
func (e *Executor) IsCancelled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isCancelled
}

// GetTaskID returns the current task ID
func (e *Executor) GetTaskID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.taskID
}

// Close closes the executor and releases resources
func (e *Executor) Close() error {
	return nil
}

// buildSettingsFromConfig builds task settings from config
func buildSettingsFromConfig(config TaskConfig) *cline.Settings {
	mode := cline.PlanActMode_PLAN
	if config.Mode == TaskModeAct {
		mode = cline.PlanActMode_ACT
	}
	
	return &cline.Settings{
		Mode: &mode,
	}
}

// getThinkingBudget returns the thinking budget based on config
func getThinkingBudget(config TaskConfig) int32 {
	if config.Thinking {
		return 0 // Let the model decide
	}
	return -1 // Disabled
}

// shouldAutoApprove determines if a message should be auto-approved
func shouldAutoApprove(askType string, config TaskConfig) bool {
	if config.Yolo {
		return true
	}

	autoApproveTypes := map[string]bool{
		"api_req_started":    true,
		"api_req_finished":   true,
		"api_req_retried":    true,
		"api_req_failed":     true,
		"deleted_api_reqs":   true,
		"checkpoint_created": true,
		"command_output":     true,
	}

	return autoApproveTypes[askType]
}

// extractMessagesFromState extracts messages from state JSON
// This is a helper function that would parse the state JSON
func extractMessagesFromState(stateJSON string) ([]*ClineMessage, error) {
	// This would parse the state JSON and extract messages
	// For now, return empty slice
	return []*ClineMessage{}, nil
}