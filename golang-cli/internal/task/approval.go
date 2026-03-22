// Package task provides task execution and tool management functionality.
package task

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UIToolApprover implements the ToolApprover interface using terminal UI prompts.
type UIToolApprover struct {
	output          *os.File
	input           *os.File
	useInteractive  bool
	defaultTimeout  time.Duration
}

// NewUIToolApprover creates a new UI-based tool approver.
func NewUIToolApprover() *UIToolApprover {
	return &UIToolApprover{
		output:         os.Stdout,
		input:          os.Stdin,
		useInteractive: true,
		defaultTimeout: 5 * time.Minute,
	}
}

// NewNonInteractiveApprover creates a non-interactive approver that auto-rejects.
func NewNonInteractiveApprover() *UIToolApprover {
	return &UIToolApprover{
		output:         os.Stdout,
		input:          os.Stdin,
		useInteractive: false,
		defaultTimeout: 5 * time.Minute,
	}
}

// SetInteractive enables or disables interactive prompts.
func (a *UIToolApprover) SetInteractive(interactive bool) {
	a.useInteractive = interactive
}

// SetOutput sets the output file for prompts.
func (a *UIToolApprover) SetOutput(output *os.File) {
	a.output = output
}

// SetInput sets the input file for prompts.
func (a *UIToolApprover) SetInput(input *os.File) {
	a.input = input
}

// SetTimeout sets the default approval timeout.
func (a *UIToolApprover) SetTimeout(timeout time.Duration) {
	a.defaultTimeout = timeout
}

// RequestApproval requests approval for a tool using an interactive prompt.
func (a *UIToolApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	if !a.useInteractive {
		return false, fmt.Errorf("non-interactive mode: approval rejected")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// Create the approval model
	model := NewToolApprovalModel(req, a.defaultTimeout)

	// Run the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithInput(a.input),
		tea.WithOutput(a.output),
	)

	// Run in a goroutine to handle context cancellation
	resultChan := make(chan approvalResult, 1)
	go func() {
		m, err := p.Run()
		if err != nil {
			resultChan <- approvalResult{approved: false, err: err}
			return
		}

		am, ok := m.(*ToolApprovalModel)
		if !ok {
			resultChan <- approvalResult{approved: false, err: fmt.Errorf("unexpected model type")}
			return
		}

		resultChan <- approvalResult{
			approved: am.IsApproved(),
			err:      nil,
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return result.approved, result.err
	case <-ctx.Done():
		p.Quit()
		return false, ctx.Err()
	}
}

// approvalResult represents the result of an approval request.
type approvalResult struct {
	approved bool
	err      error
}

// DisplayToolRequest displays the tool request to the user without requesting approval.
func (a *UIToolApprover) DisplayToolRequest(req ToolRequest) error {
	if !a.useInteractive {
		return nil
	}

	display := formatToolRequestForDisplay(req)
	_, err := fmt.Fprintln(a.output, display)
	return err
}

// ToolApprovalModel is the Bubble Tea model for tool approval prompts.
type ToolApprovalModel struct {
	request       ToolRequest
	timeout       time.Duration
	width         int
	height        int
	approved      bool
	rejected      bool
	quitting      bool
	showDetails   bool
	countdown     time.Duration
	startTime     time.Time
}

// NewToolApprovalModel creates a new approval model for the given request.
func NewToolApprovalModel(req ToolRequest, timeout time.Duration) *ToolApprovalModel {
	return &ToolApprovalModel{
		request:   req,
		timeout:   timeout,
		startTime: time.Now(),
		countdown: timeout,
	}
}

// Init initializes the approval model.
func (m *ToolApprovalModel) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
		waitForKey(),
	)
}

// tick returns a command that ticks every second for the countdown.
func (m *ToolApprovalModel) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// waitForKey returns a command that waits for any key press.
func waitForKey() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return keyCheckMsg{}
	})
}

// tickMsg is sent on every tick.
type tickMsg time.Time

// keyCheckMsg is sent periodically to check for key input.
type keyCheckMsg struct{}

