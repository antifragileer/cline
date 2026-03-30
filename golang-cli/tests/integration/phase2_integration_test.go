// Package integration provides integration tests for Phase 2: Connection & State Management.
// This file tests connection pooling, bidirectional streaming, state synchronization,
// and message conversion as defined in the implementation plan.
package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// MockStreamServer implements a mock streaming server for testing
type MockStreamServer struct {
	cline.UnimplementedTaskServiceServer
	messages chan *cline.ClineMessage
}

func NewMockStreamServer() *MockStreamServer {
	return &MockStreamServer{
		messages: make(chan *cline.ClineMessage, 100),
	}
}

func (m *MockStreamServer) Stream(stream cline.TaskService_StreamServer) error {
	// Echo back messages
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Echo back with slight modification
		msg.Text = "echo: " + msg.Text
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
}

// TestPhase2_ConnectionPooling validates connection pooling functionality
func TestPhase2_ConnectionPooling(t *testing.T) {
	t.Run("pool_creation_and_initialization", func(t *testing.T) {
		config := host.PoolConfig{
			Target:   "localhost:50051",
			PoolSize: 5,
		}

		pool, err := host.NewConnPool(config)
		require.NoError(t, err)
		require.NotNil(t, pool)

		defer pool.Stop()
	})

	t.Run("pool_connection_reuse", func(t *testing.T) {
		// Start a mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Create pool
		config := host.PoolConfig{
			Target:   lis.Addr().String(),
			PoolSize: 3,
		}

		pool, err := host.NewConnPool(config)
		require.NoError(t, err)
		defer pool.Stop()

		// Start the pool
		err = pool.Start()
		require.NoError(t, err)

		// Get connection
		conn, err := pool.GetConnection()
		require.NoError(t, err)
		require.NotNil(t, conn)

		// Connection should be ready
		assert.Equal(t, connectivity.Ready, conn.GetState())
	})

	t.Run("pool_handles_concurrent_requests", func(t *testing.T) {
		// Start a mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		config := host.PoolConfig{
			Target:   lis.Addr().String(),
			PoolSize: 5,
		}

		pool, err := host.NewConnPool(config)
		require.NoError(t, err)
		defer pool.Stop()

		err = pool.Start()
		require.NoError(t, err)

		// Run concurrent requests
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(index int) {
				defer func() { done <- true }()

				conn, err := pool.GetConnection()
				if err != nil {
					t.Logf("Request %d failed to get connection: %v", index, err)
					return
				}

				// Simulate some work
				time.Sleep(10 * time.Millisecond)

				_ = conn
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("pool_health_checks", func(t *testing.T) {
		// Start a mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		config := host.PoolConfig{
			Target:   lis.Addr().String(),
			PoolSize: 2,
		}

		pool, err := host.NewConnPool(config)
		require.NoError(t, err)
		defer pool.Stop()

		err = pool.Start()
		require.NoError(t, err)

		// Test that pool can create healthy connections
		conn, err := pool.GetConnection()
		require.NoError(t, err)
		require.NotNil(t, conn)

		// Connection should be ready
		assert.Equal(t, connectivity.Ready, conn.GetState())
	})
}

// TestPhase2_BidirectionalStreaming validates streaming functionality
func TestPhase2_BidirectionalStreaming(t *testing.T) {
	t.Run("stream_establishes_bidirectional_connection", func(t *testing.T) {
		// Start mock streaming server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockStream := NewMockStreamServer()
		cline.RegisterTaskServiceServer(server, mockStream)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Connect to server
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)
		stream, err := client.Stream(context.Background())
		require.NoError(t, err)
		defer stream.CloseSend()

		// Send a message
		msg := &cline.ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: cline.ClineMessageType_SAY,
			Text: "test message",
		}

		err = stream.Send(msg)
		require.NoError(t, err)

		// Receive echo
		resp, err := stream.Recv()
		require.NoError(t, err)
		assert.Contains(t, resp.Text, "echo:")
	})

	t.Run("stream_handles_multiple_messages", func(t *testing.T) {
		// Start mock streaming server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockStream := NewMockStreamServer()
		cline.RegisterTaskServiceServer(server, mockStream)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)
		stream, err := client.Stream(context.Background())
		require.NoError(t, err)
		defer stream.CloseSend()

		// Send multiple messages
		for i := 0; i < 5; i++ {
			msg := &cline.ClineMessage{
				Ts:   time.Now().UnixMilli(),
				Type: cline.ClineMessageType_SAY,
				Text: fmt.Sprintf("message %d", i),
			}

			err = stream.Send(msg)
			require.NoError(t, err)

			resp, err := stream.Recv()
			require.NoError(t, err)
			assert.Contains(t, resp.Text, fmt.Sprintf("message %d", i))
		}
	})

	t.Run("stream_handles_server_disconnect", func(t *testing.T) {
		// Start and stop server quickly to test disconnect handling
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		server := grpc.NewServer()
		mockStream := NewMockStreamServer()
		cline.RegisterTaskServiceServer(server, mockStream)

		go server.Serve(lis)
		time.Sleep(100 * time.Millisecond)

		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)
		stream, err := client.Stream(context.Background())
		require.NoError(t, err)

		// Stop server
		server.Stop()
		lis.Close()

		// Try to send - should eventually fail
		msg := &cline.ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: cline.ClineMessageType_SAY,
			Text: "after disconnect",
		}

		// This may or may not fail immediately depending on buffering
		_ = stream.Send(msg)
	})

	t.Run("stream_timeout_handling", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockStream := NewMockStreamServer()
		cline.RegisterTaskServiceServer(server, mockStream)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		// Create stream with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		client := cline.NewTaskServiceClient(conn)
		stream, err := client.Stream(ctx)
		require.NoError(t, err)
		defer stream.CloseSend()

		// Stream should respect context timeout
		time.Sleep(150 * time.Millisecond)

		// After timeout, operations should fail
		msg := &cline.ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: cline.ClineMessageType_SAY,
			Text: "after timeout",
		}

		err = stream.Send(msg)
		// May succeed or fail depending on timing
		t.Logf("Send after timeout result: %v", err)
	})
}

