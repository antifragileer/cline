// Package task provides task execution and conversation management functionality
// for the Cline CLI.
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// ConversationStorageKey is the key used to store conversation data in file storage
const ConversationStorageKey = "conversation_history"

// DefaultMaxMessages is the default maximum number of messages before truncation
const DefaultMaxMessages = 10000

// DefaultMaxMessageSize is the maximum size in bytes for a single message content
const DefaultMaxMessageSize = 1024 * 1024 // 1MB

// DefaultPageSize is the default number of messages per page
const DefaultPageSize = 50

// ExportFormat represents the format for conversation export
type ExportFormat string

const (
	// ExportFormatMarkdown exports conversation as markdown
	ExportFormatMarkdown ExportFormat = "markdown"
	// ExportFormatJSON exports conversation as JSON
	ExportFormatJSON ExportFormat = "json"
	// ExportFormatPlainText exports conversation as plain text
	ExportFormatPlainText ExportFormat = "plaintext"
)

// ConversationMessage represents a message in the conversation history.
// This struct matches the proto definitions for ClineMessage serialization.
type ConversationMessage struct {
	// ID is the unique identifier for the message
	ID string `json:"id"`

	// Timestamp is when the message was created (matches proto ts field)
	Timestamp int64 `json:"ts"`

	// Type is the message type (ASK or SAY in proto terms)
	Type string `json:"type"`

	// AskType is set when Type is "ask" (matches proto ClineAsk enum)
	AskType string `json:"ask_type,omitempty"`

	// SayType is set when Type is "say" (matches proto ClineSay enum)
	SayType string `json:"say_type,omitempty"`

	// Content is the message text content
	Content string `json:"content"`

	// Reasoning contains the model's reasoning process
	Reasoning string `json:"reasoning,omitempty"`

	// Images contains base64-encoded image data
	Images []string `json:"images,omitempty"`

	// Files contains file paths referenced in the message
	Files []string `json:"files,omitempty"`

	// Partial indicates if this is a partial/streaming message
	Partial bool `json:"partial"`

	// LastCheckpointHash stores the git checkpoint hash
	LastCheckpointHash string `json:"last_checkpoint_hash,omitempty"`

	// IsCheckpointCheckedOut indicates if this is a checked out checkpoint
	IsCheckpointCheckedOut bool `json:"is_checkpoint_checked_out,omitempty"`

	// IsOperationOutsideWorkspace indicates if the operation is outside workspace
	IsOperationOutsideWorkspace bool `json:"is_operation_outside_workspace,omitempty"`

	// ConversationHistoryIndex tracks position in conversation history
	ConversationHistoryIndex int32 `json:"conversation_history_index,omitempty"`

	// DeletedRange stores truncated range information
	DeletedRange *ConversationHistoryDeletedRange `json:"deleted_range,omitempty"`

	// ToolName is set for tool-related messages
	ToolName string `json:"tool_name,omitempty"`

	// ToolInput contains tool arguments
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`

	// ToolResult contains tool execution result
	ToolResult string `json:"tool_result,omitempty"`

	// Metadata contains additional type-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Language is set for code blocks
	Language string `json:"language,omitempty"`
}

// ConversationHistoryDeletedRange represents a deleted range in conversation history
type ConversationHistoryDeletedRange struct {
	StartIndex int32 `json:"start_index"`
	EndIndex   int32 `json:"end_index"`
}

// PaginationParams defines pagination parameters for listing messages
type PaginationParams struct {
	// Offset is the number of messages to skip
	Offset int

	// Limit is the maximum number of messages to return
	Limit int

	// Direction determines message ordering: "asc" (oldest first) or "desc" (newest first)
	Direction string
}

// SearchParams defines search parameters for finding messages
type SearchParams struct {
	// Query is the search query string
	Query string

	// CaseSensitive determines if search is case-sensitive
	CaseSensitive bool

	// SearchInContent searches in message content
	SearchInContent bool

	// SearchInToolResults searches in tool results
	SearchInToolResults bool

	// MessageTypes filters by message types (empty = all types)
	MessageTypes []string

	// StartTime filters messages after this time
	StartTime *time.Time

	// EndTime filters messages before this time
	EndTime *time.Time

	// Pagination for search results
	Pagination PaginationParams
}

// ConversationStats holds statistics about the conversation
type ConversationStats struct {
	// TotalMessages is the total number of messages
	TotalMessages int `json:"total_messages"`

	// TotalTokens is the estimated total token count
	TotalTokens int `json:"total_tokens"`

	// UserMessages is the count of user messages
	UserMessages int `json:"user_messages"`

	// AIMessages is the count of AI messages
	AIMessages int `json:"ai_messages"`

	// ToolMessages is the count of tool-related messages
	ToolMessages int `json:"tool_messages"`

	// FirstMessageTime is the timestamp of the first message
	FirstMessageTime int64 `json:"first_message_time"`

	// LastMessageTime is the timestamp of the last message
	LastMessageTime int64 `json:"last_message_time"`

	// StorageSize is the approximate storage size in bytes
	StorageSize int64 `json:"storage_size"`
}

// ConversationManager handles conversation history storage, retrieval, and management
type ConversationManager struct {
	// storage is the file storage backend
	storage storage.FileStorage

	// messages holds the in-memory message cache
	messages []*ConversationMessage

	// mu protects concurrent access to messages
	mu sync.RWMutex

	// maxMessages is the maximum number of messages before truncation
	maxMessages int

	// maxMessageSize is the maximum size for a single message
	maxMessageSize int

	// dirty tracks if there are unsaved changes
	dirty bool

	// taskID identifies which task this conversation belongs to
	taskID string

	// dataDir is the directory for conversation data files
	dataDir string
}

// NewConversationManager creates a new conversation manager instance
func NewConversationManager(taskID string, dataDir string) (*ConversationManager, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required")
	}

	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".cline", "data", "tasks", taskID)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create conversation directory: %w", err)
	}

	storagePath := filepath.Join(dataDir, "conversation.json")
	fileStorage, err := storage.NewClineFileStorage(storagePath, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation storage: %w", err)
	}

	cm := &ConversationManager{
		storage:        fileStorage,
		messages:       make([]*ConversationMessage, 0),
		maxMessages:    DefaultMaxMessages,
		maxMessageSize: DefaultMaxMessageSize,
		taskID:         taskID,
		dataDir:        dataDir,
	}

	// Load existing messages
	if err := cm.load(); err != nil {
		return nil, fmt.Errorf("failed to load conversation: %w", err)
	}

	return cm, nil
}

// NewConversationManagerWithStorage creates a conversation manager with existing storage
func NewConversationManagerWithStorage(taskID string, fileStorage storage.FileStorage) (*ConversationManager, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required")
	}

	cm := &ConversationManager{
		storage:        fileStorage,
		messages:       make([]*ConversationMessage, 0),
		maxMessages:    DefaultMaxMessages,
		maxMessageSize: DefaultMaxMessageSize,
		taskID:         taskID,
		dataDir:        "",
	}

	// Load existing messages
	if err := cm.load(); err != nil {
		return nil, fmt.Errorf("failed to load conversation: %w", err)
	}

	return cm, nil
}

// Close closes the conversation manager and releases resources
func (cm *ConversationManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Flush any pending changes
	if cm.dirty {
		if err := cm.save(); err != nil {
			return fmt.Errorf("failed to save conversation on close: %w", err)
		}
	}

	return cm.storage.Close()
}

// AddMessage adds a new message to the conversation
func (cm *ConversationManager) AddMessage(msg *ConversationMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Validate and truncate message content if too large
	if len(msg.Content) > cm.maxMessageSize {
		msg.Content = msg.Content[:cm.maxMessageSize] + "\n... [content truncated]"
	}

	// Set timestamp if not set
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	// Generate ID if not set
	if msg.ID == "" {
		msg.ID = generateMessageID()
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check for and update partial messages of the same type
	if msg.Partial && len(cm.messages) > 0 {
		lastMsg := cm.messages[len(cm.messages)-1]
		if lastMsg.Type == msg.Type && lastMsg.Partial {
			// Update the last message instead of adding a new one
			*lastMsg = *msg
			cm.dirty = true
			return cm.save()
		}
	}

	// Add message to the list
	cm.messages = append(cm.messages, msg)
	cm.dirty = true

	// Check if truncation is needed
	if len(cm.messages) > cm.maxMessages {
		cm.truncateConversation()
	}

	return cm.save()
}

// AddMessageFromTUI converts a TUI message and adds it to the conversation
// Note: This is a placeholder that can be implemented when the tui package is available
func (cm *ConversationManager) AddMessageFromTUI(tuiMsg interface{}) error {
	// Type assertion would be done here when tui.Message is available
	// For now, return an error indicating this needs implementation
	return fmt.Errorf("AddMessageFromTUI requires tui package implementation")
}

// GetMessage retrieves a single message by ID
func (cm *ConversationManager) GetMessage(id string) (*ConversationMessage, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, msg := range cm.messages {
		if msg.ID == id {
			// Return a copy to prevent external modification
			return copyMessage(msg), true
		}
	}

	return nil, false
}

// GetMessages retrieves messages with pagination support (lazy loading)
func (cm *ConversationManager) GetMessages(params PaginationParams) ([]*ConversationMessage, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Set defaults
	if params.Limit <= 0 {
		params.Limit = DefaultPageSize
	}
	if params.Direction == "" {
		params.Direction = "asc"
	}

	total := len(cm.messages)
	if total == 0 {
		return []*ConversationMessage{}, nil
	}

	// Calculate slice bounds
	var start, end int
	if params.Direction == "desc" {
		// Newest first
		end = total - params.Offset
		start = end - params.Limit
		if start < 0 {
			start = 0
		}
		if end > total {
			end = total
		}
		if start >= end {
			return []*ConversationMessage{}, nil
		}
		// Reverse the slice for desc order
		result := make([]*ConversationMessage, 0, end-start)
		for i := end - 1; i >= start; i-- {
			result = append(result, copyMessage(cm.messages[i]))
		}
		return result, nil
	}

	// Oldest first (asc)
	start = params.Offset
	end = start + params.Limit
	if start > total {
		return []*ConversationMessage{}, nil
	}
	if end > total {
		end = total
	}

	result := make([]*ConversationMessage, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, copyMessage(cm.messages[i]))
	}

	return result, nil
}

// GetMessageRange retrieves messages within a specific index range (inclusive)
func (cm *ConversationManager) GetMessageRange(startIdx, endIdx int) ([]*ConversationMessage, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := len(cm.messages)
	if total == 0 {
		return []*ConversationMessage{}, nil
	}

	// Validate bounds
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= total {
		endIdx = total - 1
	}
	if startIdx > endIdx {
		return []*ConversationMessage{}, nil
	}

	result := make([]*ConversationMessage, 0, endIdx-startIdx+1)
	for i := startIdx; i <= endIdx; i++ {
		result = append(result, copyMessage(cm.messages[i]))
	}

	return result, nil
}

// GetAllMessages returns all messages (use with caution for large conversations)
func (cm *ConversationManager) GetAllMessages() ([]*ConversationMessage, error) {
	return cm.GetMessages(PaginationParams{
		Offset:    0,
		Limit:     cm.maxMessages,
		Direction: "asc",
	})
}

// GetLastMessage returns the most recent message
func (cm *ConversationManager) GetLastMessage() (*ConversationMessage, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if len(cm.messages) == 0 {
		return nil, false
	}

	return copyMessage(cm.messages[len(cm.messages)-1]), true
}

// GetLastNMessages returns the last N messages
func (cm *ConversationManager) GetLastNMessages(n int) ([]*ConversationMessage, error) {
	if n <= 0 {
		return []*ConversationMessage{}, nil
	}

	total := cm.GetMessageCount()
	if n > total {
		n = total
	}

	return cm.GetMessages(PaginationParams{
		Offset:    0,
		Limit:     n,
		Direction: "desc",
	})
}

// UpdateMessage updates an existing message by ID
func (cm *ConversationManager) UpdateMessage(id string, updates *ConversationMessage) error {
	if updates == nil {
		return fmt.Errorf("updates cannot be nil")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, msg := range cm.messages {
		if msg.ID == id {
			// Preserve the original ID and timestamp
			updates.ID = msg.ID
			updates.Timestamp = msg.Timestamp

			cm.messages[i] = updates
			cm.dirty = true
			return cm.save()
		}
	}

	return fmt.Errorf("message not found: %s", id)
}

// DeleteMessage removes a message by ID
func (cm *ConversationManager) DeleteMessage(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, msg := range cm.messages {
		if msg.ID == id {
			// Remove message by slicing
			cm.messages = append(cm.messages[:i], cm.messages[i+1:]...)
			cm.dirty = true
			return cm.save()
		}
	}

	return fmt.Errorf("message not found: %s", id)
}

// DeleteMessageRange removes messages within a specific index range
func (cm *ConversationManager) DeleteMessageRange(startIdx, endIdx int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	total := len(cm.messages)
	if startIdx < 0 || endIdx >= total || startIdx > endIdx {
		return fmt.Errorf("invalid range: %d-%d (total: %d)", startIdx, endIdx, total)
	}

	// Remove the range
	cm.messages = append(cm.messages[:startIdx], cm.messages[endIdx+1:]...)
	cm.dirty = true

	// Record the deleted range for context tracking
	if startIdx < len(cm.messages) {
		cm.messages[startIdx].DeletedRange = &ConversationHistoryDeletedRange{
			StartIndex: int32(startIdx),
			EndIndex:   int32(endIdx),
		}
	}

	return cm.save()
}

// SearchMessages searches for messages matching the search parameters
func (cm *ConversationManager) SearchMessages(params SearchParams) ([]*ConversationMessage, int, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var results []*ConversationMessage

	// Prepare search query
	query := params.Query
	if !params.CaseSensitive {
		query = strings.ToLower(query)
	}

	for _, msg := range cm.messages {
		// Filter by message type
		if len(params.MessageTypes) > 0 {
			found := false
			for _, mt := range params.MessageTypes {
				if strings.EqualFold(msg.Type, mt) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by time range
		if params.StartTime != nil && msg.Timestamp < params.StartTime.UnixMilli() {
			continue
		}
		if params.EndTime != nil && msg.Timestamp > params.EndTime.UnixMilli() {
			continue
		}

		// Search in content
		match := false
		if params.SearchInContent {
			content := msg.Content
			if !params.CaseSensitive {
				content = strings.ToLower(content)
			}
			if strings.Contains(content, query) {
				match = true
			}
		}

		// Search in tool results
		if !match && params.SearchInToolResults && msg.ToolResult != "" {
			toolResult := msg.ToolResult
			if !params.CaseSensitive {
				toolResult = strings.ToLower(toolResult)
			}
			if strings.Contains(toolResult, query) {
				match = true
			}
		}

		// Also search in reasoning if present
		if !match && msg.Reasoning != "" {
			reasoning := msg.Reasoning
			if !params.CaseSensitive {
				reasoning = strings.ToLower(reasoning)
			}
			if strings.Contains(reasoning, query) {
				match = true
			}
		}

		if match {
			results = append(results, copyMessage(msg))
		}
	}

	total := len(results)

	// Apply pagination to results
	if params.Pagination.Limit <= 0 {
		params.Pagination.Limit = DefaultPageSize
	}

	start := params.Pagination.Offset
	if start > total {
		return []*ConversationMessage{}, total, nil
	}

	end := start + params.Pagination.Limit
	if end > total {
		end = total
	}

	return results[start:end], total, nil
}

// SearchWithRegex searches messages using a regular expression pattern
func (cm *ConversationManager) SearchWithRegex(pattern string, caseSensitive bool) ([]*ConversationMessage, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	flags := "(?i)"
	if caseSensitive {
		flags = ""
	}

	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var results []*ConversationMessage
	for _, msg := range cm.messages {
		if re.MatchString(msg.Content) || re.MatchString(msg.Reasoning) || re.MatchString(msg.ToolResult) {
			results = append(results, copyMessage(msg))
		}
	}

	return results, nil
}

// Export exports the conversation to a file in the specified format
func (cm *ConversationManager) Export(format ExportFormat, outputPath string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var content []byte
	var err error

	switch format {
	case ExportFormatJSON:
		content, err = cm.exportJSON()
	case ExportFormatMarkdown:
		content, err = cm.exportMarkdown()
	case ExportFormatPlainText:
		content, err = cm.exportPlainText()
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to export conversation: %w", err)
	}

	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ExportToString exports the conversation to a string in the specified format
func (cm *ConversationManager) ExportToString(format ExportFormat) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var content []byte
	var err error

	switch format {
	case ExportFormatJSON:
		content, err = cm.exportJSON()
	case ExportFormatMarkdown:
		content, err = cm.exportMarkdown()
	case ExportFormatPlainText:
		content, err = cm.exportPlainText()
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to export conversation: %w", err)
	}

	return string(content), nil
}

// GetStats returns statistics about the conversation
func (cm *ConversationManager) GetStats() ConversationStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ConversationStats{
		TotalMessages: len(cm.messages),
		FirstMessageTime: 0,
		LastMessageTime: 0,
	}

	if len(cm.messages) > 0 {
		stats.FirstMessageTime = cm.messages[0].Timestamp
		stats.LastMessageTime = cm.messages[len(cm.messages)-1].Timestamp
	}

	for _, msg := range cm.messages {
		// Rough token estimation (4 chars ~= 1 token)
		estimatedTokens := len(msg.Content) / 4
		stats.TotalTokens += estimatedTokens

		switch msg.Type {
		case "user":
			stats.UserMessages++
		case "say", "ask":
			stats.AIMessages++
		case "tool_use", "tool_result":
			stats.ToolMessages++
		}
	}

	// Calculate storage size
	for _, msg := range cm.messages {
		stats.StorageSize += int64(len(msg.Content))
		stats.StorageSize += int64(len(msg.Reasoning))
		stats.StorageSize += int64(len(msg.ToolResult))
	}

	return stats
}

// TruncateConversation removes old messages to keep the conversation within limits
func (cm *ConversationManager) TruncateConversation(keepMessages int) error {
	if keepMessages <= 0 {
		return fmt.Errorf("keepMessages must be positive")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.truncateWithLimit(keepMessages)
}

// Clear removes all messages from the conversation
func (cm *ConversationManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.messages = make([]*ConversationMessage, 0)
	cm.dirty = true

	return cm.save()
}

// GetMessageCount returns the total number of messages
func (cm *ConversationManager) GetMessageCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.messages)
}

// IsEmpty returns true if the conversation has no messages
func (cm *ConversationManager) IsEmpty() bool {
	return cm.GetMessageCount() == 0
}

// SetMaxMessages sets the maximum number of messages before truncation
func (cm *ConversationManager) SetMaxMessages(max int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.maxMessages = max

	// Trigger truncation if needed
	if len(cm.messages) > cm.maxMessages {
		cm.truncateConversation()
		cm.save()
	}
}

// GetTaskID returns the task ID associated with this conversation
func (cm *ConversationManager) GetTaskID() string {
	return cm.taskID
}

// load loads messages from storage
func (cm *ConversationManager) load() error {
	val, ok := cm.storage.Get(ConversationStorageKey)
	if !ok {
		// No existing conversation
		return nil
	}

	// Convert to JSON and unmarshal
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal stored data: %w", err)
	}

	var messages []*ConversationMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	cm.messages = messages
	cm.dirty = false

	return nil
}

// save persists messages to storage
func (cm *ConversationManager) save() error {
	if err := cm.storage.Set(ConversationStorageKey, cm.messages); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	cm.dirty = false
	return nil
}

// truncateConversation removes oldest messages to stay within maxMessages limit
func (cm *ConversationManager) truncateConversation() {
	if len(cm.messages) <= cm.maxMessages {
		return
	}

	// Calculate how many messages to remove
	toRemove := len(cm.messages) - cm.maxMessages

	// Keep the first message (task initialization) and remove oldest after that
	// This preserves context about the original task
	if toRemove > 0 && len(cm.messages) > 1 {
		// Always keep the first message, remove from index 1 onwards
		startRemove := 1
		endRemove := toRemove + 1
		if endRemove >= len(cm.messages) {
			endRemove = len(cm.messages) - 1
		}

		if startRemove < endRemove {
			// Record the deletion
			if endRemove < len(cm.messages) {
				cm.messages[endRemove].DeletedRange = &ConversationHistoryDeletedRange{
					StartIndex: int32(startRemove),
					EndIndex:   int32(endRemove - 1),
				}
			}

			// Remove the range
			cm.messages = append(cm.messages[:startRemove], cm.messages[endRemove:]...)
		}
	}

	cm.dirty = true
}

// truncateWithLimit removes oldest messages to stay within the specified limit
func (cm *ConversationManager) truncateWithLimit(limit int) error {
	if len(cm.messages) <= limit {
		return nil
	}

	// Calculate how many messages to remove
	toRemove := len(cm.messages) - limit

	// Remove oldest messages first, but keep the first message
	if toRemove > 0 && len(cm.messages) > 1 {
		startRemove := 1
		endRemove := toRemove + 1
		if endRemove >= len(cm.messages) {
			endRemove = len(cm.messages) - 1
		}

		if startRemove < endRemove {
			cm.messages = append(cm.messages[:startRemove], cm.messages[endRemove:]...)
			cm.dirty = true
		}
	}

	if cm.dirty {
		return cm.save()
	}

	return nil
}

// exportJSON exports conversation as JSON
func (cm *ConversationManager) exportJSON() ([]byte, error) {
	export := struct {
		TaskID     string                  `json:"task_id"`
		ExportedAt int64                   `json:"exported_at"`
		Stats      ConversationStats       `json:"stats"`
		Messages   []*ConversationMessage  `json:"messages"`
	}{
		TaskID:     cm.taskID,
		ExportedAt: time.Now().UnixMilli(),
		Stats:      cm.GetStats(),
		Messages:   cm.messages,
	}

	return json.MarshalIndent(export, "", "  ")
}

// exportMarkdown exports conversation as Markdown
func (cm *ConversationManager) exportMarkdown() ([]byte, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Conversation Export - Task: %s\n\n", cm.taskID))
	sb.WriteString(fmt.Sprintf("Exported: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	stats := cm.GetStats()
	sb.WriteString("## Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- Total Messages: %d\n", stats.TotalMessages))
	sb.WriteString(fmt.Sprintf("- User Messages: %d\n", stats.UserMessages))
	sb.WriteString(fmt.Sprintf("- AI Messages: %d\n", stats.AIMessages))
	sb.WriteString(fmt.Sprintf("- Tool Messages: %d\n", stats.ToolMessages))
	sb.WriteString(fmt.Sprintf("- Estimated Tokens: %d\n\n", stats.TotalTokens))

	sb.WriteString("---\n\n")

	for _, msg := range cm.messages {
		timestamp := time.UnixMilli(msg.Timestamp).Format("15:04:05")

		switch msg.Type {
		case "user":
			sb.WriteString(fmt.Sprintf("### 👤 User (%s)\n\n", timestamp))
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case "say", "ask":
			sb.WriteString(fmt.Sprintf("### 🤖 Assistant (%s)\n\n", timestamp))
			if msg.Reasoning != "" {
				sb.WriteString("<details>\n<summary>Reasoning</summary>\n\n")
				sb.WriteString(msg.Reasoning)
				sb.WriteString("\n\n</details>\n\n")
			}
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case "tool_use":
			sb.WriteString(fmt.Sprintf("### 🔧 Tool: %s (%s)\n\n", msg.ToolName, timestamp))
			if len(msg.ToolInput) > 0 {
				sb.WriteString("**Input:**\n```json\n")
				inputJSON, _ := json.MarshalIndent(msg.ToolInput, "", "  ")
				sb.Write(inputJSON)
				sb.WriteString("\n```\n\n")
			}

		case "tool_result":
			sb.WriteString(fmt.Sprintf("### 📋 Tool Result: %s (%s)\n\n", msg.ToolName, timestamp))
			sb.WriteString("```\n")
			sb.WriteString(msg.ToolResult)
			sb.WriteString("\n```\n\n")

		case "error":
			sb.WriteString(fmt.Sprintf("### ❌ Error (%s)\n\n", timestamp))
			sb.WriteString("```\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n```\n\n")

		default:
			sb.WriteString(fmt.Sprintf("### 📝 %s (%s)\n\n", msg.Type, timestamp))
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}

		sb.WriteString("---\n\n")
	}

	return []byte(sb.String()), nil
}

