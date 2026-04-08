package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

var (
	// ErrStreamClosed is returned when attempting to use a closed stream
	ErrStreamClosed = errors.New("stream is closed")
	// ErrStreamNotReady is returned when the stream is not ready for operations
	ErrStreamNotReady = errors.New("stream not ready")
	// ErrBackpressureExceeded is returned when backpressure limits are exceeded
	ErrBackpressureExceeded = errors.New("backpressure limit exceeded")
	// ErrReconnectionFailed is returned when all reconnection attempts fail
	ErrReconnectionFailed = errors.New("reconnection failed after maximum retries")
)

// StreamState represents the current state of a bidirectional stream
type StreamState int32

const (
	StreamStateDisconnected StreamState = iota
	StreamStateConnecting
	StreamStateReady
	StreamStateReconnecting
	StreamStateClosed
)

func (s StreamState) String() string {
	switch s {
	case StreamStateDisconnected:
		return "disconnected"
	case StreamStateConnecting:
		return "connecting"
	case StreamStateReady:
		return "ready"
	case StreamStateReconnecting:
		return "reconnecting"
	case StreamStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// StreamConfig contains configuration for a bidirectional stream
type StreamConfig struct {
	// StreamCreator is a function that creates the actual gRPC stream
	StreamCreator func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error)
	// BufferSize is the size of the send/receive buffers
	BufferSize int
	// MaxOutstandingMessages is the maximum number of messages allowed in flight
	MaxOutstandingMessages int
	// ReconnectDelay is the delay between reconnection attempts
	ReconnectDelay time.Duration
	// MaxReconnectAttempts is the maximum number of reconnection attempts
	MaxReconnectAttempts int
	// EnablePartialMessages enables support for partial/chunked messages
	EnablePartialMessages bool
	// MaxChunkSize is the maximum size of a message chunk
	MaxChunkSize int
	// OnStateChange is called when the stream state changes
	OnStateChange func(StreamState)
	// OnError is called when an error occurs
	OnError func(error)
	// OnMessageReceived is called when a message is received
	OnMessageReceived func(*ClineMessageProto)
	// TaskID is the ID of the task for this stream
	TaskID string
}

// DefaultStreamConfig returns a default stream configuration
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		BufferSize:             100,
		MaxOutstandingMessages: 50,
		ReconnectDelay:         2 * time.Second,
		MaxReconnectAttempts:   5,
		EnablePartialMessages:  true,
		MaxChunkSize:           4096,
	}
}

// BidiStream manages a bidirectional gRPC stream with reconnection support
type BidiStream struct {
	config *StreamConfig

	// State management
	state atomic.Int32

	// Channels for message flow
	sendChan    chan *ClineMessageProto
	recvChan    chan *ClineMessageProto
	ackChan     chan int64
	controlChan chan controlMessage

	// Flow control
	outstandingMsgs atomic.Int32
	backpressure    chan struct{}

	// Partial message handling
	partialMu       sync.RWMutex
	partialBuffers  map[int64][]*ClineMessageProto
	partialTimeouts map[int64]time.Time

	// Stream management
	stream     TaskService_StreamClient
	streamMu   sync.RWMutex
	cancelFunc context.CancelFunc

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Reconnection management
	reconnectCount atomic.Int32
	reconnecting   atomic.Bool

	// Graceful shutdown
	closeOnce sync.Once
}

// controlMessage is used for internal control signals
type controlMessage struct {
	action string
	value  interface{}
}

// NewBidiStream creates a new bidirectional stream with the given configuration
func NewBidiStream(config *StreamConfig) (*BidiStream, error) {
	if config == nil {
		config = DefaultStreamConfig()
	}

	if config.StreamCreator == nil {
		return nil, errors.New("stream creator is required")
	}

	if config.BufferSize <= 0 {
		config.BufferSize = DefaultStreamConfig().BufferSize
	}

	if config.MaxOutstandingMessages <= 0 {
		config.MaxOutstandingMessages = DefaultStreamConfig().MaxOutstandingMessages
	}

	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = DefaultStreamConfig().ReconnectDelay
	}

	if config.MaxReconnectAttempts <= 0 {
		config.MaxReconnectAttempts = DefaultStreamConfig().MaxReconnectAttempts
	}

	if config.MaxChunkSize <= 0 {
		config.MaxChunkSize = DefaultStreamConfig().MaxChunkSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &BidiStream{
		config:          config,
		sendChan:        make(chan *ClineMessageProto, config.BufferSize),
		recvChan:        make(chan *ClineMessageProto, config.BufferSize),
		ackChan:         make(chan int64, config.BufferSize),
		controlChan:     make(chan controlMessage, 10),
		backpressure:    make(chan struct{}, config.MaxOutstandingMessages),
		partialBuffers:  make(map[int64][]*ClineMessageProto),
		partialTimeouts: make(map[int64]time.Time),
		ctx:             ctx,
		cancel:          cancel,
	}

	s.state.Store(int32(StreamStateDisconnected))
	return s, nil
}

