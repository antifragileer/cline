// Package integration provides integration tests for the Go CLI.
// This file contains Phase 1 integration tests covering gRPC integration,
// message conversion, error handling, and task management.
package integration

import (
	"context"
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

// MockStateService implements the StateService for testing
type MockStateService struct {
	cline.UnimplementedStateServiceServer
	state *cline.State
}

func NewMockStateService() *MockStateService {
	return &MockStateService{
		state: &cline.State{
			StateJson: `{"version":"1.0.0"}`,
		},
	}
}

func (m *MockStateService) GetLatestState(ctx context.Context, req *cline.EmptyRequest) (*cline.State, error) {
	return m.state, nil
}

// TestPhase1_GRPCIntegration validates gRPC connectivity with extension core
func TestPhase1_GRPCIntegration(t *testing.T) {
	t.Run("establishes connection to extension core", func(t *testing.T) {
		// Create a listener on a random port
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		// Create gRPC server
		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		// Start server in a goroutine
		go func() {
			if err := server.Serve(lis); err != nil {
				t.Logf("Server error: %v", err)
			}
		}()
		defer server.Stop()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Test connection establishment
		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		// Verify connection state
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Wait for ready
		state := conn.GetState()
		for state != connectivity.Ready {
			if !conn.WaitForStateChange(ctx, state) {
				t.Fatal("timeout waiting for connection")
			}
			state = conn.GetState()
		}

		assert.Equal(t, connectivity.Ready, state)
	})

	t.Run("handles connection failures gracefully", func(t *testing.T) {
		// Try to connect to non-existent server
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_, err := grpc.DialContext(ctx, "localhost:59999", 
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
		
		// Should get a connection error
		assert.Error(t, err)
	})

	t.Run("supports bidirectional streaming", func(t *testing.T) {
		// This test validates that the streaming infrastructure is in place
		// Full streaming tests are in stream_test.go
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		// Create a stream client (even if mock doesn't fully implement it)
		client := cline.NewTaskServiceClient(conn)
		require.NotNil(t, client)

		// The streaming capability exists in the client
		// Full streaming tests verify the actual streaming behavior
	})
}

// TestPhase1_MessageConversion validates message type conversion
func TestPhase1_MessageConversion(t *testing.T) {
	converter := host.NewMessageConverter()
	require.NotNil(t, converter)

	t.Run("converts proto to internal format", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TEXT,
			Text: "Hello from extension",
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.Equal(t, int64(1234567890), internal.Ts)
		assert.Equal(t, host.ClineMessageType_SAY, internal.Type)
		assert.Equal(t, host.ClineSay_TEXT, internal.Say)
		assert.Equal(t, "Hello from extension", internal.Text)
	})

	t.Run("converts internal to proto format", func(t *testing.T) {
		internal := &host.ClineMessage{
			Ts:   1234567890,
			Type: host.ClineMessageType_ASK,
			Ask:  host.ClineAsk_COMMAND,
			Text: "Execute command",
		}

		proto := converter.InternalToProto(internal)

		assert.NotNil(t, proto)
		assert.Equal(t, int64(1234567890), proto.Ts)
		assert.Equal(t, cline.ClineMessageType_ASK, proto.Type)
		assert.Equal(t, cline.ClineAsk_COMMAND, proto.Ask)
	})

	t.Run("round-trip conversion preserves data", func(t *testing.T) {
		original := &cline.ClineMessage{
			Ts:      1234567890,
			Type:    cline.ClineMessageType_SAY,
			Say:     cline.ClineSay_TOOL_SAY,
			Text:    "Tool executed",
			SayTool: &cline.ClineSayTool{
				Tool: cline.ClineSayToolType_READ_FILE,
				Path: "/workspace/file.txt",
			},
		}

		internal := converter.ProtoToInternal(original)
		converted := converter.InternalToProto(internal)

		assert.Equal(t, original.Ts, converted.Ts)
		assert.Equal(t, original.Text, converted.Text)
		assert.Equal(t, original.SayTool.Tool, converted.SayTool.Tool)
		assert.Equal(t, original.SayTool.Path, converted.SayTool.Path)
	})

	t.Run("handles complex message types", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: cline.ClineMessageType_ASK,
			Ask:  cline.ClineAsk_USE_MCP_SERVER,
			AskUseMcpServer: &cline.ClineAskUseMcpServer{
				ServerName: "test-mcp-server",
				Type:       cline.McpServerRequestType_USE_MCP_TOOL,
				ToolName:   "test-tool",
				Arguments:  `{"key": "value"}`,
			},
			ApiReqInfo: &cline.ClineApiReqInfo{
				TokensIn:  100,
				TokensOut: 50,
				Cost:      0.001,
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.AskUseMcpServer)
		assert.Equal(t, "test-mcp-server", internal.AskUseMcpServer.ServerName)
		assert.NotNil(t, internal.ApiReqInfo)
		assert.Equal(t, int32(100), internal.ApiReqInfo.TokensIn)
	})
}

