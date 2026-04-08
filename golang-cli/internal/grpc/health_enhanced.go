// Package grpc provides enhanced health monitoring for gRPC connections.
package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthStatus represents the health status of a connection
type HealthStatus int

const (
	// HealthStatusUnknown means the health is not yet determined
	HealthStatusUnknown HealthStatus = iota
	// HealthStatusHealthy means the connection is healthy
	HealthStatusHealthy
	// HealthStatusDegraded means the connection is working but with issues
	HealthStatusDegraded
	// HealthStatusUnhealthy means the connection is not working
	HealthStatusUnhealthy
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDegraded:
		return "degraded"
	case HealthStatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// HealthMetrics contains health-related metrics
type HealthMetrics struct {
	// Connection metrics
	LastPingTime     time.Time
	LastPingDuration time.Duration
	AveragePingTime  time.Duration
	PingCount        int64
	FailedPingCount  int64

	// Message metrics
	MessagesSent     int64
	MessagesReceived int64
	MessagesDropped  int64
	AverageLatency   time.Duration

	// Error metrics
	ConsecutiveErrors int
	LastErrorTime     time.Time
	TotalErrors       int64

	// Timestamp
	RecordedAt time.Time
}

// HealthMonitor monitors the health of a gRPC connection
type HealthMonitor struct {
	conn          *grpc.ClientConn
	healthClient  grpc_health_v1.HealthClient
	checkInterval time.Duration
	timeout       time.Duration

	// State
	status        HealthStatus
	statusMu      sync.RWMutex
	metrics       HealthMetrics
	metricsMu     sync.RWMutex
	lastCheckTime time.Time

	// Callbacks
	onStatusChange func(HealthStatus, HealthStatus)
	onUnhealthy    func()
	onHealthy      func()

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(conn *grpc.ClientConn, checkInterval, timeout time.Duration) *HealthMonitor {
	if checkInterval <= 0 {
		checkInterval = 5 * time.Second
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HealthMonitor{
		conn:          conn,
		healthClient:  grpc_health_v1.NewHealthClient(conn),
		checkInterval: checkInterval,
		timeout:       timeout,
		status:        HealthStatusUnknown,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins health monitoring
func (h *HealthMonitor) Start() {
	h.wg.Add(1)
	go h.monitorLoop()
}

// Stop stops health monitoring
func (h *HealthMonitor) Stop() {
	h.cancel()
	h.wg.Wait()
}

// GetStatus returns the current health status
func (h *HealthMonitor) GetStatus() HealthStatus {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status
}

// GetMetrics returns the current health metrics
func (h *HealthMonitor) GetMetrics() HealthMetrics {
	h.metricsMu.RLock()
	defer h.metricsMu.RUnlock()
	return h.metrics
}

// SetStatusChangeCallback sets a callback for status changes
func (h *HealthMonitor) SetStatusChangeCallback(cb func(oldStatus, newStatus HealthStatus)) {
	h.onStatusChange = cb
}

// SetUnhealthyCallback sets a callback for when the connection becomes unhealthy
func (h *HealthMonitor) SetUnhealthyCallback(cb func()) {
	h.onUnhealthy = cb
}

// SetHealthyCallback sets a callback for when the connection becomes healthy
func (h *HealthMonitor) SetHealthyCallback(cb func()) {
	h.onHealthy = cb
}

func (h *HealthMonitor) monitorLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// Initial check
	h.checkHealth()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.checkHealth()
		}
	}
}

func (h *HealthMonitor) checkHealth() {
	h.lastCheckTime = time.Now()

	// Check connection state
	connState := h.conn.GetState()

	// Perform health check
	ctx, cancel := context.WithTimeout(h.ctx, h.timeout)
	defer cancel()

	start := time.Now()
	resp, err := h.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "", // Empty string checks overall server health
	})
	pingDuration := time.Since(start)

	// Update metrics
	h.metricsMu.Lock()
	h.metrics.LastPingTime = start
	h.metrics.LastPingDuration = pingDuration
	h.metrics.PingCount++

	// Update average ping time
	if h.metrics.PingCount == 1 {
		h.metrics.AveragePingTime = pingDuration
	} else {
		h.metrics.AveragePingTime = (h.metrics.AveragePingTime*time.Duration(h.metrics.PingCount-1) + pingDuration) / time.Duration(h.metrics.PingCount)
	}
	h.metricsMu.Unlock()

	// Determine new status
	newStatus := h.determineStatus(connState, resp, err)

	// Update status if changed
	h.statusMu.Lock()
	oldStatus := h.status
	if oldStatus != newStatus {
		h.status = newStatus
		h.statusMu.Unlock()

		// Call callbacks
		if h.onStatusChange != nil {
			h.onStatusChange(oldStatus, newStatus)
		}

		switch newStatus {
		case HealthStatusHealthy:
			if h.onHealthy != nil {
				h.onHealthy()
			}
		case HealthStatusUnhealthy:
			if h.onUnhealthy != nil {
				h.onUnhealthy()
			}
		}
	} else {
		h.statusMu.Unlock()
	}

	// Update error metrics if there was an error
	if err != nil {
		h.metricsMu.Lock()
		h.metrics.ConsecutiveErrors++
		h.metrics.LastErrorTime = time.Now()
		h.metrics.TotalErrors++
		h.metricsMu.Unlock()
	} else {
		h.metricsMu.Lock()
		h.metrics.ConsecutiveErrors = 0
		h.metricsMu.Unlock()
	}
}

