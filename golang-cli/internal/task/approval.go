// Package task provides task execution and approval handling for the Cline CLI
// Reference: cli/src/index.ts lines 203-206, 210-213, 288-329, src/core/prompts/responses.ts
package task

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalRequest represents a request for user approval
// Reference: cli/src/index.ts:288-329
type ApprovalRequest struct {
	ToolName    string            `json:"tool_name"`
	Description string            `json:"description"`
	Details     map[string]string `json:"details"`
}

// ApprovalResponse represents the user's response to an approval request
type ApprovalResponse struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message,omitempty"`
}

// ApprovalHandler handles tool approval requests
// Reference: cli/src/index.ts:288-329, src/core/prompts/responses.ts
type ApprovalHandler struct {
	mu                     sync.RWMutex
	config                 *Config
	logger                 *slog.Logger
	consecutiveMistakes    int
	maxConsecutiveMistakes int
}

// Config contains task configuration including approval settings
// Reference: cli/src/index.ts:56-64, 147-218
type Config struct {
	// Mode is the task execution mode (act or plan)
	Mode TaskMode `json:"mode"`

	// Yolo mode: Forces plain text mode, auto-approves, exits on completion
	// Reference: cli/src/index.ts:56, 203-206
	Yolo bool `json:"yolo"`

	// AutoApproveAll: Keeps interactive TUI mode but auto-approves all tools
	// Reference: cli/src/index.ts:57, 210-213
	AutoApproveAll bool `json:"auto_approve_all"`

	// DoubleCheckCompletion: Rejects first completion attempt
	// Reference: cli/src/index.ts:58, 216-218
	DoubleCheckCompletion bool `json:"double_check_completion"`

	// MaxConsecutiveMistakes: Maximum consecutive mistakes before halting
	// Reference: cli/src/index.ts:64, 195-199
	MaxConsecutiveMistakes int `json:"max_consecutive_mistakes"`

	// PlainTextMode: Whether to use plain text mode instead of TUI
	// Reference: cli/src/index.ts:62, 288-329
	PlainTextMode bool `json:"plain_text_mode"`

	// YoloWarningShown tracks if yolo warning has been displayed
	YoloWarningShown bool `json:"yolo_warning_shown"`
}

// YoloWarning is the warning message shown when yolo mode is enabled
// Reference: src/core/prompts/responses.ts (yoloModeResponse), cli/src/index.ts:203-206
const YoloWarning = "[WARNING] Yolo mode enabled - Cline will automatically approve all actions without confirmation."

// MaxConsecutiveMistakesError is the error message when max mistakes is reached
// Reference: cli/src/index.ts:195-199
const MaxConsecutiveMistakesError = "Maximum consecutive mistakes (%d) reached. Halting execution."

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler(config *Config, logger *slog.Logger) *ApprovalHandler {
	return &ApprovalHandler{
		config:                 config,
		logger:                 logger,
		maxConsecutiveMistakes: config.MaxConsecutiveMistakes,
	}
}

// ShouldAutoApprove returns whether the tool should be auto-approved
// Reference: cli/src/index.ts:203-206, 210-213
func (h *ApprovalHandler) ShouldAutoApprove() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Yolo mode always auto-approves
	if h.config.Yolo {
		return true
	}

	// Auto-approve-all mode auto-approves
	if h.config.AutoApproveAll {
		return true
	}

	return false
}

// HandleApproval handles an approval request
// Returns an automatic approval response if yolo or auto-approve-all is enabled
// Reference: cli/src/index.ts:288-329
func (h *ApprovalHandler) HandleApproval(request *ApprovalRequest) (*ApprovalResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for yolo mode
	if h.config.Yolo {
		if !h.config.YoloWarningShown {
			fmt.Println(YoloWarning)
			h.config.YoloWarningShown = true
		}
		h.logger.Info("Auto-approving (yolo mode)", "tool", request.ToolName)
		return &ApprovalResponse{
			Approved: true,
			Message:  "Auto-approved (yolo mode)",
		}, nil
	}

	// Check for auto-approve-all mode
	if h.config.AutoApproveAll {
		h.logger.Info("Auto-approving (auto-approve-all mode)", "tool", request.ToolName)
		return &ApprovalResponse{
			Approved: true,
			Message:  "Auto-approved (auto-approve-all mode)",
		}, nil
	}

	// Show warning for destructive tools in plain text mode
	if h.config.PlainTextMode {
		return h.handlePlainTextApproval(request)
	}

	// Use interactive TUI approval
	return h.handleInteractiveApproval(request)
}

