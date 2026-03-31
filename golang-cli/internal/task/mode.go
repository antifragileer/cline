// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"fmt"
	"strings"
)

// ModeHandler handles mode-specific operations for task execution
type ModeHandler struct {
	mode TaskMode
}

// NewModeHandler creates a new mode handler
func NewModeHandler(mode TaskMode) *ModeHandler {
	return &ModeHandler{
		mode: mode,
	}
}

// GetMode returns the current mode
func (mh *ModeHandler) GetMode() TaskMode {
	return mh.mode
}

// IsActMode returns true if in act mode
func (mh *ModeHandler) IsActMode() bool {
	return mh.mode == TaskModeAct
}

// IsPlanMode returns true if in plan mode
func (mh *ModeHandler) IsPlanMode() bool {
	return mh.mode == TaskModePlan
}

// GetModeDescription returns a human-readable description of the mode
func (mh *ModeHandler) GetModeDescription() string {
	switch mh.mode {
	case TaskModeAct:
		return "Act mode - Execute actions and modify files"
	case TaskModePlan:
		return "Plan mode - Plan and discuss without executing"
	default:
		return "Unknown mode"
	}
}

// GetModePromptPrefix returns a prefix to add to prompts based on mode
func (mh *ModeHandler) GetModePromptPrefix() string {
	switch mh.mode {
	case TaskModePlan:
		return "[PLAN MODE] "
	default:
		return ""
	}
}

// ValidateMode validates that a mode string is valid
func ValidateMode(mode string) (TaskMode, error) {
	switch strings.ToLower(mode) {
	case "act":
		return TaskModeAct, nil
	case "plan":
		return TaskModePlan, nil
	default:
		return "", fmt.Errorf("invalid mode: %s (must be 'act' or 'plan')", mode)
	}
}

// ModeFromString converts a string to a TaskMode, defaulting to Act if invalid
func ModeFromString(mode string) TaskMode {
	m, err := ValidateMode(mode)
	if err != nil {
		return TaskModeAct
	}
	return m
}

// ModeConfig holds configuration specific to each mode
type ModeConfig struct {
	// AllowToolExecution determines if tools can be executed
	AllowToolExecution bool

	// AllowFileModification determines if files can be modified
	AllowFileModification bool

	// AllowCommandExecution determines if commands can be executed
	AllowCommandExecution bool

	// RequireApproval determines if actions require user approval
	RequireApproval bool

	// PromptPrefix is added to all prompts in this mode
	PromptPrefix string

	// SystemPrompt is the system prompt for this mode
	SystemPrompt string
}

// GetModeConfig returns the configuration for the current mode
func (mh *ModeHandler) GetModeConfig() ModeConfig {
	switch mh.mode {
	case TaskModePlan:
		return ModeConfig{
			AllowToolExecution:    false,
			AllowFileModification: false,
			AllowCommandExecution: false,
			RequireApproval:       true,
			PromptPrefix:          "[PLAN MODE] ",
			SystemPrompt: `You are in PLAN MODE. Your role is to:
1. Analyze the request and gather information
2. Ask clarifying questions if needed
3. Create a detailed plan for implementation
4. Discuss approaches with the user
5. Do NOT execute any actions or modify files

Focus on planning, analysis, and discussion. Wait for user confirmation before proceeding to act mode.`,
		}
	case TaskModeAct:
		return ModeConfig{
			AllowToolExecution:    true,
			AllowFileModification: true,
			AllowCommandExecution: true,
			RequireApproval:       false,
			PromptPrefix:          "",
			SystemPrompt: `You are in ACT MODE. Your role is to:
1. Execute the planned actions
2. Modify files as needed
3. Run commands when necessary
4. Implement the solution
5. Report results and completion

Focus on execution and implementation. Use tools to accomplish the task.`,
		}
	default:
		return ModeConfig{
			AllowToolExecution:    true,
			AllowFileModification: true,
			AllowCommandExecution: true,
			RequireApproval:       false,
			PromptPrefix:          "",
			SystemPrompt:          "",
		}
	}
}

// WrapPrompt wraps a prompt with mode-specific prefix and context
func (mh *ModeHandler) WrapPrompt(prompt string) string {
	config := mh.GetModeConfig()

	var parts []string

	// Add system prompt if present
	if config.SystemPrompt != "" {
		parts = append(parts, config.SystemPrompt)
	}

	// Add the actual prompt with prefix
	parts = append(parts, config.PromptPrefix+prompt)

	return strings.Join(parts, "\n\n")
}

