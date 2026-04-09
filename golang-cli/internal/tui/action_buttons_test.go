package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewActionButtons(t *testing.T) {
	t.Run("creates action buttons with config", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Approve",
			SecondaryText:   "Reject",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "act", 80)

		assert.NotNil(t, ab)
		assert.Equal(t, config, ab.config)
		assert.Equal(t, "act", ab.mode)
		assert.Equal(t, 80, ab.terminalWidth)
	})

	t.Run("creates action buttons with plan mode", func(t *testing.T) {
		config := ButtonConfig{EnableButtons: false}
		ab := NewActionButtons(config, "plan", 120)

		assert.Equal(t, "plan", ab.mode)
		assert.Equal(t, 120, ab.terminalWidth)
	})
}

func TestActionButtons_SetConfig(t *testing.T) {
	t.Run("updates button configuration", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{EnableButtons: false}, "act", 80)

		newConfig := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Yes",
			SecondaryText:   "No",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab.SetConfig(newConfig)

		assert.Equal(t, newConfig, ab.config)
	})
}

func TestActionButtons_SetMode(t *testing.T) {
	t.Run("updates mode to plan", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		ab.SetMode("plan")

		assert.Equal(t, "plan", ab.mode)
	})

	t.Run("updates mode to act", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "plan", 80)

		ab.SetMode("act")

		assert.Equal(t, "act", ab.mode)
	})
}

func TestActionButtons_SetTerminalWidth(t *testing.T) {
	t.Run("updates terminal width", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		ab.SetTerminalWidth(120)

		assert.Equal(t, 120, ab.terminalWidth)
	})
}

func TestActionButtons_ShouldShow(t *testing.T) {
	t.Run("returns false when buttons disabled", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons: false,
			PrimaryText:   "Yes",
		}
		ab := NewActionButtons(config, "act", 80)

		assert.False(t, ab.ShouldShow())
	})

	t.Run("returns true when primary button exists", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons: true,
			PrimaryText:   "Yes",
			PrimaryAction: ButtonActionApprove,
		}
		ab := NewActionButtons(config, "act", 80)

		assert.True(t, ab.ShouldShow())
	})

	t.Run("returns true when secondary button exists", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			SecondaryText:   "No",
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "act", 80)

		assert.True(t, ab.ShouldShow())
	})

	t.Run("returns false when only cancel action", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			SecondaryText:   "Cancel",
			SecondaryAction: ButtonActionCancel,
		}
		ab := NewActionButtons(config, "act", 80)

		assert.False(t, ab.ShouldShow())
	})

	t.Run("returns false when no buttons configured", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons: true,
		}
		ab := NewActionButtons(config, "act", 80)

		assert.False(t, ab.ShouldShow())
	})
}

func TestActionButtons_Render(t *testing.T) {
	t.Run("renders empty when buttons disabled", func(t *testing.T) {
		config := ButtonConfig{EnableButtons: false}
		ab := NewActionButtons(config, "act", 80)

		result := ab.Render()

		assert.Empty(t, result)
	})

	t.Run("renders empty when ShouldShow is false", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			SecondaryText:   "Cancel",
			SecondaryAction: ButtonActionCancel,
		}
		ab := NewActionButtons(config, "act", 80)

		result := ab.Render()

		assert.Empty(t, result)
	})

	t.Run("renders both buttons", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Approve",
			SecondaryText:   "Reject",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "act", 80)

		result := ab.Render()

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Approve")
		assert.Contains(t, result, "Reject")
	})

	t.Run("uses act mode color", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Approve",
			SecondaryText:   "Reject",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "act", 80)

		result := ab.Render()

		assert.NotEmpty(t, result)
	})

	t.Run("uses plan mode color", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Approve",
			SecondaryText:   "Reject",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "plan", 80)

		result := ab.Render()

		assert.NotEmpty(t, result)
	})

	t.Run("handles narrow terminal", func(t *testing.T) {
		config := ButtonConfig{
			EnableButtons:   true,
			PrimaryText:     "Approve",
			SecondaryText:   "Reject",
			PrimaryAction:   ButtonActionApprove,
			SecondaryAction: ButtonActionReject,
		}
		ab := NewActionButtons(config, "act", 20)

		result := ab.Render()

		assert.NotEmpty(t, result)
	})
}

