// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModelInfo represents information about an AI model.
type ModelInfo struct {
	ID            string
	Name          string
	Description   string
	ContextWindow int
	MaxTokens     int
	Provider      string
}

// ModelSelector handles model selection UI.
type ModelSelector struct {
	models       []ModelInfo
	cursor       int
	selected     string
	provider     string
	styles       ModelSelectorStyles
	width        int
	height       int
	done         bool
	cancelled    bool
	showCustom   bool
	customValue  string
	isCustomMode bool
}

// ModelSelectorStyles holds styles for the model selector.
type ModelSelectorStyles struct {
	containerStyle   lipgloss.Style
	titleStyle       lipgloss.Style
	modelStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
	helpStyle        lipgloss.Style
	providerStyle    lipgloss.Style
	contextStyle     lipgloss.Style
	customInputStyle lipgloss.Style
}

// DefaultModelSelectorStyles returns default styles.
func DefaultModelSelectorStyles() ModelSelectorStyles {
	return ModelSelectorStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		modelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			PaddingLeft(2),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true).
			PaddingLeft(4),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),

		providerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true),

		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		customInputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SelectionBlue)).
			Padding(0, 1),
	}
}

// NewModelSelector creates a new model selector.
func NewModelSelector() *ModelSelector {
	return &ModelSelector{
		models: make([]ModelInfo, 0),
		styles: DefaultModelSelectorStyles(),
		cursor: 0,
	}
}

// NewModelSelectorWithModels creates a selector with predefined models.
func NewModelSelectorWithModels(models []ModelInfo) *ModelSelector {
	ms := &ModelSelector{
		models: models,
		styles: DefaultModelSelectorStyles(),
		cursor: 0,
	}
	ms.addCustomOption()
	return ms
}

// SetDimensions sets the terminal dimensions.
func (ms *ModelSelector) SetDimensions(width, height int) {
	ms.width = width
	ms.height = height
}

// SetProvider sets the provider and loads appropriate models.
func (ms *ModelSelector) SetProvider(provider string) {
	ms.provider = provider
	ms.models = GetModelsForProvider(provider)
	ms.addCustomOption()
	ms.cursor = 0
}

// addCustomOption adds the custom model option to the list.
func (ms *ModelSelector) addCustomOption() {
	// Add custom option at the end
	ms.models = append(ms.models, ModelInfo{
		ID:          "custom",
		Name:        "Custom Model",
		Description: "Enter a custom model ID",
		Provider:    ms.provider,
	})
}

// SetSelected sets the currently selected model.
func (ms *ModelSelector) SetSelected(modelID string) {
	ms.selected = modelID
	// Update cursor to match selected
	for i, m := range ms.models {
		if m.ID == modelID {
			ms.cursor = i
			break
		}
	}
}

// Init initializes the selector.
func (ms ModelSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the selector.
func (ms *ModelSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle custom input mode
	if ms.isCustomMode {
		return ms.handleCustomInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			ms.cancelled = true
			return ms, nil

		case tea.KeyEnter:
			return ms.handleEnter()

		case tea.KeyUp:
			if ms.cursor > 0 {
				ms.cursor--
			} else {
				ms.cursor = len(ms.models) - 1
			}
			return ms, nil

		case tea.KeyDown:
			if ms.cursor < len(ms.models)-1 {
				ms.cursor++
			} else {
				ms.cursor = 0
			}
			return ms, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "q", "Q":
				ms.cancelled = true
				return ms, nil
			case "k":
				if ms.cursor > 0 {
					ms.cursor--
				}
				return ms, nil
			case "j":
				if ms.cursor < len(ms.models)-1 {
					ms.cursor++
				}
				return ms, nil
			case "c", "C":
				// Jump to custom option
				for i, m := range ms.models {
					if m.ID == "custom" {
						ms.cursor = i
						break
					}
				}
				return ms, nil
			}
		}
	}

	return ms, nil
}

// handleEnter processes the Enter key press.
func (ms *ModelSelector) handleEnter() (tea.Model, tea.Cmd) {
	if ms.cursor >= 0 && ms.cursor < len(ms.models) {
		selected := ms.models[ms.cursor]

		if selected.ID == "custom" {
			ms.isCustomMode = true
			ms.customValue = ""
			return ms, nil
		}

		ms.selected = selected.ID
		ms.done = true
	}
	return ms, nil
}

// handleCustomInput handles input in custom model mode.
func (ms *ModelSelector) handleCustomInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			ms.isCustomMode = false
			ms.customValue = ""
			return ms, nil

		case tea.KeyEnter:
			if ms.customValue != "" {
				ms.selected = ms.customValue
				ms.done = true
				ms.isCustomMode = false
			}
			return ms, nil

		case tea.KeyBackspace:
			if len(ms.customValue) > 0 {
				ms.customValue = ms.customValue[:len(ms.customValue)-1]
			}
			return ms, nil

		case tea.KeyRunes:
			ms.customValue += msg.String()
			return ms, nil
		}
	}
	return ms, nil
}