// Update handles messages and updates the approval model.
func (m *ToolApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.approved = true
			m.quitting = true
			return m, tea.Quit

		case "n", "N", "esc":
			m.rejected = true
			m.quitting = true
			return m, tea.Quit

		case "q", "ctrl+c":
			m.rejected = true
			m.quitting = true
			return m, tea.Quit

		case "d", "?":
			m.showDetails = !m.showDetails

		case "a":
			// Approve all (yolo mode hint)
			m.approved = true
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		// Update countdown
		elapsed := time.Since(m.startTime)
		m.countdown = m.timeout - elapsed

		if m.countdown <= 0 {
			// Timeout - auto-reject
			m.rejected = true
			m.quitting = true
			return m, tea.Quit
		}

		return m, m.tick()
	}

	return m, nil
}

// View renders the approval prompt.
func (m *ToolApprovalModel) View() string {
	if m.quitting {
		if m.approved {
			return renderApprovalStatus("✓ Approved", lipgloss.Color("42"))
		}
		return renderApprovalStatus("✗ Rejected", lipgloss.Color("196"))
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7D56F4")).
		MarginBottom(1)
	b.WriteString(headerStyle.Render("🔧 Tool Approval Request"))
	b.WriteString("\n\n")

	// Tool info
	toolStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))
	b.WriteString(toolStyle.Render(fmt.Sprintf("Tool: %s", m.request.ToolName)))
	b.WriteString("\n")

	// Description
	if m.request.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			MarginTop(1)
		b.WriteString(descStyle.Render(m.request.Description))
		b.WriteString("\n")
	}

	// Parameters summary
	paramsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(1)
	b.WriteString(paramsStyle.Render(m.formatParameters()))
	b.WriteString("\n")

	// Details section (if expanded)
	if m.showDetails && len(m.request.Parameters) > 0 {
		b.WriteString(m.renderDetails())
		b.WriteString("\n")
	}

	// Countdown warning
	if m.countdown < 30*time.Second {
		countdownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		b.WriteString(countdownStyle.Render(fmt.Sprintf("⏱  Timeout in: %v", m.countdown.Round(time.Second))))
		b.WriteString("\n\n")
	}

	// Action buttons
	b.WriteString(m.renderButtons())
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
	b.WriteString(helpStyle.Render("[Y]es  [N]o  [D]etails  [A]pprove All  [Q]uit"))

	return b.String()
}

// renderButtons renders the approval buttons.
func (m *ToolApprovalModel) renderButtons() string {
	yesStyle := lipgloss.NewStyle().
		Padding(0, 3).
		MarginRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Foreground(lipgloss.Color("42"))

	noStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Foreground(lipgloss.Color("196"))

	return lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("Yes"),
		noStyle.Render("No"),
	)
}

// renderDetails renders the detailed parameters view.
func (m *ToolApprovalModel) renderDetails() string {
	var b strings.Builder

	detailsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(m.width - 4)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	b.WriteString("Parameters:\n")
	for key, value := range m.request.Parameters {
		valueStr := fmt.Sprintf("%v", value)
		// Truncate long values
		if len(valueStr) > 200 {
			valueStr = valueStr[:200] + "..."
		}
		line := fmt.Sprintf("  %s: %s", key, valueStr)
		b.WriteString(contentStyle.Render(line))
		b.WriteString("\n")
	}

	return detailsStyle.Render(b.String())
}

// formatParameters formats the parameters for display.
func (m *ToolApprovalModel) formatParameters() string {
	var parts []string

	// Show key parameters based on tool type
	switch m.request.Type {
	case ToolTypeReadFile, ToolTypeWriteFile, ToolTypeReplaceInFile:
		if path, ok := m.request.Parameters["path"].(string); ok {
			parts = append(parts, fmt.Sprintf("path: %s", path))
		}

	case ToolTypeExecuteCommand:
		if cmd, ok := m.request.Parameters["command"].(string); ok {
			// Truncate long commands
			if len(cmd) > 60 {
				cmd = cmd[:60] + "..."
			}
			parts = append(parts, fmt.Sprintf("command: %s", cmd))
		}
		if cwd, ok := m.request.Parameters["cwd"].(string); ok && cwd != "" {
			parts = append(parts, fmt.Sprintf("cwd: %s", cwd))
		}

	case ToolTypeSearchFiles:
		if path, ok := m.request.Parameters["path"].(string); ok {
			parts = append(parts, fmt.Sprintf("path: %s", path))
		}
		if regex, ok := m.request.Parameters["regex"].(string); ok {
			parts = append(parts, fmt.Sprintf("pattern: %s", regex))
		}
	}

	if len(parts) == 0 {
		return "No parameters"
	}

	return strings.Join(parts, " | ")
}

