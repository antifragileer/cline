// Package host provides gRPC client functionality for communicating with the Cline extension.
// This file contains message type converters for bidirectional communication.
package host

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
)

// MessageConverter converts between different message formats
type MessageConverter struct{}

// NewMessageConverter creates a new message converter
func NewMessageConverter() *MessageConverter {
	return &MessageConverter{}
}

// ProtoToInternal converts a proto ClineMessage to internal format
func (c *MessageConverter) ProtoToInternal(proto *cline.ClineMessage) *ClineMessage {
	if proto == nil {
		return nil
	}

	return &ClineMessage{
		Ts:                          proto.Ts,
		Type:                        ClineMessageType(proto.Type),
		Ask:                         ClineAsk(proto.Ask),
		Say:                         ClineSay(proto.Say),
		Text:                        proto.Text,
		Reasoning:                   proto.Reasoning,
		Images:                      proto.Images,
		Files:                       proto.Files,
		Partial:                     proto.Partial,
		LastCheckpointHash:          proto.LastCheckpointHash,
		IsCheckpointCheckedOut:      proto.IsCheckpointCheckedOut,
		IsOperationOutsideWorkspace: proto.IsOperationOutsideWorkspace,
		ConversationHistoryIndex:    proto.ConversationHistoryIndex,
		SayTool:                     c.convertSayTool(proto.SayTool),
		SayBrowserAction:            c.convertSayBrowserAction(proto.SayBrowserAction),
		BrowserActionResult:         c.convertBrowserActionResult(proto.BrowserActionResult),
		AskUseMcpServer:             c.convertAskUseMcpServer(proto.AskUseMcpServer),
		PlanModeResponse:            c.convertPlanModeResponse(proto.PlanModeResponse),
		AskQuestion:                 c.convertAskQuestion(proto.AskQuestion),
		AskNewTask:                  c.convertAskNewTask(proto.AskNewTask),
		ApiReqInfo:                  c.convertApiReqInfo(proto.ApiReqInfo),
		ModelInfo:                   c.convertModelInfo(proto.ModelInfo),
	}
}

// InternalToProto converts an internal ClineMessage to proto format
func (c *MessageConverter) InternalToProto(internal *ClineMessage) *cline.ClineMessage {
	if internal == nil {
		return nil
	}

	return &cline.ClineMessage{
		Ts:                          internal.Ts,
		Type:                        cline.ClineMessageType(internal.Type),
		Ask:                         cline.ClineAsk(internal.Ask),
		Say:                         cline.ClineSay(internal.Say),
		Text:                        internal.Text,
		Reasoning:                   internal.Reasoning,
		Images:                      internal.Images,
		Files:                       internal.Files,
		Partial:                     internal.Partial,
		LastCheckpointHash:          internal.LastCheckpointHash,
		IsCheckpointCheckedOut:      internal.IsCheckpointCheckedOut,
		IsOperationOutsideWorkspace: internal.IsOperationOutsideWorkspace,
		ConversationHistoryIndex:    internal.ConversationHistoryIndex,
		SayTool:                     c.convertInternalSayTool(internal.SayTool),
		SayBrowserAction:            c.convertInternalSayBrowserAction(internal.SayBrowserAction),
		BrowserActionResult:         c.convertInternalBrowserActionResult(internal.BrowserActionResult),
		AskUseMcpServer:             c.convertInternalAskUseMcpServer(internal.AskUseMcpServer),
		PlanModeResponse:            c.convertInternalPlanModeResponse(internal.PlanModeResponse),
		AskQuestion:                 c.convertInternalAskQuestion(internal.AskQuestion),
		AskNewTask:                  c.convertInternalAskNewTask(internal.AskNewTask),
		ApiReqInfo:                  c.convertInternalApiReqInfo(internal.ApiReqInfo),
		ModelInfo:                   c.convertInternalModelInfo(internal.ModelInfo),
	}
}

// Helper conversion methods

func (c *MessageConverter) convertSayTool(tool *cline.ClineSayTool) *ClineSayTool {
	if tool == nil {
		return nil
	}
	return &ClineSayTool{
		Tool:                          ClineSayToolType(tool.Tool),
		Path:                          tool.Path,
		Diff:                          tool.Diff,
		Content:                       tool.Content,
		Regex:                         tool.Regex,
		FilePattern:                   tool.FilePattern,
		OperationIsLocatedInWorkspace: tool.OperationIsLocatedInWorkspace,
	}
}