// View renders the model selector.
func (ms ModelSelector) View() string {
	// Show custom input mode if active
	if ms.isCustomMode {
		return ms.renderCustomInput()
	}

	var content strings.Builder

	// Title with provider
	title := "Select Model"
	if ms.provider != "" {
		title = fmt.Sprintf("Select Model for %s", ms.provider)
	}
	content.WriteString(ms.styles.titleStyle.Render(title))
	content.WriteString("\n\n")

	// Model list
	for i, model := range ms.models {
		isSelected := i == ms.cursor
		line := ms.renderModel(model, isSelected)

		if isSelected {
			content.WriteString(ms.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(ms.styles.modelStyle.Render(line))
		}
		content.WriteString("\n")

		// Show description for selected item
		if isSelected && model.Description != "" {
			content.WriteString(ms.styles.descriptionStyle.Render(model.Description))
			content.WriteString("\n")

			// Show context window info if available
			if model.ContextWindow > 0 {
				contextInfo := fmt.Sprintf("Context: %dk tokens", model.ContextWindow/1000)
				content.WriteString(ms.styles.contextStyle.Render(contextInfo))
				content.WriteString("\n")
			}
		}
	}

	// Help
	content.WriteString("\n")
	content.WriteString(ms.styles.helpStyle.Render("↑↓/j/k navigate • Enter select • C for custom • Esc/q cancel"))

	return ms.styles.containerStyle.Render(content.String())
}

// renderCustomInput renders the custom model input mode.
func (ms ModelSelector) renderCustomInput() string {
	var content strings.Builder

	content.WriteString(ms.styles.titleStyle.Render("Enter Custom Model ID"))
	content.WriteString("\n\n")

	content.WriteString(ms.styles.customInputStyle.Render(ms.customValue + "▌"))
	content.WriteString("\n\n")

	content.WriteString(ms.styles.helpStyle.Render("Enter to confirm • Esc to cancel"))

	return ms.styles.containerStyle.Render(content.String())
}

// renderModel renders a single model option.
func (ms *ModelSelector) renderModel(model ModelInfo, selected bool) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, "▶")
	} else {
		parts = append(parts, " ")
	}

	// Model name
	parts = append(parts, model.Name)

	// Custom indicator
	if model.ID == "custom" {
		parts = append(parts, ms.styles.descriptionStyle.Render("[custom]"))
	}

	// Current indicator
	if model.ID == ms.selected {
		parts = append(parts, ms.styles.selectedStyle.Render("(current)"))
	}

	return strings.Join(parts, " ")
}

// GetSelected returns the selected model ID.
func (ms *ModelSelector) GetSelected() string {
	return ms.selected
}

// IsDone returns true if selection is complete.
func (ms *ModelSelector) IsDone() bool {
	return ms.done
}

// IsCancelled returns true if selection was cancelled.
func (ms *ModelSelector) IsCancelled() bool {
	return ms.cancelled
}

// IsCustom returns true if a custom model was selected.
func (ms *ModelSelector) IsCustom() bool {
	for _, m := range ms.models {
		if m.ID == ms.selected {
			return false
		}
	}
	return true // Not in the predefined list means it's custom
}

// GetSelectedModel returns the full model info for the selection.
func (ms *ModelSelector) GetSelectedModel() (ModelInfo, bool) {
	for _, m := range ms.models {
		if m.ID == ms.selected {
			return m, true
		}
	}
	// Return a custom model
	return ModelInfo{
		ID:       ms.selected,
		Name:     ms.selected,
		Provider: ms.provider,
	}, ms.selected != ""
}

// Reset resets the selector state.
func (ms *ModelSelector) Reset() {
	ms.done = false
	ms.cancelled = false
	ms.cursor = 0
	ms.customValue = ""
	ms.isCustomMode = false
}

// SetModels updates the model list.
func (ms *ModelSelector) SetModels(models []ModelInfo) {
	ms.models = models
	ms.addCustomOption()
	if ms.cursor >= len(ms.models) {
		ms.cursor = 0
	}
}

// GetModelCount returns the number of models.
func (ms *ModelSelector) GetModelCount() int {
	return len(ms.models)
}

// ModelSelectionMsg is sent when model selection completes.
type ModelSelectionMsg struct {
	ModelID   string
	Model     ModelInfo
	IsCustom  bool
	Cancelled bool
}

// ToMsg converts the selector state to a message.
func (ms *ModelSelector) ToMsg() ModelSelectionMsg {
	model, _ := ms.GetSelectedModel()
	return ModelSelectionMsg{
		ModelID:   ms.selected,
		Model:     model,
		IsCustom:  ms.IsCustom(),
		Cancelled: ms.cancelled,
	}
}

