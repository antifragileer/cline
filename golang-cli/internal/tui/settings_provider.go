// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProviderOption represents an API provider option.
type ProviderOption struct {
	ID          string
	Name        string
	AuthType    string // "oauth" or "apikey"
	Description string
}

// ProviderSelector handles provider selection UI.
type ProviderSelector struct {
	providers   []ProviderOption
	cursor      int
	selected    string
	styles      ProviderSelectorStyles
	width       int
	height      int
	done        bool
	cancelled   bool
	isSelecting bool
}

// ProviderSelectorStyles holds styles for the provider selector.
type ProviderSelectorStyles struct {
	containerStyle   lipgloss.Style
	titleStyle       lipgloss.Style
	providerStyle    lipgloss.Style
	selectedStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
	helpStyle        lipgloss.Style
	oauthBadgeStyle  lipgloss.Style
	apikeyBadgeStyle lipgloss.Style
}

// DefaultProviderSelectorStyles returns default styles.
func DefaultProviderSelectorStyles() ProviderSelectorStyles {
	return ProviderSelectorStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		providerStyle: lipgloss.NewStyle().
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

		oauthBadgeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			Bold(true),

		apikeyBadgeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(WarningAmber)).
			Bold(true),
	}
}

// NewProviderSelector creates a new provider selector.
func NewProviderSelector() *ProviderSelector {
	return &ProviderSelector{
		providers: GetDefaultProviders(),
		styles:    DefaultProviderSelectorStyles(),
		cursor:    0,
	}
}

// NewProviderSelectorWithProviders creates a selector with custom providers.
func NewProviderSelectorWithProviders(providers []ProviderOption) *ProviderSelector {
	return &ProviderSelector{
		providers: providers,
		styles:    DefaultProviderSelectorStyles(),
		cursor:    0,
	}
}

// GetDefaultProviders returns the default list of providers.
func GetDefaultProviders() []ProviderOption {
	return []ProviderOption{
		{
			ID:          "cline",
			Name:        "Cline",
			AuthType:    "oauth",
			Description: "Cline hosted models with account connection",
		},
		{
			ID:          "anthropic",
			Name:        "Anthropic",
			AuthType:    "apikey",
			Description: "Direct Anthropic API access",
		},
		{
			ID:          "openai",
			Name:        "OpenAI",
			AuthType:    "apikey",
			Description: "OpenAI GPT models",
		},
		{
			ID:          "openrouter",
			Name:        "OpenRouter",
			AuthType:    "apikey",
			Description: "Multi-provider model access",
		},
		{
			ID:          "bedrock",
			Name:        "AWS Bedrock",
			AuthType:    "apikey",
			Description: "Amazon Web Services Bedrock",
		},
		{
			ID:          "gemini",
			Name:        "Google Gemini",
			AuthType:    "apikey",
			Description: "Google AI models",
		},
		{
			ID:          "ollama",
			Name:        "Ollama",
			AuthType:    "local",
			Description: "Local model hosting",
		},
		{
			ID:          "lmstudio",
			Name:        "LM Studio",
			AuthType:    "local",
			Description: "Local model hosting",
		},
	}
}

// SetDimensions sets the terminal dimensions.
func (ps *ProviderSelector) SetDimensions(width, height int) {
	ps.width = width
	ps.height = height
}

// SetSelected sets the currently selected provider.
func (ps *ProviderSelector) SetSelected(providerID string) {
	ps.selected = providerID
	// Update cursor to match selected
	for i, p := range ps.providers {
		if p.ID == providerID {
			ps.cursor = i
			break
		}
	}
}

// Init initializes the selector.
func (ps ProviderSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the selector.
func (ps *ProviderSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			ps.cancelled = true
			return ps, nil

		case tea.KeyEnter:
			if ps.cursor >= 0 && ps.cursor < len(ps.providers) {
				ps.selected = ps.providers[ps.cursor].ID
				ps.done = true
			}
			return ps, nil

		case tea.KeyUp:
			if ps.cursor > 0 {
				ps.cursor--
			} else {
				ps.cursor = len(ps.providers) - 1
			}
			return ps, nil

		case tea.KeyDown:
			if ps.cursor < len(ps.providers)-1 {
				ps.cursor++
			} else {
				ps.cursor = 0
			}
			return ps, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "q", "Q":
				ps.cancelled = true
				return ps, nil
			case "k":
				if ps.cursor > 0 {
					ps.cursor--
				}
				return ps, nil
			case "j":
				if ps.cursor < len(ps.providers)-1 {
					ps.cursor++
				}
				return ps, nil
			}
		}
	}

	return ps, nil
}

