// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// APIKeyInput handles API key input UI with masking and validation.
type APIKeyInput struct {
	value           string
	placeholder     string
	label           string
	description     string
	masked          bool
	showValue       bool
	styles          APIKeyInputStyles
	width           int
	height          int
	done            bool
	cancelled       bool
	validationError string
	isValid         bool
	validator       func(string) error
	showStrength    bool
}

// APIKeyInputStyles holds styles for the API key input.
type APIKeyInputStyles struct {
	containerStyle      lipgloss.Style
	titleStyle          lipgloss.Style
	labelStyle          lipgloss.Style
	inputStyle          lipgloss.Style
	inputFocusedStyle   lipgloss.Style
	maskedStyle         lipgloss.Style
	visibleStyle        lipgloss.Style
	helpStyle           lipgloss.Style
	errorStyle          lipgloss.Style
	successStyle        lipgloss.Style
	descriptionStyle    lipgloss.Style
	strengthWeakStyle   lipgloss.Style
	strengthFairStyle   lipgloss.Style
	strengthGoodStyle   lipgloss.Style
	strengthStrongStyle lipgloss.Style
}

// DefaultAPIKeyInputStyles returns default styles.
func DefaultAPIKeyInputStyles() APIKeyInputStyles {
	return APIKeyInputStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			Bold(true).
			MarginBottom(1),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(Gray)).
			Padding(0, 1).
			Width(50),

		inputFocusedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SelectionBlue)).
			Padding(0, 1).
			Width(50),

		maskedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		visibleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)).
			MarginTop(1),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			MarginTop(1),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true).
			MarginBottom(1),

		strengthWeakStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)).
			Bold(true),

		strengthFairStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(WarningAmber)).
			Bold(true),

		strengthGoodStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true),

		strengthStrongStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			Bold(true),
	}
}

// NewAPIKeyInput creates a new API key input.
func NewAPIKeyInput() *APIKeyInput {
	return &APIKeyInput{
		styles:      DefaultAPIKeyInputStyles(),
		masked:      true,
		placeholder: "Enter API key...",
		label:       "API Key",
		description: "Your API key will be stored securely",
		isValid:     false,
	}
}

// NewAPIKeyInputWithLabel creates a new API key input with custom label.
func NewAPIKeyInputWithLabel(label, description string) *APIKeyInput {
	input := NewAPIKeyInput()
	input.label = label
	input.description = description
	return input
}

// SetDimensions sets the terminal dimensions.
func (aki *APIKeyInput) SetDimensions(width, height int) {
	aki.width = width
	aki.height = height
}

// SetPlaceholder sets the input placeholder.
func (aki *APIKeyInput) SetPlaceholder(placeholder string) {
	aki.placeholder = placeholder
}

// SetLabel sets the input label.
func (aki *APIKeyInput) SetLabel(label string) {
	aki.label = label
}

// SetDescription sets the input description.
func (aki *APIKeyInput) SetDescription(description string) {
	aki.description = description
}

// SetMasked sets whether the input should be masked.
func (aki *APIKeyInput) SetMasked(masked bool) {
	aki.masked = masked
}

// SetValidator sets a custom validation function.
func (aki *APIKeyInput) SetValidator(validator func(string) error) {
	aki.validator = validator
}

// SetShowStrength enables password strength indicator.
func (aki *APIKeyInput) SetShowStrength(show bool) {
	aki.showStrength = show
}

// Init initializes the input.
func (aki APIKeyInput) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the input.
func (aki *APIKeyInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			aki.cancelled = true
			return aki, nil

		case tea.KeyEnter:
			if aki.value != "" {
				if err := aki.validate(); err != nil {
					aki.validationError = err.Error()
					return aki, nil
				}
				aki.done = true
				aki.validationError = ""
			}
			return aki, nil

		case tea.KeyBackspace:
			if len(aki.value) > 0 {
				aki.value = aki.value[:len(aki.value)-1]
				aki.validationError = ""
				aki.validate()
			}
			return aki, nil

		case tea.KeyCtrlU:
			// Clear input
			aki.value = ""
			aki.validationError = ""
			return aki, nil

		case tea.KeyTab:
			// Toggle visibility
			aki.showValue = !aki.showValue
			return aki, nil

		case tea.KeyRunes:
			aki.value += msg.String()
			aki.validationError = ""
			aki.validate()
			return aki, nil
		}
	}

	return aki, nil
}

// validate runs the validator if set.
func (aki *APIKeyInput) validate() error {
	if aki.validator != nil {
		err := aki.validator(aki.value)
		if err != nil {
			aki.isValid = false
			return err
		}
	}
	aki.isValid = true
	return nil
}