func (c *MessageConverter) convertInternalSayTool(tool *ClineSayTool) *cline.ClineSayTool {
	if tool == nil {
		return nil
	}
	return &cline.ClineSayTool{
		Tool:                          cline.ClineSayToolType(tool.Tool),
		Path:                          tool.Path,
		Diff:                          tool.Diff,
		Content:                       tool.Content,
		Regex:                         tool.Regex,
		FilePattern:                   tool.FilePattern,
		OperationIsLocatedInWorkspace: tool.OperationIsLocatedInWorkspace,
	}
}

func (c *MessageConverter) convertSayBrowserAction(action *cline.ClineSayBrowserAction) *ClineSayBrowserAction {
	if action == nil {
		return nil
	}
	return &ClineSayBrowserAction{
		Action:     BrowserAction(action.Action),
		Coordinate: action.Coordinate,
		Text:       action.Text,
	}
}

func (c *MessageConverter) convertInternalSayBrowserAction(action *ClineSayBrowserAction) *cline.ClineSayBrowserAction {
	if action == nil {
		return nil
	}
	return &cline.ClineSayBrowserAction{
		Action:     cline.BrowserAction(action.Action),
		Coordinate: action.Coordinate,
		Text:       action.Text,
	}
}

func (c *MessageConverter) convertBrowserActionResult(result *cline.BrowserActionResult) *BrowserActionResult {
	if result == nil {
		return nil
	}
	return &BrowserActionResult{
		Screenshot:           result.Screenshot,
		Logs:                 result.Logs,
		CurrentUrl:           result.CurrentUrl,
		CurrentMousePosition: result.CurrentMousePosition,
	}
}

func (c *MessageConverter) convertInternalBrowserActionResult(result *BrowserActionResult) *cline.BrowserActionResult {
	if result == nil {
		return nil
	}
	return &cline.BrowserActionResult{
		Screenshot:           result.Screenshot,
		Logs:                 result.Logs,
		CurrentUrl:           result.CurrentUrl,
		CurrentMousePosition: result.CurrentMousePosition,
	}
}

func (c *MessageConverter) convertAskUseMcpServer(mcp *cline.ClineAskUseMcpServer) *ClineAskUseMcpServer {
	if mcp == nil {
		return nil
	}
	return &ClineAskUseMcpServer{
		ServerName: mcp.ServerName,
		Type:       McpServerRequestType(mcp.Type),
		ToolName:   mcp.ToolName,
		Arguments:  mcp.Arguments,
		Uri:        mcp.Uri,
	}
}

func (c *MessageConverter) convertInternalAskUseMcpServer(mcp *ClineAskUseMcpServer) *cline.ClineAskUseMcpServer {
	if mcp == nil {
		return nil
	}
	return &cline.ClineAskUseMcpServer{
		ServerName: mcp.ServerName,
		Type:       cline.McpServerRequestType(mcp.Type),
		ToolName:   mcp.ToolName,
		Arguments:  mcp.Arguments,
		Uri:        mcp.Uri,
	}
}

func (c *MessageConverter) convertPlanModeResponse(resp *cline.ClinePlanModeResponse) *ClinePlanModeResponse {
	if resp == nil {
		return nil
	}
	return &ClinePlanModeResponse{
		Response: resp.Response,
		Options:  resp.Options,
		Selected: resp.Selected,
	}
}

func (c *MessageConverter) convertInternalPlanModeResponse(resp *ClinePlanModeResponse) *cline.ClinePlanModeResponse {
	if resp == nil {
		return nil
	}
	return &cline.ClinePlanModeResponse{
		Response: resp.Response,
		Options:  resp.Options,
		Selected: resp.Selected,
	}
}

func (c *MessageConverter) convertAskQuestion(q *cline.ClineAskQuestion) *ClineAskQuestion {
	if q == nil {
		return nil
	}
	return &ClineAskQuestion{
		Question: q.Question,
		Options:  q.Options,
		Selected: q.Selected,
	}
}

func (c *MessageConverter) convertInternalAskQuestion(q *ClineAskQuestion) *cline.ClineAskQuestion {
	if q == nil {
		return nil
	}
	return &cline.ClineAskQuestion{
		Question: q.Question,
		Options:  q.Options,
		Selected: q.Selected,
	}
}

func (c *MessageConverter) convertAskNewTask(t *cline.ClineAskNewTask) *ClineAskNewTask {
	if t == nil {
		return nil
	}
	return &ClineAskNewTask{
		Context: t.Context,
	}
}