// Start establishes the bidirectional stream and begins processing
func (s *BidiStream) Start() error {
	if s.getState() != StreamStateDisconnected {
		return fmt.Errorf("cannot start stream in state %s", s.getState())
	}

	s.setState(StreamStateConnecting)

	// Establish initial connection
	if err := s.connect(); err != nil {
		s.setState(StreamStateDisconnected)
		return fmt.Errorf("failed to establish initial connection: %w", err)
	}

	s.setState(StreamStateReady)

	// Start goroutines for handling the stream
	s.wg.Add(3)
	go s.sendLoop()
	go s.receiveLoop()
	go s.controlLoop()

	return nil
}

// Stop gracefully shuts down the stream
func (s *BidiStream) Stop() error {
	var closeErr error

	s.closeOnce.Do(func() {
		s.setState(StreamStateClosed)
		s.cancel()

		// Close channels
		close(s.sendChan)
		close(s.controlChan)

		// Wait for goroutines to finish
		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Goroutines finished cleanly
		case <-time.After(10 * time.Second):
			closeErr = errors.New("timeout waiting for goroutines to finish")
		}

		// Close the stream
		s.streamMu.Lock()
		if s.stream != nil {
			s.stream.CloseSend()
		}
		s.streamMu.Unlock()

		close(s.recvChan)
		close(s.ackChan)
	})

	return closeErr
}

// Send sends a message through the stream
func (s *BidiStream) Send(msg *ClineMessageProto) error {
	if msg == nil {
		return errors.New("message is nil")
	}

	if s.getState() != StreamStateReady {
		return ErrStreamNotReady
	}

	// Check backpressure
	select {
	case s.backpressure <- struct{}{}:
		// Got token, proceed
	default:
		return ErrBackpressureExceeded
	}

	// Handle partial messages
	if s.config.EnablePartialMessages && len(msg.Text) > s.config.MaxChunkSize {
		return s.sendPartialMessage(msg)
	}

	// Send complete message
	select {
	case s.sendChan <- msg:
		s.outstandingMsgs.Add(1)
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("send timeout")
	}
}

// sendPartialMessage breaks a large message into chunks and sends them
func (s *BidiStream) sendPartialMessage(msg *ClineMessageProto) error {
	chunks := chunkMessage(msg, s.config.MaxChunkSize)

	for _, chunk := range chunks {
		select {
		case s.sendChan <- chunk:
			s.outstandingMsgs.Add(1)
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(5 * time.Second):
			return errors.New("partial message send timeout")
		}
	}

	return nil
}

// chunkMessage breaks a message into chunks
func chunkMessage(msg *ClineMessageProto, chunkSize int) []*ClineMessageProto {
	text := msg.Text
	if len(text) <= chunkSize {
		return []*ClineMessageProto{msg}
	}

	numChunks := (len(text) + chunkSize - 1) / chunkSize
	chunks := make([]*ClineMessageProto, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Ts:      msg.Ts,
				Type:    msg.Type,
				Ask:     msg.Ask,
				Say:     msg.Say,
				Text:    text[start:end],
				Images:  msg.Images,
				Files:   msg.Files,
				Partial: true,
			},
		}
		chunks[i] = chunk
	}

	// Mark the last chunk as not partial
	if len(chunks) > 0 {
		chunks[len(chunks)-1].Partial = false
	}

	return chunks
}

// Receive returns a channel for receiving messages
func (s *BidiStream) Receive() <-chan *ClineMessageProto {
	return s.recvChan
}

