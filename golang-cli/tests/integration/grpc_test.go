// Package integration provides integration tests for the Go CLI.
// This package tests gRPC service implementations.
package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockTaskService implements the TaskService for testing
type MockTaskService struct {
	cline.UnimplementedTaskServiceServer
	tasks map[string]*cline.Task
}

func NewMockTaskService() *MockTaskService {
	return &MockTaskService{
		tasks: make(map[string]*cline.Task),
	}
}

func (m *MockTaskService) CreateTask(ctx context.Context, req *cline.NewTaskRequest) (*cline.Task, error) {
	task := &cline.Task{
		Id:      "test-task-1",
		Message: req.Message,
	}
	m.tasks[task.Id] = task
	return task, nil
}

func (m *MockTaskService) GetTask(ctx context.Context, req *cline.StringRequest) (*cline.Task, error) {
	task, ok := m.tasks[req.Value]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", req.Value)
	}
	return task, nil
}

// TestGRPCServerSetup tests basic gRPC server setup and teardown
func TestGRPCServerSetup(t *testing.T) {
	t.Run("starts and stops server", func(t *testing.T) {
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

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Stop the server
		server.Stop()
	})

	t.Run("accepts connections", func(t *testing.T) {
		lis, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer lis.Close()

		server := grpc.NewServer()
		mockService := NewMockTaskService()
		cline.RegisterTaskServiceServer(server, mockService)

		go server.Serve(lis)
		defer server.Stop()

		time.Sleep(100 * time.Millisecond)

		// Connect to the server
		addr := lis.Addr().String()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		// Create a client
		client := cline.NewTaskServiceClient(conn)
		require.NotNil(t, client)
	})
}

// TestGRPCServiceMethods tests gRPC service method implementations
func TestGRPCServiceMethods(t *testing.T) {
	t.Run("CreateTask creates task successfully", func(t *testing.T) {
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

		req := &cline.NewTaskRequest{
			Message: "Test task message",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := client.CreateTask(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Test task message", resp.Message)
		assert.NotEmpty(t, resp.Id)
	})

	t.Run("GetTask retrieves task successfully", func(t *testing.T) {
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

		// First create a task
		createReq := &cline.NewTaskRequest{
			Message: "Test task for retrieval",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		created, err := client.CreateTask(ctx, createReq)
		require.NoError(t, err)

		// Now retrieve it
		getReq := &cline.StringRequest{
			Value: created.Id,
		}

		retrieved, err := client.GetTask(ctx, getReq)
		require.NoError(t, err)
		assert.Equal(t, created.Id, retrieved.Id)
		assert.Equal(t, created.Message, retrieved.Message)
	})

	t.Run("GetTask returns error for non-existent task", func(t *testing.T) {
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

		req := &cline.StringRequest{
			Value: "non-existent-task-id",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.GetTask(ctx, req)
		assert.Error(t, err)
	})
}

// TestGRPCStreaming tests gRPC streaming capabilities
func TestGRPCStreaming(t *testing.T) {
	t.Run("server handles concurrent requests", func(t *testing.T) {
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

		// Create multiple concurrent clients
		numClients := 10
		done := make(chan bool, numClients)

		for i := 0; i < numClients; i++ {
			go func(clientNum int) {
				conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					t.Errorf("Client %d: failed to connect: %v", clientNum, err)
					done <- false
					return
				}
				defer conn.Close()

				client := cline.NewTaskServiceClient(conn)

				req := &cline.NewTaskRequest{
					Message: fmt.Sprintf("Task from client %d", clientNum),
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_, err = client.CreateTask(ctx, req)
				if err != nil {
					t.Errorf("Client %d: failed to create task: %v", clientNum, err)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all clients to complete
		successCount := 0
		for i := 0; i < numClients; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, numClients, successCount, "All concurrent requests should succeed")
	})
}

// TestGRPCErrorHandling tests gRPC error handling
func TestGRPCErrorHandling(t *testing.T) {
	t.Run("handles timeout gracefully", func(t *testing.T) {
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

		req := &cline.NewTaskRequest{
			Message: "Test task",
		}

		// Use a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait to ensure timeout
		time.Sleep(10 * time.Millisecond)

		_, err = client.CreateTask(ctx, req)
		// Should get a timeout error
		assert.Error(t, err)
	})

	t.Run("handles cancelled context", func(t *testing.T) {
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

		req := &cline.NewTaskRequest{
			Message: "Test task",
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = client.CreateTask(ctx, req)
		assert.Error(t, err)
	})
}

// BenchmarkGRPCPerformance benchmarks gRPC performance
func BenchmarkGRPCPerformance(b *testing.B) {
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
		req := &cline.NewTaskRequest{
			Message: fmt.Sprintf("Benchmark task %d", i),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := client.CreateTask(ctx, req)
		cancel()

		if err != nil {
			b.Errorf("Request failed: %v", err)
		}
	}
}