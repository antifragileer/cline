// Package host provides gRPC client functionality for communicating with the Cline extension.
// This file implements TaskServiceClient and other service clients using the generated proto code.
package host

import (
	"context"
	"fmt"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"google.golang.org/grpc"
)

// ProtoClient wraps the generated gRPC clients to provide a unified interface
// for communicating with the Cline extension backend.
type ProtoClient struct {
	client        *Client
	taskSvc       cline.TaskServiceClient
	stateSvc      cline.StateServiceClient
	uiSvc         cline.UiServiceClient
	fileSvc       cline.FileServiceClient
	modelsSvc     cline.ModelsServiceClient
	browserSvc    cline.BrowserServiceClient
	mcpSvc        cline.McpServiceClient
	accountSvc    cline.AccountServiceClient
	slashSvc      cline.SlashServiceClient
	webSvc        cline.WebServiceClient
	checkpointSvc cline.CheckpointsServiceClient
	worktreeSvc   cline.WorktreeServiceClient
}

// NewProtoClient creates a new ProtoClient with the given host client.
// The host client provides the connection pool and retry logic.
func NewProtoClient(client *Client) *ProtoClient {
	return &ProtoClient{
		client: client,
	}
}

// Start initializes all gRPC service clients by getting a connection from the pool.
func (p *ProtoClient) Start() error {
	conn, err := p.client.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Initialize all service clients using the same connection
	p.taskSvc = cline.NewTaskServiceClient(conn)
	p.stateSvc = cline.NewStateServiceClient(conn)
	p.uiSvc = cline.NewUiServiceClient(conn)
	p.fileSvc = cline.NewFileServiceClient(conn)
	p.modelsSvc = cline.NewModelsServiceClient(conn)
	p.browserSvc = cline.NewBrowserServiceClient(conn)
	p.mcpSvc = cline.NewMcpServiceClient(conn)
	p.accountSvc = cline.NewAccountServiceClient(conn)
	p.slashSvc = cline.NewSlashServiceClient(conn)
	p.webSvc = cline.NewWebServiceClient(conn)
	p.checkpointSvc = cline.NewCheckpointsServiceClient(conn)
	p.worktreeSvc = cline.NewWorktreeServiceClient(conn)

	return nil
}

// Stop closes the client and all connections.
func (p *ProtoClient) Stop() error {
	return p.client.Stop()
}

// TaskService returns the TaskService client.
func (p *ProtoClient) TaskService() cline.TaskServiceClient {
	return p.taskSvc
}

// StateService returns the StateService client.
func (p *ProtoClient) StateService() cline.StateServiceClient {
	return p.stateSvc
}

// UiService returns the UiService client.
func (p *ProtoClient) UiService() cline.UiServiceClient {
	return p.uiSvc
}

// FileService returns the FileService client.
func (p *ProtoClient) FileService() cline.FileServiceClient {
	return p.fileSvc
}

// ModelsService returns the ModelsService client.
func (p *ProtoClient) ModelsService() cline.ModelsServiceClient {
	return p.modelsSvc
}

// BrowserService returns the BrowserService client.
func (p *ProtoClient) BrowserService() cline.BrowserServiceClient {
	return p.browserSvc
}

// McpService returns the McpService client.
func (p *ProtoClient) McpService() cline.McpServiceClient {
	return p.mcpSvc
}

// AccountService returns the AccountService client.
func (p *ProtoClient) AccountService() cline.AccountServiceClient {
	return p.accountSvc
}

// SlashService returns the SlashService client.
func (p *ProtoClient) SlashService() cline.SlashServiceClient {
	return p.slashSvc
}

// WebService returns the WebService client.
func (p *ProtoClient) WebService() cline.WebServiceClient {
	return p.webSvc
}

// CheckpointsService returns the CheckpointsService client.
func (p *ProtoClient) CheckpointsService() cline.CheckpointsServiceClient {
	return p.checkpointSvc
}

// WorktreeService returns the WorktreeService client.
func (p *ProtoClient) WorktreeService() cline.WorktreeServiceClient {
	return p.worktreeSvc
}

// GetClient returns the underlying host client.
func (p *ProtoClient) GetClient() *Client {
	return p.client
}

