package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewSimpleApprovalHistory(t *testing.T) {
	t.Run("creates empty history", func(t *testing.T) {
		history := NewSimpleApprovalHistory()

		assert.NotNil(t, history)
		assert.Empty(t, history.decisions)
		assert.Equal(t, 0, history.GetDecisionCount())
	})
}

func TestSimpleApprovalHistory_RecordDecision(t *testing.T) {
	t.Run("records approval decision", func(t *testing.T) {
		history := NewSimpleApprovalHistory()

		history.RecordDecision("tool", "write_file", true, false)

		assert.Equal(t, 1, history.GetDecisionCount())
		decisions := history.GetRecentDecisions(1)
		assert.Len(t, decisions, 1)
		assert.Equal(t, "tool", decisions[0].ToolType)
		assert.Equal(t, "write_file", decisions[0].ToolName)
		assert.True(t, decisions[0].Approved)
		assert.False(t, decisions[0].AutoApproved)
	})

	t.Run("records rejection decision", func(t *testing.T) {
		history := NewSimpleApprovalHistory()

		history.RecordDecision("command", "rm -rf", false, false)

		decisions := history.GetRecentDecisions(1)
		assert.Len(t, decisions, 1)
		assert.False(t, decisions[0].Approved)
	})

	t.Run("records auto-approved decision", func(t *testing.T) {
		history := NewSimpleApprovalHistory()

		history.RecordDecision("browser", "navigate", true, true)

		decisions := history.GetRecentDecisions(1)
		assert.True(t, decisions[0].AutoApproved)
	})

	t.Run("maintains order of decisions", func(t *testing.T) {
		history := NewSimpleApprovalHistory()

		history.RecordDecision("tool", "tool1", true, false)
		history.RecordDecision("tool", "tool2", false, false)
		history.RecordDecision("tool", "tool3", true, false)

		decisions := history.GetRecentDecisions(3)
		assert.Len(t, decisions, 3)
		assert.Equal(t, "tool1", decisions[0].ToolName)
		assert.Equal(t, "tool2", decisions[1].ToolName)
		assert.Equal(t, "tool3", decisions[2].ToolName)
	})
}

func TestSimpleApprovalHistory_GetRecentDecisions(t *testing.T) {
	t.Run("returns all decisions when limit is zero", func(t *testing.T) {
		history := NewSimpleApprovalHistory()
		history.RecordDecision("tool", "tool1", true, false)
		history.RecordDecision("tool", "tool2", true, false)

		decisions := history.GetRecentDecisions(0)

		assert.Len(t, decisions, 2)
	})

	t.Run("returns limited decisions", func(t *testing.T) {
		history := NewSimpleApprovalHistory()
		history.RecordDecision("tool", "tool1", true, false)
		history.RecordDecision("tool", "tool2", true, false)
		history.RecordDecision("tool", "tool3", true, false)

		decisions := history.GetRecentDecisions(2)

		assert.Len(t, decisions, 2)
		assert.Equal(t, "tool2", decisions[0].ToolName)
		assert.Equal(t, "tool3", decisions[1].ToolName)
	})

	t.Run("returns all when limit exceeds count", func(t *testing.T) {
		history := NewSimpleApprovalHistory()
		history.RecordDecision("tool", "tool1", true, false)

		decisions := history.GetRecentDecisions(10)

		assert.Len(t, decisions, 1)
	})
}

func TestNewApproval(t *testing.T) {
	t.Run("creates approval with defaults", func(t *testing.T) {
		approval := NewApproval()

		assert.NotNil(t, approval)
		assert.NotNil(t, approval.history)
		assert.False(t, approval.isPending)
		assert.False(t, approval.isAutoApprove)
		assert.True(t, approval.showDetails)
		assert.NotNil(t, approval.keyboardShortcuts)

		// Check keyboard shortcuts
		assert.Equal(t, "approve", approval.keyboardShortcuts["y"])
		assert.Equal(t, "approve", approval.keyboardShortcuts["Y"])
		assert.Equal(t, "reject", approval.keyboardShortcuts["n"])
		assert.Equal(t, "reject", approval.keyboardShortcuts["N"])
		assert.Equal(t, "approve", approval.keyboardShortcuts["\r"])
		assert.Equal(t, "approve", approval.keyboardShortcuts["\n"])
	})
}