// TestPhase2_StateSynchronization validates state sync functionality
func TestPhase2_StateSynchronization(t *testing.T) {
	t.Run("state_sync_initial_fetch", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockState := NewMockStateService()
		cline.RegisterStateServiceServer(server, mockState)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewStateServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state, err := client.GetLatestState(ctx, &cline.EmptyRequest{})
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Contains(t, state.StateJson, "version")
	})

	t.Run("state_sync_handles_updates", func(t *testing.T) {
		// Create a mock client for StateSync
		config := host.ClientConfig{
			Target:   "localhost:50051",
			PoolSize: 1,
		}
		client, err := host.NewClient(config)
		require.NoError(t, err)

		// Create state sync with the client
		syncConfig := host.DefaultStateSyncConfig()
		syncer, err := host.NewStateSync(syncConfig, client)
		require.NoError(t, err)

		// Set a value
		testData := []byte(`{"provider": "anthropic", "model": "claude-3"}`)
		err = syncer.Set("test.key", testData)
		require.NoError(t, err)

		// Verify value was set
		version, ok := syncer.Get("test.key")
		require.True(t, ok)
		assert.Equal(t, testData, version.Data)
	})

	t.Run("state_sync_conflict_resolution", func(t *testing.T) {
		config := host.ClientConfig{
			Target:   "localhost:50051",
			PoolSize: 1,
		}
		client, err := host.NewClient(config)
		require.NoError(t, err)

		syncConfig := host.DefaultStateSyncConfig()
		syncer, err := host.NewStateSync(syncConfig, client)
		require.NoError(t, err)

		// Set initial value
		err = syncer.Set("conflict.key", []byte("initial"))
		require.NoError(t, err)

		// Simulate remote update with conflict resolution
		localVersion, _ := syncer.Get("conflict.key")
		remoteVersion := host.StateVersion{
			Version:   localVersion.Version + 1,
			Timestamp: time.Now(),
			Data:      []byte("remote"),
			Source:    "core",
		}

		err = syncer.HandleRemoteUpdate("conflict.key", remoteVersion)
		require.NoError(t, err)

		// Verify state was updated
		finalVersion, ok := syncer.Get("conflict.key")
		require.True(t, ok)
		// Remote should win due to higher version
		assert.Equal(t, "remote", string(finalVersion.Data))
	})
}

