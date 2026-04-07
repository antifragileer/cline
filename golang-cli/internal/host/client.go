package host

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultPoolSize        = 5
	defaultConnTimeout     = 10 * time.Second
	defaultReconnectDelay  = 5 * time.Second
	defaultMaxRetries      = 3
	defaultHealthCheckIntv = 30 * time.Second
)

// ConnectionState represents the state of a gRPC connection
type ConnectionState int32

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateClosed
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ConnPool manages a pool of gRPC connections
type ConnPool struct {
	mu          sync.RWMutex
	target      string
	connections []*grpc.ClientConn
	dialOptions []grpc.DialOption
	tlsConfig   *tls.Config
	poolSize    int

	// Connection state
	state      atomic.Int32
	retryCount atomic.Int32

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Health check
	healthCheckInterval time.Duration
	lastHealthCheck     atomic.Value

	// Reconnection
	reconnectDelay time.Duration
	maxRetries     int

	// Callbacks
	onStateChange func(ConnectionState)
	onError       func(error)
}

// PoolConfig contains configuration for the connection pool
type PoolConfig struct {
	Target              string
	PoolSize            int
	ConnTimeout         time.Duration
	ReconnectDelay      time.Duration
	MaxRetries          int
	HealthCheckInterval time.Duration
	TLSConfig           *tls.Config
	DialOptions         []grpc.DialOption
	OnStateChange       func(ConnectionState)
	OnError             func(error)
}

// NewConnPool creates a new connection pool with the given configuration
func NewConnPool(config PoolConfig) (*ConnPool, error) {
	if config.Target == "" {
		return nil, fmt.Errorf("target address is required")
	}

	poolSize := config.PoolSize
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	connTimeout := config.ConnTimeout
	if connTimeout <= 0 {
		connTimeout = defaultConnTimeout
	}

	reconnectDelay := config.ReconnectDelay
	if reconnectDelay <= 0 {
		reconnectDelay = defaultReconnectDelay
	}

	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	healthCheckIntv := config.HealthCheckInterval
	if healthCheckIntv <= 0 {
		healthCheckIntv = defaultHealthCheckIntv
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnPool{
		target:              config.Target,
		connections:         make([]*grpc.ClientConn, 0, poolSize),
		tlsConfig:           config.TLSConfig,
		poolSize:            poolSize,
		dialOptions:         config.DialOptions,
		ctx:                 ctx,
		cancel:              cancel,
		healthCheckInterval: healthCheckIntv,
		reconnectDelay:      reconnectDelay,
		maxRetries:          maxRetries,
		onStateChange:       config.OnStateChange,
		onError:             config.OnError,
	}

	pool.state.Store(int32(StateDisconnected))
	return pool, nil
}

// Start initializes the connection pool
func (p *ConnPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.getState() == StateConnected || p.getState() == StateConnecting {
		return nil
	}

	p.setState(StateConnecting)

	// Create initial connections
	for i := 0; i < p.poolSize; i++ {
		conn, err := p.createConnection()
		if err != nil {
			p.setState(StateDisconnected)
			return fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		p.connections = append(p.connections, conn)
	}

	p.setState(StateConnected)
	p.retryCount.Store(0)

	// Start health check goroutine
	p.wg.Add(1)
	go p.healthCheckLoop()

	return nil
}

// Stop closes all connections in the pool
func (p *ConnPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.setState(StateClosed)
	p.cancel()

	// Close all connections
	var lastErr error
	for _, conn := range p.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				lastErr = err
			}
		}
	}
	p.connections = p.connections[:0]

	p.wg.Wait()
	return lastErr
}

