// Package grpc provides gRPC client functionality for communicating with the Cline core extension.
// This package implements bidirectional streaming with reconnection support and context-based
// cancellation.
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var (
	// ErrConnectionFailed is returned when the gRPC connection cannot be established
	ErrConnectionFailed = errors.New("gRPC connection failed")
	// ErrConnectionClosed is returned when attempting to use a closed connection
	ErrConnectionClosed = errors.New("gRPC connection is closed")
	// ErrStreamCreationFailed is returned when a stream cannot be created
	ErrStreamCreationFailed = errors.New("failed to create gRPC stream")
	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("gRPC operation timed out")
)

// ClientConfig contains configuration for the gRPC client
type ClientConfig struct {
	// Endpoint is the gRPC server address (e.g., "localhost:50051")
	Endpoint string
	// UseTLS enables TLS encryption for the connection
	UseTLS bool
	// TLSCertPath is the path to the TLS certificate (optional)
	TLSCertPath string
	// Timeout is the default timeout for operations
	Timeout time.Duration
	// KeepaliveInterval is the interval between keepalive pings
	KeepaliveInterval time.Duration
	// MaxRetries is the maximum number of connection retry attempts
	MaxRetries int
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Endpoint:          "localhost:50051",
		UseTLS:            false,
		Timeout:           30 * time.Second,
		KeepaliveInterval: 10 * time.Second,
		MaxRetries:        5,
		RetryDelay:        1 * time.Second,
	}
}

// Client manages a gRPC connection to the Cline core extension
type Client struct {
	config     *ClientConfig
	conn       *grpc.ClientConn
	ctx        context.Context
	cancel     context.CancelFunc
	retryCount int
	mu         sync.RWMutex
	closed     bool
	dialOpts   []grpc.DialOption
}

// NewClient creates a new gRPC client with the given configuration
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	if config.Timeout <= 0 {
		config.Timeout = DefaultClientConfig().Timeout
	}

	if config.MaxRetries <= 0 {
		config.MaxRetries = DefaultClientConfig().MaxRetries
	}

	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultClientConfig().RetryDelay
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Setup dial options
	if err := client.setupDialOptions(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup dial options: %w", err)
	}

	return client, nil
}

// setupDialOptions configures the gRPC dial options
func (c *Client) setupDialOptions() error {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64*1024*1024), // 64MB
			grpc.MaxCallSendMsgSize(64*1024*1024), // 64MB
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.config.KeepaliveInterval,
			Timeout:             c.config.Timeout,
			PermitWithoutStream: true,
		}),
		grpc.WithBlock(),
		grpc.WithTimeout(c.config.Timeout),
	}

	// Configure credentials
	if c.config.UseTLS {
		if c.config.TLSCertPath != "" {
			creds, err := credentials.NewClientTLSFromFile(c.config.TLSCertPath, "")
			if err != nil {
				return fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	c.dialOpts = opts
	return nil
}

// Connect establishes the gRPC connection with retry logic
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConnectionClosed
	}

	if c.conn != nil {
		state := c.conn.GetState()
		if state == connectivity.Ready || state == connectivity.Connecting {
			return nil // Already connected or connecting
		}
	}

	var lastErr error
	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryDelay * time.Duration(attempt))
		}

		ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
		conn, err := grpc.DialContext(ctx, c.config.Endpoint, c.dialOpts...)
		cancel()

		if err != nil {
			lastErr = err
			continue
		}

		c.conn = conn
		c.retryCount = attempt
		return nil
	}

	return fmt.Errorf("%w: after %d attempts, last error: %v", ErrConnectionFailed, c.config.MaxRetries, lastErr)
}

// Close gracefully closes the gRPC connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	return nil
}

// GetConnection returns the underlying gRPC connection
func (c *Client) GetConnection() (*grpc.ClientConn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrConnectionClosed
	}

	if c.conn == nil {
		return nil, ErrConnectionFailed
	}

	state := c.conn.GetState()
	if state != connectivity.Ready {
		return nil, fmt.Errorf("connection not ready: %v", state)
	}

	return c.conn, nil
}

// WaitForReady blocks until the connection is ready or the context is cancelled
func (c *Client) WaitForReady(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		if err := c.Connect(); err != nil {
			return err
		}
		c.mu.RLock()
		conn = c.conn
		c.mu.RUnlock()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			state := conn.GetState()
			if state == connectivity.Ready {
				return nil
			}
			if !conn.WaitForStateChange(ctx, state) {
				return ctx.Err()
			}
		}
	}
}

// IsConnected returns true if the client has an active connection
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.conn == nil {
		return false
	}

	return c.conn.GetState() == connectivity.Ready
}

// GetState returns the current connection state
func (c *Client) GetState() connectivity.State {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return connectivity.Shutdown
	}

	if c.conn == nil {
		return connectivity.Idle
	}

	return c.conn.GetState()
}

// RetryCount returns the number of retry attempts made for the current connection
func (c *Client) RetryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.retryCount
}

// Reconnect closes the current connection and establishes a new one
func (c *Client) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConnectionClosed
	}

	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Reset retry count for new connection
	c.retryCount = 0

	c.mu.Unlock()
	err := c.Connect()
	c.mu.Lock()

	return err
}

// Context returns the client's context
func (c *Client) Context() context.Context {
	return c.ctx
}