func TestApproval_ShowPrompt(t *testing.T) {
	t.Run("shows prompt for tool request", func(t *testing.T) {
		approval := NewApproval()
		tool := &ToolRequest{
			ToolType:    "command",
			ToolName:    "ls",
			Description: "List files",
			Command:     "ls -la",
		}

		err := approval.ShowPrompt(tool)

		assert.NoError(t, err)
		assert.True(t, approval.IsPending())
		assert.Equal(t, tool, approval.GetCurrentRequest())
		assert.False(t, approval.decisionMade)
	})

	t.Run("resets previous state", func(t *testing.T) {
		approval := NewApproval()
		oldTool := &ToolRequest{ToolType: "old", ToolName: "old"}
		newTool := &ToolRequest{ToolType: "new", ToolName: "new"}

		approval.ShowPrompt(oldTool)
		approval.HandleInput("y") // Approve old

		approval.ShowPrompt(newTool)

		assert.Equal(t, newTool, approval.GetCurrentRequest())
		assert.False(t, approval.decisionMade)
		assert.False(t, approval.approved)
	})
}

func TestApproval_HandleInput(t *testing.T) {
	t.Run("approves on 'y'", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, done, err := approval.HandleInput("y")

		assert.NoError(t, err)
		assert.True(t, approved)
		assert.True(t, done)
		assert.Equal(t, "yesButtonClicked", <-responseChan)
	})

	t.Run("approves on 'Y'", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, done, err := approval.HandleInput("Y")

		assert.NoError(t, err)
		assert.True(t, approved)
		assert.True(t, done)
	})

	t.Run("rejects on 'n'", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, done, err := approval.HandleInput("n")

		assert.NoError(t, err)
		assert.False(t, approved)
		assert.True(t, done)
		assert.Equal(t, "noButtonClicked", <-responseChan)
	})

	t.Run("rejects on 'N'", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, done, err := approval.HandleInput("N")

		assert.NoError(t, err)
		assert.False(t, approved)
		assert.True(t, done)
	})

	t.Run("approves on Enter", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, done, err := approval.HandleInput("\r")

		assert.NoError(t, err)
		assert.True(t, approved)
		assert.True(t, done)
	})

	t.Run("returns not done for unknown key", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})

		approved, done, err := approval.HandleInput("x")

		assert.NoError(t, err)
		assert.False(t, approved)
		assert.False(t, done)
		assert.True(t, approval.IsPending()) // Still pending
	})

	t.Run("returns done when not pending", func(t *testing.T) {
		approval := NewApproval()
		// Don't show prompt

		approved, done, err := approval.HandleInput("y")

		assert.NoError(t, err)
		assert.False(t, approved)
		assert.True(t, done)
	})

	t.Run("records decision in history", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test", Command: "ls"})

		approval.HandleInput("y")

		decisions := approval.GetHistory().GetRecentDecisions(1)
		assert.Len(t, decisions, 1)
		assert.Equal(t, "tool", decisions[0].ToolType)
		assert.Equal(t, "test", decisions[0].ToolName)
		assert.True(t, decisions[0].Approved)
	})
}

func TestApproval_IsAutoApproveEnabled(t *testing.T) {
	t.Run("returns false by default", func(t *testing.T) {
		approval := NewApproval()

		assert.False(t, approval.IsAutoApproveEnabled())
	})

	t.Run("returns true when enabled", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)

		assert.True(t, approval.IsAutoApproveEnabled())
	})
}