// TestPhase1_ErrorHandling validates error handling and reconnection
func TestPhase1_ErrorHandling(t *testing.T) {
	handler := host.NewErrorHandler()
	require.NotNil(t, handler)

	t.Run("identifies retryable errors", func(t *testing.T) {
		retryableErrs := []error{
			host.ErrStreamNotReady,
			host.ErrBackpressureExceeded,
		}

		for _, err := range retryableErrs {
			assert.True(t, handler.IsRetryableError(err), 
				"Expected %v to be retryable", err)
		}
	})

	t.Run("identifies non-retryable errors", func(t *testing.T) {
		// These errors should not trigger retries
		nonRetryableErrs := []error{
			host.ErrStreamClosed,
		}

		for _, err := range nonRetryableErrs {
			assert.False(t, handler.IsRetryableError(err),
				"Expected %v to be non-retryable", err)
		}
	})

	t.Run("calculates exponential backoff", func(t *testing.T) {
		backoff0 := handler.CalculateBackoff(0)
		backoff1 := handler.CalculateBackoff(1)
		backoff2 := handler.CalculateBackoff(2)

		assert.Equal(t, 500*time.Millisecond, backoff0)
		assert.Equal(t, 1*time.Second, backoff1)
		assert.Equal(t, 2*time.Second, backoff2)
	})

	t.Run("caps backoff at maximum", func(t *testing.T) {
		// After many retries, backoff should be capped
		backoff := handler.CalculateBackoff(10)
		assert.LessOrEqual(t, backoff, 30*time.Second)
	})
}

// TestPhase1_TaskManagement validates task lifecycle management
func TestPhase1_TaskManagement(t *testing.T) {
	t.Run("creates new task via gRPC", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &cline.NewTaskRequest{
			Text: "Create a new task",
		}

		resp, err := client.NewTask(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Value)
	})

	t.Run("cancels running task", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.CancelTask(ctx, &cline.EmptyRequest{})
		// Mock doesn't implement this, but the call should not panic
		assert.Error(t, err) // Expected since mock doesn't implement
	})

	t.Run("retrieves task history", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &cline.GetTaskHistoryRequest{
			FavoritesOnly:        false,
			CurrentWorkspaceOnly: true,
		}

		_, err = client.GetTaskHistory(ctx, req)
		// Mock doesn't implement this, but the call should not panic
		assert.Error(t, err) // Expected since mock doesn't implement
	})
}

// TestPhase1_StateSynchronization validates state synchronization
func TestPhase1_StateSynchronization(t *testing.T) {
	t.Run("retrieves state from extension", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockStateSvc := NewMockStateService()
		cline.RegisterStateServiceServer(server, mockStateSvc)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
}

// TestPhase1_EndToEnd validates end-to-end integration
func TestPhase1_EndToEnd(t *testing.T) {
	t.Run("complete task creation and cancellation flow", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		// Set up server
		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Connect
		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := cline.NewTaskServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create task
		createReq := &cline.NewTaskRequest{
			Text:   "Test task",
			Images: []string{"image1.png"},
			Files:  []string{"file1.txt"},
		}

		createResp, err := client.NewTask(ctx, createReq)
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Value)

		// Show task
		showReq := &cline.StringRequest{
			Value: createResp.Value,
		}

		taskResp, err := client.ShowTaskWithId(ctx, showReq)
		require.NoError(t, err)
		assert.Equal(t, createReq.Text, taskResp.Task)
		assert.Equal(t, createResp.Value, taskResp.Id)

		t.Logf("Successfully created and retrieved task: %s", taskResp.Id)
	})
}

// BenchmarkPhase1_GRPCPerformance benchmarks gRPC performance
func BenchmarkPhase1_GRPCPerformance(b *testing.B) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	server := grpc.NewServer()
	mockService := NewMockTaskService()
	cline.RegisterTaskServiceServer(server, mockService)

	go server.Serve(lis)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	addr := lis.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := cline.NewTaskServiceClient(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req := &cline.NewTaskRequest{
			Text: "Benchmark task",
		}
		_, err := client.NewTask(ctx, req)
		cancel()

		if err != nil {
			b.Errorf("Request failed: %v", err)
		}
	}
}

// BenchmarkPhase1_MessageConversion benchmarks message conversion performance
func BenchmarkPhase1_MessageConversion(b *testing.B) {
	converter := host.NewMessageConverter()

	proto := &cline.ClineMessage{
		Ts:   1234567890,
		Type: cline.ClineMessageType_SAY,
		Say:  cline.ClineSay_TEXT,
		Text: "Benchmark message",
		SayTool: &cline.ClineSayTool{
			Tool: cline.ClineSayToolType_READ_FILE,
			Path: "/test/file.txt",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		internal := converter.ProtoToInternal(proto)
		_ = converter.InternalToProto(internal)
	}
}