// View renders the provider selector.
func (ps ProviderSelector) View() string {
	var content strings.Builder

	// Title
	content.WriteString(ps.styles.titleStyle.Render("Select API Provider"))
	content.WriteString("\n\n")

	// Provider list
	for i, provider := range ps.providers {
		isSelected := i == ps.cursor
		line := ps.renderProvider(provider, isSelected)

		if isSelected {
			content.WriteString(ps.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(ps.styles.providerStyle.Render(line))
		}
		content.WriteString("\n")

		// Show description for selected item
		if isSelected {
			content.WriteString(ps.styles.descriptionStyle.Render(provider.Description))
			content.WriteString("\n")
		}
	}

	// Help
	content.WriteString("\n")
	content.WriteString(ps.styles.helpStyle.Render("↑↓/j/k navigate • Enter select • Esc/q cancel"))

	return ps.styles.containerStyle.Render(content.String())
}

// renderProvider renders a single provider option.
func (ps *ProviderSelector) renderProvider(provider ProviderOption, selected bool) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, "▶")
	} else {
		parts = append(parts, " ")
	}

	// Provider name
	parts = append(parts, provider.Name)

	// Auth type badge
	badge := ps.formatAuthBadge(provider.AuthType)
	parts = append(parts, badge)

	// Current indicator
	if provider.ID == ps.selected {
		parts = append(parts, ps.styles.selectedStyle.Render("(current)"))
	}

	return strings.Join(parts, " ")
}

// formatAuthBadge formats the authentication type badge.
func (ps *ProviderSelector) formatAuthBadge(authType string) string {
	switch authType {
	case "oauth":
		return ps.styles.oauthBadgeStyle.Render("[OAuth]")
	case "apikey":
		return ps.styles.apikeyBadgeStyle.Render("[API Key]")
	case "local":
		return ps.styles.descriptionStyle.Render("[Local]")
	default:
		return ps.styles.descriptionStyle.Render("[" + authType + "]")
	}
}

// GetSelected returns the selected provider ID.
func (ps *ProviderSelector) GetSelected() string {
	return ps.selected
}

// IsDone returns true if selection is complete.
func (ps *ProviderSelector) IsDone() bool {
	return ps.done
}

// IsCancelled returns true if selection was cancelled.
func (ps *ProviderSelector) IsCancelled() bool {
	return ps.cancelled
}

// GetSelectedProvider returns the full provider info for the selection.
func (ps *ProviderSelector) GetSelectedProvider() (ProviderOption, bool) {
	for _, p := range ps.providers {
		if p.ID == ps.selected {
			return p, true
		}
	}
	return ProviderOption{}, false
}

// GetSelectedProviderAtCursor returns the provider at current cursor position.
func (ps *ProviderSelector) GetSelectedProviderAtCursor() (ProviderOption, bool) {
	if ps.cursor >= 0 && ps.cursor < len(ps.providers) {
		return ps.providers[ps.cursor], true
	}
	return ProviderOption{}, false
}

// RequiresAPIKey returns true if the selected provider requires an API key.
func (ps *ProviderSelector) RequiresAPIKey() bool {
	if provider, ok := ps.GetSelectedProvider(); ok {
		return provider.AuthType == "apikey"
	}
	return false
}

// RequiresOAuth returns true if the selected provider requires OAuth.
func (ps *ProviderSelector) RequiresOAuth() bool {
	if provider, ok := ps.GetSelectedProvider(); ok {
		return provider.AuthType == "oauth"
	}
	return false
}

// Reset resets the selector state.
func (ps *ProviderSelector) Reset() {
	ps.done = false
	ps.cancelled = false
	ps.cursor = 0
}

// SetProviders updates the provider list.
func (ps *ProviderSelector) SetProviders(providers []ProviderOption) {
	ps.providers = providers
	if ps.cursor >= len(providers) {
		ps.cursor = 0
	}
}

// GetProviderCount returns the number of providers.
func (ps *ProviderSelector) GetProviderCount() int {
	return len(ps.providers)
}

// GetProviderAt returns the provider at the given index.
func (ps *ProviderSelector) GetProviderAt(index int) (ProviderOption, bool) {
	if index >= 0 && index < len(ps.providers) {
		return ps.providers[index], true
	}
	return ProviderOption{}, false
}

// ProviderSelectionMsg is sent when provider selection completes.
type ProviderSelectionMsg struct {
	ProviderID string
	Provider   ProviderOption
	Cancelled  bool
}

// ToMsg converts the selector state to a message.
func (ps *ProviderSelector) ToMsg() ProviderSelectionMsg {
	provider, _ := ps.GetSelectedProvider()
	return ProviderSelectionMsg{
		ProviderID: ps.selected,
		Provider:   provider,
		Cancelled:  ps.cancelled,
	}
}

// ValidateProvider checks if a provider ID is valid.
func ValidateProvider(providerID string) bool {
	providers := GetDefaultProviders()
	for _, p := range providers {
		if p.ID == providerID {
			return true
		}
	}
	return false
}

// GetProviderByID returns a provider by ID.
func GetProviderByID(providerID string) (ProviderOption, error) {
	providers := GetDefaultProviders()
	for _, p := range providers {
		if p.ID == providerID {
			return p, nil
		}
	}
	return ProviderOption{}, fmt.Errorf("provider not found: %s", providerID)
}