func TestActionButtons_renderButton(t *testing.T) {
	t.Run("renders button with text and shortcut", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		result := ab.renderButton("Test", "1", 20, lipgloss.Color("#00D9FF"))

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Test")
		assert.Contains(t, result, "(1)")
	})

	t.Run("handles narrow width", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		result := ab.renderButton("Test", "1", 5, lipgloss.Color("#00D9FF"))

		assert.NotEmpty(t, result)
	})

	t.Run("handles zero width", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		result := ab.renderButton("Test", "1", 0, lipgloss.Color("#00D9FF"))

		assert.NotEmpty(t, result)
	})

	t.Run("pads button to width", func(t *testing.T) {
		ab := NewActionButtons(ButtonConfig{}, "act", 80)

		result := ab.renderButton("A", "1", 20, lipgloss.Color("#00D9FF"))

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "A")
	})
}

func TestGetButtonConfig(t *testing.T) {
	t.Run("returns config for api_req_failed", func(t *testing.T) {
		config := GetButtonConfig("say", "api_req_failed", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Retry", config.PrimaryText)
		assert.Equal(t, "Start New Task", config.SecondaryText)
		assert.Equal(t, ButtonActionRetry, config.PrimaryAction)
		assert.Equal(t, ButtonActionNewTask, config.SecondaryAction)
		assert.True(t, config.SendingDisabled)
	})

	t.Run("returns config for mistake_limit_reached", func(t *testing.T) {
		config := GetButtonConfig("say", "mistake_limit_reached", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Proceed Anyways", config.PrimaryText)
		assert.Equal(t, "Start New Task", config.SecondaryText)
		assert.Equal(t, ButtonActionProceed, config.PrimaryAction)
		assert.False(t, config.SendingDisabled)
	})

	t.Run("returns config for streaming state", func(t *testing.T) {
		config := GetButtonConfig("say", "", true, false)

		assert.True(t, config.EnableButtons)
		assert.True(t, config.SendingDisabled)
		assert.Equal(t, "Cancel", config.SecondaryText)
		assert.Equal(t, ButtonActionCancel, config.SecondaryAction)
	})

	t.Run("returns config for ask tool", func(t *testing.T) {
		config := GetButtonConfig("ask", "tool", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Approve", config.PrimaryText)
		assert.Equal(t, "Reject", config.SecondaryText)
		assert.Equal(t, ButtonActionApprove, config.PrimaryAction)
		assert.Equal(t, ButtonActionReject, config.SecondaryAction)
	})

	t.Run("returns config for ask command", func(t *testing.T) {
		config := GetButtonConfig("ask", "command", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Run Command", config.PrimaryText)
		assert.Equal(t, "Reject", config.SecondaryText)
	})

	t.Run("returns config for ask command_output", func(t *testing.T) {
		config := GetButtonConfig("ask", "command_output", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Proceed While Running", config.PrimaryText)
		assert.Empty(t, config.SecondaryText)
		assert.Equal(t, ButtonActionProceed, config.PrimaryAction)
	})

	t.Run("returns config for ask browser_action_launch", func(t *testing.T) {
		config := GetButtonConfig("ask", "browser_action_launch", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Approve", config.PrimaryText)
		assert.Equal(t, "Reject", config.SecondaryText)
	})

	t.Run("returns config for ask use_mcp_server", func(t *testing.T) {
		config := GetButtonConfig("ask", "use_mcp_server", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Approve", config.PrimaryText)
		assert.Equal(t, "Reject", config.SecondaryText)
	})

	t.Run("returns config for ask completion_result", func(t *testing.T) {
		config := GetButtonConfig("ask", "completion_result", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Start New Task", config.PrimaryText)
		assert.Equal(t, "Exit", config.SecondaryText)
		assert.Equal(t, ButtonActionNewTask, config.PrimaryAction)
	})

	t.Run("returns config for ask resume_task", func(t *testing.T) {
		config := GetButtonConfig("ask", "resume_task", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Resume Task", config.PrimaryText)
		assert.Equal(t, "Exit", config.SecondaryText)
		assert.Equal(t, ButtonActionProceed, config.PrimaryAction)
	})

	t.Run("returns config for ask resume_completed_task", func(t *testing.T) {
		config := GetButtonConfig("ask", "resume_completed_task", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Start New Task", config.PrimaryText)
		assert.Equal(t, "Exit", config.SecondaryText)
		assert.Equal(t, ButtonActionNewTask, config.PrimaryAction)
	})

	t.Run("returns config for ask new_task", func(t *testing.T) {
		config := GetButtonConfig("ask", "new_task", false, false)

		assert.True(t, config.EnableButtons)
		assert.Equal(t, "Start New Task with Context", config.PrimaryText)
		assert.Equal(t, "Exit", config.SecondaryText)
	})

	t.Run("returns config for ask followup", func(t *testing.T) {
		config := GetButtonConfig("ask", "followup", false, false)

		assert.False(t, config.EnableButtons)
	})

	t.Run("returns config for ask plan_mode_respond", func(t *testing.T) {
		config := GetButtonConfig("ask", "plan_mode_respond", false, false)

		assert.False(t, config.EnableButtons)
	})

	t.Run("returns config for say api_req_started", func(t *testing.T) {
		config := GetButtonConfig("say", "api_req_started", false, false)

		assert.True(t, config.EnableButtons)
		assert.True(t, config.SendingDisabled)
		assert.Equal(t, "Cancel", config.SecondaryText)
		assert.Equal(t, ButtonActionCancel, config.SecondaryAction)
	})

	t.Run("returns default config", func(t *testing.T) {
		config := GetButtonConfig("unknown", "unknown", false, false)

		assert.False(t, config.EnableButtons)
		assert.False(t, config.SendingDisabled)
	})
}

func TestHandleButtonAction(t *testing.T) {
	t.Run("returns yesButtonClicked for approve", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionApprove, "tool")

		assert.Equal(t, "yesButtonClicked", result)
	})

	t.Run("returns yesButtonClicked for proceed", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionProceed, "command")

		assert.Equal(t, "yesButtonClicked", result)
	})

	t.Run("returns noButtonClicked for reject", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionReject, "tool")

		assert.Equal(t, "noButtonClicked", result)
	})

	t.Run("returns yesButtonClicked for new task", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionNewTask, "completion_result")

		assert.Equal(t, "yesButtonClicked", result)
	})

	t.Run("returns yesButtonClicked for retry", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionRetry, "api_req_failed")

		assert.Equal(t, "yesButtonClicked", result)
	})

	t.Run("returns noButtonClicked for cancel", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionCancel, "api_req_started")

		assert.Equal(t, "noButtonClicked", result)
	})

	t.Run("returns noButtonClicked for unknown action", func(t *testing.T) {
		result := HandleButtonAction(ButtonActionType("unknown"), "tool")

		assert.Equal(t, "noButtonClicked", result)
	})
}

func TestButtonActionType_Constants(t *testing.T) {
	t.Run("button action type constants", func(t *testing.T) {
		assert.Equal(t, ButtonActionType("approve"), ButtonActionApprove)
		assert.Equal(t, ButtonActionType("reject"), ButtonActionReject)
		assert.Equal(t, ButtonActionType("proceed"), ButtonActionProceed)
		assert.Equal(t, ButtonActionType("new_task"), ButtonActionNewTask)
		assert.Equal(t, ButtonActionType("cancel"), ButtonActionCancel)
		assert.Equal(t, ButtonActionType("retry"), ButtonActionRetry)
	})
}