func (h *HealthMonitor) determineStatus(connState connectivity.State, resp *grpc_health_v1.HealthCheckResponse, err error) HealthStatus {
	// If connection is not ready, it's unhealthy
	if connState != connectivity.Ready {
		return HealthStatusUnhealthy
	}

	// If there was an error checking health, it's degraded or unhealthy
	if err != nil {
		// Check if it's a timeout or connection error
		if h.metrics.ConsecutiveErrors > 2 {
			return HealthStatusUnhealthy
		}
		return HealthStatusDegraded
	}

	// Check health check response status
	switch resp.Status {
	case grpc_health_v1.HealthCheckResponse_SERVING:
		// If ping time is high, mark as degraded
		if h.metrics.LastPingDuration > 500*time.Millisecond {
			return HealthStatusDegraded
		}
		return HealthStatusHealthy
	case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
		return HealthStatusUnhealthy
	case grpc_health_v1.HealthCheckResponse_UNKNOWN:
		return HealthStatusDegraded
	default:
		return HealthStatusUnknown
	}
}

// UpdateMessageMetrics updates message-related metrics
func (h *HealthMonitor) UpdateMessageMetrics(sent, received, dropped int64, avgLatency time.Duration) {
	h.metricsMu.Lock()
	defer h.metricsMu.Unlock()

	h.metrics.MessagesSent += sent
	h.metrics.MessagesReceived += received
	h.metrics.MessagesDropped += dropped
	h.metrics.AverageLatency = avgLatency
	h.metrics.RecordedAt = time.Now()
}

// IsHealthy returns true if the connection is healthy
func (h *HealthMonitor) IsHealthy() bool {
	return h.GetStatus() == HealthStatusHealthy
}

// GetLastCheckTime returns the time of the last health check
func (h *HealthMonitor) GetLastCheckTime() time.Time {
	return h.lastCheckTime
}

// WaitForHealthy blocks until the connection is healthy or context is cancelled
func (h *HealthMonitor) WaitForHealthy(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if h.IsHealthy() {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ConnectionHealthReport provides a comprehensive health report
type ConnectionHealthReport struct {
	Status          HealthStatus
	ConnectionState connectivity.State
	Metrics         HealthMetrics
	LastCheckTime   time.Time
	Recommendations []string
}

// GenerateReport generates a comprehensive health report
func (h *HealthMonitor) GenerateReport() ConnectionHealthReport {
	h.statusMu.RLock()
	status := h.status
	h.statusMu.RUnlock()

	h.metricsMu.RLock()
	metrics := h.metrics
	h.metricsMu.RUnlock()

	report := ConnectionHealthReport{
		Status:          status,
		ConnectionState: h.conn.GetState(),
		Metrics:         metrics,
		LastCheckTime:   h.lastCheckTime,
		Recommendations: make([]string, 0),
	}

	// Generate recommendations based on health
	switch status {
	case HealthStatusUnhealthy:
		report.Recommendations = append(report.Recommendations,
			"Connection is unhealthy - consider reconnecting",
			"Check network connectivity to the server",
		)
	case HealthStatusDegraded:
		report.Recommendations = append(report.Recommendations,
			"Connection is degraded - monitor closely",
			"High latency detected - check network conditions",
		)
	}

	if metrics.ConsecutiveErrors > 5 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Multiple consecutive errors (%d) - may need reconnection", metrics.ConsecutiveErrors),
		)
	}

	if metrics.MessagesDropped > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Messages have been dropped (%d) - check flow control", metrics.MessagesDropped),
		)
	}

	return report
}

// String returns a string representation of the health report
func (r ConnectionHealthReport) String() string {
	return fmt.Sprintf(
		"Health Report:\n"+
			"  Status: %s\n"+
			"  Connection State: %s\n"+
			"  Last Check: %s\n"+
			"  Ping Count: %d (failed: %d)\n"+
			"  Messages: sent=%d, received=%d, dropped=%d\n"+
			"  Errors: consecutive=%d, total=%d",
		r.Status,
		r.ConnectionState,
		r.LastCheckTime.Format(time.RFC3339),
		r.Metrics.PingCount,
		r.Metrics.FailedPingCount,
		r.Metrics.MessagesSent,
		r.Metrics.MessagesReceived,
		r.Metrics.MessagesDropped,
		r.Metrics.ConsecutiveErrors,
		r.Metrics.TotalErrors,
	)
}