// exportPlainText exports conversation as plain text
func (cm *ConversationManager) exportPlainText() ([]byte, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Conversation Export - Task: %s\n", cm.taskID))
	sb.WriteString(fmt.Sprintf("Exported: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	for _, msg := range cm.messages {
		timestamp := time.UnixMilli(msg.Timestamp).Format("15:04:05")

		switch msg.Type {
		case "user":
			sb.WriteString(fmt.Sprintf("[%s] USER: %s\n\n", timestamp, msg.Content))

		case "say", "ask":
			sb.WriteString(fmt.Sprintf("[%s] ASSISTANT: %s\n\n", timestamp, msg.Content))

		case "tool_use":
			sb.WriteString(fmt.Sprintf("[%s] TOOL %s: ", timestamp, msg.ToolName))
			inputJSON, _ := json.Marshal(msg.ToolInput)
			sb.WriteString(string(inputJSON))
			sb.WriteString("\n\n")

		case "tool_result":
			sb.WriteString(fmt.Sprintf("[%s] RESULT: %s\n\n", timestamp, msg.ToolResult))

		case "error":
			sb.WriteString(fmt.Sprintf("[%s] ERROR: %s\n\n", timestamp, msg.Content))

		default:
			sb.WriteString(fmt.Sprintf("[%s] %s: %s\n\n", timestamp, msg.Type, msg.Content))
		}
	}

	return []byte(sb.String()), nil
}

// copyMessage creates a deep copy of a message
func copyMessage(msg *ConversationMessage) *ConversationMessage {
	if msg == nil {
		return nil
	}

	msgCopy := &ConversationMessage{
		ID:                          msg.ID,
		Timestamp:                   msg.Timestamp,
		Type:                        msg.Type,
		AskType:                     msg.AskType,
		SayType:                     msg.SayType,
		Content:                     msg.Content,
		Reasoning:                   msg.Reasoning,
		Partial:                     msg.Partial,
		LastCheckpointHash:          msg.LastCheckpointHash,
		IsCheckpointCheckedOut:      msg.IsCheckpointCheckedOut,
		IsOperationOutsideWorkspace: msg.IsOperationOutsideWorkspace,
		ConversationHistoryIndex:    msg.ConversationHistoryIndex,
		ToolName:                    msg.ToolName,
		ToolResult:                  msg.ToolResult,
		Language:                    msg.Language,
		Metadata:                    make(map[string]interface{}),
	}

	// Deep copy slices
	if len(msg.Images) > 0 {
		msgCopy.Images = make([]string, len(msg.Images))
		copy(msgCopy.Images, msg.Images)
	}
	if len(msg.Files) > 0 {
		msgCopy.Files = make([]string, len(msg.Files))
		copy(msgCopy.Files, msg.Files)
	}

	// Deep copy metadata
	for k, v := range msg.Metadata {
		msgCopy.Metadata[k] = v
	}

	// Deep copy tool input
	if msg.ToolInput != nil {
		msgCopy.ToolInput = make(map[string]interface{})
		for k, v := range msg.ToolInput {
			msgCopy.ToolInput[k] = v
		}
	}

	// Deep copy deleted range
	if msg.DeletedRange != nil {
		msgCopy.DeletedRange = &ConversationHistoryDeletedRange{
			StartIndex: msg.DeletedRange.StartIndex,
			EndIndex:   msg.DeletedRange.EndIndex,
		}
	}

	return msgCopy
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond())
}

