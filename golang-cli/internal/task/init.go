// Package task provides task initialization and management functionality for the Cline CLI.
// This package handles creating new tasks, loading configurations, validating inputs,
// establishing gRPC streams, and persisting task history.
package task

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
	"google.golang.org/grpc"
)

// TaskMode represents the execution mode for a task
type TaskMode string

const (
	// TaskModeAct represents act mode (execute actions)
	TaskModeAct TaskMode = "act"
	// TaskModePlan represents plan mode (planning only)
	TaskModePlan TaskMode = "plan"
)

// TaskConfig holds all configuration options for task initialization
type TaskConfig struct {
	// Mode specifies whether to run in act or plan mode
	Mode TaskMode `json:"mode"`

	// Prompt is the task prompt/message
	Prompt string `json:"prompt"`

	// Images is a list of image file paths to attach
	Images []string `json:"images"`

	// Model specifies the model to use for the task
	Model string `json:"model"`

	// Timeout is the maximum duration for task execution
	Timeout time.Duration `json:"timeout"`

	// Yolo enables auto-approval without confirmation
	Yolo bool `json:"yolo"`

	// Thinking enables thinking mode
	Thinking bool `json:"thinking"`

	// Cwd is the current working directory for the task
	Cwd string `json:"cwd"`

	// ConfigPath is the path to a custom configuration file
	ConfigPath string `json:"configPath"`

	// Verbose enables verbose output
	Verbose bool `json:"verbose"`

	// TaskID specifies a task ID to resume (if empty, a new one will be generated)
	TaskID string `json:"taskId"`
}

// ImageData represents validated image data ready for transmission
type ImageData struct {
	// Path is the original file path
	Path string `json:"path"`

	// Content is the base64-encoded image content
	Content string `json:"content"`

	// MimeType is the detected MIME type
	MimeType string `json:"mimeType"`

	// Size is the file size in bytes
	Size int64 `json:"size"`
}

// NewTaskRequest represents a request to create a new task
type NewTaskRequest struct {
	TaskID    string            `json:"taskId"`
	Prompt    string            `json:"prompt"`
	Mode      TaskMode          `json:"mode"`
	Images    []ImageData       `json:"images,omitempty"`
	Model     string            `json:"model"`
	Timeout   int64             `json:"timeout"` // seconds
	Yolo      bool              `json:"yolo"`
	Thinking  bool              `json:"thinking"`
	Cwd       string            `json:"cwd"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewTaskResponse represents a response from creating a new task
type NewTaskResponse struct {
	Success   bool   `json:"success"`
	TaskID    string `json:"taskId"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// TaskHistoryEntry represents a single entry in the task history
type TaskHistoryEntry struct {
	TaskID    string   `json:"taskId"`
	Prompt    string   `json:"prompt"`
	Mode      TaskMode `json:"mode"`
	Cwd       string   `json:"cwd"`
	Model     string   `json:"model"`
	CreatedAt int64    `json:"createdAt"`
	Status    string   `json:"status"`
}

// InitOptions provides options for task initialization
type InitOptions struct {
	// Config is the layered configuration
	Config *config.LayeredConfig

	// Storage is the storage context
	Storage *storage.StorageContext

	// Client is the gRPC client
	Client *host.Client

	// UI is the UI interface (optional)
	UI UIInterface

	// Output is the output writer for logging
	Output io.Writer
}

// UIInterface defines the interface for UI operations
type UIInterface interface {
	// Initialize sets up the UI for a new task
	Initialize(taskID string, prompt string) error

	// UpdateStatus updates the UI with the current status
	UpdateStatus(status string) error

	// ShowError displays an error message
	ShowError(err error) error

	// Close cleans up the UI
	Close() error
}

// Initializer handles task initialization
type Initializer struct {
	config  *config.LayeredConfig
	storage *storage.StorageContext
	client  *host.Client
	ui      UIInterface
	output  io.Writer
}

// NewInitializer creates a new task initializer
func NewInitializer(opts InitOptions) (*Initializer, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if opts.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	output := opts.Output
	if output == nil {
		output = io.Discard
	}

	return &Initializer{
		config:  opts.Config,
		storage: opts.Storage,
		client:  opts.Client,
		ui:      opts.UI,
		output:  output,
	}, nil
}

// Init initializes a new task with the given configuration
func (i *Initializer) Init(ctx context.Context, cfg TaskConfig) (*NewTaskResponse, error) {
	// Step 1: Generate or validate task ID
	taskID, err := i.generateTaskID(cfg.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate task ID: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(i.output, "Task ID: %s\n", taskID)
	}

	// Step 2: Set working directory
	cwd, err := i.setWorkingDirectory(cfg.Cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to set working directory: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(i.output, "Working directory: %s\n", cwd)
	}

	// Step 3: Load and validate images
	images, err := i.loadAndValidateImages(cfg.Images)
	if err != nil {
		return nil, fmt.Errorf("failed to load images: %w", err)
	}

	if cfg.Verbose && len(images) > 0 {
		fmt.Fprintf(i.output, "Loaded %d image(s)\n", len(images))
	}

	// Step 4: Load configuration
	if err := i.loadConfiguration(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(i.output, "Configuration loaded\n")
	}

	// Step 5: Initialize UI
	if i.ui != nil {
		if err := i.ui.Initialize(taskID, cfg.Prompt); err != nil {
			return nil, fmt.Errorf("failed to initialize UI: %w", err)
		}
	}

	// Step 6: Establish gRPC stream and send NewTask request
	var response *NewTaskResponse
	if i.client != nil {
		response, err = i.sendNewTaskRequest(ctx, taskID, cfg, images)
		if err != nil {
			if i.ui != nil {
				_ = i.ui.ShowError(err)
			}
			return nil, fmt.Errorf("failed to send new task request: %w", err)
		}

		if cfg.Verbose {
			fmt.Fprintf(i.output, "NewTask request sent successfully\n")
		}
	} else {
		// No client, create local response
		response = &NewTaskResponse{
			Success:   true,
			TaskID:    taskID,
			Message:   "Task created locally",
			Timestamp: time.Now().Unix(),
		}
	}

	// Step 7: Persist to history
	if err := i.persistToHistory(taskID, cfg); err != nil {
		// Log but don't fail - history is not critical
		fmt.Fprintf(i.output, "Warning: failed to persist task to history: %v\n", err)
	}

	return response, nil
}

// generateTaskID generates a new UUID task ID or validates the provided one
func (i *Initializer) generateTaskID(providedID string) (string, error) {
	if providedID != "" {
		// Validate the provided ID format (basic UUID format check)
		if isValidUUID(providedID) {
			return providedID, nil
		}
		// Not a valid UUID, but we'll still accept it as a custom ID
		return providedID, nil
	}

	// Generate new UUID v4
	uuid, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	return uuid, nil
}

// generateUUID generates a UUID v4 (random UUID)
func generateUUID() (string, error) {
	u := make([]byte, 16)
	if _, err := rand.Read(u); err != nil {
		return "", err
	}

	// Set version (4) and variant bits
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16]), nil
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(s string) bool {
	// Remove dashes and check length
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return false
	}

	// Check if it's valid hex
	_, err := hex.DecodeString(s)
	return err == nil
}