func TestApproval_SetAutoApprove(t *testing.T) {
	t.Run("enables auto-approve", func(t *testing.T) {
		approval := NewApproval()

		approval.SetAutoApprove(true)

		assert.True(t, approval.isAutoApprove)
	})

	t.Run("disables auto-approve", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)

		approval.SetAutoApprove(false)

		assert.False(t, approval.isAutoApprove)
	})
}

func TestApproval_AutoApprove(t *testing.T) {
	t.Run("auto-approves when enabled", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		responseChan := make(chan string, 1)
		approval.SetResponseChannel(responseChan)

		approved, err := approval.AutoApprove()

		assert.NoError(t, err)
		assert.True(t, approved)
		assert.False(t, approval.IsPending())
		assert.Equal(t, "yesButtonClicked", <-responseChan)
	})

	t.Run("records as auto-approved in history", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})

		approval.AutoApprove()

		decisions := approval.GetHistory().GetRecentDecisions(1)
		assert.Len(t, decisions, 1)
		assert.True(t, decisions[0].AutoApproved)
	})

	t.Run("rejects dangerous operations when auto-approve disabled", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(false)
		approval.ShowPrompt(&ToolRequest{
			ToolType:    "command",
			ToolName:    "rm",
			IsDangerous: true,
		})

		approved, err := approval.AutoApprove()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous operation requires manual approval")
		assert.False(t, approved)
	})

	t.Run("approves dangerous operations when auto-approve enabled", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)
		approval.ShowPrompt(&ToolRequest{
			ToolType:    "command",
			ToolName:    "rm",
			IsDangerous: true,
		})

		approved, err := approval.AutoApprove()

		assert.NoError(t, err)
		assert.True(t, approved)
	})

	t.Run("returns error when no pending request", func(t *testing.T) {
		approval := NewApproval()
		approval.SetAutoApprove(true)

		approved, err := approval.AutoApprove()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no pending approval request")
		assert.False(t, approved)
	})
}

func TestApproval_IsPending(t *testing.T) {
	t.Run("returns false initially", func(t *testing.T) {
		approval := NewApproval()

		assert.False(t, approval.IsPending())
	})

	t.Run("returns true when prompt shown", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})

		assert.True(t, approval.IsPending())
	})

	t.Run("returns false after decision", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		approval.HandleInput("y")

		assert.False(t, approval.IsPending())
	})
}

func TestApproval_GetCurrentRequest(t *testing.T) {
	t.Run("returns nil initially", func(t *testing.T) {
		approval := NewApproval()

		assert.Nil(t, approval.GetCurrentRequest())
	})

	t.Run("returns current request", func(t *testing.T) {
		approval := NewApproval()
		tool := &ToolRequest{ToolType: "tool", ToolName: "test"}
		approval.ShowPrompt(tool)

		assert.Equal(t, tool, approval.GetCurrentRequest())
	})
}

func TestApproval_GetHistory(t *testing.T) {
	t.Run("returns history", func(t *testing.T) {
		approval := NewApproval()

		history := approval.GetHistory()

		assert.NotNil(t, history)
	})
}

func TestApproval_Reset(t *testing.T) {
	t.Run("clears approval state", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		approval.HandleInput("y")

		approval.Reset()

		assert.Nil(t, approval.GetCurrentRequest())
		assert.False(t, approval.IsPending())
		assert.False(t, approval.decisionMade)
		assert.False(t, approval.approved)
	})
}

