package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAPIKeyInputStyles(t *testing.T) {
	styles := DefaultAPIKeyInputStyles()

	assert.NotZero(t, styles.containerStyle)
	assert.NotZero(t, styles.titleStyle)
	assert.NotZero(t, styles.labelStyle)
	assert.NotZero(t, styles.inputStyle)
	assert.NotZero(t, styles.inputFocusedStyle)
	assert.NotZero(t, styles.maskedStyle)
	assert.NotZero(t, styles.visibleStyle)
	assert.NotZero(t, styles.helpStyle)
	assert.NotZero(t, styles.errorStyle)
	assert.NotZero(t, styles.successStyle)
	assert.NotZero(t, styles.descriptionStyle)
	assert.NotZero(t, styles.strengthWeakStyle)
	assert.NotZero(t, styles.strengthFairStyle)
	assert.NotZero(t, styles.strengthGoodStyle)
	assert.NotZero(t, styles.strengthStrongStyle)
}

func TestNewAPIKeyInput(t *testing.T) {
	input := NewAPIKeyInput()

	assert.NotNil(t, input)
	assert.Equal(t, "Enter API key...", input.placeholder)
	assert.Equal(t, "API Key", input.label)
	assert.Equal(t, "Your API key will be stored securely", input.description)
	assert.True(t, input.masked)
	assert.False(t, input.isValid)
	assert.NotZero(t, input.styles)
}

func TestNewAPIKeyInputWithLabel(t *testing.T) {
	input := NewAPIKeyInputWithLabel("Custom Label", "Custom Description")

	assert.Equal(t, "Custom Label", input.label)
	assert.Equal(t, "Custom Description", input.description)
}

func TestAPIKeyInput_SetDimensions(t *testing.T) {
	input := NewAPIKeyInput()
	input.SetDimensions(100, 50)

	assert.Equal(t, 100, input.width)
	assert.Equal(t, 50, input.height)
}

func TestAPIKeyInput_SetPlaceholder(t *testing.T) {
	input := NewAPIKeyInput()
	input.SetPlaceholder("Custom placeholder")

	assert.Equal(t, "Custom placeholder", input.placeholder)
}

func TestAPIKeyInput_SetLabel(t *testing.T) {
	input := NewAPIKeyInput()
	input.SetLabel("Custom Label")

	assert.Equal(t, "Custom Label", input.label)
}

func TestAPIKeyInput_SetDescription(t *testing.T) {
	input := NewAPIKeyInput()
	input.SetDescription("Custom Description")

	assert.Equal(t, "Custom Description", input.description)
}

func TestAPIKeyInput_SetMasked(t *testing.T) {
	input := NewAPIKeyInput()
	
	// Default is true
	assert.True(t, input.masked)
	
	input.SetMasked(false)
	assert.False(t, input.masked)
	
	input.SetMasked(true)
	assert.True(t, input.masked)
}

func TestAPIKeyInput_SetValidator(t *testing.T) {
	input := NewAPIKeyInput()
	
	validator := func(s string) error {
		if len(s) < 5 {
			return fmt.Errorf("too short")
		}
		return nil
	}
	
	input.SetValidator(validator)
	assert.NotNil(t, input.validator)
	
	// Test validation
	input.value = "ab"
	input.validate()
	assert.False(t, input.isValid)
	
	input.value = "valid key"
	input.validate()
	assert.True(t, input.isValid)
}

func TestAPIKeyInput_SetShowStrength(t *testing.T) {
	input := NewAPIKeyInput()
	
	assert.False(t, input.showStrength)
	
	input.SetShowStrength(true)
	assert.True(t, input.showStrength)
}

func TestAPIKeyInput_Init(t *testing.T) {
	input := NewAPIKeyInput()
	cmd := input.Init()
	
	assert.Nil(t, cmd)
}

func TestAPIKeyInput_Update_Esc(t *testing.T) {
	input := NewAPIKeyInput()
	
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, _ = input.Update(msg)
	
	assert.True(t, input.IsCancelled())
}

func TestAPIKeyInput_Update_Enter(t *testing.T) {
	t.Run("saves with valid input", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = "valid-api-key-12345"
		
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _ = input.Update(msg)
		
		assert.True(t, input.IsDone())
		assert.True(t, input.IsValid())
	})
	
	t.Run("does not save with empty input", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = ""
		
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _ = input.Update(msg)
		
		assert.False(t, input.IsDone())
	})
	
	t.Run("validates on enter", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.SetValidator(func(s string) error {
			return fmt.Errorf("invalid")
		})
		input.value = "test"
		
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, _ = input.Update(msg)
		
		assert.False(t, input.IsDone())
		assert.NotEmpty(t, input.validationError)
	})
}

