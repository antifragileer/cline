// Package host provides gRPC client functionality for communicating with the Cline extension.
package host

import (
	"context"
	"sync"
	"time"
)

// RequestKey uniquely identifies a request for coalescing
type RequestKey string

// PendingRequest represents a request that is currently in-flight
type PendingRequest struct {
	// Channel for waiting callers to receive the result
	done chan struct{}

	// Result of the request
	result interface{}
	err    error

	// Number of waiters
	waiterCount int32

	// Timestamp when request was started
	startTime time.Time
}

// RequestCoalescer deduplicates concurrent identical requests
type RequestCoalescer struct {
	mu sync.RWMutex

	// pending tracks in-flight requests
	pending map[RequestKey]*PendingRequest

	// config
	config RequestCoalescerConfig
}

// RequestCoalescerConfig contains configuration for request coalescing
type RequestCoalescerConfig struct {
	// RequestTimeout is the maximum time to wait for a request
	RequestTimeout time.Duration

	// CleanupInterval is how often to clean up stale pending requests
	CleanupInterval time.Duration

	// MaxPendingAge is the maximum age of a pending request before cleanup
	MaxPendingAge time.Duration
}

// DefaultRequestCoalescerConfig returns a default configuration
func DefaultRequestCoalescerConfig() RequestCoalescerConfig {
	return RequestCoalescerConfig{
		RequestTimeout:  30 * time.Second,
		CleanupInterval: 5 * time.Minute,
		MaxPendingAge:   10 * time.Minute,
	}
}

// NewRequestCoalescer creates a new request coalescer
func NewRequestCoalescer(config RequestCoalescerConfig) *RequestCoalescer {
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = DefaultRequestCoalescerConfig().RequestTimeout
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = DefaultRequestCoalescerConfig().CleanupInterval
	}
	if config.MaxPendingAge <= 0 {
		config.MaxPendingAge = DefaultRequestCoalescerConfig().MaxPendingAge
	}

	rc := &RequestCoalescer{
		pending: make(map[RequestKey]*PendingRequest),
		config:  config,
	}

	// Start cleanup goroutine
	go rc.cleanupLoop()

	return rc
}