func (c *MessageConverter) convertInternalAskNewTask(t *ClineAskNewTask) *cline.ClineAskNewTask {
	if t == nil {
		return nil
	}
	return &cline.ClineAskNewTask{
		Context: t.Context,
	}
}

func (c *MessageConverter) convertApiReqInfo(info *cline.ClineApiReqInfo) *ClineApiReqInfo {
	if info == nil {
		return nil
	}
	return &ClineApiReqInfo{
		Request:                info.Request,
		TokensIn:               info.TokensIn,
		TokensOut:              info.TokensOut,
		CacheWrites:            info.CacheWrites,
		CacheReads:             info.CacheReads,
		Cost:                   info.Cost,
		CancelReason:           ClineApiReqCancelReason(info.CancelReason),
		StreamingFailedMessage: info.StreamingFailedMessage,
		RetryStatus:            c.convertRetryStatus(info.RetryStatus),
	}
}

func (c *MessageConverter) convertInternalApiReqInfo(info *ClineApiReqInfo) *cline.ClineApiReqInfo {
	if info == nil {
		return nil
	}
	return &cline.ClineApiReqInfo{
		Request:                info.Request,
		TokensIn:               info.TokensIn,
		TokensOut:              info.TokensOut,
		CacheWrites:            info.CacheWrites,
		CacheReads:             info.CacheReads,
		Cost:                   info.Cost,
		CancelReason:           cline.ClineApiReqCancelReason(info.CancelReason),
		StreamingFailedMessage: info.StreamingFailedMessage,
		RetryStatus:            c.convertInternalRetryStatus(info.RetryStatus),
	}
}

func (c *MessageConverter) convertRetryStatus(status *cline.ApiReqRetryStatus) *ApiReqRetryStatus {
	if status == nil {
		return nil
	}
	return &ApiReqRetryStatus{
		Attempt:      status.Attempt,
		MaxAttempts:  status.MaxAttempts,
		DelaySec:     status.DelaySec,
		ErrorSnippet: status.ErrorSnippet,
	}
}

func (c *MessageConverter) convertInternalRetryStatus(status *ApiReqRetryStatus) *cline.ApiReqRetryStatus {
	if status == nil {
		return nil
	}
	return &cline.ApiReqRetryStatus{
		Attempt:      status.Attempt,
		MaxAttempts:  status.MaxAttempts,
		DelaySec:     status.DelaySec,
		ErrorSnippet: status.ErrorSnippet,
	}
}

func (c *MessageConverter) convertModelInfo(info *cline.ClineModelInfo) *ClineModelInfo {
	if info == nil {
		return nil
	}
	return &ClineModelInfo{
		ProviderId: info.ProviderId,
		ModelId:    info.ModelId,
	}
}

func (c *MessageConverter) convertInternalModelInfo(info *ClineModelInfo) *cline.ClineModelInfo {
	if info == nil {
		return nil
	}
	return &cline.ClineModelInfo{
		ProviderId: info.ProviderId,
		ModelId:    info.ModelId,
	}
}

// CreateMessageFromJSON creates a ClineMessage from JSON bytes
func (c *MessageConverter) CreateMessageFromJSON(data []byte) (*ClineMessageProto, error) {
	var msg ClineMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return &ClineMessageProto{ClineMessage: &msg}, nil
}

// ConvertTimestamp converts milliseconds to time.Time
func (c *MessageConverter) ConvertTimestamp(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

// CreateTimestamp creates a timestamp from current time
func (c *MessageConverter) CreateTimestamp() int64 {
	return time.Now().UnixMilli()
}

// IsCompleteMessage checks if a message is complete (not partial)
func (c *MessageConverter) IsCompleteMessage(msg *ClineMessageProto) bool {
	if msg == nil || msg.ClineMessage == nil {
		return false
	}
	return !msg.Partial
}

// ExtractMessageType extracts the type of message
func (c *MessageConverter) ExtractMessageType(msg *ClineMessageProto) string {
	if msg == nil || msg.ClineMessage == nil {
		return "unknown"
	}

	switch msg.Type {
	case ClineMessageType_ASK:
		return fmt.Sprintf("ask:%d", msg.Ask)
	case ClineMessageType_SAY:
		return fmt.Sprintf("say:%d", msg.Say)
	default:
		return "unknown"
	}
}