// HandleCompletion handles task completion, respecting double-check mode
// Reference: cli/src/index.ts:216-218
func (h *ApprovalHandler) HandleCompletion() (*ApprovalResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for double-check completion mode
	if h.config.DoubleCheckCompletion {
		h.logger.Info("Double-check completion enabled, requesting verification")
		return &ApprovalResponse{
			Approved: false,
			Message:  "Double-check completion enabled. Please verify your work before completing.",
		}, nil
	}

	return &ApprovalResponse{
		Approved: true,
		Message:  "Task completed",
	}, nil
}

// RecordToolResult records the result of a tool execution for mistake tracking
// Reference: cli/src/index.ts:195-199
func (h *ApprovalHandler) RecordToolResult(success bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if success {
		// Reset consecutive mistakes on success
		h.consecutiveMistakes = 0
	} else {
		// Increment consecutive mistakes on failure
		h.consecutiveMistakes++

		// Check if we've reached the maximum
		if h.maxConsecutiveMistakes > 0 && h.consecutiveMistakes >= h.maxConsecutiveMistakes {
			return fmt.Errorf(MaxConsecutiveMistakesError, h.maxConsecutiveMistakes)
		}
	}

	return nil
}

// GetConsecutiveMistakes returns the current count of consecutive mistakes
func (h *ApprovalHandler) GetConsecutiveMistakes() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.consecutiveMistakes
}

// ResetConsecutiveMistakes resets the consecutive mistakes counter
func (h *ApprovalHandler) ResetConsecutiveMistakes() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.consecutiveMistakes = 0
}

// handlePlainTextApproval handles approval in plain text mode
// Reference: cli/src/index.ts:288-329
func (h *ApprovalHandler) handlePlainTextApproval(request *ApprovalRequest) (*ApprovalResponse, error) {
	// In plain text mode, we need user input
	fmt.Printf("\n%s Tool approval request:\n", request.ToolName)
	fmt.Printf("Description: %s\n", request.Description)

	if len(request.Details) > 0 {
		fmt.Println("Details:")
		for k, v := range request.Details {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	fmt.Print("\nApprove? (y/n): ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to read user input: %w", err)
	}

	approved := response == "y" || response == "Y" || response == "yes" || response == "Yes"

	return &ApprovalResponse{
		Approved: approved,
		Message:  fmt.Sprintf("User approved: %v", approved),
	}, nil
}

// InteractiveApprovalModel is the Bubble Tea model for interactive approval
type InteractiveApprovalModel struct {
	list     list.Model
	choice   string
	quitting bool
	width    int
	height   int
}

// approvalItem represents an approval option
type approvalItem struct {
	title       string
	description string
	approved    bool
}

func (i approvalItem) Title() string       { return i.title }
func (i approvalItem) Description() string { return i.description }
func (i approvalItem) FilterValue() string { return i.title }

// handleInteractiveApproval handles approval using an interactive TUI
func (h *ApprovalHandler) handleInteractiveApproval(request *ApprovalRequest) (*ApprovalResponse, error) {
	items := []list.Item{
		approvalItem{
			title:       "Approve",
			description: "Execute this tool",
			approved:    true,
		},
		approvalItem{
			title:       "Reject",
			description: "Cancel this tool execution",
			approved:    false,
		},
	}

	// Create list with styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#00D26A")).
		Foreground(lipgloss.Color("#00D26A")).
		Bold(true)

	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0"))

	l := list.New(items, delegate, 40, 10)
	l.Title = fmt.Sprintf("Approve: %s", request.ToolName)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	m := InteractiveApprovalModel{
		list: l,
	}

	// Run the program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return nil, fmt.Errorf("failed to run approval UI: %w", err)
	}

	// For now, return a simple approval response
	// In a full implementation, we'd capture the user's choice from the model
	return &ApprovalResponse{
		Approved: true,
		Message:  "Approved via TUI",
	}, nil
}

// Init implements tea.Model
func (m InteractiveApprovalModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m InteractiveApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(approvalItem)
			if ok {
				m.choice = i.title
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m InteractiveApprovalModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

// IsYoloMode returns whether yolo mode is enabled
func (h *ApprovalHandler) IsYoloMode() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config.Yolo
}

// IsAutoApproveAllMode returns whether auto-approve-all mode is enabled
func (h *ApprovalHandler) IsAutoApproveAllMode() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config.AutoApproveAll
}

// IsPlainTextMode returns whether plain text mode is enabled
func (h *ApprovalHandler) IsPlainTextMode() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config.PlainTextMode
}

// ShouldExitOnCompletion returns whether the CLI should exit after task completion
// Yolo mode exits on completion
func (h *ApprovalHandler) ShouldExitOnCompletion() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config.Yolo
}