// GetConnection returns a connection from the pool
func (p *ConnPool) GetConnection() (*grpc.ClientConn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.getState() != StateConnected {
		return nil, fmt.Errorf("connection pool not connected (state: %s)", p.getState())
	}

	// Simple round-robin selection
	if len(p.connections) == 0 {
		return nil, fmt.Errorf("no available connections")
	}

	// Find first ready connection
	for _, conn := range p.connections {
		if conn != nil && p.isConnectionReady(conn) {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("no ready connections available")
}

// createConnection creates a new gRPC connection
func (p *ConnPool) createConnection() (*grpc.ClientConn, error) {
	opts := make([]grpc.DialOption, 0, len(p.dialOptions)+3)

	// Add transport credentials
	if p.tlsConfig != nil {
		creds := credentials.NewTLS(p.tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add dial timeout
	ctx, cancel := context.WithTimeout(p.ctx, p.connTimeout())
	defer cancel()

	// Add block option to wait for connection to be ready
	opts = append(opts, grpc.WithBlock())

	// Append user-provided options
	opts = append(opts, p.dialOptions...)

	conn, err := grpc.DialContext(ctx, p.target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", p.target, err)
	}

	return conn, nil
}

// healthCheckLoop periodically checks connection health
func (p *ConnPool) healthCheckLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

// performHealthCheck checks all connections and triggers reconnection if needed
func (p *ConnPool) performHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.getState() == StateClosed {
		return
	}

	p.lastHealthCheck.Store(time.Now())

	needsReconnect := false
	for i, conn := range p.connections {
		if conn == nil || !p.isConnectionReady(conn) {
			needsReconnect = true
			// Attempt to recreate this connection
			newConn, err := p.createConnection()
			if err != nil {
				if p.onError != nil {
					p.onError(fmt.Errorf("health check: failed to recreate connection %d: %w", i, err))
				}
				continue
			}
			if conn != nil {
				conn.Close()
			}
			p.connections[i] = newConn
		}
	}

	if needsReconnect {
		p.setState(StateReconnecting)
		p.setState(StateConnected)
	}
}

// isConnectionReady checks if a connection is in Ready state
func (p *ConnPool) isConnectionReady(conn *grpc.ClientConn) bool {
	return conn.GetState() == connectivity.Ready
}

// Reconnect forces a reconnection of all connections in the pool
func (p *ConnPool) Reconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.getState() == StateClosed {
		return fmt.Errorf("cannot reconnect: pool is closed")
	}

	if p.getState() != StateConnected {
		return fmt.Errorf("cannot reconnect: pool is not connected (state: %s)", p.getState())
	}

	p.setState(StateReconnecting)

	// Close and recreate all connections
	for i, conn := range p.connections {
		if conn != nil {
			conn.Close()
		}

		newConn, err := p.createConnection()
		if err != nil {
			p.setState(StateDisconnected)
			return fmt.Errorf("failed to recreate connection %d: %w", i, err)
		}
		p.connections[i] = newConn
	}

	p.setState(StateConnected)
	p.retryCount.Store(0)
	return nil
}

// GetState returns the current state of the connection pool
func (p *ConnPool) GetState() ConnectionState {
	return p.getState()
}

func (p *ConnPool) getState() ConnectionState {
	return ConnectionState(p.state.Load())
}

func (p *ConnPool) setState(state ConnectionState) {
	oldState := p.getState()
	if oldState == state {
		return
	}
	p.state.Store(int32(state))
	if p.onStateChange != nil {
		p.onStateChange(state)
	}
}

func (p *ConnPool) connTimeout() time.Duration {
	return defaultConnTimeout
}

// WaitForReady blocks until the pool is connected or context is cancelled
func (p *ConnPool) WaitForReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if p.GetState() == StateConnected {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Stats returns pool statistics
func (p *ConnPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	readyCount := 0
	for _, conn := range p.connections {
		if conn != nil && p.isConnectionReady(conn) {
			readyCount++
		}
	}

	lastHealthCheck, _ := p.lastHealthCheck.Load().(time.Time)

	return PoolStats{
		TotalConnections: len(p.connections),
		ReadyConnections: readyCount,
		State:            p.getState(),
		RetryCount:       int(p.retryCount.Load()),
		LastHealthCheck:  lastHealthCheck,
	}
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalConnections int
	ReadyConnections int
	State            ConnectionState
	RetryCount       int
	LastHealthCheck  time.Time
}

// Client is a high-level gRPC client with connection pooling
type Client struct {
	pool   *ConnPool
	target string
}

// ClientConfig contains configuration for the gRPC client
type ClientConfig struct {
	Target              string
	PoolSize            int
	ConnTimeout         time.Duration
	ReconnectDelay      time.Duration
	MaxRetries          int
	HealthCheckInterval time.Duration
	TLSConfig           *tls.Config
	DialOptions         []grpc.DialOption
	OnStateChange       func(ConnectionState)
	OnError             func(error)
}

// NewClient creates a new gRPC client with connection pooling
func NewClient(config ClientConfig) (*Client, error) {
	poolConfig := PoolConfig{
		Target:              config.Target,
		PoolSize:            config.PoolSize,
		ConnTimeout:         config.ConnTimeout,
		ReconnectDelay:      config.ReconnectDelay,
		MaxRetries:          config.MaxRetries,
		HealthCheckInterval: config.HealthCheckInterval,
		TLSConfig:           config.TLSConfig,
		DialOptions:         config.DialOptions,
		OnStateChange:       config.OnStateChange,
		OnError:             config.OnError,
	}

	pool, err := NewConnPool(poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &Client{
		pool:   pool,
		target: config.Target,
	}, nil
}

// Start initializes the client and connection pool
func (c *Client) Start() error {
	if err := c.pool.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}
	return nil
}

// Stop closes the client and all connections
func (c *Client) Stop() error {
	return c.pool.Stop()
}

// GetPool returns the underlying connection pool
func (c *Client) GetPool() *ConnPool {
	return c.pool
}

// WithRetry executes the given function with retry logic
func (c *Client) WithRetry(ctx context.Context, fn func(*grpc.ClientConn) error) error {
	var lastErr error

	for attempt := 0; attempt <= c.pool.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.pool.reconnectDelay):
			}
		}

		conn, err := c.pool.GetConnection()
		if err != nil {
			lastErr = err
			c.pool.retryCount.Add(1)
			continue
		}

		if err := fn(conn); err != nil {
			lastErr = err
			// Check if this is a connection error that warrants a retry
			if isRetryableError(err) {
				c.pool.retryCount.Add(1)
				continue
			}
			return err
		}

		// Success
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific gRPC error codes that warrant retry
	// This is a simplified check - in production, you'd use status.Code()
	errStr := err.Error()
	retryableStrs := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"deadline exceeded",
		"unavailable",
		"resource exhausted",
	}

	for _, s := range retryableStrs {
		if contains(errStr, s) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// WaitForReady blocks until the client is connected
func (c *Client) WaitForReady(ctx context.Context) error {
	return c.pool.WaitForReady(ctx)
}

// Stats returns client statistics
func (c *Client) Stats() PoolStats {
	return c.pool.Stats()
}

// ForceReconnect forces a reconnection of all connections
func (c *Client) ForceReconnect() error {
	return c.pool.Reconnect()
}