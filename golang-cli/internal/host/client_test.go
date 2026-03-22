package host

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// mockGRPCConn is a mock implementation of grpc.Conn for testing
type mockGRPCConn struct {
	mu         sync.RWMutex
	state      connectivity.State
	target     string
	closed     bool
	closeErr   error
}

func (m *mockGRPCConn) GetState() connectivity.State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *mockGRPCConn) setState(s connectivity.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

func (m *mockGRPCConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
}

func TestNewConnPool(t *testing.T) {
	tests := []struct {
		name    string
		config  PoolConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: PoolConfig{
				Target:              "localhost:50051",
				PoolSize:            3,
				ConnTimeout:         5 * time.Second,
				ReconnectDelay:      2 * time.Second,
				MaxRetries:          2,
				HealthCheckInterval: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty target",
			config: PoolConfig{
				Target: "",
			},
			wantErr: true,
		},
		{
			name: "defaults applied",
			config: PoolConfig{
				Target: "localhost:50051",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewConnPool(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewConnPool() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewConnPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if pool == nil {
				t.Error("NewConnPool() returned nil pool")
				return
			}
			if pool.GetState() != StateDisconnected {
				t.Errorf("expected initial state %v, got %v", StateDisconnected, pool.GetState())
			}
		})
	}
}

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{StateReconnecting, "reconnecting"},
		{StateClosed, "closed"},
		{ConnectionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("ConnectionState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnPoolStateTransitions(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target: "localhost:50051",
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Initial state
	if pool.GetState() != StateDisconnected {
		t.Errorf("initial state = %v, want %v", pool.GetState(), StateDisconnected)
	}

	// Test state callback
	stateChanges := []ConnectionState{}
	pool.onStateChange = func(s ConnectionState) {
		stateChanges = append(stateChanges, s)
	}

	// Set state manually
	pool.setState(StateConnecting)
	if pool.GetState() != StateConnecting {
		t.Errorf("state after setting connecting = %v, want %v", pool.GetState(), StateConnecting)
	}

	pool.setState(StateConnected)
	if pool.GetState() != StateConnected {
		t.Errorf("state after setting connected = %v, want %v", pool.GetState(), StateConnected)
	}

	if len(stateChanges) != 2 {
		t.Errorf("expected 2 state changes, got %d", len(stateChanges))
	}
}

func TestClientConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: ClientConfig{
				Target:   "localhost:50051",
				PoolSize: 3,
				TLSConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			wantErr: false,
		},
		{
			name: "empty target",
			config: ClientConfig{
				Target: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}
			if client.target != tt.config.Target {
				t.Errorf("client.target = %v, want %v", client.target, tt.config.Target)
			}
		})
	}
}

func TestRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("connection refused"),
			want: true,
		},
		{
			name: "connection reset",
			err:  fmt.Errorf("connection reset by peer"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  fmt.Errorf("broken pipe"),
			want: true,
		},
		{
			name: "deadline exceeded",
			err:  fmt.Errorf("context deadline exceeded"),
			want: true,
		},
		{
			name: "unavailable",
			err:  fmt.Errorf("service unavailable"),
			want: true,
		},
		{
			name: "resource exhausted",
			err:  fmt.Errorf("resource exhausted"),
			want: true,
		},
		{
			name: "random error",
			err:  fmt.Errorf("some random error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
		{"hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_in_%s", tt.substr, tt.s), func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestPoolStats(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target:   "localhost:50051",
		PoolSize: 5,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	stats := pool.Stats()

	if stats.TotalConnections != 0 {
		t.Errorf("expected 0 total connections before start, got %d", stats.TotalConnections)
	}
	if stats.ReadyConnections != 0 {
		t.Errorf("expected 0 ready connections before start, got %d", stats.ReadyConnections)
	}
	if stats.State != StateDisconnected {
		t.Errorf("expected state disconnected, got %v", stats.State)
	}
}

func TestWaitForReady(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target: "localhost:50051",
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Without starting, it should timeout
	err = pool.WaitForReady(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got %v", err)
	}
}

func TestConnPoolReconnect(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target: "localhost:50051",
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Stop the pool to set state to closed
	pool.Stop()

	// Reconnect on closed pool should fail
	err = pool.Reconnect()
	if err == nil {
		t.Error("expected error when reconnecting closed pool")
	} else if err.Error() != "cannot reconnect: pool is closed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConnPoolStop(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target:   "localhost:50051",
		PoolSize: 2,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Stop on non-started pool should not panic
	err = pool.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping pool: %v", err)
	}

	if pool.GetState() != StateClosed {
		t.Errorf("expected state closed after stop, got %v", pool.GetState())
	}
}

func TestDefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"defaultPoolSize", defaultPoolSize, 5},
		{"defaultConnTimeout", defaultConnTimeout, 10 * time.Second},
		{"defaultReconnectDelay", defaultReconnectDelay, 5 * time.Second},
		{"defaultMaxRetries", defaultMaxRetries, 3},
		{"defaultHealthCheckIntv", defaultHealthCheckIntv, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.got.(type) {
			case int:
				if v != tt.expected.(int) {
					t.Errorf("%s = %v, want %v", tt.name, v, tt.expected)
				}
			case time.Duration:
				if v != tt.expected.(time.Duration) {
					t.Errorf("%s = %v, want %v", tt.name, v, tt.expected)
				}
			}
		})
	}
}

