// Package grpc provides reconnection logic and connection pooling for resilient
// communication with the Cline core extension.
package grpc

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ReconnectPolicy defines the reconnection strategy
type ReconnectPolicy struct {
	// MaxAttempts is the maximum number of reconnection attempts (0 = unlimited)
	MaxAttempts int
	// InitialDelay is the initial delay between reconnection attempts
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between reconnection attempts
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// ResetInterval is the interval after which the retry count resets
	ResetInterval time.Duration
}

// DefaultReconnectPolicy returns a default reconnection policy
func DefaultReconnectPolicy() *ReconnectPolicy {
	return &ReconnectPolicy{
		MaxAttempts:       0, // Unlimited
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		ResetInterval:     5 * time.Minute,
	}
}

// ConnectionPool manages a pool of gRPC connections with automatic reconnection
type ConnectionPool struct {
	config         *ClientConfig
	policy         *ReconnectPolicy
	connections    map[string]*PooledConnection
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	wg             sync.WaitGroup
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	ID         string
	Client     *Client
	LastUsed   time.Time
	Reconnects int
	mu         sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *ClientConfig, policy *ReconnectPolicy) *ConnectionPool {
	if policy == nil {
		policy = DefaultReconnectPolicy()
	}

	pool := &ConnectionPool{
		config:        config,
		policy:        policy,
		connections:   make(map[string]*PooledConnection),
		cleanupTicker: time.NewTicker(1 * time.Minute),
		stopCleanup:   make(chan struct{}),
	}

	// Start cleanup goroutine
	pool.wg.Add(1)
	go pool.cleanupLoop()

	return pool
}

// GetConnection gets or creates a connection for the given ID
func (p *ConnectionPool) GetConnection(id string) (*PooledConnection, error) {
	p.mu.RLock()
	conn, exists := p.connections[id]
	p.mu.RUnlock()

	if exists {
		conn.mu.Lock()
		conn.LastUsed = time.Now()
		conn.mu.Unlock()

		// Check if connection is healthy
		if conn.Client.IsConnected() {
			return conn, nil
		}

		// Try to reconnect
		if err := conn.Client.Reconnect(); err != nil {
			// Remove unhealthy connection
			p.RemoveConnection(id)
			return nil, fmt.Errorf("failed to reconnect: %w", err)
		}

		return conn, nil
	}

	// Create new connection
	client, err := NewClient(p.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	conn = &PooledConnection{
		ID:       id,
		Client:   client,
		LastUsed: time.Now(),
	}

	p.mu.Lock()
	p.connections[id] = conn
	p.mu.Unlock()

	return conn, nil
}

// RemoveConnection removes a connection from the pool
func (p *ConnectionPool) RemoveConnection(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[id]; exists {
		conn.Client.Close()
		delete(p.connections, id)
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	close(p.stopCleanup)
	p.cleanupTicker.Stop()
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for id, conn := range p.connections {
		if err := conn.Client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection %s: %w", id, err))
		}
	}

	p.connections = make(map[string]*PooledConnection)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// cleanupLoop periodically cleans up idle connections
func (p *ConnectionPool) cleanupLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCleanup:
			return
		case <-p.cleanupTicker.C:
			p.cleanupIdleConnections()
		}
	}
}

// cleanupIdleConnections removes connections that have been idle for too long
func (p *ConnectionPool) cleanupIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	idleTimeout := 5 * time.Minute
	now := time.Now()

	for id, conn := range p.connections {
		conn.mu.RLock()
		lastUsed := conn.LastUsed
		conn.mu.RUnlock()

		if now.Sub(lastUsed) > idleTimeout {
			conn.Client.Close()
			delete(p.connections, id)
		}
	}
}

// ConnectionCount returns the number of connections in the pool
func (p *ConnectionPool) ConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// ReconnectManager handles automatic reconnection for a single connection
type ReconnectManager struct {
	client         *Client
	policy         *ReconnectPolicy
	retryCount     int
	lastRetry      time.Time
	mu             sync.Mutex
	reconnecting   bool
	onReconnect    func()
	onDisconnect   func(error)
	stopMonitoring chan struct{}
	wg             sync.WaitGroup
}

// NewReconnectManager creates a new reconnection manager
func NewReconnectManager(client *Client, policy *ReconnectPolicy) *ReconnectManager {
	if policy == nil {
		policy = DefaultReconnectPolicy()
	}

	return &ReconnectManager{
		client:         client,
		policy:         policy,
		stopMonitoring: make(chan struct{}),
	}
}

// SetCallbacks sets the reconnection callbacks
func (r *ReconnectManager) SetCallbacks(onReconnect func(), onDisconnect func(error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onReconnect = onReconnect
	r.onDisconnect = onDisconnect
}

// StartMonitoring starts monitoring the connection state
func (r *ReconnectManager) StartMonitoring() {
	r.wg.Add(1)
	go r.monitorLoop()
}

// StopMonitoring stops monitoring the connection
func (r *ReconnectManager) StopMonitoring() {
	close(r.stopMonitoring)
	r.wg.Wait()
}

// monitorLoop monitors the connection and triggers reconnection when needed
func (r *ReconnectManager) monitorLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopMonitoring:
			return
		case <-ticker.C:
			r.checkAndReconnect()
		}
	}
}