// setWorkingDirectory sets and validates the working directory
func (i *Initializer) setWorkingDirectory(cwd string) (string, error) {
	if cwd == "" {
		// Use current directory
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Expand path if it contains ~
	if strings.HasPrefix(cwd, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		cwd = filepath.Join(homeDir, cwd[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate directory exists and is accessible
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("working directory not accessible: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	return absPath, nil
}

// loadAndValidateImages loads and validates image files
func (i *Initializer) loadAndValidateImages(paths []string) ([]ImageData, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	images := make([]ImageData, 0, len(paths))

	for _, path := range paths {
		image, err := i.loadImage(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", path, err)
		}
		images = append(images, *image)
	}

	return images, nil
}

// LoadAndValidateImage is an exported function to load and validate an image file
// This is useful for testing image loading functionality without creating an Initializer
func LoadAndValidateImage(path string) (*ImageData, error) {
	return loadImageInternal(path)
}

// loadImageInternal loads and validates a single image file
func loadImageInternal(path string) (*ImageData, error) {
	// Expand path if needed
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check file exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("file not accessible: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(absPath))
	mimeType := getMimeType(ext)
	if mimeType == "" {
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(content)

	return &ImageData{
		Path:     absPath,
		Content:  encoded,
		MimeType: mimeType,
		Size:     info.Size(),
	}, nil
}

// loadImage loads and validates a single image file
func (i *Initializer) loadImage(path string) (*ImageData, error) {
	return loadImageInternal(path)
}

// getMimeType returns the MIME type for a given file extension
func getMimeType(ext string) string {
	switch strings.ToLower(ext) {
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

// loadConfiguration loads and merges configuration from all sources
func (i *Initializer) loadConfiguration(cfg *TaskConfig) error {
	// Set defaults from config if not already set
	if cfg.Mode == "" {
		mode := i.config.GetString("mode")
		if mode == "plan" {
			cfg.Mode = TaskModePlan
		} else {
			cfg.Mode = TaskModeAct
		}
	}

	if cfg.Model == "" {
		cfg.Model = i.config.GetString("model")
	}

	if cfg.Timeout == 0 {
		timeout := i.config.GetInt("api.timeout")
		if timeout > 0 {
			cfg.Timeout = time.Duration(timeout) * time.Second
		}
	}

	// Load from config file if specified
	if cfg.ConfigPath != "" {
		if err := i.loadConfigFile(cfg.ConfigPath, cfg); err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
	}

	return nil
}

// loadConfigFile loads configuration from a JSON file
func (i *Initializer) loadConfigFile(path string, cfg *TaskConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var fileConfig TaskConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge file config into provided config (file values don't override CLI flags)
	if cfg.Mode == "" && fileConfig.Mode != "" {
		cfg.Mode = fileConfig.Mode
	}
	if cfg.Model == "" && fileConfig.Model != "" {
		cfg.Model = fileConfig.Model
	}
	if cfg.Timeout == 0 && fileConfig.Timeout > 0 {
		cfg.Timeout = fileConfig.Timeout
	}
	if !cfg.Yolo && fileConfig.Yolo {
		cfg.Yolo = fileConfig.Yolo
	}
	if !cfg.Thinking && fileConfig.Thinking {
		cfg.Thinking = fileConfig.Thinking
	}

	return nil
}

// sendNewTaskRequest sends the NewTask request via gRPC using the TaskStreamHandler
func (i *Initializer) sendNewTaskRequest(ctx context.Context, taskID string, cfg TaskConfig, images []ImageData) (*NewTaskResponse, error) {
	if i.client == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}

	// Create a TaskStreamHandler for bidirectional communication
	handler := host.NewTaskStreamHandler(taskID)

	// Set up message callbacks
	handler.OnTextMessage = func(text string) {
		if cfg.Verbose {
			fmt.Fprintf(i.output, "Assistant: %s\n", text)
		}
	}

	handler.OnCompletion = func(result string) {
		if cfg.Verbose {
			fmt.Fprintf(i.output, "Task completed: %s\n", result)
		}
	}

	handler.OnError = func(err error) {
		fmt.Fprintf(i.output, "Error: %v\n", err)
	}

	// Create a StreamCreator function using the client
	streamCreator := func(ctx context.Context, conn *grpc.ClientConn) (host.TaskService_StreamClient, error) {
		return CreateTaskStream(ctx, conn, taskID)
	}

	// Start the handler
	if err := handler.Start(streamCreator); err != nil {
		return nil, fmt.Errorf("failed to start task stream handler: %w", err)
	}
	defer handler.Stop()

	// Send the initial task message
	imageContents := make([]string, 0, len(images))
	for _, img := range images {
		imageContents = append(imageContents, img.Content)
	}

	if err := handler.SendMessage(cfg.Prompt, imageContents, nil); err != nil {
		return nil, fmt.Errorf("failed to send task message: %w", err)
	}

	// Wait for completion or timeout
	if cfg.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()

		if err := handler.WaitForCompletion(ctx); err != nil {
			return nil, fmt.Errorf("task execution failed: %w", err)
		}
	} else {
		if err := handler.WaitForCompletion(ctx); err != nil {
			return nil, fmt.Errorf("task execution failed: %w", err)
		}
	}

	return &NewTaskResponse{
		Success:   true,
		TaskID:    taskID,
		Message:   "Task completed successfully",
		Timestamp: time.Now().Unix(),
	}, nil
}

// persistToHistory saves the task to history
func (i *Initializer) persistToHistory(taskID string, cfg TaskConfig) error {
	entry := TaskHistoryEntry{
		TaskID:    taskID,
		Prompt:    cfg.Prompt,
		Mode:      cfg.Mode,
		Cwd:       cfg.Cwd,
		Model:     cfg.Model,
		CreatedAt: time.Now().Unix(),
		Status:    "initialized",
	}

	// Get existing history
	var history []TaskHistoryEntry
	if data, ok := i.storage.GlobalState.Get("taskHistory"); ok {
		if histData, err := json.Marshal(data); err == nil {
			_ = json.Unmarshal(histData, &history)
		}
	}

	// Add new entry
	history = append(history, entry)

	// Keep only last 100 entries
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	// Save history
	if err := i.storage.GlobalState.Set("taskHistory", history); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// mustMarshalJSON marshals a value to JSON, panicking on error
func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return data
}

// getVersion returns the CLI version
func getVersion() string {
	// This would typically be set at build time
	// For now, return a placeholder
	return "0.1.0"
}

// Close cleans up the initializer
func (i *Initializer) Close() error {
	if i.ui != nil {
		return i.ui.Close()
	}
	return nil
}

// ============================================================================
// Task Stream Functions
// ============================================================================

// CreateTaskStream creates a bidirectional stream for task communication using the generated proto types
// This function is used by the task initializer to establish communication with the core extension
func CreateTaskStream(ctx context.Context, conn *grpc.ClientConn, taskID string) (host.TaskService_StreamClient, error) {
	// Use the StreamCreator from the host package
	streamCreator := host.CreateTaskStreamCreator(taskID)
	return streamCreator(ctx, conn)
}

// HandleTaskStream manages the bidirectional stream for a task, routing messages between
// the CLI and the core extension. It handles incoming messages (tool requests, responses)
// and sends user input back.
func HandleTaskStream(ctx context.Context, stream host.TaskService_StreamClient, handlers TaskStreamHandlers) error {
	// Create channels for coordinating send/receive
	errChan := make(chan error, 2)
	doneChan := make(chan struct{})

	// Start receive goroutine
	go func() {
		defer close(doneChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("receive error: %w", err)
				return
			}

			if msg == nil || msg.ClineMessage == nil {
				continue
			}

			// Route the message to appropriate handler
			if err := routeMessage(msg, handlers); err != nil {
				errChan <- fmt.Errorf("message routing error: %w", err)
				return
			}
		}
	}()

	// Wait for completion or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	}
}

// TaskStreamHandlers contains callback functions for handling different message types
type TaskStreamHandlers struct {
	// OnTextMessage is called when a text message is received
	OnTextMessage func(text string)
	// OnToolRequest is called when a tool execution is requested
	OnToolRequest func(tool *host.ClineSayTool) error
	// OnCommandRequest is called when a command execution is requested
	OnCommandRequest func(command string) error
	// OnCompletion is called when the task completes
	OnCompletion func(result string)
	// OnError is called when an error occurs
	OnError func(err error)
	// OnAskQuestion is called when a question is asked
	OnAskQuestion func(question *host.ClineAskQuestion) (string, error)
	// SendResponse is called to send a response back to the core extension
	SendResponse func(responseType string, text string, images []string, files []string) error
}

// routeMessage routes an incoming message to the appropriate handler
func routeMessage(msg *host.ClineMessageProto, handlers TaskStreamHandlers) error {
	if msg == nil || msg.ClineMessage == nil {
		return nil
	}

	switch msg.Type {
	case host.ClineMessageType_SAY:
		return handleSayMessage(msg, handlers)
	case host.ClineMessageType_ASK:
		return handleAskMessage(msg, handlers)
	default:
		return fmt.Errorf("unknown message type: %v", msg.Type)
	}
}

// handleSayMessage handles say messages from the core extension
func handleSayMessage(msg *host.ClineMessageProto, handlers TaskStreamHandlers) error {
	switch msg.Say {
	case host.ClineSay_TEXT:
		if handlers.OnTextMessage != nil {
			handlers.OnTextMessage(msg.Text)
		}
	case host.ClineSay_TOOL_SAY:
		if msg.SayTool != nil && handlers.OnToolRequest != nil {
			if err := handlers.OnToolRequest(msg.SayTool); err != nil {
				return err
			}
		}
	case host.ClineSay_COMMAND_SAY:
		if handlers.OnCommandRequest != nil {
			if err := handlers.OnCommandRequest(msg.Text); err != nil {
				return err
			}
		}
	case host.ClineSay_COMPLETION_RESULT_SAY:
		if handlers.OnCompletion != nil {
			handlers.OnCompletion(msg.Text)
		}
	case host.ClineSay_ERROR:
		if handlers.OnError != nil {
			handlers.OnError(fmt.Errorf("%s", msg.Text))
		}
	}
	return nil
}

// handleAskMessage handles ask messages from the core extension
func handleAskMessage(msg *host.ClineMessageProto, handlers TaskStreamHandlers) error {
	switch msg.Ask {
	case host.ClineAsk_FOLLOWUP, host.ClineAsk_PLAN_MODE_RESPOND, host.ClineAsk_ACT_MODE_RESPOND:
		if msg.AskQuestion != nil && handlers.OnAskQuestion != nil {
			response, err := handlers.OnAskQuestion(msg.AskQuestion)
			if err != nil {
				return err
			}
			if handlers.SendResponse != nil {
				return handlers.SendResponse("messageResponse", response, nil, nil)
			}
		}
	case host.ClineAsk_COMMAND:
		// Command approval request - auto-approve for now
		if handlers.SendResponse != nil {
			return handlers.SendResponse("yesButtonClicked", "", nil, nil)
		}
	case host.ClineAsk_TOOL:
		// Tool approval request - auto-approve for now
		if handlers.SendResponse != nil {
			return handlers.SendResponse("yesButtonClicked", "", nil, nil)
		}
	case host.ClineAsk_COMPLETION_RESULT:
		if handlers.OnCompletion != nil {
			handlers.OnCompletion(msg.Text)
		}
	}
	return nil
}