// ConversationStore manages multiple conversation managers for different tasks
type ConversationStore struct {
	// conversations maps task IDs to conversation managers
	conversations map[string]*ConversationManager

	// mu protects concurrent access
	mu sync.RWMutex

	// dataDir is the base directory for all conversations
	dataDir string

	// storage instances for reuse
	storageCache map[string]storage.FileStorage
}

// NewConversationStore creates a new conversation store
func NewConversationStore(dataDir string) (*ConversationStore, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".cline", "data", "tasks")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create conversation store directory: %w", err)
	}

	return &ConversationStore{
		conversations: make(map[string]*ConversationManager),
		dataDir:       dataDir,
		storageCache:  make(map[string]storage.FileStorage),
	}, nil
}

// GetOrCreateManager gets an existing conversation manager or creates a new one
func (cs *ConversationStore) GetOrCreateManager(taskID string) (*ConversationManager, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cm, ok := cs.conversations[taskID]; ok {
		return cm, nil
	}

	cm, err := NewConversationManager(taskID, filepath.Join(cs.dataDir, taskID))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation manager for task %s: %w", taskID, err)
	}

	cs.conversations[taskID] = cm
	return cm, nil
}

// GetManager gets an existing conversation manager without creating
func (cs *ConversationStore) GetManager(taskID string) (*ConversationManager, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cm, ok := cs.conversations[taskID]
	return cm, ok
}