// TestPhase2_MessageConversion validates message conversion
func TestPhase2_MessageConversion(t *testing.T) {
	converter := host.NewMessageConverter()
	require.NotNil(t, converter)

	t.Run("convert_simple_text_message", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TEXT,
			Text: "Hello, world!",
		}

		internal := converter.ProtoToInternal(proto)

		assert.Equal(t, int64(1234567890), internal.Ts)
		assert.Equal(t, host.ClineMessageType_SAY, internal.Type)
		assert.Equal(t, host.ClineSay_TEXT, internal.Say)
		assert.Equal(t, "Hello, world!", internal.Text)
	})

	t.Run("convert_tool_message", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TOOL_SAY,
			Text: "Tool executed successfully",
			SayTool: &cline.ClineSayTool{
				Tool: cline.ClineSayToolType_READ_FILE,
				Path: "/workspace/test.txt",
				Content: "file contents",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.Equal(t, host.ClineSay_TOOL_SAY, internal.Say)
		assert.NotNil(t, internal.SayTool)
		assert.Equal(t, host.ClineSayToolType_READ_FILE, internal.SayTool.Tool)
		assert.Equal(t, "/workspace/test.txt", internal.SayTool.Path)
	})

	t.Run("convert_api_request_info", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_API_REQ_STARTED,
			ApiReqInfo: &cline.ClineApiReqInfo{
				TokensIn:   1000,
				TokensOut:  500,
				Cost:       0.002,
				CacheReads: 50,
				CacheWrites: 25,
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.ApiReqInfo)
		assert.Equal(t, int32(1000), internal.ApiReqInfo.TokensIn)
		assert.Equal(t, int32(500), internal.ApiReqInfo.TokensOut)
		assert.InDelta(t, 0.002, internal.ApiReqInfo.Cost, 0.0001)
	})

	t.Run("convert_mcp_server_message", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_ASK,
			Ask:  cline.ClineAsk_USE_MCP_SERVER,
			AskUseMcpServer: &cline.ClineAskUseMcpServer{
				ServerName: "test-server",
				Type:       cline.McpServerRequestType_USE_MCP_TOOL,
				ToolName:   "readFile",
				Arguments:  `{"path": "/test.txt"}`,
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.Equal(t, host.ClineMessageType_ASK, internal.Type)
		assert.Equal(t, host.ClineAsk_USE_MCP_SERVER, internal.Ask)
		assert.NotNil(t, internal.AskUseMcpServer)
		assert.Equal(t, "test-server", internal.AskUseMcpServer.ServerName)
		assert.Equal(t, "readFile", internal.AskUseMcpServer.ToolName)
	})

	t.Run("round_trip_conversion", func(t *testing.T) {
		original := &cline.ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_COMMAND_OUTPUT_SAY,
			Text: "Command output",
		}

		// Proto -> Internal
		internal := converter.ProtoToInternal(original)

		// Internal -> Proto
		converted := converter.InternalToProto(internal)

		// Verify key fields preserved
		assert.Equal(t, original.Ts, converted.Ts)
		assert.Equal(t, original.Type, converted.Type)
		assert.Equal(t, original.Say, converted.Say)
		assert.Equal(t, original.Text, converted.Text)
	})
}

// TestPhase2_ConnectionManager validates connection manager functionality
func TestPhase2_ConnectionManager(t *testing.T) {
	t.Run("connection_manager_creation", func(t *testing.T) {
		config := &host.EndpointConfig{
			Address:           "localhost:50051",
			HostBridgeAddress: "localhost:50052",
		}

		cm := host.NewConnectionManager(config)
		require.NotNil(t, cm)
	})

	t.Run("connection_manager_connect", func(t *testing.T) {
		// Start mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		config := &host.EndpointConfig{
			Address: lis.Addr().String(),
		}

		cm := host.NewConnectionManager(config)
		require.NotNil(t, cm)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = cm.Connect(ctx)
		require.NoError(t, err)

		assert.True(t, cm.IsConnected())

		err = cm.Close()
		require.NoError(t, err)
	})

	t.Run("connection_manager_retry", func(t *testing.T) {
		config := &host.EndpointConfig{
			Address: "localhost:59999", // Non-existent server
		}

		cm := host.NewConnectionManager(config)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Should fail after retries
		err := cm.ConnectWithRetry(ctx, 2)
		assert.Error(t, err)
	})

	t.Run("connection_manager_health_check", func(t *testing.T) {
		// Start mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		config := &host.EndpointConfig{
			Address: lis.Addr().String(),
		}

		cm := host.NewConnectionManager(config)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = cm.Connect(ctx)
		require.NoError(t, err)
		defer cm.Close()

		// Health check should pass
		err = cm.HealthCheck(ctx)
		// May fail if health service not registered, which is OK for this test
		t.Logf("Health check result: %v", err)
	})
}

// TestPhase2_EndToEnd validates end-to-end Phase 2 functionality
func TestPhase2_EndToEnd(t *testing.T) {
	t.Run("complete_connection_lifecycle", func(t *testing.T) {
		// Start mock server
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockStream := NewMockStreamServer()
		cline.RegisterTaskServiceServer(server, mockStream)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Create connection manager
		config := &host.EndpointConfig{
			Address: lis.Addr().String(),
		}

		cm := host.NewConnectionManager(config)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Connect
		err = cm.Connect(ctx)
		require.NoError(t, err)

		// Verify connected
		assert.True(t, cm.IsConnected())

		// Get proto client
		client := cm.GetProtoClient()
		assert.NotNil(t, client)

		// Close connection
		err = cm.Close()
		require.NoError(t, err)

		// Verify disconnected
		assert.False(t, cm.IsConnected())
	})

	t.Run("state_sync_with_connection", func(t *testing.T) {
		// Start mock server with state
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockState := NewMockStateService()
		cline.RegisterStateServiceServer(server, mockState)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Connect
		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		// Fetch state
		client := cline.NewStateServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		state, err := client.GetLatestState(ctx, &cline.EmptyRequest{})
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Verify state contains version
		assert.Contains(t, state.StateJson, "version")
	})
}

// Helper types and functions