// checkAndReconnect checks the connection state and reconnects if necessary
func (r *ReconnectManager) checkAndReconnect() {
	state := r.client.GetState()

	if state == connectivity.Ready {
		// Reset retry count on successful connection
		r.mu.Lock()
		if r.retryCount > 0 && time.Since(r.lastRetry) > r.policy.ResetInterval {
			r.retryCount = 0
		}
		r.mu.Unlock()
		return
	}

	// Connection is not ready, try to reconnect
	if err := r.attemptReconnect(); err != nil {
		r.mu.Lock()
		onDisconnect := r.onDisconnect
		r.mu.Unlock()

		if onDisconnect != nil {
			onDisconnect(err)
		}
	}
}

// attemptReconnect attempts to reconnect with backoff
func (r *ReconnectManager) attemptReconnect() error {
	r.mu.Lock()
	if r.reconnecting {
		r.mu.Unlock()
		return errors.New("reconnection already in progress")
	}
	r.reconnecting = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.reconnecting = false
		r.mu.Unlock()
	}()

	// Calculate backoff delay
	delay := r.calculateBackoff()

	// Check max attempts
	if r.policy.MaxAttempts > 0 && r.retryCount >= r.policy.MaxAttempts {
		return fmt.Errorf("max reconnection attempts (%d) exceeded", r.policy.MaxAttempts)
	}

	// Wait before reconnecting
	select {
	case <-r.stopMonitoring:
		return errors.New("monitoring stopped")
	case <-time.After(delay):
	}

	// Attempt reconnection
	if err := r.client.Reconnect(); err != nil {
		r.mu.Lock()
		r.retryCount++
		r.lastRetry = time.Now()
		r.mu.Unlock()
		return fmt.Errorf("reconnection failed: %w", err)
	}

	// Reconnection successful
	r.mu.Lock()
	r.retryCount = 0
	onReconnect := r.onReconnect
	r.mu.Unlock()

	if onReconnect != nil {
		onReconnect()
	}

	return nil
}

// calculateBackoff calculates the backoff delay
func (r *ReconnectManager) calculateBackoff() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	delay := r.policy.InitialDelay
	for i := 0; i < r.retryCount; i++ {
		delay = time.Duration(float64(delay) * r.policy.BackoffMultiplier)
		if delay > r.policy.MaxDelay {
			delay = r.policy.MaxDelay
			break
		}
	}

	return delay
}

// GetRetryCount returns the current retry count
func (r *ReconnectManager) GetRetryCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.retryCount
}

// IsReconnecting returns true if reconnection is in progress
func (r *ReconnectManager) IsReconnecting() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reconnecting
}

// ResilientClient wraps a gRPC client with automatic reconnection
type ResilientClient struct {
	client   *Client
	manager  *ReconnectManager
	streamMu sync.RWMutex
	streams  map[string]*BidiStream
}

// NewResilientClient creates a new resilient client
func NewResilientClient(config *ClientConfig, policy *ReconnectPolicy) (*ResilientClient, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}

	manager := NewReconnectManager(client, policy)
	rc := &ResilientClient{
		client:  client,
		manager: manager,
		streams: make(map[string]*BidiStream),
	}

	// Set up callbacks
	manager.SetCallbacks(
		func() { rc.onReconnect() },
		func(err error) { rc.onDisconnect(err) },
	)

	return rc, nil
}

// Connect establishes the initial connection
func (rc *ResilientClient) Connect() error {
	if err := rc.client.Connect(); err != nil {
		return err
	}

	rc.manager.StartMonitoring()
	return nil
}

// Close closes the client and all streams
func (rc *ResilientClient) Close() error {
	rc.manager.StopMonitoring()

	rc.streamMu.Lock()
	for _, stream := range rc.streams {
		stream.Stop()
	}
	rc.streams = make(map[string]*BidiStream)
	rc.streamMu.Unlock()

	return rc.client.Close()
}

// CreateStream creates a new bidirectional stream
func (rc *ResilientClient) CreateStream(id string, config *StreamConfig) (*BidiStream, error) {
	stream, err := NewBidiStream(config)
	if err != nil {
		return nil, err
	}

	rc.streamMu.Lock()
	rc.streams[id] = stream
	rc.streamMu.Unlock()

	return stream, nil
}

// RemoveStream removes a stream from tracking
func (rc *ResilientClient) RemoveStream(id string) {
	rc.streamMu.Lock()
	if stream, exists := rc.streams[id]; exists {
		stream.Stop()
		delete(rc.streams, id)
	}
	rc.streamMu.Unlock()
}

// onReconnect is called when the connection is reestablished
func (rc *ResilientClient) onReconnect() {
	// Restart all streams
	rc.streamMu.RLock()
	streams := make(map[string]*BidiStream, len(rc.streams))
	for id, stream := range rc.streams {
		streams[id] = stream
	}
	rc.streamMu.RUnlock()

	for _, stream := range streams {
		stream.ForceReconnect()
	}
}

// onDisconnect is called when the connection is lost
func (rc *ResilientClient) onDisconnect(err error) {
	// Streams will handle their own reconnection via the stream's reconnect logic
	_ = err // Log this in production
}

// GetConnection returns the underlying connection
func (rc *ResilientClient) GetConnection() (*grpc.ClientConn, error) {
	return rc.client.GetConnection()
}

// IsConnected returns true if the client is connected
func (rc *ResilientClient) IsConnected() bool {
	return rc.client.IsConnected()
}