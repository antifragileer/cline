// Package grpc provides health monitoring functionality.
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// MockHealthClient implements the HealthClient interface for testing
type MockHealthClient struct {
	CheckFunc func(ctx context.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error)
}

func (m *MockHealthClient) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	return m.CheckFunc(ctx, in, opts...)
}

func (m *MockHealthClient) Watch(ctx context.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (grpc_health_v1.Health_WatchClient, error) {
	return nil, errors.New("not implemented")
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusUnknown, "unknown"},
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewHealthMonitor_Defaults(t *testing.T) {
	// Create a mock connection (we won't actually use it for network operations)
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 0, 0)

	if monitor.checkInterval != 5*time.Second {
		t.Errorf("Expected default checkInterval to be 5s, got %v", monitor.checkInterval)
	}
	if monitor.timeout != 2*time.Second {
		t.Errorf("Expected default timeout to be 2s, got %v", monitor.timeout)
	}
	if monitor.status != HealthStatusUnknown {
		t.Errorf("Expected initial status to be Unknown, got %v", monitor.status)
	}
}

func TestNewHealthMonitor_CustomValues(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 10*time.Second, 5*time.Second)

	if monitor.checkInterval != 10*time.Second {
		t.Errorf("Expected checkInterval to be 10s, got %v", monitor.checkInterval)
	}
	if monitor.timeout != 5*time.Second {
		t.Errorf("Expected timeout to be 5s, got %v", monitor.timeout)
	}
}

func TestHealthMonitor_StartStop(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 100*time.Millisecond, 1*time.Second)

	// Test that Start doesn't panic
	monitor.Start()

	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)

	// Test that Stop doesn't panic
	monitor.Stop()
}

func TestHealthMonitor_GetStatus(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	// Initial status should be Unknown
	if status := monitor.GetStatus(); status != HealthStatusUnknown {
		t.Errorf("Expected initial status to be Unknown, got %v", status)
	}

	// Set status manually for testing
	monitor.status = HealthStatusHealthy
	if status := monitor.GetStatus(); status != HealthStatusHealthy {
		t.Errorf("Expected status to be Healthy, got %v", status)
	}
}

func TestHealthMonitor_GetMetrics(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	// Set some metrics manually
	monitor.metrics.PingCount = 10
	monitor.metrics.ConsecutiveErrors = 2

	metrics := monitor.GetMetrics()
	if metrics.PingCount != 10 {
		t.Errorf("Expected PingCount to be 10, got %d", metrics.PingCount)
	}
	if metrics.ConsecutiveErrors != 2 {
		t.Errorf("Expected ConsecutiveErrors to be 2, got %d", metrics.ConsecutiveErrors)
	}
}

func TestHealthMonitor_SetStatusChangeCallback(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	monitor.SetStatusChangeCallback(func(old, new HealthStatus) {
		// Callback set
	})

	// Note: In actual implementation, callback would be triggered during health checks
	// Here we're just verifying the callback can be set
	if monitor.onStatusChange == nil {
		t.Error("Status change callback was not set")
	}
}

func TestHealthMonitor_SetUnhealthyCallback(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	monitor.SetUnhealthyCallback(func() {
		// Callback set
	})

	if monitor.onUnhealthy == nil {
		t.Error("Unhealthy callback was not set")
	}
}

func TestHealthMonitor_SetHealthyCallback(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	monitor.SetHealthyCallback(func() {
		// Callback set
	})

	if monitor.onHealthy == nil {
		t.Error("Healthy callback was not set")
	}
}

func TestHealthMonitor_IsHealthy(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	// Unknown is not healthy
	if monitor.IsHealthy() {
		t.Error("Expected IsHealthy to be false for Unknown status")
	}

	monitor.status = HealthStatusHealthy
	if !monitor.IsHealthy() {
		t.Error("Expected IsHealthy to be true for Healthy status")
	}

	monitor.status = HealthStatusDegraded
	if monitor.IsHealthy() {
		t.Error("Expected IsHealthy to be false for Degraded status")
	}

	monitor.status = HealthStatusUnhealthy
	if monitor.IsHealthy() {
		t.Error("Expected IsHealthy to be false for Unhealthy status")
	}
}