// Acknowledge acknowledges receipt of a message
func (s *BidiStream) Acknowledge(ts int64) {
	select {
	case s.ackChan <- ts:
	case <-s.ctx.Done():
	}
}

// GetState returns the current stream state
func (s *BidiStream) GetState() StreamState {
	return s.getState()
}

func (s *BidiStream) getState() StreamState {
	return StreamState(s.state.Load())
}

func (s *BidiStream) setState(state StreamState) {
	oldState := s.getState()
	if oldState == state {
		return
	}

	s.state.Store(int32(state))

	if s.config.OnStateChange != nil {
		s.config.OnStateChange(state)
	}
}

// connect establishes the gRPC stream connection
func (s *BidiStream) connect() error {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.cancelFunc = cancel

	// Pass nil for connection - StreamCreator is responsible for establishing the connection
	stream, err := s.config.StreamCreator(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	s.stream = stream
	return nil
}

// sendLoop handles sending messages to the stream
func (s *BidiStream) sendLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-s.sendChan:
			if !ok {
				return
			}

			if err := s.sendMessage(msg); err != nil {
				s.handleError(fmt.Errorf("send error: %w", err))
				s.releaseBackpressure(msg.Ts)
			}
		}
	}
}

// sendMessage sends a single message to the stream
func (s *BidiStream) sendMessage(msg *ClineMessageProto) error {
	s.streamMu.RLock()
	stream := s.stream
	s.streamMu.RUnlock()

	if stream == nil {
		return ErrStreamNotReady
	}

	return stream.Send(msg)
}

// receiveLoop handles receiving messages from the stream
func (s *BidiStream) receiveLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		msg, err := s.receiveMessage()
		if err != nil {
			if err == io.EOF || errors.Is(err, context.Canceled) {
				return
			}

			s.handleError(fmt.Errorf("receive error: %w", err))

			// Trigger reconnection on receive errors
			if s.getState() == StreamStateReady {
				s.triggerReconnection()
			}
			continue
		}

		if msg == nil {
			continue
		}

		// Handle partial messages
		if msg.Partial {
			msg = s.handlePartialMessage(msg)
			if msg == nil {
				// Not complete yet
				continue
			}
		}

		// Send to receive channel
		select {
		case s.recvChan <- msg:
			if s.config.OnMessageReceived != nil {
				s.config.OnMessageReceived(msg)
			}
		case <-s.ctx.Done():
			return
		}

		// Release backpressure for acknowledged messages
		s.releaseBackpressure(msg.Ts)
	}
}

// receiveMessage receives a single message from the stream
func (s *BidiStream) receiveMessage() (*ClineMessageProto, error) {
	s.streamMu.RLock()
	stream := s.stream
	s.streamMu.RUnlock()

	if stream == nil {
		return nil, ErrStreamNotReady
	}

	return stream.Recv()
}

// handlePartialMessage processes partial message chunks
func (s *BidiStream) handlePartialMessage(chunk *ClineMessageProto) *ClineMessageProto {
	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	ts := chunk.Ts

	// Initialize buffer for this message
	if _, exists := s.partialBuffers[ts]; !exists {
		s.partialBuffers[ts] = make([]*ClineMessageProto, 0)
		s.partialTimeouts[ts] = time.Now().Add(30 * time.Second)
	}

	// Add chunk to buffer
	s.partialBuffers[ts] = append(s.partialBuffers[ts], chunk)

	// Check if we have all chunks (simple heuristic: non-partial chunk marks completion)
	if !chunk.Partial {
		// Reassemble message
		msg := s.reassembleMessage(ts)
		delete(s.partialBuffers, ts)
		delete(s.partialTimeouts, ts)
		return msg
	}

	return nil
}

// reassembleMessage combines chunks into a complete message
func (s *BidiStream) reassembleMessage(ts int64) *ClineMessageProto {
	chunks := s.partialBuffers[ts]
	if len(chunks) == 0 {
		return nil
	}

	// Reassemble text
	var fullText string
	for _, chunk := range chunks {
		fullText += chunk.Text
	}

	// Create complete message from first chunk
	first := chunks[0]
	return &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:      first.Ts,
			Type:    first.Type,
			Ask:     first.Ask,
			Say:     first.Say,
			Text:    fullText,
			Images:  first.Images,
			Files:   first.Files,
			Partial: false,
		},
	}
}