func TestApproval_GetToolDetails(t *testing.T) {
	t.Run("returns empty when no request", func(t *testing.T) {
		approval := NewApproval()

		details := approval.GetToolDetails()

		assert.Empty(t, details)
	})

	t.Run("returns tool details", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{
			ToolType:    "command",
			ToolName:    "ls",
			Description: "List directory contents",
			Command:     "ls -la /home",
			FilePath:    "/home",
			IsDangerous: true,
		})

		details := approval.GetToolDetails()

		assert.Contains(t, details, "Tool: ls")
		assert.Contains(t, details, "Type: command")
		assert.Contains(t, details, "Description: List directory contents")
		assert.Contains(t, details, "Command: ls -la /home")
		assert.Contains(t, details, "File: /home")
		assert.Contains(t, details, "Warning")
	})

	t.Run("omits empty fields", func(t *testing.T) {
		approval := NewApproval()
		approval.ShowPrompt(&ToolRequest{
			ToolType: "tool",
			ToolName: "simple",
		})

		details := approval.GetToolDetails()

		assert.Contains(t, details, "Tool: simple")
		assert.NotContains(t, details, "Description")
		assert.NotContains(t, details, "Command")
		assert.NotContains(t, details, "File")
	})
}

func TestApproval_SetResponseChannel(t *testing.T) {
	t.Run("sets response channel", func(t *testing.T) {
		approval := NewApproval()
		ch := make(chan string, 1)

		approval.SetResponseChannel(ch)

		// Trigger a response
		approval.ShowPrompt(&ToolRequest{ToolType: "tool", ToolName: "test"})
		approval.HandleInput("y")

		assert.Equal(t, "yesButtonClicked", <-ch)
	})
}

func TestNewApprovalModel(t *testing.T) {
	t.Run("creates approval model", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Test Title", "Test Message", "Test Details")

		assert.NotNil(t, model)
		assert.Equal(t, ApprovalTypeTool, model.requestType)
		assert.Equal(t, "Test Title", model.title)
		assert.Equal(t, "Test Message", model.message)
		assert.Equal(t, "Test Details", model.details)
		assert.Equal(t, 0, model.selected)
		assert.True(t, model.showDetails)
		assert.Len(t, model.options, 3)
		assert.False(t, model.done)
	})
}

func TestApprovalModel_Update(t *testing.T) {
	t.Run("approves on 'y'", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

		assert.True(t, newModel.(ApprovalModel).IsDone())
		assert.Equal(t, ApprovalYes, newModel.(ApprovalModel).GetResult())
	})

	t.Run("rejects on 'n'", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

		assert.True(t, newModel.(ApprovalModel).IsDone())
		assert.Equal(t, ApprovalNo, newModel.(ApprovalModel).GetResult())
	})

	t.Run("always on 'a'", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

		assert.True(t, newModel.(ApprovalModel).IsDone())
		assert.Equal(t, ApprovalAlways, newModel.(ApprovalModel).GetResult())
	})

	t.Run("quits on 'q'", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

		assert.True(t, newModel.(ApprovalModel).IsDone())
		assert.Equal(t, ApprovalNo, newModel.(ApprovalModel).GetResult())
		assert.NotNil(t, cmd)
	})

	t.Run("navigates left", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")
		model.selected = 1

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})

		assert.Equal(t, 0, newModel.(ApprovalModel).selected)
	})

	t.Run("navigates right", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})

		assert.Equal(t, 1, newModel.(ApprovalModel).selected)
	})

	t.Run("toggles details on 'd'", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")
		model.showDetails = true

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

		assert.False(t, newModel.(ApprovalModel).showDetails)
	})

	t.Run("cycles with tab", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")
		model.selected = 0

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.Equal(t, 1, newModel.(ApprovalModel).selected)
	})

	t.Run("cycles backward with shift+tab", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")
		model.selected = 0

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

		assert.Equal(t, 2, newModel.(ApprovalModel).selected)
	})
}

