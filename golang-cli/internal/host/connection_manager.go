// Package host provides gRPC client functionality for communicating with the Cline extension.
package host

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ConnectionManager manages the gRPC connection to the Cline core extension
type ConnectionManager struct {
	config     *EndpointConfig
	conn       *grpc.ClientConn
	protoClient *ProtoClient
	healthClient grpc_health_v1.HealthClient
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *EndpointConfig) *ConnectionManager {
	if config == nil {
		config = &EndpointConfig{
			Address:           fmt.Sprintf("localhost:%d", DefaultProtoBusPort),
			HostBridgeAddress: fmt.Sprintf("localhost:%d", DefaultHostBridgePort),
		}
	}
	return &ConnectionManager{
		config: config,
	}
}

// Connect establishes a connection to the gRPC server with retry logic
func (cm *ConnectionManager) Connect(ctx context.Context) error {
	if cm.conn != nil {
		return nil
	}

	// Set up dial options
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy": "round_robin",
			"healthCheckConfig": {
				"serviceName": ""
			}
		}`),
	}

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(connectCtx, cm.config.Address, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", cm.config.Address, err)
	}

	cm.conn = conn
	cm.healthClient = grpc_health_v1.NewHealthClient(conn)
	
	// Create a connection pool with the existing connection
	pool, _ := NewConnPool(PoolConfig{
		Target: cm.config.Address,
		PoolSize: 1,
	})
	pool.connections = []*grpc.ClientConn{conn}
	pool.state.Store(int32(StateConnected))
	
	cm.protoClient = NewProtoClient(&Client{
		pool:   pool,
		target: cm.config.Address,
	})

	return nil
}

// ConnectWithRetry attempts to connect with exponential backoff
func (cm *ConnectionManager) ConnectWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 500ms, 1s, 2s, 4s...
			backoff := time.Duration(500*(1<<(attempt-1))) * time.Millisecond
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := cm.Connect(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries+1, lastErr)
}

// IsConnected returns true if the connection is ready
func (cm *ConnectionManager) IsConnected() bool {
	if cm.conn == nil {
		return false
	}
	
	state := cm.conn.GetState()
	return state == connectivity.Ready
}

// HealthCheck performs a health check on the connection
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	if cm.healthClient == nil {
		return fmt.Errorf("not connected")
	}

	resp, err := cm.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service not serving: %v", resp.Status)
	}

	return nil
}

// GetProtoClient returns the ProtoClient for making RPC calls
func (cm *ConnectionManager) GetProtoClient() *ProtoClient {
	return cm.protoClient
}

// GetConnection returns the raw gRPC connection
func (cm *ConnectionManager) GetConnection() *grpc.ClientConn {
	return cm.conn
}

// Close closes the connection
func (cm *ConnectionManager) Close() error {
	if cm.conn != nil {
		err := cm.conn.Close()
		cm.conn = nil
		cm.protoClient = nil
		cm.healthClient = nil
		return err
	}
	return nil
}

// WaitForReady blocks until the connection is ready or context is cancelled
func (cm *ConnectionManager) WaitForReady(ctx context.Context) error {
	if cm.conn == nil {
		return fmt.Errorf("not connected")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			state := cm.conn.GetState()
			if state == connectivity.Ready {
				return nil
			}
			
			// Wait for state change
			if !cm.conn.WaitForStateChange(ctx, state) {
				return ctx.Err()
			}
		}
	}
}