func TestAPIKeyInput_Update_Backspace(t *testing.T) {
	input := NewAPIKeyInput()
	input.value = "test"
	
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	_, _ = input.Update(msg)
	
	assert.Equal(t, "tes", input.value)
}

func TestAPIKeyInput_Update_CtrlU(t *testing.T) {
	input := NewAPIKeyInput()
	input.value = "test value"
	
	msg := tea.KeyMsg{Type: tea.KeyCtrlU}
	_, _ = input.Update(msg)
	
	assert.Empty(t, input.value)
}

func TestAPIKeyInput_Update_Tab(t *testing.T) {
	input := NewAPIKeyInput()
	
	// Default is hidden
	assert.False(t, input.showValue)
	
	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, _ = input.Update(msg)
	
	assert.True(t, input.showValue)
	
	// Toggle back
	_, _ = input.Update(msg)
	assert.False(t, input.showValue)
}

func TestAPIKeyInput_Update_Runes(t *testing.T) {
	input := NewAPIKeyInput()
	
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, _ = input.Update(msg)
	
	assert.Equal(t, "a", input.value)
	
	// Add more characters
	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b', 'c'}}
	_, _ = input.Update(msg2)
	
	assert.Equal(t, "abc", input.value)
}

func TestAPIKeyInput_formatValue(t *testing.T) {
	t.Run("shows placeholder when empty", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = ""
		
		result := input.formatValue()
		assert.Equal(t, "Enter API key...", result)
	})
	
	t.Run("masks value when masked and not showing", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = "secret123"
		input.masked = true
		input.showValue = false
		
		result := input.formatValue()
		assert.Equal(t, "•••••••••▌", result)
	})
	
	t.Run("shows value when not masked", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = "visible123"
		input.masked = false
		
		result := input.formatValue()
		assert.Equal(t, "visible123▌", result)
	})
	
	t.Run("shows value when showValue is true", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = "shown123"
		input.masked = true
		input.showValue = true
		
		result := input.formatValue()
		assert.Equal(t, "shown123▌", result)
	})
}

func TestAPIKeyInput_calculateStrength(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{"empty", "", 0},
		{"short", "abc", 0},
		{"medium length", "abcdefgh", 1}, // >= 8 chars
		{"long", "abcdefghijklmnop", 2},  // >= 16 chars
		{"very long", strings.Repeat("a", 32), 3}, // >= 32 chars
		{"with mixed case", "Abcdefgh", 2}, // length + mixed case
		// Note: "abc123!@" is only 8 chars, so it gets 1 for length, but needs >= 16 for second length point
		// Mixed case gives +1, but digits+special needs both, so 2 total
		{"with numbers and special", "abc123!@", 2},
		// "Abcdef123!@" - length >= 8 but < 16, so 1 point
		// Mixed case: +1, digits+special: +1 = 3 total
		{"strong", "Abcdef123!@", 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := NewAPIKeyInput()
			input.value = tt.value
			
			strength := input.calculateStrength()
			assert.Equal(t, tt.expected, strength)
		})
	}
}

func TestAPIKeyInput_renderStrength(t *testing.T) {
	input := NewAPIKeyInput()
	
	// Test with empty value (strength 0)
	input.value = ""
	result := input.renderStrength()
	assert.Contains(t, result, "Strength:")
	// Empty value gives strength 0 which shows "Weak" with all empty bars
	
	// Test with strong password
	input.value = "Abcdefghijklmnop123!@#"
	result = input.renderStrength()
	assert.Contains(t, result, "Strength:")
	assert.Contains(t, result, "█")
}

func TestAPIKeyInput_View(t *testing.T) {
	t.Run("renders basic view", func(t *testing.T) {
		input := NewAPIKeyInput()
		view := input.View()
		
		assert.Contains(t, view, "API Key")
		assert.Contains(t, view, "Enter API key...")
	})
	
	t.Run("renders with value", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.value = "test123"
		
		view := input.View()
		
		assert.Contains(t, view, "Tab to show/hide")
	})
	
	t.Run("renders with strength", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.SetShowStrength(true)
		input.value = "Abcdefgh123!"
		
		view := input.View()
		
		assert.Contains(t, view, "Strength:")
	})
}

func TestAPIKeyInput_GetValue(t *testing.T) {
	input := NewAPIKeyInput()
	input.value = "my-secret-key"
	
	assert.Equal(t, "my-secret-key", input.GetValue())
}

func TestAPIKeyInput_IsDone(t *testing.T) {
	input := NewAPIKeyInput()
	assert.False(t, input.IsDone())
	
	input.done = true
	assert.True(t, input.IsDone())
}

func TestAPIKeyInput_IsCancelled(t *testing.T) {
	input := NewAPIKeyInput()
	assert.False(t, input.IsCancelled())
	
	input.cancelled = true
	assert.True(t, input.IsCancelled())
}

