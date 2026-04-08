// Package host provides gRPC client and streaming functionality for the Cline CLI.
// This file contains proto-generated types manually created to match the proto definitions.
package host

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// Common Types (from common.proto)
// ============================================================================

// Empty represents an empty message
type Empty struct{}

// EmptyRequest represents an empty request
type EmptyRequest struct{}

// StringRequest represents a string request
type StringRequest struct {
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

// StringArrayRequest represents a string array request
type StringArrayRequest struct {
	Value []string `protobuf:"bytes,2,rep,name=value,proto3" json:"value,omitempty"`
}

// Int64Request represents an int64 request
type Int64Request struct {
	Value int64 `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

// Int64 represents an int64 value
type Int64 struct {
	Value int64 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

// String represents a string value
type String struct {
	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

// BooleanRequest represents a boolean request
type BooleanRequest struct {
	Value bool `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

// Boolean represents a boolean value
type Boolean struct {
	Value bool `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

// KeyValuePair represents a key-value pair
type KeyValuePair struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

// Metadata represents request metadata
type Metadata struct{}

// ============================================================================
// UI Types (from ui.proto)
// ============================================================================

// ClineMessageType represents the type of Cline message
type ClineMessageType int32

const (
	ClineMessageType_ASK ClineMessageType = 0
	ClineMessageType_SAY ClineMessageType = 1
)

// ClineAsk represents ask types
type ClineAsk int32

const (
	ClineAsk_FOLLOWUP              ClineAsk = 0
	ClineAsk_PLAN_MODE_RESPOND     ClineAsk = 1
	ClineAsk_COMMAND               ClineAsk = 2
	ClineAsk_COMMAND_OUTPUT        ClineAsk = 3
	ClineAsk_COMPLETION_RESULT     ClineAsk = 4
	ClineAsk_TOOL                  ClineAsk = 5
	ClineAsk_API_REQ_FAILED        ClineAsk = 6
	ClineAsk_RESUME_TASK           ClineAsk = 7
	ClineAsk_RESUME_COMPLETED_TASK ClineAsk = 8
	ClineAsk_MISTAKE_LIMIT_REACHED ClineAsk = 9
	ClineAsk_BROWSER_ACTION_LAUNCH ClineAsk = 10
	ClineAsk_USE_MCP_SERVER        ClineAsk = 11
	ClineAsk_NEW_TASK              ClineAsk = 12
	ClineAsk_CONDENSE              ClineAsk = 13
	ClineAsk_REPORT_BUG            ClineAsk = 14
	ClineAsk_SUMMARIZE_TASK        ClineAsk = 15
	ClineAsk_ACT_MODE_RESPOND      ClineAsk = 16
	ClineAsk_USE_SUBAGENTS         ClineAsk = 17
)

// ClineSay represents say types
type ClineSay int32

const (
	ClineSay_TASK                       ClineSay = 0
	ClineSay_ERROR                      ClineSay = 1
	ClineSay_API_REQ_STARTED            ClineSay = 2
	ClineSay_API_REQ_FINISHED           ClineSay = 3
	ClineSay_TEXT                       ClineSay = 4
	ClineSay_REASONING                  ClineSay = 5
	ClineSay_COMPLETION_RESULT_SAY      ClineSay = 6
	ClineSay_USER_FEEDBACK              ClineSay = 7
	ClineSay_USER_FEEDBACK_DIFF         ClineSay = 8
	ClineSay_API_REQ_RETRIED            ClineSay = 9
	ClineSay_COMMAND_SAY                ClineSay = 10
	ClineSay_COMMAND_OUTPUT_SAY         ClineSay = 11
	ClineSay_TOOL_SAY                   ClineSay = 12
	ClineSay_SHELL_INTEGRATION_WARNING  ClineSay = 13
	ClineSay_BROWSER_ACTION_LAUNCH_SAY  ClineSay = 14
	ClineSay_BROWSER_ACTION             ClineSay = 15
	ClineSay_BROWSER_ACTION_RESULT      ClineSay = 16
	ClineSay_MCP_SERVER_REQUEST_STARTED ClineSay = 17
	ClineSay_MCP_SERVER_RESPONSE        ClineSay = 18
	ClineSay_MCP_NOTIFICATION           ClineSay = 19
	ClineSay_USE_MCP_SERVER_SAY         ClineSay = 20
	ClineSay_DIFF_ERROR                 ClineSay = 21
	ClineSay_DELETED_API_REQS           ClineSay = 22
	ClineSay_CLINEIGNORE_ERROR          ClineSay = 23
	ClineSay_CHECKPOINT_CREATED         ClineSay = 24
	ClineSay_LOAD_MCP_DOCUMENTATION     ClineSay = 25
	ClineSay_INFO                       ClineSay = 26
	ClineSay_TASK_PROGRESS              ClineSay = 27
	ClineSay_ERROR_RETRY                ClineSay = 28
	ClineSay_GENERATE_EXPLANATION       ClineSay = 29
	ClineSay_HOOK_STATUS                ClineSay = 30
	ClineSay_HOOK_OUTPUT_STREAM         ClineSay = 31
	ClineSay_COMMAND_PERMISSION_DENIED  ClineSay = 32
	ClineSay_CONDITIONAL_RULES_APPLIED  ClineSay = 33
	ClineSay_SUBAGENT_STATUS            ClineSay = 34
	ClineSay_USE_SUBAGENTS_SAY          ClineSay = 35
	ClineSay_SUBAGENT_USAGE             ClineSay = 36
)

// ClineSayToolType represents tool types for ClineSayTool
type ClineSayToolType int32

const (
	ClineSayToolType_EDITED_EXISTING_FILE       ClineSayToolType = 0
	ClineSayToolType_NEW_FILE_CREATED           ClineSayToolType = 1
	ClineSayToolType_READ_FILE                  ClineSayToolType = 2
	ClineSayToolType_LIST_FILES_TOP_LEVEL       ClineSayToolType = 3
	ClineSayToolType_LIST_FILES_RECURSIVE       ClineSayToolType = 4
	ClineSayToolType_LIST_CODE_DEFINITION_NAMES ClineSayToolType = 5
	ClineSayToolType_SEARCH_FILES               ClineSayToolType = 6
	ClineSayToolType_WEB_FETCH                  ClineSayToolType = 7
	ClineSayToolType_FILE_DELETED               ClineSayToolType = 8
)

// BrowserAction represents browser actions
type BrowserAction int32

const (
	BrowserAction_LAUNCH      BrowserAction = 0
	BrowserAction_CLICK       BrowserAction = 1
	BrowserAction_TYPE        BrowserAction = 2
	BrowserAction_SCROLL_DOWN BrowserAction = 3
	BrowserAction_SCROLL_UP   BrowserAction = 4
	BrowserAction_CLOSE       BrowserAction = 5
)

// McpServerRequestType represents MCP server request types
type McpServerRequestType int32

const (
	McpServerRequestType_USE_MCP_TOOL        McpServerRequestType = 0
	McpServerRequestType_ACCESS_MCP_RESOURCE McpServerRequestType = 1
)

// ClineApiReqCancelReason represents API request cancel reasons
type ClineApiReqCancelReason int32

const (
	ClineApiReqCancelReason_STREAMING_FAILED  ClineApiReqCancelReason = 0
	ClineApiReqCancelReason_USER_CANCELLED    ClineApiReqCancelReason = 1
	ClineApiReqCancelReason_RETRIES_EXHAUSTED ClineApiReqCancelReason = 2
)

// ConversationHistoryDeletedRange represents a deleted range in conversation history
type ConversationHistoryDeletedRange struct {
	StartIndex int32 `protobuf:"varint,1,opt,name=start_index,json=startIndex,proto3" json:"start_index,omitempty"`
	EndIndex   int32 `protobuf:"varint,2,opt,name=end_index,json=endIndex,proto3" json:"end_index,omitempty"`
}

// ClineSayTool represents a tool execution message
type ClineSayTool struct {
	Tool                          ClineSayToolType `protobuf:"varint,1,opt,name=tool,proto3,enum=cline.ClineSayToolType" json:"tool,omitempty"`
	Path                          string           `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	Diff                          string           `protobuf:"bytes,3,opt,name=diff,proto3" json:"diff,omitempty"`
	Content                       string           `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	Regex                         string           `protobuf:"bytes,5,opt,name=regex,proto3" json:"regex,omitempty"`
	FilePattern                   string           `protobuf:"bytes,6,opt,name=file_pattern,json=filePattern,proto3" json:"file_pattern,omitempty"`
	OperationIsLocatedInWorkspace bool             `protobuf:"varint,7,opt,name=operation_is_located_in_workspace,json=operationIsLocatedInWorkspace,proto3" json:"operation_is_located_in_workspace,omitempty"`
}

// ClineSayBrowserAction represents a browser action message
type ClineSayBrowserAction struct {
	Action     BrowserAction `protobuf:"varint,1,opt,name=action,proto3,enum=cline.BrowserAction" json:"action,omitempty"`
	Coordinate string        `protobuf:"bytes,2,opt,name=coordinate,proto3" json:"coordinate,omitempty"`
	Text       string        `protobuf:"bytes,3,opt,name=text,proto3" json:"text,omitempty"`
}

// BrowserActionResult represents the result of a browser action
type BrowserActionResult struct {
	Screenshot           string `protobuf:"bytes,1,opt,name=screenshot,proto3" json:"screenshot,omitempty"`
	Logs                 string `protobuf:"bytes,2,opt,name=logs,proto3" json:"logs,omitempty"`
	CurrentUrl           string `protobuf:"bytes,3,opt,name=current_url,json=currentUrl,proto3" json:"current_url,omitempty"`
	CurrentMousePosition string `protobuf:"bytes,4,opt,name=current_mouse_position,json=currentMousePosition,proto3" json:"current_mouse_position,omitempty"`
}

// ClineAskUseMcpServer represents an MCP server usage request
type ClineAskUseMcpServer struct {
	ServerName string               `protobuf:"bytes,1,opt,name=server_name,json=serverName,proto3" json:"server_name,omitempty"`
	Type       McpServerRequestType `protobuf:"varint,2,opt,name=type,proto3,enum=cline.McpServerRequestType" json:"type,omitempty"`
	ToolName   string               `protobuf:"bytes,3,opt,name=tool_name,json=toolName,proto3" json:"tool_name,omitempty"`
	Arguments  string               `protobuf:"bytes,4,opt,name=arguments,proto3" json:"arguments,omitempty"`
	Uri        string               `protobuf:"bytes,5,opt,name=uri,proto3" json:"uri,omitempty"`
}

// ClinePlanModeResponse represents a plan mode response
type ClinePlanModeResponse struct {
	Response string   `protobuf:"bytes,1,opt,name=response,proto3" json:"response,omitempty"`
	Options  []string `protobuf:"bytes,2,rep,name=options,proto3" json:"options,omitempty"`
	Selected string   `protobuf:"bytes,3,opt,name=selected,proto3" json:"selected,omitempty"`
}

// ClineAskQuestion represents a question ask
type ClineAskQuestion struct {
	Question string   `protobuf:"bytes,1,opt,name=question,proto3" json:"question,omitempty"`
	Options  []string `protobuf:"bytes,2,rep,name=options,proto3" json:"options,omitempty"`
	Selected string   `protobuf:"bytes,3,opt,name=selected,proto3" json:"selected,omitempty"`
}

// ClineAskNewTask represents a new task ask
type ClineAskNewTask struct {
	Context string `protobuf:"bytes,1,opt,name=context,proto3" json:"context,omitempty"`
}

// ApiReqRetryStatus represents API request retry status
type ApiReqRetryStatus struct {
	Attempt      int32  `protobuf:"varint,1,opt,name=attempt,proto3" json:"attempt,omitempty"`
	MaxAttempts  int32  `protobuf:"varint,2,opt,name=max_attempts,json=maxAttempts,proto3" json:"max_attempts,omitempty"`
	DelaySec     int32  `protobuf:"varint,3,opt,name=delay_sec,json=delaySec,proto3" json:"delay_sec,omitempty"`
	ErrorSnippet string `protobuf:"bytes,4,opt,name=error_snippet,json=errorSnippet,proto3" json:"error_snippet,omitempty"`
}

// ClineApiReqInfo represents API request information
type ClineApiReqInfo struct {
	Request                string                  `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	TokensIn               int32                   `protobuf:"varint,2,opt,name=tokens_in,json=tokensIn,proto3" json:"tokens_in,omitempty"`
	TokensOut              int32                   `protobuf:"varint,3,opt,name=tokens_out,json=tokensOut,proto3" json:"tokens_out,omitempty"`
	CacheWrites            int32                   `protobuf:"varint,4,opt,name=cache_writes,json=cacheWrites,proto3" json:"cache_writes,omitempty"`
	CacheReads             int32                   `protobuf:"varint,5,opt,name=cache_reads,json=cacheReads,proto3" json:"cache_reads,omitempty"`
	Cost                   float64                 `protobuf:"fixed64,6,opt,name=cost,proto3" json:"cost,omitempty"`
	CancelReason           ClineApiReqCancelReason `protobuf:"varint,7,opt,name=cancel_reason,json=cancelReason,proto3,enum=cline.ClineApiReqCancelReason" json:"cancel_reason,omitempty"`
	StreamingFailedMessage string                  `protobuf:"bytes,8,opt,name=streaming_failed_message,json=streamingFailedMessage,proto3" json:"streaming_failed_message,omitempty"`
	RetryStatus            *ApiReqRetryStatus      `protobuf:"bytes,9,opt,name=retry_status,json=retryStatus,proto3" json:"retry_status,omitempty"`
}

// ClineModelInfo represents model information
type ClineModelInfo struct {
	ProviderId string `protobuf:"bytes,1,opt,name=provider_id,json=providerId,proto3" json:"provider_id,omitempty"`
	ModelId    string `protobuf:"bytes,2,opt,name=model_id,json=modelId,proto3" json:"model_id,omitempty"`
}

// ClineMessage represents the main message type for task communication
type ClineMessage struct {
	Ts                              int64                            `protobuf:"varint,1,opt,name=ts,proto3" json:"ts,omitempty"`
	Type                            ClineMessageType                 `protobuf:"varint,2,opt,name=type,proto3,enum=cline.ClineMessageType" json:"type,omitempty"`
	Ask                             ClineAsk                         `protobuf:"varint,3,opt,name=ask,proto3,enum=cline.ClineAsk" json:"ask,omitempty"`
	Say                             ClineSay                         `protobuf:"varint,4,opt,name=say,proto3,enum=cline.ClineSay" json:"say,omitempty"`
	Text                            string                           `protobuf:"bytes,5,opt,name=text,proto3" json:"text,omitempty"`
	Reasoning                       string                           `protobuf:"bytes,6,opt,name=reasoning,proto3" json:"reasoning,omitempty"`
	Images                          []string                         `protobuf:"bytes,7,rep,name=images,proto3" json:"images,omitempty"`
	Files                           []string                         `protobuf:"bytes,8,rep,name=files,proto3" json:"files,omitempty"`
	Partial                         bool                             `protobuf:"varint,9,opt,name=partial,proto3" json:"partial,omitempty"`
	LastCheckpointHash              string                           `protobuf:"bytes,10,opt,name=last_checkpoint_hash,json=lastCheckpointHash,proto3" json:"last_checkpoint_hash,omitempty"`
	IsCheckpointCheckedOut          bool                             `protobuf:"varint,11,opt,name=is_checkpoint_checked_out,json=isCheckpointCheckedOut,proto3" json:"is_checkpoint_checked_out,omitempty"`
	IsOperationOutsideWorkspace     bool                             `protobuf:"varint,12,opt,name=is_operation_outside_workspace,json=isOperationOutsideWorkspace,proto3" json:"is_operation_outside_workspace,omitempty"`
	ConversationHistoryIndex        int32                            `protobuf:"varint,13,opt,name=conversation_history_index,json=conversationHistoryIndex,proto3" json:"conversation_history_index,omitempty"`
	ConversationHistoryDeletedRange *ConversationHistoryDeletedRange `protobuf:"bytes,14,opt,name=conversation_history_deleted_range,json=conversationHistoryDeletedRange,proto3" json:"conversation_history_deleted_range,omitempty"`
	SayTool                         *ClineSayTool                    `protobuf:"bytes,15,opt,name=say_tool,json=sayTool,proto3" json:"say_tool,omitempty"`
	SayBrowserAction                *ClineSayBrowserAction           `protobuf:"bytes,16,opt,name=say_browser_action,json=sayBrowserAction,proto3" json:"say_browser_action,omitempty"`
	BrowserActionResult             *BrowserActionResult             `protobuf:"bytes,17,opt,name=browser_action_result,json=browserActionResult,proto3" json:"browser_action_result,omitempty"`
	AskUseMcpServer                 *ClineAskUseMcpServer            `protobuf:"bytes,18,opt,name=ask_use_mcp_server,json=askUseMcpServer,proto3" json:"ask_use_mcp_server,omitempty"`
	PlanModeResponse                *ClinePlanModeResponse           `protobuf:"bytes,19,opt,name=plan_mode_response,json=planModeResponse,proto3" json:"plan_mode_response,omitempty"`
	AskQuestion                     *ClineAskQuestion                `protobuf:"bytes,20,opt,name=ask_question,json=askQuestion,proto3" json:"ask_question,omitempty"`
	AskNewTask                      *ClineAskNewTask                 `protobuf:"bytes,21,opt,name=ask_new_task,json=askNewTask,proto3" json:"ask_new_task,omitempty"`
	ApiReqInfo                      *ClineApiReqInfo                 `protobuf:"bytes,22,opt,name=api_req_info,json=apiReqInfo,proto3" json:"api_req_info,omitempty"`
	ModelInfo                       *ClineModelInfo                  `protobuf:"bytes,23,opt,name=model_info,json=modelInfo,proto3" json:"model_info,omitempty"`
}

// ProtoMessage is an interface for proto messages
type ProtoMessage interface {
	Reset()
	String() string
	ProtoMessage()
}

// ============================================================================
// Task Service Types (from task.proto)
// ============================================================================

// NewTaskRequest represents a request to create a new task
type NewTaskRequest struct {
	Metadata     *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Text         string    `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
	Images       []string  `protobuf:"bytes,3,rep,name=images,proto3" json:"images,omitempty"`
	Files        []string  `protobuf:"bytes,4,rep,name=files,proto3" json:"files,omitempty"`
	TaskSettings *Settings `protobuf:"bytes,5,opt,name=task_settings,json=taskSettings,proto3" json:"task_settings,omitempty"`
}

// Settings represents task settings
type Settings struct {
	// Add settings fields as needed based on state.proto
}

// AskResponseRequest represents a request to respond to an ask
type AskResponseRequest struct {
	Metadata     *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	ResponseType string    `protobuf:"bytes,2,opt,name=response_type,json=responseType,proto3" json:"response_type,omitempty"`
	Text         string    `protobuf:"bytes,3,opt,name=text,proto3" json:"text,omitempty"`
	Images       []string  `protobuf:"bytes,4,rep,name=images,proto3" json:"images,omitempty"`
	Files        []string  `protobuf:"bytes,5,rep,name=files,proto3" json:"files,omitempty"`
}

// ============================================================================
// ProtoMessage wrapper for ClineMessage to implement proto.Message interface
// ============================================================================

// ClineMessageProto wraps ClineMessage for gRPC streaming
type ClineMessageProto struct {
	*ClineMessage
}

// Reset resets the message
func (m *ClineMessageProto) Reset() {
	if m.ClineMessage != nil {
		*m.ClineMessage = ClineMessage{}
	}
}

// String returns the string representation
func (m *ClineMessageProto) String() string {
	if m.ClineMessage == nil {
		return ""
	}
	return fmt.Sprintf("ClineMessage{ts: %d, type: %v, ask: %v, say: %v, text: %s}",
		m.Ts, m.Type, m.Ask, m.Say, m.Text)
}

// ProtoMessage marks this as a proto message
func (m *ClineMessageProto) ProtoMessage() {}

// GetTs returns the timestamp
func (m *ClineMessageProto) GetTs() int64 {
	if m != nil && m.ClineMessage != nil {
		return m.Ts
	}
	return 0
}

// GetText returns the text content
func (m *ClineMessageProto) GetText() string {
	if m != nil && m.ClineMessage != nil {
		return m.Text
	}
	return ""
}

// GetPartial returns whether this is a partial message
func (m *ClineMessageProto) GetPartial() bool {
	if m != nil && m.ClineMessage != nil {
		return m.Partial
	}
	return false
}

// GetType returns the message type
func (m *ClineMessageProto) GetType() ClineMessageType {
	if m != nil && m.ClineMessage != nil {
		return m.Type
	}
	return ClineMessageType_ASK
}

// GetAsk returns the ask type
func (m *ClineMessageProto) GetAsk() ClineAsk {
	if m != nil && m.ClineMessage != nil {
		return m.Ask
	}
	return ClineAsk_FOLLOWUP
}

// GetSay returns the say type
func (m *ClineMessageProto) GetSay() ClineSay {
	if m != nil && m.ClineMessage != nil {
		return m.Say
	}
	return ClineSay_TASK
}

// ============================================================================
// TaskService Client Interface
// ============================================================================

// TaskServiceClient is the client API for TaskService service
type TaskServiceClient interface {
	// CancelTask cancels the currently running task
	CancelTask(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error)
	// CancelBackgroundCommand cancels the currently running background command
	CancelBackgroundCommand(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error)
	// ClearTask clears the current task
	ClearTask(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error)
	// GetTotalTasksSize gets the total size of all tasks
	GetTotalTasksSize(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Int64, error)
	// DeleteTasksWithIds deletes multiple tasks with the given IDs
	DeleteTasksWithIds(ctx context.Context, in *StringArrayRequest, opts ...grpc.CallOption) (*Empty, error)
	// NewTask creates a new task with the given text and optional images
	NewTask(ctx context.Context, in *NewTaskRequest, opts ...grpc.CallOption) (*String, error)
	// ShowTaskWithId shows a task with the specified ID
	ShowTaskWithId(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*TaskResponse, error)
	// ExportTaskWithId exports a task with the given ID to markdown
	ExportTaskWithId(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*Empty, error)
	// ToggleTaskFavorite toggles the favorite status of a task
	ToggleTaskFavorite(ctx context.Context, in *TaskFavoriteRequest, opts ...grpc.CallOption) (*Empty, error)
	// GetTaskHistory gets filtered task history
	GetTaskHistory(ctx context.Context, in *GetTaskHistoryRequest, opts ...grpc.CallOption) (*TaskHistoryArray, error)
	// AskResponse sends a response to a previous ask operation
	AskResponse(ctx context.Context, in *AskResponseRequest, opts ...grpc.CallOption) (*Empty, error)
	// TaskFeedback records task feedback (thumbs up/down)
	TaskFeedback(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*Empty, error)
	// TaskCompletionViewChanges shows task completion changes diff in a view
	TaskCompletionViewChanges(ctx context.Context, in *Int64Request, opts ...grpc.CallOption) (*Empty, error)
	// ExecuteQuickWin executes a quick win task with command and title
	ExecuteQuickWin(ctx context.Context, in *ExecuteQuickWinRequest, opts ...grpc.CallOption) (*Empty, error)
	// DeleteAllTaskHistory deletes all task history
	DeleteAllTaskHistory(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*DeleteAllTaskHistoryCount, error)
	// ExplainChanges explains changes with AI and adds inline comments to the diff view
	ExplainChanges(ctx context.Context, in *ExplainChangesRequest, opts ...grpc.CallOption) (*Empty, error)
	// Stream establishes a bidirectional stream for task communication
	Stream(ctx context.Context, opts ...grpc.CallOption) (TaskService_StreamClient, error)
}

// TaskService_StreamClient is the client API for bidirectional streaming
type TaskService_StreamClient interface {
	Send(*ClineMessageProto) error
	Recv() (*ClineMessageProto, error)
	grpc.ClientStream
}

// TaskResponse represents a task response
type TaskResponse struct {
	Id          string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Task        string  `protobuf:"bytes,2,opt,name=task,proto3" json:"task,omitempty"`
	Ts          int64   `protobuf:"varint,3,opt,name=ts,proto3" json:"ts,omitempty"`
	IsFavorited bool    `protobuf:"varint,4,opt,name=is_favorited,json=isFavorited,proto3" json:"is_favorited,omitempty"`
	Size        int64   `protobuf:"varint,5,opt,name=size,proto3" json:"size,omitempty"`
	TotalCost   float64 `protobuf:"fixed64,6,opt,name=total_cost,json=totalCost,proto3" json:"total_cost,omitempty"`
	TokensIn    int32   `protobuf:"varint,7,opt,name=tokens_in,json=tokensIn,proto3" json:"tokens_in,omitempty"`
	TokensOut   int32   `protobuf:"varint,8,opt,name=tokens_out,json=tokensOut,proto3" json:"tokens_out,omitempty"`
	CacheWrites int32   `protobuf:"varint,9,opt,name=cache_writes,json=cacheWrites,proto3" json:"cache_writes,omitempty"`
	CacheReads  int32   `protobuf:"varint,10,opt,name=cache_reads,json=cacheReads,proto3" json:"cache_reads,omitempty"`
	ModelId     string  `protobuf:"bytes,11,opt,name=model_id,json=modelId,proto3" json:"model_id,omitempty"`
}

// TaskFavoriteRequest represents a request to toggle task favorite
type TaskFavoriteRequest struct {
	Metadata    *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	TaskId      string    `protobuf:"bytes,2,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	IsFavorited bool      `protobuf:"varint,3,opt,name=is_favorited,json=isFavorited,proto3" json:"is_favorited,omitempty"`
}

// GetTaskHistoryRequest represents a request to get task history
type GetTaskHistoryRequest struct {
	Metadata             *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	FavoritesOnly        bool      `protobuf:"varint,2,opt,name=favorites_only,json=favoritesOnly,proto3" json:"favorites_only,omitempty"`
	SearchQuery          string    `protobuf:"bytes,3,opt,name=search_query,json=searchQuery,proto3" json:"search_query,omitempty"`
	SortBy               string    `protobuf:"varint,4,opt,name=sort_by,json=sortBy,proto3" json:"sort_by,omitempty"`
	CurrentWorkspaceOnly bool      `protobuf:"varint,5,opt,name=current_workspace_only,json=currentWorkspaceOnly,proto3" json:"current_workspace_only,omitempty"`
}

// TaskHistoryArray represents a task history array response
type TaskHistoryArray struct {
	Tasks      []*TaskItem `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
	TotalCount int32       `protobuf:"varint,2,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
}

// TaskItem represents a task item in history
type TaskItem struct {
	Id          string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Task        string  `protobuf:"bytes,2,opt,name=task,proto3" json:"task,omitempty"`
	Ts          int64   `protobuf:"varint,3,opt,name=ts,proto3" json:"ts,omitempty"`
	IsFavorited bool    `protobuf:"varint,4,opt,name=is_favorited,json=isFavorited,proto3" json:"is_favorited,omitempty"`
	Size        int64   `protobuf:"varint,5,opt,name=size,proto3" json:"size,omitempty"`
	TotalCost   float64 `protobuf:"fixed64,6,opt,name=total_cost,json=totalCost,proto3" json:"total_cost,omitempty"`
	TokensIn    int32   `protobuf:"varint,7,opt,name=tokens_in,json=tokensIn,proto3" json:"tokens_in,omitempty"`
	TokensOut   int32   `protobuf:"varint,8,opt,name=tokens_out,json=tokensOut,proto3" json:"tokens_out,omitempty"`
	CacheWrites int32   `protobuf:"varint,9,opt,name=cache_writes,json=cacheWrites,proto3" json:"cache_writes,omitempty"`
	CacheReads  int32   `protobuf:"varint,10,opt,name=cache_reads,json=cacheReads,proto3" json:"cache_reads,omitempty"`
	ModelId     string  `protobuf:"bytes,11,opt,name=model_id,json=modelId,proto3" json:"model_id,omitempty"`
}

// ExecuteQuickWinRequest represents a request to execute a quick win
type ExecuteQuickWinRequest struct {
	Metadata *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Command  string    `protobuf:"bytes,2,opt,name=command,proto3" json:"command,omitempty"`
	Title    string    `protobuf:"bytes,3,opt,name=title,proto3" json:"title,omitempty"`
}

// DeleteAllTaskHistoryCount represents the result of deleting all task history
type DeleteAllTaskHistoryCount struct {
	TasksDeleted int32 `protobuf:"varint,1,opt,name=tasks_deleted,json=tasksDeleted,proto3" json:"tasks_deleted,omitempty"`
}

// ExplainChangesRequest represents a request to explain changes
type ExplainChangesRequest struct {
	Metadata  *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	MessageTs int64     `protobuf:"varint,2,opt,name=message_ts,json=messageTs,proto3" json:"message_ts,omitempty"`
}

// ============================================================================
// Timestamp helpers
// ============================================================================

// ToTimestamp converts a Go time.Time to a protobuf timestamp
func ToTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// FromTimestamp converts a protobuf timestamp to Go time.Time
func FromTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