// Do executes the given function, coalescing identical concurrent requests
// The key identifies the request for deduplication
func (rc *RequestCoalescer) Do(ctx context.Context, key RequestKey, fn func() (interface{}, error)) (interface{}, error) {
	// Fast path: check if there's already a pending request
	rc.mu.RLock()
	pending, exists := rc.pending[key]
	rc.mu.RUnlock()

	if exists {
		// Wait for the existing request to complete
		select {
		case <-pending.done:
			return pending.result, pending.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Slow path: we might need to create a new request
	rc.mu.Lock()

	// Double-check after acquiring write lock
	if pending, exists := rc.pending[key]; exists {
		rc.mu.Unlock()
		select {
		case <-pending.done:
			return pending.result, pending.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Create new pending request
	pending = &PendingRequest{
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
	rc.pending[key] = pending
	rc.mu.Unlock()

	// Execute the request
	go rc.executeRequest(key, pending, fn)

	// Wait for result
	select {
	case <-pending.done:
		return pending.result, pending.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// DoAsync executes the function asynchronously without waiting
func (rc *RequestCoalescer) DoAsync(key RequestKey, fn func() (interface{}, error)) <-chan struct{} {
	rc.mu.RLock()
	pending, exists := rc.pending[key]
	rc.mu.RUnlock()

	if exists {
		return pending.done
	}

	rc.mu.Lock()
	if pending, exists := rc.pending[key]; exists {
		rc.mu.Unlock()
		return pending.done
	}

	pending = &PendingRequest{
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
	rc.pending[key] = pending
	rc.mu.Unlock()

	go rc.executeRequest(key, pending, fn)

	return pending.done
}

// executeRequest runs the function and broadcasts the result
func (rc *RequestCoalescer) executeRequest(key RequestKey, pending *PendingRequest, fn func() (interface{}, error)) {
	// Execute the function
	pending.result, pending.err = fn()

	// Broadcast completion
	close(pending.done)

	// Remove from pending after a short delay to allow stragglers to get the result
	time.AfterFunc(100*time.Millisecond, func() {
		rc.mu.Lock()
		// Only remove if it's still our request (not replaced by a new one)
		if current, exists := rc.pending[key]; exists && current == pending {
			delete(rc.pending, key)
		}
		rc.mu.Unlock()
	})
}

// GetPendingCount returns the number of currently pending requests
func (rc *RequestCoalescer) GetPendingCount() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.pending)
}

// cleanupLoop periodically removes stale pending requests
func (rc *RequestCoalescer) cleanupLoop() {
	ticker := time.NewTicker(rc.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rc.cleanup()
	}
}

// cleanup removes stale pending requests
func (rc *RequestCoalescer) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	for key, pending := range rc.pending {
		// Check if request is too old
		if now.Sub(pending.startTime) > rc.config.MaxPendingAge {
			// Only remove if done (don't interrupt active requests)
			select {
			case <-pending.done:
				delete(rc.pending, key)
			default:
				// Still active, check if it should be timed out
				if now.Sub(pending.startTime) > rc.config.RequestTimeout*2 {
					// Force remove very old requests
					delete(rc.pending, key)
				}
			}
		}
	}
}

// CoalescedCall represents a coalesced function call
type CoalescedCall struct {
	coalescer *RequestCoalescer
	key       RequestKey
	fn        func() (interface{}, error)
}

// NewCoalescedCall creates a new coalesced call
func NewCoalescedCall(coalescer *RequestCoalescer, key RequestKey, fn func() (interface{}, error)) *CoalescedCall {
	return &CoalescedCall{
		coalescer: coalescer,
		key:       key,
		fn:        fn,
	}
}

// Execute runs the coalesced call
func (cc *CoalescedCall) Execute(ctx context.Context) (interface{}, error) {
	return cc.coalescer.Do(ctx, cc.key, cc.fn)
}

// StreamRequestCoalescer is specialized for coalescing stream requests
type StreamRequestCoalescer struct {
	*RequestCoalescer
	streamKeys sync.Map // map[streamID]RequestKey
}

// NewStreamRequestCoalescer creates a new stream request coalescer
func NewStreamRequestCoalescer(config RequestCoalescerConfig) *StreamRequestCoalescer {
	return &StreamRequestCoalescer{
		RequestCoalescer: NewRequestCoalescer(config),
	}
}

// CoalesceStream coalesces stream establishment requests
func (src *StreamRequestCoalescer) CoalesceStream(streamID string, fn func() (interface{}, error)) (interface{}, error) {
	key := RequestKey("stream:" + streamID)
	src.streamKeys.Store(streamID, key)
	return src.Do(context.Background(), key, fn)
}

// InvalidateStream removes a stream from coalescing
func (src *StreamRequestCoalescer) InvalidateStream(streamID string) {
	key := RequestKey("stream:" + streamID)
	src.streamKeys.Delete(streamID)
	src.mu.Lock()
	delete(src.pending, key)
	src.mu.Unlock()
}

// ResponseCache provides simple response caching for idempotent requests
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[RequestKey]*CacheEntry
	config  ResponseCacheConfig
}

// CacheEntry represents a cached response
type CacheEntry struct {
	result    interface{}
	err       error
	timestamp time.Time
	ttl       time.Duration
}

// ResponseCacheConfig contains cache configuration
type ResponseCacheConfig struct {
	DefaultTTL    time.Duration
	MaxEntries    int
	CleanupInterval time.Duration
}

// DefaultResponseCacheConfig returns default cache configuration
func DefaultResponseCacheConfig() ResponseCacheConfig {
	return ResponseCacheConfig{
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewResponseCache creates a new response cache
func NewResponseCache(config ResponseCacheConfig) *ResponseCache {
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = DefaultResponseCacheConfig().DefaultTTL
	}
	if config.MaxEntries <= 0 {
		config.MaxEntries = DefaultResponseCacheConfig().MaxEntries
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = DefaultResponseCacheConfig().CleanupInterval
	}

	rc := &ResponseCache{
		entries: make(map[RequestKey]*CacheEntry),
		config:  config,
	}

	go rc.cleanupLoop()

	return rc
}

// Get retrieves a cached response if available and not expired
func (rc *ResponseCache) Get(key RequestKey) (interface{}, error, bool) {
	rc.mu.RLock()
	entry, exists := rc.entries[key]
	rc.mu.RUnlock()

	if !exists {
		return nil, nil, false
	}

	// Check if expired
	if time.Since(entry.timestamp) > entry.ttl {
		rc.mu.Lock()
		delete(rc.entries, key)
		rc.mu.Unlock()
		return nil, nil, false
	}

	return entry.result, entry.err, true
}

// Set stores a response in the cache
func (rc *ResponseCache) Set(key RequestKey, result interface{}, err error, ttl time.Duration) {
	if ttl <= 0 {
		ttl = rc.config.DefaultTTL
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Evict oldest entries if at capacity
	if len(rc.entries) >= rc.config.MaxEntries {
		rc.evictOldest(100) // Evict 10% of entries
	}

	rc.entries[key] = &CacheEntry{
		result:    result,
		err:       err,
		timestamp: time.Now(),
		ttl:       ttl,
	}
}

// Invalidate removes an entry from the cache
func (rc *ResponseCache) Invalidate(key RequestKey) {
	rc.mu.Lock()
	delete(rc.entries, key)
	rc.mu.Unlock()
}

// Clear removes all entries from the cache
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	rc.entries = make(map[RequestKey]*CacheEntry)
	rc.mu.Unlock()
}

// evictOldest removes the oldest N entries
func (rc *ResponseCache) evictOldest(count int) {
	type kv struct {
		key   RequestKey
		entry *CacheEntry
	}

	// Collect all entries
	var allEntries []kv
	for k, v := range rc.entries {
		allEntries = append(allEntries, kv{k, v})
	}

	// Sort by timestamp (oldest first) - simple bubble sort for small N
	for i := 0; i < len(allEntries) && i < count; i++ {
		oldestIdx := i
		for j := i + 1; j < len(allEntries); j++ {
			if allEntries[j].entry.timestamp.Before(allEntries[oldestIdx].entry.timestamp) {
				oldestIdx = j
			}
		}
		if oldestIdx != i {
			allEntries[i], allEntries[oldestIdx] = allEntries[oldestIdx], allEntries[i]
		}
	}

	// Remove oldest entries
	for i := 0; i < count && i < len(allEntries); i++ {
		delete(rc.entries, allEntries[i].key)
	}
}

// cleanupLoop periodically removes expired entries
func (rc *ResponseCache) cleanupLoop() {
	ticker := time.NewTicker(rc.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rc.cleanup()
	}
}

// cleanup removes expired entries
func (rc *ResponseCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	for key, entry := range rc.entries {
		if now.Sub(entry.timestamp) > entry.ttl {
			delete(rc.entries, key)
		}
	}
}