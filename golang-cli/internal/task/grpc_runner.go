// Package task provides task execution functionality for the Cline CLI.
// This file implements the gRPC-based task runner that connects to the Cline core extension.
package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/cline/cline/golang-cli/internal/errorservice"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/session"
	"github.com/cline/cline/golang-cli/internal/telemetry"
)

// GRPCRunner executes tasks using gRPC communication with the Cline core extension
// Reference: cli/src/utils/plain-text-task.ts
type GRPCRunner struct {
	mu              sync.RWMutex
	config          *Config
	approvalHandler *ApprovalHandler
	telemetry       telemetry.Service
	errorService    errorservice.Service
	session         session.Manager
	logger          *slog.Logger

	// Task state
	taskID      string
	isRunning   bool
	isCancelled bool
	exitCode    int

	// gRPC connection
	grpcClient    *host.Client
	taskClient    host.TaskServiceClient
	streamHandler *host.TaskStreamHandler

	// Message handling
	messageHandler MessageHandler

	// Results
	taskResult string
	taskError  error
	completed  bool

	// Channels for communication
	doneChan     chan struct{}
	messageChan  chan *host.ClineMessageProto
	responseChan chan *host.AskResponseRequest
}

// NewGRPCRunner creates a new gRPC-based task runner
func NewGRPCRunner(
	config *Config,
	telemetry telemetry.Service,
	errorService errorservice.Service,
	logger *slog.Logger,
) *GRPCRunner {
	if config == nil {
		config = &Config{Mode: TaskModeAct}
	}

	return &GRPCRunner{
		config:          config,
		approvalHandler: NewApprovalHandler(config, logger),
		telemetry:       telemetry,
		errorService:    errorService,
		session:         session.Get(),
		logger:          logger,
		taskID:          generateTaskID(),
		doneChan:        make(chan struct{}),
		messageChan:     make(chan *host.ClineMessageProto, 100),
		responseChan:    make(chan *host.AskResponseRequest, 10),
	}
}

// SetGRPCClient sets the gRPC client for the runner
func (r *GRPCRunner) SetGRPCClient(client *host.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.grpcClient = client
}

// StartTask starts a new task with the given prompt and options
// Reference: cli/src/utils/plain-text-task.ts:148-151
func (r *GRPCRunner) StartTask(ctx context.Context, opts TaskOptions) error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("task already running")
	}
	r.isRunning = true
	r.isCancelled = false
	r.completed = false
	r.exitCode = 0
	r.mu.Unlock()

	// Ensure cleanup when done
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
		close(r.doneChan)
	}()

	// Record telemetry
	if r.telemetry != nil {
		_ = r.telemetry.CaptureTaskCreated(r.taskID, "default")
	}
	r.session.StartAPICall()
	defer r.session.EndAPICall()

	r.logger.Info("Starting task", "task_id", r.taskID, "prompt", opts.Prompt)

	// Get gRPC connection
	conn, err := r.grpcClient.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Create task client
	r.taskClient = host.NewTaskServiceClient(conn)

	// Create new task via RPC
	req := &host.NewTaskRequest{
		Text:   opts.Prompt,
		Images: opts.Images,
		Files:  opts.Files,
	}

	if opts.Timeout > 0 {
		req.TaskSettings = &host.Settings{}
		// Note: Settings struct is minimal, timeout would be handled via context
	}

	// Call NewTask RPC
	resp, err := r.taskClient.NewTask(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Store the task ID from the response
	if resp != nil && resp.Value != "" {
		r.taskID = resp.Value
		r.logger.Info("Task created", "task_id", r.taskID)
	}

	// Emit task started message
	if r.messageHandler != nil {
		r.messageHandler.OnInfo(fmt.Sprintf("Task: %s", opts.Prompt))
	}

	// Start bidirectional stream for communication
	if err := r.startStream(ctx); err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	// Wait for completion or cancellation
	return r.waitForCompletion(ctx)
}

// ResumeTask resumes an existing task with the given task ID
// Reference: cli/src/utils/plain-text-task.ts:135-147
func (r *GRPCRunner) ResumeTask(ctx context.Context, taskID string, message string) error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("task already running")
	}
	r.isRunning = true
	r.isCancelled = false
	r.completed = false
	r.taskID = taskID
	r.mu.Unlock()

	// Ensure cleanup when done
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
		close(r.doneChan)
	}()

	// Record telemetry
	if r.telemetry != nil {
		_ = r.telemetry.CaptureResumeTask(true)
	}
	r.session.StartAPICall()
	defer r.session.EndAPICall()

	r.logger.Info("Resuming task", "task_id", taskID, "message", message)

	// Get gRPC connection
	conn, err := r.grpcClient.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Create task client
	r.taskClient = host.NewTaskServiceClient(conn)

	// Show task with ID via RPC
	_, err = r.taskClient.ShowTaskWithId(ctx, &host.StringRequest{Value: taskID})
	if err != nil {
		return fmt.Errorf("failed to show task: %w", err)
	}

	r.logger.Info("Task loaded", "task_id", taskID)

	// Emit task started message
	if r.messageHandler != nil {
		if message != "" {
			r.messageHandler.OnInfo(fmt.Sprintf("Resuming task %s with: %s", taskID, message))
		} else {
			r.messageHandler.OnInfo(fmt.Sprintf("Resuming task %s", taskID))
		}
	}

	// Start bidirectional stream for communication
	if err := r.startStream(ctx); err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	// Send resume message if provided
	if message != "" {
		if err := r.SendMessage(ctx, message); err != nil {
			r.logger.Warn("Failed to send resume message", "error", err)
		}
	}

	// Wait for completion or cancellation
	return r.waitForCompletion(ctx)
}