// controlLoop handles control messages and cleanup
func (s *BidiStream) controlLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupPartialMessages()
		}
	}
}

// cleanupPartialMessages removes stale partial message buffers
func (s *BidiStream) cleanupPartialMessages() {
	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	now := time.Now()
	for ts, timeout := range s.partialTimeouts {
		if now.After(timeout) {
			delete(s.partialBuffers, ts)
			delete(s.partialTimeouts, ts)
		}
	}
}

// triggerReconnection initiates the reconnection process
func (s *BidiStream) triggerReconnection() {
	if s.reconnecting.Swap(true) {
		// Already reconnecting
		return
	}

	go s.reconnect()
}

// reconnect attempts to reconnect the stream
func (s *BidiStream) reconnect() {
	defer s.reconnecting.Store(false)

	s.setState(StreamStateReconnecting)

	for attempt := 0; attempt < s.config.MaxReconnectAttempts; attempt++ {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(s.config.ReconnectDelay * time.Duration(attempt+1)):
		}

		if err := s.connect(); err != nil {
			s.reconnectCount.Add(1)
			if s.config.OnError != nil {
				s.config.OnError(fmt.Errorf("reconnection attempt %d failed: %w", attempt+1, err))
			}
			continue
		}

		// Reconnection successful
		s.reconnectCount.Store(0)
		s.setState(StreamStateReady)
		return
	}

	// All reconnection attempts failed
	s.handleError(ErrReconnectionFailed)
	s.setState(StreamStateDisconnected)
}

// handleError handles errors and invokes the error callback
func (s *BidiStream) handleError(err error) {
	if s.config.OnError != nil {
		s.config.OnError(err)
	}
}

// releaseBackpressure releases a backpressure token
func (s *BidiStream) releaseBackpressure(ts int64) {
	select {
	case <-s.backpressure:
		s.outstandingMsgs.Add(-1)
	default:
	}
}

// Stats returns stream statistics
func (s *BidiStream) Stats() StreamStats {
	return StreamStats{
		State:               s.getState(),
		OutstandingMessages: int(s.outstandingMsgs.Load()),
		ReconnectCount:      int(s.reconnectCount.Load()),
		SendQueueLen:        len(s.sendChan),
		RecvQueueLen:        len(s.recvChan),
	}
}

// StreamStats contains stream statistics
type StreamStats struct {
	State               StreamState
	OutstandingMessages int
	ReconnectCount      int
	SendQueueLen        int
	RecvQueueLen        int
}