// CanExecuteTool checks if a tool can be executed in the current mode
func (mh *ModeHandler) CanExecuteTool(toolName string) bool {
	config := mh.GetModeConfig()

	if !config.AllowToolExecution {
		return false
	}

	// In plan mode, certain tools are still allowed (like read_file for analysis)
	if mh.mode == TaskModePlan {
		allowedToolsInPlan := map[string]bool{
			"read_file":                  true,
			"search_files":               true,
			"list_files":                 true,
			"list_code_definition_names": true,
		}

		if allowedToolsInPlan[toolName] {
			return true
		}

		return false
	}

	return true
}

// CanModifyFile checks if files can be modified in the current mode
func (mh *ModeHandler) CanModifyFile() bool {
	return mh.GetModeConfig().AllowFileModification
}

// CanExecuteCommand checks if commands can be executed in the current mode
func (mh *ModeHandler) CanExecuteCommand() bool {
	return mh.GetModeConfig().AllowCommandExecution
}

// ModeSwitchRequest represents a request to switch modes
type ModeSwitchRequest struct {
	// TargetMode is the mode to switch to
	TargetMode TaskMode

	// Reason is the reason for the switch
	Reason string

	// PreserveContext determines if context should be preserved
	PreserveContext bool
}

// ModeSwitchResult represents the result of a mode switch
type ModeSwitchResult struct {
	// Success indicates if the switch was successful
	Success bool

	// PreviousMode is the mode before switching
	PreviousMode TaskMode

	// CurrentMode is the mode after switching
	CurrentMode TaskMode

	// Message describes the result
	Message string
}

// SwitchMode switches to a new mode
func (mh *ModeHandler) SwitchMode(request ModeSwitchRequest) ModeSwitchResult {
	previousMode := mh.mode

	// Validate target mode
	if request.TargetMode != TaskModeAct && request.TargetMode != TaskModePlan {
		return ModeSwitchResult{
			Success:      false,
			PreviousMode: previousMode,
			CurrentMode:  mh.mode,
			Message:      fmt.Sprintf("Invalid target mode: %s", request.TargetMode),
		}
	}

	// Perform the switch
	mh.mode = request.TargetMode

	return ModeSwitchResult{
		Success:     true,
		PreviousMode: previousMode,
		CurrentMode:  mh.mode,
		Message: fmt.Sprintf("Switched from %s to %s mode%s",
			previousMode,
			mh.mode,
			func() string {
				if request.Reason != "" {
					return ": " + request.Reason
				}
				return ""
			}()),
	}
}

// ModeFlags holds flags related to mode configuration
type ModeFlags struct {
	// Act enables act mode
	Act bool

	// Plan enables plan mode
	Plan bool

	// Yolo enables auto-approval (yolo mode)
	Yolo bool

	// AutoApproveAll enables auto-approval for all actions
	AutoApproveAll bool
}

// ParseModeFlags parses mode flags and returns the appropriate mode
func ParseModeFlags(flags ModeFlags) (TaskMode, error) {
	// Check for mutually exclusive flags
	if flags.Act && flags.Plan {
		return "", fmt.Errorf("cannot use both --act and --plan flags")
	}

	// Determine mode
	if flags.Plan {
		return TaskModePlan, nil
	}

	// Default to act mode
	return TaskModeAct, nil
}

// GetModeFromFlags determines the mode from command flags
func GetModeFromFlags(actFlag, planFlag bool) TaskMode {
	flags := ModeFlags{
		Act:  actFlag,
		Plan: planFlag,
	}

	mode, _ := ParseModeFlags(flags)
	return mode
}

// ModeCapabilities describes what a mode can do
type ModeCapabilities struct {
	// CanReadFiles indicates if the mode can read files
	CanReadFiles bool

	// CanWriteFiles indicates if the mode can write files
	CanWriteFiles bool

	// CanExecuteCommands indicates if the mode can execute commands
	CanExecuteCommands bool

	// CanUseTools indicates if the mode can use tools
	CanUseTools bool

	// RequiresUserApproval indicates if actions require approval
	RequiresUserApproval bool
}

// GetCapabilities returns the capabilities of the current mode
func (mh *ModeHandler) GetCapabilities() ModeCapabilities {
	config := mh.GetModeConfig()

	return ModeCapabilities{
		CanReadFiles:         true, // All modes can read files
		CanWriteFiles:        config.AllowFileModification,
		CanExecuteCommands:   config.AllowCommandExecution,
		CanUseTools:          config.AllowToolExecution,
		RequiresUserApproval: config.RequireApproval,
	}
}

// ModeSummary provides a summary of mode behavior
type ModeSummary struct {
	Mode         TaskMode           `json:"mode"`
	Description  string             `json:"description"`
	Capabilities ModeCapabilities   `json:"capabilities"`
}

// GetModeSummary returns a summary of the current mode
func (mh *ModeHandler) GetModeSummary() ModeSummary {
	return ModeSummary{
		Mode:         mh.mode,
		Description:  mh.GetModeDescription(),
		Capabilities: mh.GetCapabilities(),
	}
}