// RemoveManager removes a conversation manager from the store
func (cs *ConversationStore) RemoveManager(taskID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cm, ok := cs.conversations[taskID]; ok {
		if err := cm.Close(); err != nil {
			return fmt.Errorf("failed to close conversation manager: %w", err)
		}
		delete(cs.conversations, taskID)
	}

	return nil
}

// ListTaskIDs returns all task IDs with conversations
func (cs *ConversationStore) ListTaskIDs() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	taskIDs := make([]string, 0, len(cs.conversations))
	for taskID := range cs.conversations {
		taskIDs = append(taskIDs, taskID)
	}

	sort.Strings(taskIDs)
	return taskIDs
}

// Close closes all conversation managers in the store
func (cs *ConversationStore) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var errs []string
	for taskID, cm := range cs.conversations {
		if err := cm.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("task %s: %v", taskID, err))
		}
	}

	cs.conversations = make(map[string]*ConversationManager)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing conversations: %s", strings.Join(errs, "; "))
	}

	return nil
}

// DeleteConversation deletes all data for a specific task
func (cs *ConversationStore) DeleteConversation(taskID string) error {
	// Remove from cache first
	if err := cs.RemoveManager(taskID); err != nil {
		return err
	}

	// Delete the directory
	taskDir := filepath.Join(cs.dataDir, taskID)
	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("failed to delete conversation directory: %w", err)
	}

	return nil
}