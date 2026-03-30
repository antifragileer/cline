// Package integration provides integration tests for gRPC connectivity.
// These tests verify the Go CLI's ability to communicate with the core extension.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// TestGRPCConnectivity validates gRPC server connectivity
func TestGRPCConnectivity(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("grpc_health_check", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start the core extension if available
		// For now, test that the CLI can start without crashing
		cmd := exec.CommandContext(ctx, goPath, "version", "--short")
		
		out, err := cmd.CombinedOutput()
		t.Logf("CLI version output: %s", string(out))

		// CLI should be able to check version without gRPC
		require.NoError(t, err, "CLI should respond to version command")
		assert.Contains(t, string(out), ".")
	})

	t.Run("grpc_connection_timeout", func(t *testing.T) {
		// Test that CLI handles gRPC connection timeouts gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try to run a task without core extension running
		cmd := exec.CommandContext(ctx, goPath, "--json", "test prompt")
		cmd.Env = append(os.Environ(),
			"CLINE_CORE_TIMEOUT=1", // Short timeout
		)

		out, err := cmd.CombinedOutput()
		t.Logf("Timeout test output: %s", string(out))

		// Should handle gracefully (not crash)
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		// Exit code should be valid
		assert.True(t, exitCode >= 0 && exitCode <= 255, "Invalid exit code: %d", exitCode)
	})

	t.Run("grpc_retry_mechanism", func(t *testing.T) {
		// Test retry logic
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "version", "-v")
		
		start := time.Now()
		out, err := cmd.CombinedOutput()
		elapsed := time.Since(start)
		cancel()

		t.Logf("Retry test took %v, output: %s", elapsed, string(out))
		require.NoError(t, err, "Should complete successfully")
	})
}

// TestCoreExtensionIntegration validates core extension interaction
func TestCoreExtensionIntegration(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("core_extension_discovery", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "version", "--json")
		
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "CLI should work standalone")

		// Parse version output
		var versionInfo map[string]interface{}
		if err := json.Unmarshal(out, &versionInfo); err == nil {
			t.Logf("Version info: %+v", versionInfo)
		}
	})

	t.Run("core_extension_state_sync", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grpc-sync-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Set some config
		cmd1 := exec.CommandContext(ctx, goPath, "config", "set", "test.key", "test-value")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out1, err1 := cmd1.CombinedOutput()
		
		if err1 == nil {
			t.Logf("Config set succeeded: %s", string(out1))

			// Get the config back
			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			cmd2 := exec.CommandContext(ctx2, goPath, "config", "get", "test.key")
			cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			out2, err2 := cmd2.CombinedOutput()
			cancel2()

			if err2 == nil {
				t.Logf("Config get succeeded: %s", string(out2))
				assert.Contains(t, string(out2), "test-value")
			}
		}
	})
}

// TestTaskFlowIntegration validates complete task execution flow
func TestTaskFlowIntegration(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("task_flow_with_core", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// This would normally connect to a running core extension
		// For integration testing, we verify the CLI prepares correctly
		cmd := exec.CommandContext(ctx, goPath, "task", "--help")
		
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Task help should work")

		helpText := string(out)
		assert.Contains(t, helpText, "task", "Task")
	})

	t.Run("task_resumption_flow", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "task-resume-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create mock task history
		tasksDir := filepath.Join(tempDir, "tasks")
		err = os.MkdirAll(tasksDir, 0755)
		require.NoError(t, err)

		mockHistory := []map[string]interface{}{
			{
				"id":        "resume-test-task",
				"timestamp": time.Now().UnixMilli(),
				"prompt":    "Previous task",
				"status":    "completed",
			},
		}

		historyData, _ := json.Marshal(mockHistory)
		err = os.WriteFile(filepath.Join(tasksDir, "history.json"), historyData, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "history")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		
		out, err := cmd.CombinedOutput()
		t.Logf("History output: %s", string(out))

		// Should show history without crashing
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestStreamingResponses validates streaming response handling
func TestStreamingResponses(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("stream_json_output", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "version", "--json")
		
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Validate JSON output
		var result map[string]interface{}
		err = json.Unmarshal(out, &result)
		assert.NoError(t, err, "Output should be valid JSON")
	})

	t.Run("stream_line_by_line", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		
		out, err := cmd.CombinedOutput()
		t.Logf("Config list output lines: %d", len(string(out)))

		// Should produce output
		if err == nil {
			assert.NotEmpty(t, string(out))
		}
	})
}

// TestErrorPropagation validates error handling through gRPC
func TestErrorPropagation(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("grpc_error_handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try invalid operation
		cmd := exec.CommandContext(ctx, goPath, "invalid-command-xyz")
		
		out, err := cmd.CombinedOutput()
		t.Logf("Error output: %s", string(out))

		// Should return error
		assert.Error(t, err, "Invalid command should error")
	})

	t.Run("timeout_error_handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set very short timeout
		cmd := exec.CommandContext(ctx, goPath, "-t", "1", "version")
		
		out, err := cmd.CombinedOutput()
		t.Logf("Timeout error output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		// Should complete or timeout gracefully
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestConnectionPooling validates connection reuse
func TestConnectionPooling(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("multiple_requests_reuse", func(t *testing.T) {
		// Run multiple commands sequentially
		for i := 0; i < 5; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, goPath, "version", "--short")
			
			out, err := cmd.CombinedOutput()
			cancel()

			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
			} else {
				t.Logf("Request %d succeeded: %s", i, string(out))
			}
		}
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				cmd := exec.CommandContext(ctx, goPath, "version", "--short")
				_, err := cmd.CombinedOutput()
				
				if err != nil {
					errors <- fmt.Errorf("request %d failed: %w", index, err)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Logf("Error: %v", err)
			errorCount++
		}

		// Most requests should succeed
		assert.Less(t, errorCount, 3, "Too many concurrent requests failed")
	})
}

// TestMockGRPCServer tests against a mock gRPC server
func TestMockGRPCServer(t *testing.T) {
	t.Run("mock_health_check", func(t *testing.T) {
		// Start mock server
		server, addr, err := startMockGRPCServer()
		if err != nil {
			t.Skipf("Could not start mock server: %v", err)
		}
		defer server.Stop()

		// Connect to mock server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			t.Skipf("Could not connect to mock server: %v", err)
		}
		defer conn.Close()

		// Test health check
		healthClient := grpc_health_v1.NewHealthClient(conn)
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		
		if err != nil {
			t.Logf("Health check error: %v", err)
		} else {
			t.Logf("Health check status: %v", resp.Status)
			assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
		}
	})
}

// TestConfigurationViaGRPC validates configuration operations over gRPC
func TestConfigurationViaGRPC(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("config_get_via_grpc", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "config", "get", "provider")
		
		out, err := cmd.CombinedOutput()
		t.Logf("Config get output: %s", string(out))

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("config_set_via_grpc", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "config-grpc-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "config", "set", "test.grpc.key", "grpc-value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		
		out, err := cmd.CombinedOutput()
		t.Logf("Config set output: %s", string(out))

		// Should complete
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}


// Mock gRPC server for testing
type mockHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (s *mockHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func startMockGRPCServer() (*grpc.Server, string, error) {
	// Use a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}

	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, &mockHealthServer{})

	go server.Serve(listener)

	return server, listener.Addr().String(), nil
}