// IsApproved returns true if the user approved the tool.
func (m *ToolApprovalModel) IsApproved() bool {
	return m.approved && !m.rejected
}

// IsRejected returns true if the user rejected the tool.
func (m *ToolApprovalModel) IsRejected() bool {
	return m.rejected
}

// renderApprovalStatus renders the final approval status.
func renderApprovalStatus(text string, color lipgloss.Color) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Padding(1, 2)
	return style.Render(text)
}

// formatToolRequestForDisplay formats a tool request for non-interactive display.
func formatToolRequestForDisplay(req ToolRequest) string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7D56F4"))

	b.WriteString(headerStyle.Render(fmt.Sprintf("Tool Request: %s", req.ToolName)))
	b.WriteString("\n")

	if req.Description != "" {
		b.WriteString(req.Description)
		b.WriteString("\n")
	}

	b.WriteString("Parameters:\n")
	for key, value := range req.Parameters {
		b.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
	}

	return b.String()
}

// AutoApprover automatically approves all tool requests.
type AutoApprover struct{}

// NewAutoApprover creates a new auto-approver.
func NewAutoApprover() *AutoApprover {
	return &AutoApprover{}
}

// RequestApproval always returns true (approved).
func (a *AutoApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return true, nil
}

// DisplayToolRequest is a no-op for auto-approver.
func (a *AutoApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}

// RejectAllApprover automatically rejects all tool requests.
type RejectAllApprover struct{}

// NewRejectAllApprover creates a new reject-all approver.
func NewRejectAllApprover() *RejectAllApprover {
	return &RejectAllApprover{}
}

// RequestApproval always returns false (rejected).
func (a *RejectAllApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return false, nil
}

// DisplayToolRequest is a no-op for reject-all approver.
func (a *RejectAllApprover) DisplayToolRequest(req ToolRequest) error {
	return nil
}

// ConditionalApprover approves based on a condition function.
type ConditionalApprover struct {
	condition func(ToolRequest) bool
}

// NewConditionalApprover creates a new conditional approver.
func NewConditionalApprover(condition func(ToolRequest) bool) *ConditionalApprover {
	return &ConditionalApprover{condition: condition}
}

// RequestApproval returns true if the condition is met.
func (a *ConditionalApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	return a.condition(req), nil
}

// DisplayToolRequest displays the request but doesn't wait for input.
func (a *ConditionalApprover) DisplayToolRequest(req ToolRequest) error {
	fmt.Println(formatToolRequestForDisplay(req))
	return nil
}

// DelegatingApprover delegates to different approvers based on tool type.
type DelegatingApprover struct {
	defaultApprover ToolApprover
	approvers       map[ToolType]ToolApprover
}

// NewDelegatingApprover creates a new delegating approver.
func NewDelegatingApprover(defaultApprover ToolApprover) *DelegatingApprover {
	return &DelegatingApprover{
		defaultApprover: defaultApprover,
		approvers:       make(map[ToolType]ToolApprover),
	}
}

// RegisterApprover registers an approver for a specific tool type.
func (a *DelegatingApprover) RegisterApprover(toolType ToolType, approver ToolApprover) {
	a.approvers[toolType] = approver
}

// RequestApproval delegates to the appropriate approver.
func (a *DelegatingApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	approver, ok := a.approvers[req.Type]
	if !ok {
		approver = a.defaultApprover
	}
	return approver.RequestApproval(ctx, req)
}

// DisplayToolRequest delegates to the appropriate approver.
func (a *DelegatingApprover) DisplayToolRequest(req ToolRequest) error {
	approver, ok := a.approvers[req.Type]
	if !ok {
		approver = a.defaultApprover
	}
	return approver.DisplayToolRequest(req)
}