// ============================================================================
// TaskService Convenience Methods
// ============================================================================

// NewTask creates a new task with the given prompt and optional images.
func (p *ProtoClient) NewTask(ctx context.Context, text string, images, files []string, settings *cline.Settings) (string, error) {
	req := &cline.NewTaskRequest{
		Metadata: &cline.Metadata{},
		Text:     text,
		Images:   images,
		Files:    files,
	}
	if settings != nil {
		req.TaskSettings = settings
	}

	resp, err := p.taskSvc.NewTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create new task: %w", err)
	}
	return resp.GetValue(), nil
}

// CancelTask cancels the currently running task.
func (p *ProtoClient) CancelTask(ctx context.Context) error {
	_, err := p.taskSvc.CancelTask(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}
	return nil
}

// CancelBackgroundCommand cancels the currently running background command.
func (p *ProtoClient) CancelBackgroundCommand(ctx context.Context) error {
	_, err := p.taskSvc.CancelBackgroundCommand(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to cancel background command: %w", err)
	}
	return nil
}

// ClearTask clears the current task.
func (p *ProtoClient) ClearTask(ctx context.Context) error {
	_, err := p.taskSvc.ClearTask(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to clear task: %w", err)
	}
	return nil
}

// ShowTaskWithId shows a task with the specified ID.
func (p *ProtoClient) ShowTaskWithId(ctx context.Context, taskId string) (*cline.TaskResponse, error) {
	resp, err := p.taskSvc.ShowTaskWithId(ctx, &cline.StringRequest{Value: taskId})
	if err != nil {
		return nil, fmt.Errorf("failed to show task: %w", err)
	}
	return resp, nil
}

// ExportTaskWithId exports a task with the given ID to markdown.
func (p *ProtoClient) ExportTaskWithId(ctx context.Context, taskId string) error {
	_, err := p.taskSvc.ExportTaskWithId(ctx, &cline.StringRequest{Value: taskId})
	if err != nil {
		return fmt.Errorf("failed to export task: %w", err)
	}
	return nil
}

// ToggleTaskFavorite toggles the favorite status of a task.
func (p *ProtoClient) ToggleTaskFavorite(ctx context.Context, taskId string, isFavorited bool) error {
	req := &cline.TaskFavoriteRequest{
		Metadata:    &cline.Metadata{},
		TaskId:      taskId,
		IsFavorited: isFavorited,
	}
	_, err := p.taskSvc.ToggleTaskFavorite(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to toggle task favorite: %w", err)
	}
	return nil
}

// GetTaskHistory gets filtered task history.
func (p *ProtoClient) GetTaskHistory(ctx context.Context, favoritesOnly bool, searchQuery, sortBy string, currentWorkspaceOnly bool) (*cline.TaskHistoryArray, error) {
	req := &cline.GetTaskHistoryRequest{
		Metadata:             &cline.Metadata{},
		FavoritesOnly:        favoritesOnly,
		SearchQuery:          searchQuery,
		SortBy:               sortBy,
		CurrentWorkspaceOnly: currentWorkspaceOnly,
	}
	resp, err := p.taskSvc.GetTaskHistory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task history: %w", err)
	}
	return resp, nil
}

// AskResponse sends a response to a previous ask operation.
func (p *ProtoClient) AskResponse(ctx context.Context, responseType, text string, images, files []string) error {
	req := &cline.AskResponseRequest{
		Metadata:     &cline.Metadata{},
		ResponseType: responseType,
		Text:         text,
		Images:       images,
		Files:        files,
	}
	_, err := p.taskSvc.AskResponse(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send ask response: %w", err)
	}
	return nil
}

// TaskFeedback records task feedback (thumbs up/down).
func (p *ProtoClient) TaskFeedback(ctx context.Context, feedback string) error {
	_, err := p.taskSvc.TaskFeedback(ctx, &cline.StringRequest{Value: feedback})
	if err != nil {
		return fmt.Errorf("failed to send task feedback: %w", err)
	}
	return nil
}

// TaskCompletionViewChanges shows task completion changes diff in a view.
func (p *ProtoClient) TaskCompletionViewChanges(ctx context.Context, messageTs int64) error {
	_, err := p.taskSvc.TaskCompletionViewChanges(ctx, &cline.Int64Request{Value: messageTs})
	if err != nil {
		return fmt.Errorf("failed to view task completion changes: %w", err)
	}
	return nil
}