// WaitForReady blocks until the stream is ready or context is cancelled
func (s *BidiStream) WaitForReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if s.getState() == StreamStateReady {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// IsReady returns true if the stream is ready for sending/receiving
func (s *BidiStream) IsReady() bool {
	return s.getState() == StreamStateReady
}

// ForceReconnect forces a reconnection of the stream
func (s *BidiStream) ForceReconnect() error {
	if s.getState() == StreamStateClosed {
		return ErrStreamClosed
	}

	s.triggerReconnection()
	return nil
}

// ============================================================================
// TaskServiceClient Implementation
// ============================================================================

// taskServiceClient implements TaskServiceClient
type taskServiceClient struct {
	cc grpc.ClientConnInterface
}

// NewTaskServiceClient creates a new TaskServiceClient
func NewTaskServiceClient(cc grpc.ClientConnInterface) TaskServiceClient {
	return &taskServiceClient{cc}
}

func (c *taskServiceClient) CancelTask(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/CancelTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) CancelBackgroundCommand(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/CancelBackgroundCommand", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ClearTask(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ClearTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) GetTotalTasksSize(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*Int64, error) {
	out := new(Int64)
	err := c.cc.Invoke(ctx, "/cline.TaskService/GetTotalTasksSize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) DeleteTasksWithIds(ctx context.Context, in *StringArrayRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/DeleteTasksWithIds", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) NewTask(ctx context.Context, in *NewTaskRequest, opts ...grpc.CallOption) (*String, error) {
	out := new(String)
	err := c.cc.Invoke(ctx, "/cline.TaskService/NewTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ShowTaskWithId(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*TaskResponse, error) {
	out := new(TaskResponse)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ShowTaskWithId", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ExportTaskWithId(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ExportTaskWithId", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ToggleTaskFavorite(ctx context.Context, in *TaskFavoriteRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ToggleTaskFavorite", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) GetTaskHistory(ctx context.Context, in *GetTaskHistoryRequest, opts ...grpc.CallOption) (*TaskHistoryArray, error) {
	out := new(TaskHistoryArray)
	err := c.cc.Invoke(ctx, "/cline.TaskService/GetTaskHistory", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) AskResponse(ctx context.Context, in *AskResponseRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/AskResponse", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) TaskFeedback(ctx context.Context, in *StringRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/TaskFeedback", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) TaskCompletionViewChanges(ctx context.Context, in *Int64Request, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/TaskCompletionViewChanges", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ExecuteQuickWin(ctx context.Context, in *ExecuteQuickWinRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ExecuteQuickWin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) DeleteAllTaskHistory(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*DeleteAllTaskHistoryCount, error) {
	out := new(DeleteAllTaskHistoryCount)
	err := c.cc.Invoke(ctx, "/cline.TaskService/DeleteAllTaskHistory", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) ExplainChanges(ctx context.Context, in *ExplainChangesRequest, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cline.TaskService/ExplainChanges", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *taskServiceClient) Stream(ctx context.Context, opts ...grpc.CallOption) (TaskService_StreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &TaskService_ServiceDesc.Streams[0], "/cline.TaskService/Stream", opts...)
	if err != nil {
		return nil, err
	}
	x := &taskServiceStreamClient{stream}
	return x, nil
}

// TaskService_ServiceDesc is the grpc.ServiceDesc for TaskService service
var TaskService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cline.TaskService",
	HandlerType: (*interface{})(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CancelTask",
			Handler:    nil,
		},
		{
			MethodName: "CancelBackgroundCommand",
			Handler:    nil,
		},
		{
			MethodName: "ClearTask",
			Handler:    nil,
		},
		{
			MethodName: "GetTotalTasksSize",
			Handler:    nil,
		},
		{
			MethodName: "DeleteTasksWithIds",
			Handler:    nil,
		},
		{
			MethodName: "NewTask",
			Handler:    nil,
		},
		{
			MethodName: "ShowTaskWithId",
			Handler:    nil,
		},
		{
			MethodName: "ExportTaskWithId",
			Handler:    nil,
		},
		{
			MethodName: "ToggleTaskFavorite",
			Handler:    nil,
		},
		{
			MethodName: "GetTaskHistory",
			Handler:    nil,
		},
		{
			MethodName: "AskResponse",
			Handler:    nil,
		},
		{
			MethodName: "TaskFeedback",
			Handler:    nil,
		},
		{
			MethodName: "TaskCompletionViewChanges",
			Handler:    nil,
		},
		{
			MethodName: "ExecuteQuickWin",
			Handler:    nil,
		},
		{
			MethodName: "DeleteAllTaskHistory",
			Handler:    nil,
		},
		{
			MethodName: "ExplainChanges",
			Handler:    nil,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Stream",
			Handler:       nil,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "cline/task.proto",
}

// taskServiceStreamClient implements TaskService_StreamClient
type taskServiceStreamClient struct {
	grpc.ClientStream
}

func (x *taskServiceStreamClient) Send(m *ClineMessageProto) error {
	return x.ClientStream.SendMsg(m)
}

func (x *taskServiceStreamClient) Recv() (*ClineMessageProto, error) {
	m := new(ClineMessageProto)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ============================================================================
// Message Routing and Task Stream Handler
// ============================================================================

// TaskStreamHandler handles bidirectional communication for a task
type TaskStreamHandler struct {
	stream      *BidiStream
	taskID      string
	messageChan chan *ClineMessageProto
	errorChan   chan error
	doneChan    chan struct{}

	// Callbacks for different message types
	OnTextMessage    func(text string)
	OnToolRequest    func(tool *ClineSayTool) error
	OnCommandRequest func(command string) error
	OnCompletion     func(result string)
	OnError          func(err error)
	OnAskQuestion    func(question *ClineAskQuestion) (string, error)
}

// NewTaskStreamHandler creates a new task stream handler
func NewTaskStreamHandler(taskID string) *TaskStreamHandler {
	return &TaskStreamHandler{
		taskID:      taskID,
		messageChan: make(chan *ClineMessageProto, 100),
		errorChan:   make(chan error, 10),
		doneChan:    make(chan struct{}),
	}
}

// Start starts the task stream handler with the given client
func (h *TaskStreamHandler) Start(streamCreator func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error)) error {
	config := &StreamConfig{
		StreamCreator: streamCreator,
		TaskID:        h.taskID,
		OnMessageReceived: func(msg *ClineMessageProto) {
			select {
			case h.messageChan <- msg:
			case <-h.doneChan:
			}
		},
		OnError: func(err error) {
			select {
			case h.errorChan <- err:
			case <-h.doneChan:
			}
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		return fmt.Errorf("failed to create bidirectional stream: %w", err)
	}

	h.stream = stream

	if err := stream.Start(); err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	// Start message routing goroutine
	go h.routeMessages()

	return nil
}

// Stop stops the task stream handler
func (h *TaskStreamHandler) Stop() error {
	close(h.doneChan)
	if h.stream != nil {
		return h.stream.Stop()
	}
	return nil
}

// SendMessage sends a user message to the task
func (h *TaskStreamHandler) SendMessage(text string, images []string, files []string) error {
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:     time.Now().UnixMilli(),
			Type:   ClineMessageType_SAY,
			Say:    ClineSay_TEXT,
			Text:   text,
			Images: images,
			Files:  files,
		},
	}
	return h.stream.Send(msg)
}

// SendAskResponse sends a response to an ask
func (h *TaskStreamHandler) SendAskResponse(responseType string, text string, images []string, files []string) error {
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:     time.Now().UnixMilli(),
			Type:   ClineMessageType_SAY,
			Say:    ClineSay_USER_FEEDBACK,
			Text:   text,
			Images: images,
			Files:  files,
		},
	}
	return h.stream.Send(msg)
}

// routeMessages routes incoming messages to appropriate handlers
func (h *TaskStreamHandler) routeMessages() {
	for {
		select {
		case <-h.doneChan:
			return
		case msg := <-h.messageChan:
			h.handleMessage(msg)
		case err := <-h.errorChan:
			if h.OnError != nil {
				h.OnError(err)
			}
		}
	}
}

// handleMessage handles a single incoming message
func (h *TaskStreamHandler) handleMessage(msg *ClineMessageProto) {
	if msg == nil {
		return
	}

	switch msg.Type {
	case ClineMessageType_SAY:
		h.handleSayMessage(msg)
	case ClineMessageType_ASK:
		h.handleAskMessage(msg)
	}
}

// handleSayMessage handles say messages from the core extension
func (h *TaskStreamHandler) handleSayMessage(msg *ClineMessageProto) {
	switch msg.Say {
	case ClineSay_TEXT:
		if h.OnTextMessage != nil {
			h.OnTextMessage(msg.Text)
		}
	case ClineSay_TOOL_SAY:
		if msg.SayTool != nil && h.OnToolRequest != nil {
			if err := h.OnToolRequest(msg.SayTool); err != nil {
				if h.OnError != nil {
					h.OnError(err)
				}
			}
		}
	case ClineSay_COMMAND_SAY:
		if h.OnCommandRequest != nil {
			if err := h.OnCommandRequest(msg.Text); err != nil {
				if h.OnError != nil {
					h.OnError(err)
				}
			}
		}
	case ClineSay_COMPLETION_RESULT_SAY:
		if h.OnCompletion != nil {
			h.OnCompletion(msg.Text)
		}
	case ClineSay_ERROR:
		if h.OnError != nil {
			h.OnError(errors.New(msg.Text))
		}
	}
}

// handleAskMessage handles ask messages from the core extension
func (h *TaskStreamHandler) handleAskMessage(msg *ClineMessageProto) {
	switch msg.Ask {
	case ClineAsk_FOLLOWUP, ClineAsk_PLAN_MODE_RESPOND, ClineAsk_ACT_MODE_RESPOND:
		// These are questions that need responses
		if msg.AskQuestion != nil && h.OnAskQuestion != nil {
			response, err := h.OnAskQuestion(msg.AskQuestion)
			if err != nil {
				if h.OnError != nil {
					h.OnError(err)
				}
				return
			}
			// Send response back
			h.SendAskResponse("messageResponse", response, nil, nil)
		}
	case ClineAsk_COMMAND:
		// Command approval request
		h.handleApprovalRequest(msg, "command")
	case ClineAsk_TOOL:
		// Tool approval request
		h.handleApprovalRequest(msg, "tool")
	case ClineAsk_BROWSER_ACTION_LAUNCH:
		// Browser action approval request
		h.handleApprovalRequest(msg, "browser_action_launch")
	case ClineAsk_USE_MCP_SERVER:
		// MCP server approval request
		h.handleApprovalRequest(msg, "use_mcp_server")
	case ClineAsk_COMPLETION_RESULT:
		if h.OnCompletion != nil {
			h.OnCompletion(msg.Text)
		}
		// Also handle as approval request for task completion
		h.handleApprovalRequest(msg, "completion_result")
	case ClineAsk_RESUME_TASK:
		h.handleApprovalRequest(msg, "resume_task")
	case ClineAsk_RESUME_COMPLETED_TASK:
		h.handleApprovalRequest(msg, "resume_completed_task")
	case ClineAsk_NEW_TASK:
		h.handleApprovalRequest(msg, "new_task")
	}
}

// handleApprovalRequest handles approval requests with proper callback integration
func (h *TaskStreamHandler) handleApprovalRequest(msg *ClineMessageProto, askType string) {
	var response string
	var err error

	// Determine the appropriate callback based on ask type
	switch askType {
	case "command":
		if h.OnCommandRequest != nil {
			err = h.OnCommandRequest(msg.Text)
		}
	case "tool":
		if msg.SayTool != nil && h.OnToolRequest != nil {
			err = h.OnToolRequest(msg.SayTool)
		}
	default:
		// For other approval types, use OnAskQuestion if available
		if h.OnAskQuestion != nil {
			question := &ClineAskQuestion{
				Question: msg.Text,
			}
			response, err = h.OnAskQuestion(question)
		}
	}

	if err != nil {
		// User rejected or error occurred
		if h.OnError != nil {
			h.OnError(err)
		}
		h.SendAskResponse("noButtonClicked", err.Error(), nil, nil)
		return
	}

	// Use response from callback if provided, otherwise default to approval
	if response == "" {
		response = "yesButtonClicked"
	}
	h.SendAskResponse(response, "", nil, nil)
}

// WaitForCompletion blocks until the task completes or an error occurs
func (h *TaskStreamHandler) WaitForCompletion(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.doneChan:
			return nil
		case err := <-h.errorChan:
			return err
		case msg := <-h.messageChan:
			if msg != nil {
				// Check for completion
				if msg.Type == ClineMessageType_SAY && msg.Say == ClineSay_COMPLETION_RESULT_SAY {
					return nil
				}
				if msg.Type == ClineMessageType_ASK && msg.Ask == ClineAsk_COMPLETION_RESULT {
					return nil
				}
			}
		}
	}
}

// ============================================================================
// JSON Serialization Helpers
// ============================================================================

// MessageToJSON converts a ClineMessageProto to JSON
func MessageToJSON(msg *ClineMessageProto) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}
	return json.Marshal(msg)
}

// JSONToMessage converts JSON to a ClineMessageProto
func JSONToMessage(data []byte) (*ClineMessageProto, error) {
	var msg ClineMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &ClineMessageProto{ClineMessage: &msg}, nil
}

// ============================================================================
// StreamCreator Function
// ============================================================================

// CreateTaskStreamCreator creates a StreamCreator function for task communication
// This function returns a configured StreamCreator that uses the generated TaskService_StreamClient
func CreateTaskStreamCreator(taskID string) func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
	return func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
		// Create a new TaskService client
		client := NewTaskServiceClient(conn)

		// Establish the bidirectional stream
		stream, err := client.Stream(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to establish task stream: %w", err)
		}

		// Send initial task ID message to identify this stream
		initMsg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Ts:   time.Now().UnixMilli(),
				Type: ClineMessageType_SAY,
				Say:  ClineSay_TEXT,
				Text: taskID,
			},
		}

		if err := stream.Send(initMsg); err != nil {
			return nil, fmt.Errorf("failed to send task ID: %w", err)
		}

		return stream, nil
	}
}
