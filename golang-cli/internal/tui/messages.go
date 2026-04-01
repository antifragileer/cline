// Package tui provides message types for TUI communication.
package tui

// ChatUpdateMsg is sent when the chat needs to be updated
type ChatUpdateMsg struct {
	Messages []Message
}

// StreamChunkMsg is sent when a stream chunk is received
type StreamChunkMsg struct {
	Content string
	Done    bool
}

// StreamMessageMsg is sent when a complete stream message is received
type StreamMessageMsg struct {
	Message *Message
}

// StreamStateMsg represents the state of a stream
type StreamStateMsg struct {
	State   int
	Message string
}

// StatusUpdateMsg is sent when the status bar needs updating
type StatusUpdateMsg struct {
	Status  string
	Message string
}

// ProgressUpdateMsg is sent for progress updates
type ProgressUpdateMsg struct {
	Current int
	Total   int
	Message string
}

// ChatErrorMsg is sent when a chat error occurs
type ChatErrorMsg struct {
	Error error
}

// ApprovalRequestMsg is sent when approval is requested
type ApprovalRequestMsg struct {
	AskType  string
	Text     string
	Response chan<- string
}

// ChatResponse represents a response to a chat message
type ChatResponse struct {
	Text string
}

// ChatCancelledMsg is sent when chat is cancelled
type ChatCancelledMsg struct{}

// ChatCompletedMsg is sent when chat is completed
type ChatCompletedMsg struct {
	Success bool
	Summary string
}
