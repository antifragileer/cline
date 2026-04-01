// Package grpc provides health check functionality for monitoring gRPC connection state.
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/connectivity"
)

// HealthChecker monitors the health of a gRPC connection
type HealthChecker struct {
	client        *Client
	checkInterval time.Duration
	timeout       time.Duration
	onUnhealthy   func(error)
	onHealthy     func()
	stopChan      chan struct{}
	running       bool
}

// NewHealthChecker creates a new health checker for the given client
func NewHealthChecker(client *Client, checkInterval, timeout time.Duration) *HealthChecker {
	if checkInterval <= 0 {
		checkInterval = 10 * time.Second
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &HealthChecker{
		client:        client,
		checkInterval: checkInterval,
		timeout:       timeout,
		stopChan:      make(chan struct{}),
	}
}

// SetCallbacks sets the health change callbacks
func (h *HealthChecker) SetCallbacks(onUnhealthy func(error), onHealthy func()) {
	h.onUnhealthy = onUnhealthy
	h.onHealthy = onHealthy
}

// Start begins health check polling
func (h *HealthChecker) Start() {
	if h.running {
		return
	}
	h.running = true

	go h.checkLoop()
}

// Stop stops health check polling
func (h *HealthChecker) Stop() {
	if !h.running {
		return
	}
	h.running = false
	close(h.stopChan)
}

// checkLoop runs the health check loop
func (h *HealthChecker) checkLoop() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// Initial check
	h.performCheck()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.performCheck()
		}
	}
}

// performCheck performs a single health check
func (h *HealthChecker) performCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	state := h.client.GetState()

	switch state {
	case connectivity.Ready:
		if h.onHealthy != nil {
			h.onHealthy()
		}
	case connectivity.Idle, connectivity.Connecting:
		// Transient states, wait for next check
	case connectivity.TransientFailure, connectivity.Shutdown:
		if h.onUnhealthy != nil {
			h.onUnhealthy(context.Canceled)
		}
	}

	// Try to reconnect if not ready
	if state != connectivity.Ready && state != connectivity.Shutdown {
		if err := h.client.Connect(); err != nil {
			if h.onUnhealthy != nil {
				h.onUnhealthy(err)
			}
		}
	}

	_ = ctx
}

// IsRunning returns true if the health checker is running
func (h *HealthChecker) IsRunning() bool {
	return h.running
}

// ConnectionState provides a comprehensive view of connection health
type ConnectionState struct {
	Connected       bool
	State           connectivity.State
	RetryCount      int
	LastHealthCheck time.Time
	Latency         time.Duration
}

// GetConnectionState returns detailed connection state information
func (h *HealthChecker) GetConnectionState() ConnectionState {
	state := ConnectionState{
		Connected:       h.client.IsConnected(),
		State:           h.client.GetState(),
		RetryCount:      h.client.RetryCount(),
		LastHealthCheck: time.Now(),
	}

	// Measure latency with a quick check
	start := time.Now()
	if conn, err := h.client.GetConnection(); err == nil && conn != nil {
		state.Latency = time.Since(start)
	}

	return state
}