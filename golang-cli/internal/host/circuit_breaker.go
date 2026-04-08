// Package host provides gRPC client functionality for communicating with the Cline extension.
package host

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int32

const (
	// CircuitStateClosed means requests flow normally
	CircuitStateClosed CircuitState = iota
	// CircuitStateOpen means requests are blocked
	CircuitStateOpen
	// CircuitStateHalfOpen means testing if service recovered
	CircuitStateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitStateClosed:
		return "closed"
	case CircuitStateOpen:
		return "open"
	case CircuitStateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig contains configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// ResetTimeout is the duration to wait before trying again (half-open)
	ResetTimeout time.Duration
	// HalfOpenMaxRequests is the number of requests to test in half-open state
	HalfOpenMaxRequests int
}

// DefaultCircuitBreakerConfig returns a default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		ResetTimeout:        30 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern for gRPC connections
type CircuitBreaker struct {
	config CircuitBreakerConfig

	// State
	state atomic.Int32

	// Metrics
	failures    atomic.Int32
	successes   atomic.Int32
	lastFailure atomic.Value // time.Time

	// Half-open tracking
	halfOpenRequests atomic.Int32

	// Callbacks
	onStateChange func(CircuitState)
	onFailure     func(error)

	// Synchronization
	mu sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultCircuitBreakerConfig().FailureThreshold
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = DefaultCircuitBreakerConfig().ResetTimeout
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = DefaultCircuitBreakerConfig().HalfOpenMaxRequests
	}

	cb := &CircuitBreaker{
		config: config,
	}
	cb.state.Store(int32(CircuitStateClosed))
	return cb
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn()

	if err != nil {
		cb.RecordFailure(err)
	} else {
		cb.RecordSuccess()
	}

	return err
}

// CanExecute returns true if the circuit allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	state := cb.getState()

	switch state {
	case CircuitStateClosed:
		return true

	case CircuitStateOpen:
		// Check if reset timeout has elapsed
		lastFailure, ok := cb.lastFailure.Load().(time.Time)
		if !ok {
			return false
		}

		if time.Since(lastFailure) > cb.config.ResetTimeout {
			// Transition to half-open
			cb.transitionTo(CircuitStateHalfOpen)
			cb.halfOpenRequests.Store(0)
			return true
		}
		return false

	case CircuitStateHalfOpen:
		// Allow limited requests in half-open state
		count := cb.halfOpenRequests.Add(1)
		return count <= int32(cb.config.HalfOpenMaxRequests)

	default:
		return false
	}
}

// RecordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) RecordFailure(err error) {
	cb.lastFailure.Store(time.Now())

	state := cb.getState()

	switch state {
	case CircuitStateClosed:
		failures := cb.failures.Add(1)
		if int(failures) >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitStateOpen)
		}

	case CircuitStateHalfOpen:
		// Any failure in half-open immediately opens the circuit
		cb.transitionTo(CircuitStateOpen)
		cb.halfOpenRequests.Store(0)
	}

	if cb.onFailure != nil {
		cb.onFailure(err)
	}
}

// RecordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) RecordSuccess() {
	state := cb.getState()

	switch state {
	case CircuitStateClosed:
		// Reset failure count on success
		cb.failures.Store(0)

	case CircuitStateHalfOpen:
		successes := cb.successes.Add(1)
		// If all half-open requests succeed, close the circuit
		if int(successes) >= cb.config.HalfOpenMaxRequests {
			cb.transitionTo(CircuitStateClosed)
			cb.failures.Store(0)
			cb.successes.Store(0)
			cb.halfOpenRequests.Store(0)
		}
	}
}

// GetState returns the current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.getState()
}

// IsOpen returns true if the circuit is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.getState() == CircuitStateOpen
}

// IsClosed returns true if the circuit is closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.getState() == CircuitStateClosed
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.halfOpenRequests.Store(0)
	cb.transitionTo(CircuitStateClosed)
}

// SetOnStateChange sets the state change callback
func (cb *CircuitBreaker) SetOnStateChange(fn func(CircuitState)) {
	cb.onStateChange = fn
}

// SetOnFailure sets the failure callback
func (cb *CircuitBreaker) SetOnFailure(fn func(error)) {
	cb.onFailure = fn
}

// GetMetrics returns current metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		State:             cb.getState(),
		Failures:          int(cb.failures.Load()),
		Successes:         int(cb.successes.Load()),
		HalfOpenRequests:  int(cb.halfOpenRequests.Load()),
		LastFailureTime:   cb.getLastFailureTime(),
	}
}

// CircuitBreakerMetrics contains circuit breaker metrics
type CircuitBreakerMetrics struct {
	State             CircuitState
	Failures          int
	Successes         int
	HalfOpenRequests  int
	LastFailureTime   time.Time
}

func (cb *CircuitBreaker) getState() CircuitState {
	return CircuitState(cb.state.Load())
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.getState()
	if oldState == newState {
		return
	}

	cb.state.Store(int32(newState))

	if cb.onStateChange != nil {
		cb.onStateChange(newState)
	}
}

func (cb *CircuitBreaker) getLastFailureTime() time.Time {
	t, _ := cb.lastFailure.Load().(time.Time)
	return t
}

// CircuitBreakerWrapper wraps a function with circuit breaker protection
type CircuitBreakerWrapper struct {
	breaker *CircuitBreaker
}

// NewCircuitBreakerWrapper creates a new wrapper
func NewCircuitBreakerWrapper(config CircuitBreakerConfig) *CircuitBreakerWrapper {
	return &CircuitBreakerWrapper{
		breaker: NewCircuitBreaker(config),
	}
}

// Wrap wraps a function with circuit breaker protection
func (w *CircuitBreakerWrapper) Wrap(fn func() error) func() error {
	return func() error {
		return w.breaker.Execute(context.Background(), fn)
	}
}

// GetBreaker returns the underlying circuit breaker
func (w *CircuitBreakerWrapper) GetBreaker() *CircuitBreaker {
	return w.breaker
}