func TestApprovalModel_View(t *testing.T) {
	t.Run("renders loading when no dimensions", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		view := model.View()

		assert.Equal(t, "Loading...", view)
	})

	t.Run("renders modal with dimensions", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "Details")
		model.SetDimensions(100, 50)

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Title")
		assert.Contains(t, view, "Message")
	})

	t.Run("renders without details when hidden", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "Details")
		model.SetDimensions(100, 50)
		model.showDetails = false

		view := model.View()

		assert.NotContains(t, view, "Details")
	})

	t.Run("truncates long details", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")
		model.SetDimensions(100, 50)
		// Create very long details
		longDetails := ""
		for i := 0; i < 400; i++ {
			longDetails += "a"
		}
		model.details = longDetails

		view := model.View()

		assert.Contains(t, view, "...")
	})
}

func TestApprovalModel_wrapText(t *testing.T) {
	t.Run("wraps text at width", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		wrapped := model.wrapText("Hello World This Is Long", 10)

		assert.Contains(t, wrapped, "\n")
	})

	t.Run("returns empty for empty text", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		wrapped := model.wrapText("", 10)

		assert.Empty(t, wrapped)
	})

	t.Run("returns original for zero width", func(t *testing.T) {
		model := NewApprovalModel(ApprovalTypeTool, "Title", "Message", "")

		wrapped := model.wrapText("Hello", 0)

		assert.Equal(t, "Hello", wrapped)
	})
}

func TestApprovalModel_getIcon(t *testing.T) {
	tests := []struct {
		approvalType ApprovalType
		expected     string
	}{
		{ApprovalTypeCommand, "⚡"},
		{ApprovalTypeTool, "🔧"},
		{ApprovalTypeEdit, "✏️"},
		{ApprovalTypeBrowser, "🌐"},
		{ApprovalType("unknown"), "❓"},
	}

	for _, tt := range tests {
		t.Run(string(tt.approvalType), func(t *testing.T) {
			model := NewApprovalModel(tt.approvalType, "Title", "Message", "")

			icon := model.getIcon()

			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestShowApprovalPrompt(t *testing.T) {
	// Note: This test would require a TTY, so we just verify the function exists
	t.Run("function exists", func(t *testing.T) {
		// Function signature is correct
		var _ func(ApprovalType, string, string, string) (ApprovalResponse, error) = ShowApprovalPrompt
	})
}

func TestCommandApprovalPrompt(t *testing.T) {
	// Note: This test would require a TTY
	t.Run("function exists", func(t *testing.T) {
		var _ func(string, bool) (ApprovalResponse, error) = CommandApprovalPrompt
	})
}

func TestToolApprovalPrompt(t *testing.T) {
	// Note: This test would require a TTY
	t.Run("function exists", func(t *testing.T) {
		var _ func(string, map[string]interface{}) (ApprovalResponse, error) = ToolApprovalPrompt
	})
}

func TestEditApprovalPrompt(t *testing.T) {
	// Note: This test would require a TTY
	t.Run("function exists", func(t *testing.T) {
		var _ func(string, string) (ApprovalResponse, error) = EditApprovalPrompt
	})
}

func TestBrowserApprovalPrompt(t *testing.T) {
	// Note: This test would require a TTY
	t.Run("function exists", func(t *testing.T) {
		var _ func(string, string) (ApprovalResponse, error) = BrowserApprovalPrompt
	})
}

func TestApprovalResponse(t *testing.T) {
	t.Run("approval response constants", func(t *testing.T) {
		assert.Equal(t, ApprovalResponse("yes"), ApprovalYes)
		assert.Equal(t, ApprovalResponse("no"), ApprovalNo)
		assert.Equal(t, ApprovalResponse("always"), ApprovalAlways)
	})
}

func TestApprovalType(t *testing.T) {
	t.Run("approval type constants", func(t *testing.T) {
		assert.Equal(t, ApprovalType("command"), ApprovalTypeCommand)
		assert.Equal(t, ApprovalType("tool"), ApprovalTypeTool)
		assert.Equal(t, ApprovalType("edit"), ApprovalTypeEdit)
		assert.Equal(t, ApprovalType("browser"), ApprovalTypeBrowser)
	})
}