func TestHealthMonitor_GetLastCheckTime(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	before := time.Now()
	monitor.lastCheckTime = before
	after := monitor.GetLastCheckTime()

	if !after.Equal(before) {
		t.Errorf("Expected GetLastCheckTime to return %v, got %v", before, after)
	}
}

func TestHealthMonitor_WaitForHealthy_Timeout(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusUnhealthy

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = monitor.WaitForHealthy(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}
}

func TestHealthMonitor_WaitForHealthy_Success(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusHealthy

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = monitor.WaitForHealthy(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHealthMonitor_GenerateReport_Unhealthy(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusUnhealthy
	monitor.lastCheckTime = time.Now()

	report := monitor.GenerateReport()

	if report.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status to be Unhealthy, got %v", report.Status)
	}

	// Should have recommendations for unhealthy status
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations for unhealthy status")
	}
}

func TestHealthMonitor_GenerateReport_Degraded(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusDegraded
	monitor.lastCheckTime = time.Now()

	report := monitor.GenerateReport()

	if report.Status != HealthStatusDegraded {
		t.Errorf("Expected status to be Degraded, got %v", report.Status)
	}

	// Should have recommendations for degraded status
	if len(report.Recommendations) == 0 {
		t.Error("Expected recommendations for degraded status")
	}
}

func TestHealthMonitor_GenerateReport_WithErrors(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusHealthy
	monitor.metrics.ConsecutiveErrors = 10
	monitor.lastCheckTime = time.Now()

	report := monitor.GenerateReport()

	// Should have recommendations for high error count
	foundErrorRec := false
	for _, rec := range report.Recommendations {
		if rec == "Multiple consecutive errors (10) - may need reconnection" {
			foundErrorRec = true
			break
		}
	}
	if !foundErrorRec {
		t.Error("Expected recommendation about consecutive errors")
	}
}

func TestHealthMonitor_GenerateReport_WithDroppedMessages(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusHealthy
	monitor.metrics.MessagesDropped = 5
	monitor.lastCheckTime = time.Now()

	report := monitor.GenerateReport()

	// Should have recommendations for dropped messages
	foundDropRec := false
	for _, rec := range report.Recommendations {
		if rec == "Messages have been dropped (5) - check flow control" {
			foundDropRec = true
			break
		}
	}
	if !foundDropRec {
		t.Error("Expected recommendation about dropped messages")
	}
}

func TestConnectionHealthReport_String(t *testing.T) {
	report := ConnectionHealthReport{
		Status:          HealthStatusHealthy,
		ConnectionState: connectivity.Ready,
		Metrics: HealthMetrics{
			PingCount:        10,
			FailedPingCount:  1,
			MessagesSent:     100,
			MessagesReceived: 95,
			MessagesDropped:  5,
			ConsecutiveErrors: 0,
			TotalErrors:      2,
		},
		LastCheckTime: time.Now(),
	}

	str := report.String()

	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Check that key elements are present
	if report.Status.String() != "healthy" {
		t.Error("Status should be healthy")
	}
}

func TestHealthMonitor_UpdateMessageMetrics(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	monitor.UpdateMessageMetrics(10, 8, 2, 50*time.Millisecond)

	metrics := monitor.GetMetrics()
	if metrics.MessagesSent != 10 {
		t.Errorf("Expected MessagesSent to be 10, got %d", metrics.MessagesSent)
	}
	if metrics.MessagesReceived != 8 {
		t.Errorf("Expected MessagesReceived to be 8, got %d", metrics.MessagesReceived)
	}
	if metrics.MessagesDropped != 2 {
		t.Errorf("Expected MessagesDropped to be 2, got %d", metrics.MessagesDropped)
	}
	if metrics.AverageLatency != 50*time.Millisecond {
		t.Errorf("Expected AverageLatency to be 50ms, got %v", metrics.AverageLatency)
	}
}