// GetModelsForProvider returns available models for a provider.
func GetModelsForProvider(provider string) []ModelInfo {
	switch provider {
	case "anthropic":
		return []ModelInfo{
			{
				ID:            "claude-sonnet-4-20250514",
				Name:          "Claude Sonnet 4",
				Description:   "Balanced performance and speed",
				ContextWindow: 200000,
				Provider:      provider,
			},
			{
				ID:            "claude-opus-4-20250514",
				Name:          "Claude Opus 4",
				Description:   "Maximum capability for complex tasks",
				ContextWindow: 200000,
				Provider:      provider,
			},
			{
				ID:            "claude-3-5-sonnet-20241022",
				Name:          "Claude 3.5 Sonnet",
				Description:   "Previous generation",
				ContextWindow: 200000,
				Provider:      provider,
			},
		}
	case "openai":
		return []ModelInfo{
			{
				ID:            "gpt-4o",
				Name:          "GPT-4o",
				Description:   "Latest multimodal model",
				ContextWindow: 128000,
				Provider:      provider,
			},
			{
				ID:            "gpt-4o-mini",
				Name:          "GPT-4o Mini",
				Description:   "Fast and cost-effective",
				ContextWindow: 128000,
				Provider:      provider,
			},
			{
				ID:            "gpt-4-turbo",
				Name:          "GPT-4 Turbo",
				Description:   "High capability",
				ContextWindow: 128000,
				Provider:      provider,
			},
			{
				ID:            "gpt-3.5-turbo",
				Name:          "GPT-3.5 Turbo",
				Description:   "Fast and efficient",
				ContextWindow: 16385,
				Provider:      provider,
			},
		}
	case "openrouter":
		return []ModelInfo{
			{
				ID:            "anthropic/claude-sonnet-4",
				Name:          "Claude Sonnet 4 (OR)",
				Description:   "Via OpenRouter",
				ContextWindow: 200000,
				Provider:      provider,
			},
			{
				ID:            "openai/gpt-4o",
				Name:          "GPT-4o (OR)",
				Description:   "Via OpenRouter",
				ContextWindow: 128000,
				Provider:      provider,
			},
			{
				ID:            "meta-llama/llama-3.1-405b",
				Name:          "Llama 3.1 405B",
				Description:   "Open source model",
				ContextWindow: 128000,
				Provider:      provider,
			},
		}
	case "bedrock":
		return []ModelInfo{
			{
				ID:            "anthropic.claude-sonnet-4-20250514-v1:0",
				Name:          "Claude Sonnet 4",
				Description:   "AWS Bedrock hosted",
				ContextWindow: 200000,
				Provider:      provider,
			},
			{
				ID:            "anthropic.claude-opus-4-20250514-v1:0",
				Name:          "Claude Opus 4",
				Description:   "AWS Bedrock hosted",
				ContextWindow: 200000,
				Provider:      provider,
			},
		}
	case "gemini":
		return []ModelInfo{
			{
				ID:            "gemini-1.5-pro",
				Name:          "Gemini 1.5 Pro",
				Description:   "Google AI model",
				ContextWindow: 2000000,
				Provider:      provider,
			},
			{
				ID:            "gemini-1.5-flash",
				Name:          "Gemini 1.5 Flash",
				Description:   "Fast Google AI model",
				ContextWindow: 1000000,
				Provider:      provider,
			},
		}
	case "ollama":
		return []ModelInfo{
			{
				ID:          "llama3.1",
				Name:        "Llama 3.1",
				Description: "Local Llama model",
				Provider:    provider,
			},
			{
				ID:          "codellama",
				Name:        "Code Llama",
				Description: "Code-specialized model",
				Provider:    provider,
			},
			{
				ID:          "mistral",
				Name:        "Mistral",
				Description: "Mistral model",
				Provider:    provider,
			},
		}
	case "lmstudio":
		return []ModelInfo{
			{
				ID:          "local",
				Name:        "Local Model",
				Description: "LM Studio hosted model",
				Provider:    provider,
			},
		}
	case "cline":
		return []ModelInfo{
			{
				ID:            "claude-sonnet-4",
				Name:          "Claude Sonnet 4",
				Description:   "Cline hosted - balanced performance",
				ContextWindow: 200000,
				Provider:      provider,
			},
			{
				ID:            "claude-opus-4",
				Name:          "Claude Opus 4",
				Description:   "Cline hosted - maximum capability",
				ContextWindow: 200000,
				Provider:      provider,
			},
		}
	default:
		return []ModelInfo{
			{
				ID:          "default",
				Name:        "Default Model",
				Description: "Provider default",
				Provider:    provider,
			},
		}
	}
}

// ValidateModel checks if a model ID is valid for a provider.
func ValidateModel(provider, modelID string) bool {
	models := GetModelsForProvider(provider)
	for _, m := range models {
		if m.ID == modelID {
			return true
		}
	}
	return true // Allow custom models
}