func TestWithRetry(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Target:   "localhost:50051",
		PoolSize: 1,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test with canceled context
	cancel() // Cancel immediately
	err = client.WithRetry(ctx, func(conn *grpc.ClientConn) error {
		return nil
	})
	if err != context.Canceled {
		t.Errorf("expected context canceled, got %v", err)
	}
}

func TestClientForceReconnect(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Target: "localhost:50051",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Stop the client to set state to closed
	client.Stop()

	// ForceReconnect on closed client should fail (because pool is closed)
	err = client.ForceReconnect()
	if err == nil {
		t.Error("expected error when forcing reconnect on closed client")
	}
}

func TestDialOptions(t *testing.T) {
	customOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	pool, err := NewConnPool(PoolConfig{
		Target:      "localhost:50051",
		DialOptions: customOpts,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if len(pool.dialOptions) != len(customOpts) {
		t.Errorf("expected %d dial options, got %d", len(customOpts), len(pool.dialOptions))
	}
}

func TestConcurrentAccess(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target:   "localhost:50051",
		PoolSize: 5,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent state reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pool.GetState()
			_ = pool.Stats()
		}()
	}

	wg.Wait()
}

func TestOnErrorCallback(t *testing.T) {
	var receivedError error
	var mu sync.Mutex

	pool, err := NewConnPool(PoolConfig{
		Target: "localhost:50051",
		OnError: func(err error) {
			mu.Lock()
			defer mu.Unlock()
			receivedError = err
		},
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if pool.onError == nil {
		t.Error("expected onError callback to be set")
	}

	// Verify callback is stored
	mu.Lock()
	if receivedError != nil {
		t.Error("expected no error initially")
	}
	mu.Unlock()
}

func TestTLSConfiguration(t *testing.T) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "localhost",
	}

	pool, err := NewConnPool(PoolConfig{
		Target:    "localhost:50051",
		TLSConfig: tlsConfig,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if pool.tlsConfig != tlsConfig {
		t.Error("expected TLS config to be set")
	}
}

func TestClientGetPool(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Target: "localhost:50051",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	pool := client.GetPool()
	if pool == nil {
		t.Error("expected non-nil pool")
	}
}

func TestHealthCheckInterval(t *testing.T) {
	customInterval := 5 * time.Second

	pool, err := NewConnPool(PoolConfig{
		Target:              "localhost:50051",
		HealthCheckInterval: customInterval,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if pool.healthCheckInterval != customInterval {
		t.Errorf("expected health check interval %v, got %v", customInterval, pool.healthCheckInterval)
	}
}

func TestMaxRetriesConfiguration(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target:     "localhost:50051",
		MaxRetries: 10,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if pool.maxRetries != 10 {
		t.Errorf("expected max retries 10, got %d", pool.maxRetries)
	}
}

func TestReconnectDelayConfiguration(t *testing.T) {
	pool, err := NewConnPool(PoolConfig{
		Target:         "localhost:50051",
		ReconnectDelay: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	if pool.reconnectDelay != 2*time.Second {
		t.Errorf("expected reconnect delay %v, got %v", 2*time.Second, pool.reconnectDelay)
	}
}