// startStream establishes the bidirectional gRPC stream
func (r *GRPCRunner) startStream(ctx context.Context) error {
	// Get gRPC connection
	_, err := r.grpcClient.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Create stream handler with callbacks
	r.streamHandler = host.NewTaskStreamHandler(r.taskID)

	// Set up callback handlers
	r.streamHandler.OnTextMessage = func(text string) {
		if r.messageHandler != nil {
			r.messageHandler.OnText(text, false)
		}
	}

	r.streamHandler.OnToolRequest = func(tool *host.ClineSayTool) error {
		if r.messageHandler != nil {
			response, err := r.messageHandler.OnToolUse(string(tool.Tool), map[string]interface{}{
				"path":    tool.Path,
				"diff":    tool.Diff,
				"content": tool.Content,
			})
			if err != nil {
				return err
			}
			// Tool approved, continue
			if !response {
				return fmt.Errorf("tool rejected by user")
			}
			return nil
		}
		return nil
	}

	r.streamHandler.OnCommandRequest = func(command string) error {
		if r.messageHandler != nil {
			response, err := r.messageHandler.OnCommand(command, true)
			if err != nil {
				return err
			}
			// Command approved, execute
			if response != "execute" {
				return fmt.Errorf("command rejected by user")
			}
			return nil
		}
		return nil
	}

	r.streamHandler.OnCompletion = func(result string) {
		r.mu.Lock()
		r.completed = true
		r.taskResult = result
		r.mu.Unlock()

		if r.messageHandler != nil {
			r.messageHandler.OnCompletion(true, result)
		}
	}

	r.streamHandler.OnError = func(err error) {
		r.mu.Lock()
		r.taskError = err
		r.exitCode = 1
		r.mu.Unlock()

		if r.messageHandler != nil {
			r.messageHandler.OnError(err)
		}
	}

	r.streamHandler.OnAskQuestion = func(question *host.ClineAskQuestion) (string, error) {
		if r.messageHandler != nil {
			return r.messageHandler.OnAsk("followup", question.Question)
		}
		return "", nil
	}

	// Create stream creator
	streamCreator := func(ctx context.Context, conn *grpc.ClientConn) (host.TaskService_StreamClient, error) {
		client := host.NewTaskServiceClient(conn)
		return client.Stream(ctx)
	}

	// Start the stream handler
	if err := r.streamHandler.Start(streamCreator); err != nil {
		return fmt.Errorf("failed to start stream handler: %w", err)
	}

	return nil
}

// sendAskResponse sends a response to an ASK message
func (r *GRPCRunner) sendAskResponse(ctx context.Context, responseType, text string) error {
	if r.taskClient == nil {
		return fmt.Errorf("task client not initialized")
	}

	req := &host.AskResponseRequest{
		ResponseType: responseType,
		Text:         text,
	}

	_, err := r.taskClient.AskResponse(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send ask response: %w", err)
	}

	return nil
}

// SendMessage sends a user message to the task
func (r *GRPCRunner) SendMessage(ctx context.Context, message string) error {
	if r.streamHandler == nil {
		return fmt.Errorf("stream handler not initialized")
	}

	return r.streamHandler.SendMessage(message, nil, nil)
}

// SendApproval sends an approval response
func (r *GRPCRunner) SendApproval(ctx context.Context, approved bool) error {
	responseType := "noButtonClicked"
	if approved {
		responseType = "yesButtonClicked"
	}
	return r.sendAskResponse(ctx, responseType, "")
}

// CancelTask cancels the current task
func (r *GRPCRunner) CancelTask() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.isCancelled = true

	if r.taskClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.taskClient.CancelTask(ctx, &host.EmptyRequest{})
		if err != nil {
			r.logger.Warn("Failed to cancel task", "error", err)
			return err
		}
	}

	if r.streamHandler != nil {
		return r.streamHandler.Stop()
	}

	return nil
}

// waitForCompletion waits for the task to complete or be cancelled
func (r *GRPCRunner) waitForCompletion(ctx context.Context) error {
	// Create a ticker to periodically check state
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.CancelTask()
			return ctx.Err()

		case <-r.doneChan:
			// Task runner stopped
			r.mu.RLock()
			err := r.taskError
			r.mu.RUnlock()
			return err

		case <-ticker.C:
			r.mu.RLock()
			completed := r.completed
			cancelled := r.isCancelled
			taskErr := r.taskError
			r.mu.RUnlock()

			if cancelled {
				return fmt.Errorf("task cancelled")
			}

			if completed {
				return taskErr
			}
		}
	}
}

// IsRunning returns whether a task is currently running
func (r *GRPCRunner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRunning
}

// IsCancelled returns whether the task has been cancelled
func (r *GRPCRunner) IsCancelled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isCancelled
}

// GetTaskID returns the current task ID
func (r *GRPCRunner) GetTaskID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskID
}

// GetExitCode returns the task exit code
func (r *GRPCRunner) GetExitCode() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.exitCode
}

// SetMessageHandler sets the message handler for the runner
func (r *GRPCRunner) SetMessageHandler(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandler = handler
}

// TaskOptions contains options for starting a task
type TaskOptions struct {
	Prompt  string
	Images  []string
	Files   []string
	Timeout time.Duration
	Model   string
	Mode    TaskMode
	Yolo    bool
}