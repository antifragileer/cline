// Package task provides tool approver implementations for the Cline CLI.
package task

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AutoApprover automatically approves all tool requests.
type AutoApprover struct{}

// NewAutoApprover creates a new auto-approver that approves all requests.
func NewAutoApprover() *AutoApprover {
	return &AutoApprover{}
}

// RequestApproval always returns true (auto-approve).
func (a *AutoApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return true, nil
}

// DisplayToolRequest is a no-op for auto-approver.
func (a *AutoApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}

// RejectAllApprover rejects all tool requests.
type RejectAllApprover struct{}

// NewRejectAllApprover creates a new reject-all approver.
func NewRejectAllApprover() *RejectAllApprover {
	return &RejectAllApprover{}
}

// RequestApproval always returns false (reject all).
func (r *RejectAllApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return false, nil
}

// DisplayToolRequest is a no-op for reject-all approver.
func (r *RejectAllApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}

// ConditionalApprover approves based on a condition function.
type ConditionalApprover struct {
	condition func(req ToolRequest) bool
}

// NewConditionalApprover creates a new conditional approver.
func NewConditionalApprover(condition func(req ToolRequest) bool) *ConditionalApprover {
	return &ConditionalApprover{
		condition: condition,
	}
}

// RequestApproval returns true if the condition is met.
func (c *ConditionalApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return c.condition(req), nil
}

// DisplayToolRequest is a no-op for conditional approver.
func (c *ConditionalApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}

// DelegatingApprover delegates approval to different approvers based on tool type.
type DelegatingApprover struct {
	defaultApprover ToolApprover
	approvers       map[ToolType]ToolApprover
	mu              sync.RWMutex
}

// NewDelegatingApprover creates a new delegating approver.
func NewDelegatingApprover(defaultApprover ToolApprover) *DelegatingApprover {
	return &DelegatingApprover{
		defaultApprover: defaultApprover,
		approvers:       make(map[ToolType]ToolApprover),
	}
}

// RegisterApprover registers an approver for a specific tool type.
func (d *DelegatingApprover) RegisterApprover(toolType ToolType, approver ToolApprover) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.approvers[toolType] = approver
}

// RequestApproval delegates to the appropriate approver based on tool type.
func (d *DelegatingApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if approver, ok := d.approvers[req.Type]; ok {
		return approver.RequestApproval(ctx, req)
	}

	return d.defaultApprover.RequestApproval(ctx, req)
}

// DisplayToolRequest displays the tool request.
func (d *DelegatingApprover) DisplayToolRequest(req ToolRequest) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if approver, ok := d.approvers[req.Type]; ok {
		return approver.DisplayToolRequest(req)
	}

	return d.defaultApprover.DisplayToolRequest(req)
}

// LoggingApprover wraps another approver and logs approval decisions.
type LoggingApprover struct {
	inner  ToolApprover
	logger func(string)
}

// NewLoggingApprover creates a new logging approver wrapper.
func NewLoggingApprover(inner ToolApprover, logger func(string)) *LoggingApprover {
	return &LoggingApprover{
		inner:  inner,
		logger: logger,
	}
}

// RequestApproval logs and delegates to the inner approver.
func (l *LoggingApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	l.logger(fmt.Sprintf("Approval requested for %s (type: %s)", req.ToolName, req.Type))

	approved, err := l.inner.RequestApproval(ctx, req)
	
	status := "rejected"
	if approved {
		status = "approved"
	}
	if err != nil {
		status = fmt.Sprintf("error: %v", err)
	}
	
	l.logger(fmt.Sprintf("Tool %s: %s", req.ToolName, status))
	
	return approved, err
}

// DisplayToolRequest delegates to the inner approver.
func (l *LoggingApprover) DisplayToolRequest(req ToolRequest) error {
	return l.inner.DisplayToolRequest(req)
}

// TimeoutApprover wraps an approver with a timeout.
type TimeoutApprover struct {
	inner   ToolApprover
	timeout time.Duration
}

// NewTimeoutApprover creates a new timeout approver wrapper.
func NewTimeoutApprover(inner ToolApprover, timeout time.Duration) *TimeoutApprover {
	return &TimeoutApprover{
		inner:   inner,
		timeout: timeout,
	}
}

// RequestApproval applies a timeout to the approval request.
func (t *TimeoutApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	return t.inner.RequestApproval(timeoutCtx, req)
}

// DisplayToolRequest delegates to the inner approver.
func (t *TimeoutApprover) DisplayToolRequest(req ToolRequest) error {
	return t.inner.DisplayToolRequest(req)
}

// BatchApprover handles batch approval of multiple tools.
type BatchApprover struct {
	inner        ToolApprover
	batchSize    int
	autoApprove  bool
	pendingCount int
	mu           sync.Mutex
}

// NewBatchApprover creates a new batch approver.
func NewBatchApprover(inner ToolApprover, batchSize int) *BatchApprover {
	return &BatchApprover{
		inner:     inner,
		batchSize: batchSize,
	}
}

// SetAutoApprove enables or disables auto-approval for batch operations.
func (b *BatchApprover) SetAutoApprove(autoApprove bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.autoApprove = autoApprove
}

// RequestApproval handles approval, potentially in batch mode.
func (b *BatchApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	b.mu.Lock()
	autoApprove := b.autoApprove
	b.pendingCount++
	b.mu.Unlock()

	if autoApprove {
		return true, nil
	}

	return b.inner.RequestApproval(ctx, req)
}

// DisplayToolRequest delegates to the inner approver.
func (b *BatchApprover) DisplayToolRequest(req ToolRequest) error {
	return b.inner.DisplayToolRequest(req)
}

// NonInteractiveApprover is an approver for non-interactive environments.
type NonInteractiveApprover struct {
	defaultApproval bool
}

// NewNonInteractiveApprover creates a new non-interactive approver.
// It defaults to rejecting all requests unless configured otherwise.
func NewNonInteractiveApprover() *NonInteractiveApprover {
	return &NonInteractiveApprover{
		defaultApproval: false,
	}
}

// SetDefaultApproval sets the default approval decision.
func (n *NonInteractiveApprover) SetDefaultApproval(approve bool) {
	n.defaultApproval = approve
}

// RequestApproval returns the default approval decision.
func (n *NonInteractiveApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return n.defaultApproval, nil
}

// DisplayToolRequest is a no-op for non-interactive approver.
func (n *NonInteractiveApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}