// View renders the API key input.
func (aki APIKeyInput) View() string {
	var content strings.Builder

	// Title
	content.WriteString(aki.styles.titleStyle.Render(aki.label))
	content.WriteString("\n")

	// Description
	if aki.description != "" {
		content.WriteString(aki.styles.descriptionStyle.Render(aki.description))
		content.WriteString("\n")
	}

	// Input field
	displayValue := aki.formatValue()
	inputStyle := aki.styles.inputStyle
	if aki.validationError == "" && aki.value != "" {
		inputStyle = aki.styles.inputFocusedStyle
	}
	content.WriteString(inputStyle.Render(displayValue))
	content.WriteString("\n")

	// Toggle hint
	if aki.value != "" {
		toggleHint := "Tab to show/hide"
		if aki.showValue {
			toggleHint = "Tab to hide"
		}
		content.WriteString(aki.styles.helpStyle.Render(toggleHint))
		content.WriteString("\n")
	}

	// Strength indicator
	if aki.showStrength && aki.value != "" {
		content.WriteString("\n")
		content.WriteString(aki.renderStrength())
		content.WriteString("\n")
	}

	// Validation error
	if aki.validationError != "" {
		content.WriteString("\n")
		content.WriteString(aki.styles.errorStyle.Render("✗ " + aki.validationError))
		content.WriteString("\n")
	}

	// Success indicator
	if aki.isValid && aki.value != "" && aki.validationError == "" {
		content.WriteString("\n")
		content.WriteString(aki.styles.successStyle.Render("✓ Valid API key format"))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	content.WriteString(aki.styles.helpStyle.Render("Enter to save • Esc to cancel • Ctrl+U to clear"))

	return aki.styles.containerStyle.Render(content.String())
}

// formatValue formats the input value for display.
func (aki *APIKeyInput) formatValue() string {
	if aki.value == "" {
		return aki.placeholder
	}

	if aki.masked && !aki.showValue {
		// Show masked value with cursor
		return strings.Repeat("•", len(aki.value)) + "▌"
	}

	return aki.value + "▌"
}

// renderStrength renders the password strength indicator.
func (aki *APIKeyInput) renderStrength() string {
	strength := aki.calculateStrength()

	var label string
	var style lipgloss.Style

	switch strength {
	case 0, 1:
		label = "Weak"
		style = aki.styles.strengthWeakStyle
	case 2:
		label = "Fair"
		style = aki.styles.strengthFairStyle
	case 3:
		label = "Good"
		style = aki.styles.strengthGoodStyle
	case 4, 5:
		label = "Strong"
		style = aki.styles.strengthStrongStyle
	}

	bar := strings.Repeat("█", strength) + strings.Repeat("░", 5-strength)
	return fmt.Sprintf("Strength: %s %s", bar, style.Render(label))
}

// calculateStrength calculates the password strength (0-5).
func (aki *APIKeyInput) calculateStrength() int {
	if len(aki.value) == 0 {
		return 0
	}

	strength := 0

	// Length check
	if len(aki.value) >= 8 {
		strength++
	}
	if len(aki.value) >= 16 {
		strength++
	}
	if len(aki.value) >= 32 {
		strength++
	}

	// Complexity checks
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, r := range aki.value {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if hasUpper && hasLower {
		strength++
	}
	if hasDigit && hasSpecial {
		strength++
	}

	if strength > 5 {
		strength = 5
	}

	return strength
}

// GetValue returns the entered API key.
func (aki *APIKeyInput) GetValue() string {
	return aki.value
}

// IsDone returns true if input is complete.
func (aki *APIKeyInput) IsDone() bool {
	return aki.done
}

// IsCancelled returns true if input was cancelled.
func (aki *APIKeyInput) IsCancelled() bool {
	return aki.cancelled
}

// IsValid returns true if the input is valid.
func (aki *APIKeyInput) IsValid() bool {
	return aki.isValid
}

// Reset resets the input state.
func (aki *APIKeyInput) Reset() {
	aki.value = ""
	aki.done = false
	aki.cancelled = false
	aki.validationError = ""
	aki.isValid = false
	aki.showValue = false
}

// SetValue sets the initial value.
func (aki *APIKeyInput) SetValue(value string) {
	aki.value = value
	aki.validate()
}

// HasValue returns true if there's input.
func (aki *APIKeyInput) HasValue() bool {
	return aki.value != ""
}

// GetValidationError returns the validation error if any.
func (aki *APIKeyInput) GetValidationError() string {
	return aki.validationError
}

// MaskValue masks the value for display.
func MaskValue(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("•", len(value))
	}
	return value[:4] + strings.Repeat("•", len(value)-8) + value[len(value)-4:]
}

// ValidateAPIKeyFormat performs basic API key format validation.
func ValidateAPIKeyFormat(key string) error {
	if len(key) < 10 {
		return fmt.Errorf("API key too short")
	}

	// Common API key patterns
	// Anthropic: sk-ant-...
	// OpenAI: sk-...
	// OpenRouter: sk-or-...

	if strings.HasPrefix(key, "sk-ant-") && len(key) < 20 {
		return fmt.Errorf("Anthropic API key appears incomplete")
	}

	if strings.HasPrefix(key, "sk-or-") && len(key) < 20 {
		return fmt.Errorf("OpenRouter API key appears incomplete")
	}

	if strings.HasPrefix(key, "sk-") && !strings.HasPrefix(key, "sk-ant-") && !strings.HasPrefix(key, "sk-or-") {
		if len(key) < 20 {
			return fmt.Errorf("OpenAI API key appears incomplete")
		}
	}

	return nil
}

// GetAPIKeyHint returns a hint about the expected format for a provider.
func GetAPIKeyHint(provider string) string {
	switch provider {
	case "anthropic":
		return "Format: sk-ant-..."
	case "openai":
		return "Format: sk-..."
	case "openrouter":
		return "Format: sk-or-..."
	case "gemini":
		return "Format: AI..."
	case "bedrock":
		return "AWS credentials required"
	default:
		return "Enter your API key"
	}
}

// APIKeyInputMsg is sent when API key input completes.
type APIKeyInputMsg struct {
	Value     string
	Valid     bool
	Cancelled bool
}

// ToMsg converts the input state to a message.
func (aki *APIKeyInput) ToMsg() APIKeyInputMsg {
	return APIKeyInputMsg{
		Value:     aki.value,
		Valid:     aki.isValid,
		Cancelled: aki.cancelled,
	}
}