func TestAPIKeyInput_IsValid(t *testing.T) {
	input := NewAPIKeyInput()
	assert.False(t, input.IsValid())
	
	input.isValid = true
	assert.True(t, input.IsValid())
}

func TestAPIKeyInput_Reset(t *testing.T) {
	input := NewAPIKeyInput()
	input.value = "test"
	input.done = true
	input.cancelled = true
	input.validationError = "error"
	input.isValid = true
	input.showValue = true
	
	input.Reset()
	
	assert.Empty(t, input.value)
	assert.False(t, input.done)
	assert.False(t, input.cancelled)
	assert.Empty(t, input.validationError)
	assert.False(t, input.isValid)
	assert.False(t, input.showValue)
}

func TestAPIKeyInput_SetValue(t *testing.T) {
	input := NewAPIKeyInput()
	input.SetValue("new-value")
	
	assert.Equal(t, "new-value", input.value)
}

func TestAPIKeyInput_HasValue(t *testing.T) {
	input := NewAPIKeyInput()
	assert.False(t, input.HasValue())
	
	input.value = "has value"
	assert.True(t, input.HasValue())
}

func TestAPIKeyInput_GetValidationError(t *testing.T) {
	input := NewAPIKeyInput()
	assert.Empty(t, input.GetValidationError())
	
	input.validationError = "some error"
	assert.Equal(t, "some error", input.GetValidationError())
}

func TestMaskValue(t *testing.T) {
	t.Run("short value", func(t *testing.T) {
		result := MaskValue("abc")
		assert.Equal(t, "•••", result)
	})
	
	t.Run("long value", func(t *testing.T) {
		result := MaskValue("abcdefghijklmnopqrstuvwxyz")
		// First 4 chars + mask + last 4 chars
		// 26 chars total: abcd + (26-8=18 dots) + wxyz
		assert.Equal(t, "abcd••••••••••••••••••wxyz", result)
	})
	
	t.Run("exactly 8 chars", func(t *testing.T) {
		result := MaskValue("abcdefgh")
		assert.Equal(t, "••••••••", result)
	})
}

func TestValidateAPIKeyFormat(t *testing.T) {
	t.Run("empty key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})
	
	t.Run("short key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("short")
		assert.Error(t, err)
	})
	
	t.Run("valid key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("this-is-a-valid-key-with-enough-length")
		assert.NoError(t, err)
	})
	
	t.Run("incomplete anthropic key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-ant-short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Anthropic")
	})
	
	t.Run("valid anthropic key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-ant-api03-valid-key-with-sufficient-length")
		assert.NoError(t, err)
	})
	
	t.Run("incomplete openrouter key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-or-short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OpenRouter")
	})
	
	t.Run("valid openrouter key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-or-v1-valid-key-with-sufficient-length")
		assert.NoError(t, err)
	})
	
	t.Run("incomplete openai key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-short")
		assert.Error(t, err)
		// Short keys get "API key too short" before pattern matching
		assert.NotNil(t, err)
	})
	
	t.Run("valid openai key", func(t *testing.T) {
		err := ValidateAPIKeyFormat("sk-validopenaikeywithsufficientlength")
		assert.NoError(t, err)
	})
}

func TestGetAPIKeyHint(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"anthropic", "Format: sk-ant-..."},
		{"openai", "Format: sk-..."},
		{"openrouter", "Format: sk-or-..."},
		{"gemini", "Format: AI..."},
		{"bedrock", "AWS credentials required"},
		{"unknown", "Enter your API key"},
		{"", "Enter your API key"},
	}
	
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := GetAPIKeyHint(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKeyInput_ToMsg(t *testing.T) {
	input := NewAPIKeyInput()
	input.value = "test-value"
	input.isValid = true
	input.cancelled = false
	
	msg := input.ToMsg()
	
	assert.Equal(t, "test-value", msg.Value)
	assert.True(t, msg.Valid)
	assert.False(t, msg.Cancelled)
}

func TestAPIKeyInput_validate(t *testing.T) {
	t.Run("no validator", func(t *testing.T) {
		input := NewAPIKeyInput()
		err := input.validate()
		
		assert.NoError(t, err)
		assert.True(t, input.isValid)
	})
	
	t.Run("with passing validator", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.SetValidator(func(s string) error { return nil })
		
		err := input.validate()
		
		assert.NoError(t, err)
		assert.True(t, input.isValid)
	})
	
	t.Run("with failing validator", func(t *testing.T) {
		input := NewAPIKeyInput()
		input.SetValidator(func(s string) error { return fmt.Errorf("invalid") })
		
		err := input.validate()
		
		assert.Error(t, err)
		assert.False(t, input.isValid)
	})
}