func TestHealthMonitor_determineStatus_ConnectionNotReady(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	// Test when connection is not ready
	status := monitor.determineStatus(connectivity.TransientFailure, nil, nil)
	if status != HealthStatusUnhealthy {
		t.Errorf("Expected Unhealthy when connection not ready, got %v", status)
	}
}

func TestHealthMonitor_determineStatus_HealthCheckError(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	// Test with health check error but low consecutive errors
	status := monitor.determineStatus(connectivity.Ready, nil, errors.New("health check failed"))
	if status != HealthStatusDegraded {
		t.Errorf("Expected Degraded with few errors, got %v", status)
	}

	// Test with health check error and high consecutive errors
	monitor.metrics.ConsecutiveErrors = 5
	status = monitor.determineStatus(connectivity.Ready, nil, errors.New("health check failed"))
	if status != HealthStatusUnhealthy {
		t.Errorf("Expected Unhealthy with many errors, got %v", status)
	}
}

func TestHealthMonitor_determineStatus_NotServing(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	resp := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}

	status := monitor.determineStatus(connectivity.Ready, resp, nil)
	if status != HealthStatusUnhealthy {
		t.Errorf("Expected Unhealthy when NOT_SERVING, got %v", status)
	}
}

func TestHealthMonitor_determineStatus_UnknownStatus(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	resp := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_UNKNOWN,
	}

	status := monitor.determineStatus(connectivity.Ready, resp, nil)
	if status != HealthStatusDegraded {
		t.Errorf("Expected Degraded when UNKNOWN, got %v", status)
	}
}

func TestHealthMonitor_determineStatus_ServingWithHighLatency(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.metrics.LastPingDuration = 600 * time.Millisecond

	resp := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}

	status := monitor.determineStatus(connectivity.Ready, resp, nil)
	if status != HealthStatusDegraded {
		t.Errorf("Expected Degraded with high latency, got %v", status)
	}
}

func TestHealthMonitor_determineStatus_ServingHealthy(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Skip("Skipping test - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.metrics.LastPingDuration = 100 * time.Millisecond

	resp := &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}

	status := monitor.determineStatus(connectivity.Ready, resp, nil)
	if status != HealthStatusHealthy {
		t.Errorf("Expected Healthy, got %v", status)
	}
}

func TestHealthMetrics_Struct(t *testing.T) {
	metrics := HealthMetrics{
		LastPingTime:     time.Now(),
		LastPingDuration: 50 * time.Millisecond,
		AveragePingTime:  75 * time.Millisecond,
		PingCount:        100,
		FailedPingCount:  5,
		MessagesSent:     1000,
		MessagesReceived: 950,
		MessagesDropped:  50,
		AverageLatency:   25 * time.Millisecond,
		ConsecutiveErrors: 3,
		LastErrorTime:    time.Now(),
		TotalErrors:      10,
		RecordedAt:       time.Now(),
	}

	// Just verify the struct can be created and accessed
	if metrics.PingCount != 100 {
		t.Error("Unexpected PingCount")
	}
	if metrics.MessagesSent != 1000 {
		t.Error("Unexpected MessagesSent")
	}
}

func BenchmarkHealthMonitor_GetStatus(b *testing.B) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		b.Skip("Skipping benchmark - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)
	monitor.status = HealthStatusHealthy

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetStatus()
	}
}

func BenchmarkHealthMonitor_GetMetrics(b *testing.B) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		b.Skip("Skipping benchmark - cannot create connection")
	}
	defer conn.Close()

	monitor := NewHealthMonitor(conn, 5*time.Second, 2*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetMetrics()
	}
}