// ExecuteQuickWin executes a quick win task with command and title.
func (p *ProtoClient) ExecuteQuickWin(ctx context.Context, command, title string) error {
	req := &cline.ExecuteQuickWinRequest{
		Metadata: &cline.Metadata{},
		Command:  command,
		Title:    title,
	}
	_, err := p.taskSvc.ExecuteQuickWin(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to execute quick win: %w", err)
	}
	return nil
}

// DeleteAllTaskHistory deletes all task history.
func (p *ProtoClient) DeleteAllTaskHistory(ctx context.Context) (int32, error) {
	resp, err := p.taskSvc.DeleteAllTaskHistory(ctx, &cline.EmptyRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to delete all task history: %w", err)
	}
	return resp.GetTasksDeleted(), nil
}

// ExplainChanges explains changes with AI and adds inline comments to the diff view.
func (p *ProtoClient) ExplainChanges(ctx context.Context, messageTs int64) error {
	req := &cline.ExplainChangesRequest{
		Metadata:  &cline.Metadata{},
		MessageTs: messageTs,
	}
	_, err := p.taskSvc.ExplainChanges(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to explain changes: %w", err)
	}
	return nil
}

// GetTotalTasksSize gets the total size of all tasks.
func (p *ProtoClient) GetTotalTasksSize(ctx context.Context) (int64, error) {
	resp, err := p.taskSvc.GetTotalTasksSize(ctx, &cline.EmptyRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to get total tasks size: %w", err)
	}
	return resp.GetValue(), nil
}

// DeleteTasksWithIds deletes multiple tasks with the given IDs.
func (p *ProtoClient) DeleteTasksWithIds(ctx context.Context, taskIds []string) error {
	_, err := p.taskSvc.DeleteTasksWithIds(ctx, &cline.StringArrayRequest{Value: taskIds})
	if err != nil {
		return fmt.Errorf("failed to delete tasks: %w", err)
	}
	return nil
}

// ============================================================================
// StateService Convenience Methods
// ============================================================================

// GetLatestState gets the latest state from the server.
func (p *ProtoClient) GetLatestState(ctx context.Context) (*cline.State, error) {
	resp, err := p.stateSvc.GetLatestState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest state: %w", err)
	}
	return resp, nil
}

// UpdateSettings updates settings on the server.
func (p *ProtoClient) UpdateSettings(ctx context.Context, settings *cline.Settings) error {
	req := &cline.UpdateSettingsRequest{
		Metadata: &cline.Metadata{},
	}
	// Note: Settings field mapping would be done here based on actual requirements
	_, err := p.stateSvc.UpdateSettings(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}
	return nil
}

// ============================================================================
// UiService Convenience Methods
// ============================================================================

// InitializeWebview initializes the webview when it launches.
func (p *ProtoClient) InitializeWebview(ctx context.Context) error {
	_, err := p.uiSvc.InitializeWebview(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to initialize webview: %w", err)
	}
	return nil
}

// OpenUrl opens a URL in the default browser.
func (p *ProtoClient) OpenUrl(ctx context.Context, url string) error {
	_, err := p.uiSvc.OpenUrl(ctx, &cline.StringRequest{Value: url})
	if err != nil {
		return fmt.Errorf("failed to open URL: %w", err)
	}
	return nil
}

// ============================================================================
// Streaming Methods
// ============================================================================

// SubscribeToPartialMessage subscribes to partial message updates (streaming Cline messages).
func (p *ProtoClient) SubscribeToPartialMessage(ctx context.Context) (grpc.ServerStreamingClient[cline.ClineMessage], error) {
	stream, err := p.uiSvc.SubscribeToPartialMessage(ctx, &cline.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to partial messages: %w", err)
	}
	return stream, nil
}

// SubscribeToState subscribes to state updates from the server.
func (p *ProtoClient) SubscribeToState(ctx context.Context) (grpc.ServerStreamingClient[cline.State], error) {
	stream, err := p.stateSvc.SubscribeToState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to state: %w", err)
	}
	return stream, nil
}
