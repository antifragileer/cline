// Package grpc provides bidirectional streaming support with reconnection and
// graceful error handling for the Cline CLI.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

// StreamState represents the current state of a bidirectional stream
type StreamState int32

const (
	// StreamStateDisconnected indicates the stream is not connected
	StreamStateDisconnected StreamState = iota
	// StreamStateConnecting indicates the stream is establishing connection
	StreamStateConnecting
	// StreamStateReady indicates the stream is ready for communication
	StreamStateReady
	// StreamStateReconnecting indicates the stream is reconnecting
	StreamStateReconnecting
	// StreamStateClosed indicates the stream has been closed
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
	StreamCreator func(ctx context.Context, conn *grpc.ClientConn) (grpc.BidiStreamingClient[Message, Message], error)
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
	OnMessageReceived func(*Message)
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

// Message represents a generic message that can be sent/received on the stream
type Message struct {
	// ID is the unique message identifier
	ID string `json:"id"`
	// Type is the message type
	Type string `json:"type"`
	// Payload is the message payload
	Payload []byte `json:"payload"`
	// Partial indicates if this is a partial message chunk
	Partial bool `json:"partial"`
	// Sequence is the sequence number for partial messages
	Sequence int `json:"sequence"`
	// TotalChunks is the total number of chunks for partial messages
	TotalChunks int `json:"total_chunks"`
	// Timestamp is the message timestamp
	Timestamp int64 `json:"timestamp"`
}

// BidiStream manages a bidirectional gRPC stream with reconnection support
type BidiStream struct {
	config *StreamConfig

	// State management
	state atomic.Int32

	// Channels for message flow
	sendChan    chan *Message
	recvChan    chan *Message
	ackChan     chan string
	controlChan chan controlMessage

	// Flow control
	outstandingMsgs atomic.Int32
	backpressure    chan struct{}

	// Partial message handling
	partialMu       sync.RWMutex
	partialBuffers  map[string][]*Message
	partialTimeouts map[string]time.Time

	// Stream management
	stream     grpc.BidiStreamingClient[Message, Message]
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
		sendChan:        make(chan *Message, config.BufferSize),
		recvChan:        make(chan *Message, config.BufferSize),
		ackChan:         make(chan string, config.BufferSize),
		controlChan:     make(chan controlMessage, 10),
		backpressure:    make(chan struct{}, config.MaxOutstandingMessages),
		partialBuffers:  make(map[string][]*Message),
		partialTimeouts: make(map[string]time.Time),
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
func (s *BidiStream) Send(msg *Message) error {
	if msg == nil {
		return errors.New("message is nil")
	}

	if s.getState() != StreamStateReady {
		return errors.New("stream not ready")
	}

	// Check backpressure
	select {
	case s.backpressure <- struct{}{}:
		// Got token, proceed
	default:
		return errors.New("backpressure limit exceeded")
	}

	// Handle partial messages
	if s.config.EnablePartialMessages && len(msg.Payload) > s.config.MaxChunkSize {
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
func (s *BidiStream) sendPartialMessage(msg *Message) error {
	chunks := s.chunkMessage(msg)

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
func (s *BidiStream) chunkMessage(msg *Message) []*Message {
	payload := msg.Payload
	if len(payload) <= s.config.MaxChunkSize {
		return []*Message{msg}
	}

	numChunks := (len(payload) + s.config.MaxChunkSize - 1) / s.config.MaxChunkSize
	chunks := make([]*Message, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * s.config.MaxChunkSize
		end := start + s.config.MaxChunkSize
		if end > len(payload) {
			end = len(payload)
		}

		chunk := &Message{
			ID:          msg.ID,
			Type:        msg.Type,
			Payload:     payload[start:end],
			Partial:     true,
			Sequence:    i,
			TotalChunks: numChunks,
			Timestamp:   msg.Timestamp,
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
func (s *BidiStream) Receive() <-chan *Message {
	return s.recvChan
}

// Acknowledge acknowledges receipt of a message
func (s *BidiStream) Acknowledge(msgID string) {
	select {
	case s.ackChan <- msgID:
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

	// Note: StreamCreator is responsible for establishing the connection
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
				s.releaseBackpressure(msg.ID)
			}
		}
	}
}

// sendMessage sends a single message to the stream
func (s *BidiStream) sendMessage(msg *Message) error {
	s.streamMu.RLock()
	stream := s.stream
	s.streamMu.RUnlock()

	if stream == nil {
		return errors.New("stream not ready")
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
		s.releaseBackpressure(msg.ID)
	}
}

// receiveMessage receives a single message from the stream
func (s *BidiStream) receiveMessage() (*Message, error) {
	s.streamMu.RLock()
	stream := s.stream
	s.streamMu.RUnlock()

	if stream == nil {
		return nil, errors.New("stream not ready")
	}

	return stream.Recv()
}

// handlePartialMessage processes partial message chunks
func (s *BidiStream) handlePartialMessage(chunk *Message) *Message {
	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	msgID := chunk.ID

	// Initialize buffer for this message
	if _, exists := s.partialBuffers[msgID]; !exists {
		s.partialBuffers[msgID] = make([]*Message, 0)
		s.partialTimeouts[msgID] = time.Now().Add(30 * time.Second)
	}

	// Add chunk to buffer
	s.partialBuffers[msgID] = append(s.partialBuffers[msgID], chunk)

	// Check if we have all chunks
	if !chunk.Partial && len(s.partialBuffers[msgID]) == chunk.TotalChunks {
		// Reassemble message
		msg := s.reassembleMessage(msgID)
		delete(s.partialBuffers, msgID)
		delete(s.partialTimeouts, msgID)
		return msg
	}

	return nil
}

// reassembleMessage combines chunks into a complete message
func (s *BidiStream) reassembleMessage(msgID string) *Message {
	chunks := s.partialBuffers[msgID]
	if len(chunks) == 0 {
		return nil
	}

	// Calculate total payload size
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk.Payload)
	}

	// Reassemble payload
	payload := make([]byte, 0, totalSize)
	for _, chunk := range chunks {
		payload = append(payload, chunk.Payload...)
	}

	// Create complete message from first chunk
	first := chunks[0]
	return &Message{
		ID:        first.ID,
		Type:      first.Type,
		Payload:   payload,
		Partial:   false,
		Timestamp: first.Timestamp,
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
	for msgID, timeout := range s.partialTimeouts {
		if now.After(timeout) {
			delete(s.partialBuffers, msgID)
			delete(s.partialTimeouts, msgID)
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
	s.handleError(errors.New("reconnection failed after maximum retries"))
	s.setState(StreamStateDisconnected)
}

// handleError handles errors and invokes the error callback
func (s *BidiStream) handleError(err error) {
	if s.config.OnError != nil {
		s.config.OnError(err)
	}
}

// releaseBackpressure releases a backpressure token
func (s *BidiStream) releaseBackpressure(msgID string) {
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
		return errors.New("stream is closed")
	}

	s.triggerReconnection()
	return nil
}