// TimeoutApprover wraps an approver with a timeout.
type TimeoutApprover struct {
	approver ToolApprover
	timeout  time.Duration
}

// NewTimeoutApprover creates a new timeout-wrapped approver.
func NewTimeoutApprover(approver ToolApprover, timeout time.Duration) *TimeoutApprover {
	return &TimeoutApprover{
		approver: approver,
		timeout:  timeout,
	}
}

// RequestApproval requests approval with a timeout.
func (a *TimeoutApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	return a.approver.RequestApproval(ctx, req)
}

// DisplayToolRequest displays the tool request.
func (a *TimeoutApprover) DisplayToolRequest(req ToolRequest) error {
	return a.approver.DisplayToolRequest(req)
}

// LoggingApprover logs all approval requests.
type LoggingApprover struct {
	approver ToolApprover
	logger   func(string)
}

// NewLoggingApprover creates a new logging approver.
func NewLoggingApprover(approver ToolApprover, logger func(string)) *LoggingApprover {
	if logger == nil {
		logger = func(s string) { fmt.Println(s) }
	}
	return &LoggingApprover{
		approver: approver,
		logger:   logger,
	}
}

// RequestApproval logs and delegates.
func (a *LoggingApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	a.logger(fmt.Sprintf("Approval requested for tool: %s (type: %s)", req.ToolName, req.Type))
	
	approved, err := a.approver.RequestApproval(ctx, req)
	
	if err != nil {
		a.logger(fmt.Sprintf("Approval error: %v", err))
	} else if approved {
		a.logger(fmt.Sprintf("Tool approved: %s", req.ToolName))
	} else {
		a.logger(fmt.Sprintf("Tool rejected: %s", req.ToolName))
	}
	
	return approved, err
}

// DisplayToolRequest displays and logs.
func (a *LoggingApprover) DisplayToolRequest(req ToolRequest) error {
	a.logger(fmt.Sprintf("Displaying tool request: %s", req.ToolName))
	return a.approver.DisplayToolRequest(req)
}

// BatchApprover approves multiple tools at once.
type BatchApprover struct {
	uiApprover    *UIToolApprover
	pendingReqs   []ToolRequest
	batchSize     int
	autoApprove   bool
}

// NewBatchApprover creates a new batch approver.
func NewBatchApprover(uiApprover *UIToolApprover, batchSize int) *BatchApprover {
	return &BatchApprover{
		uiApprover:  uiApprover,
		pendingReqs: make([]ToolRequest, 0, batchSize),
		batchSize:   batchSize,
		autoApprove: false,
	}
}

// SetAutoApprove enables or disables auto-approval for the batch.
func (a *BatchApprover) SetAutoApprove(autoApprove bool) {
	a.autoApprove = autoApprove
}

// RequestApproval adds to batch and approves if batch is full.
func (a *BatchApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	a.pendingReqs = append(a.pendingReqs, req)

	// If batch is full or auto-approve is on, approve all
	if a.autoApprove || len(a.pendingReqs) >= a.batchSize {
		return a.approveBatch(ctx)
	}

	// Otherwise, ask for this specific tool
	return a.uiApprover.RequestApproval(ctx, req)
}

// approveBatch approves all pending requests.
func (a *BatchApprover) approveBatch(ctx context.Context) (bool, error) {
	// For simplicity, approve all pending requests
	// In a real implementation, this would show a batch approval UI
	
	fmt.Printf("Approving batch of %d tools\n", len(a.pendingReqs))
	a.pendingReqs = a.pendingReqs[:0] // Clear batch
	
	return true, nil
}

// DisplayToolRequest displays the tool request.
func (a *BatchApprover) DisplayToolRequest(req ToolRequest) error {
	return a.uiApprover.DisplayToolRequest(req)
}

// Flush approves any remaining pending requests.
func (a *BatchApprover) Flush(ctx context.Context) (bool, error) {
	if len(a.pendingReqs) > 0 {
		return a.approveBatch(ctx)
	}
	return true, nil
}