package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDefaultProviderSelectorStyles(t *testing.T) {
	styles := DefaultProviderSelectorStyles()

	assert.NotZero(t, styles.containerStyle)
	assert.NotZero(t, styles.titleStyle)
	assert.NotZero(t, styles.providerStyle)
	assert.NotZero(t, styles.selectedStyle)
	assert.NotZero(t, styles.descriptionStyle)
	assert.NotZero(t, styles.helpStyle)
	assert.NotZero(t, styles.oauthBadgeStyle)
	assert.NotZero(t, styles.apikeyBadgeStyle)
}

func TestNewProviderSelector(t *testing.T) {
	selector := NewProviderSelector()

	assert.NotNil(t, selector)
	assert.NotEmpty(t, selector.providers)
	assert.Equal(t, 0, selector.cursor)
	assert.False(t, selector.done)
	assert.False(t, selector.cancelled)
}

func TestNewProviderSelectorWithProviders(t *testing.T) {
	customProviders := []ProviderOption{
		{ID: "custom1", Name: "Custom 1", AuthType: "apikey"},
		{ID: "custom2", Name: "Custom 2", AuthType: "oauth"},
	}

	selector := NewProviderSelectorWithProviders(customProviders)

	assert.Equal(t, 2, len(selector.providers))
	assert.Equal(t, "custom1", selector.providers[0].ID)
}

func TestGetDefaultProviders(t *testing.T) {
	providers := GetDefaultProviders()

	assert.NotEmpty(t, providers)
	assert.GreaterOrEqual(t, len(providers), 5)

	// Check for expected providers
	providerIDs := make(map[string]bool)
	for _, p := range providers {
		providerIDs[p.ID] = true
	}

	assert.True(t, providerIDs["cline"])
	assert.True(t, providerIDs["anthropic"])
	assert.True(t, providerIDs["openai"])
	assert.True(t, providerIDs["openrouter"])
}

func TestProviderSelector_SetDimensions(t *testing.T) {
	selector := NewProviderSelector()
	selector.SetDimensions(100, 50)

	assert.Equal(t, 100, selector.width)
	assert.Equal(t, 50, selector.height)
}

func TestProviderSelector_SetSelected(t *testing.T) {
	selector := NewProviderSelector()

	selector.SetSelected("anthropic")

	assert.Equal(t, "anthropic", selector.selected)
	// Cursor should be updated to match
	assert.Equal(t, 1, selector.cursor) // anthropic is at index 1
}

func TestProviderSelector_SetSelected_NotFound(t *testing.T) {
	selector := NewProviderSelector()

	selector.SetSelected("nonexistent")

	assert.Equal(t, "nonexistent", selector.selected)
	// Cursor should remain unchanged
	assert.Equal(t, 0, selector.cursor)
}

func TestProviderSelector_Init(t *testing.T) {
	selector := NewProviderSelector()
	cmd := selector.Init()

	assert.Nil(t, cmd)
}

func TestProviderSelector_Update_Esc(t *testing.T) {
	selector := NewProviderSelector()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, _ = selector.Update(msg)

	assert.True(t, selector.IsCancelled())
}

func TestProviderSelector_Update_Enter(t *testing.T) {
	selector := NewProviderSelector()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = selector.Update(msg)

	assert.True(t, selector.IsDone())
	assert.Equal(t, "cline", selector.GetSelected()) // First provider is selected by default
}

func TestProviderSelector_Update_Up(t *testing.T) {
	selector := NewProviderSelector()
	selector.cursor = 2

	msg := tea.KeyMsg{Type: tea.KeyUp}
	_, _ = selector.Update(msg)

	assert.Equal(t, 1, selector.cursor)
}

func TestProviderSelector_Update_Up_Wrap(t *testing.T) {
	selector := NewProviderSelector()
	selector.cursor = 0

	msg := tea.KeyMsg{Type: tea.KeyUp}
	_, _ = selector.Update(msg)

	assert.Equal(t, len(selector.providers)-1, selector.cursor)
}

func TestProviderSelector_Update_Down(t *testing.T) {
	selector := NewProviderSelector()

	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, _ = selector.Update(msg)

	assert.Equal(t, 1, selector.cursor)
}

func TestProviderSelector_Update_Down_Wrap(t *testing.T) {
	selector := NewProviderSelector()
	selector.cursor = len(selector.providers) - 1

	msg := tea.KeyMsg{Type: tea.KeyDown}
	_, _ = selector.Update(msg)

	assert.Equal(t, 0, selector.cursor)
}

func TestProviderSelector_Update_Runes(t *testing.T) {
	t.Run("q key cancels", func(t *testing.T) {
		selector := NewProviderSelector()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, _ = selector.Update(msg)

		assert.True(t, selector.IsCancelled())
	})

	t.Run("Q key cancels", func(t *testing.T) {
		selector := NewProviderSelector()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}
		_, _ = selector.Update(msg)

		assert.True(t, selector.IsCancelled())
	})

	t.Run("k key moves up", func(t *testing.T) {
		selector := NewProviderSelector()
		selector.cursor = 2

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		_, _ = selector.Update(msg)

		assert.Equal(t, 1, selector.cursor)
	})

	t.Run("j key moves down", func(t *testing.T) {
		selector := NewProviderSelector()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		_, _ = selector.Update(msg)

		assert.Equal(t, 1, selector.cursor)
	})
}

func TestProviderSelector_View(t *testing.T) {
	selector := NewProviderSelector()

	view := selector.View()

	assert.Contains(t, view, "Select API Provider")
	assert.Contains(t, view, "Cline")
	assert.Contains(t, view, "Anthropic")
}

func TestProviderSelector_renderProvider(t *testing.T) {
	selector := NewProviderSelector()

	provider := ProviderOption{
		ID:       "test",
		Name:     "Test Provider",
		AuthType: "apikey",
	}

	t.Run("selected provider", func(t *testing.T) {
		result := selector.renderProvider(provider, true)
		assert.Contains(t, result, "▶")
		assert.Contains(t, result, "Test Provider")
	})

	t.Run("unselected provider", func(t *testing.T) {
		result := selector.renderProvider(provider, false)
		assert.Contains(t, result, "Test Provider")
	})
}

func TestProviderSelector_formatAuthBadge(t *testing.T) {
	selector := NewProviderSelector()

	t.Run("oauth badge", func(t *testing.T) {
		badge := selector.formatAuthBadge("oauth")
		assert.Contains(t, badge, "OAuth")
	})

	t.Run("apikey badge", func(t *testing.T) {
		badge := selector.formatAuthBadge("apikey")
		assert.Contains(t, badge, "API Key")
	})

	t.Run("local badge", func(t *testing.T) {
		badge := selector.formatAuthBadge("local")
		assert.Contains(t, badge, "Local")
	})

	t.Run("unknown badge", func(t *testing.T) {
		badge := selector.formatAuthBadge("unknown")
		assert.Contains(t, badge, "unknown")
	})
}

func TestProviderSelector_GetSelected(t *testing.T) {
	selector := NewProviderSelector()
	assert.Empty(t, selector.GetSelected())

	selector.selected = "anthropic"
	assert.Equal(t, "anthropic", selector.GetSelected())
}

func TestProviderSelector_IsDone(t *testing.T) {
	selector := NewProviderSelector()
	assert.False(t, selector.IsDone())

	selector.done = true
	assert.True(t, selector.IsDone())
}

func TestProviderSelector_IsCancelled(t *testing.T) {
	selector := NewProviderSelector()
	assert.False(t, selector.IsCancelled())

	selector.cancelled = true
	assert.True(t, selector.IsCancelled())
}

func TestProviderSelector_GetSelectedProvider(t *testing.T) {
	selector := NewProviderSelector()

	t.Run("returns provider when selected", func(t *testing.T) {
		selector.selected = "anthropic"
		provider, ok := selector.GetSelectedProvider()

		assert.True(t, ok)
		assert.Equal(t, "anthropic", provider.ID)
	})

	t.Run("returns false when not found", func(t *testing.T) {
		selector.selected = "nonexistent"
		_, ok := selector.GetSelectedProvider()

		assert.False(t, ok)
	})
}

func TestProviderSelector_GetSelectedProviderAtCursor(t *testing.T) {
	selector := NewProviderSelector()
	selector.cursor = 1 // Anthropic

	provider, ok := selector.GetSelectedProviderAtCursor()

	assert.True(t, ok)
	assert.Equal(t, "anthropic", provider.ID)
}

func TestProviderSelector_RequiresAPIKey(t *testing.T) {
	selector := NewProviderSelector()

	t.Run("returns true for apikey provider", func(t *testing.T) {
		selector.selected = "anthropic"
		assert.True(t, selector.RequiresAPIKey())
	})

	t.Run("returns false for oauth provider", func(t *testing.T) {
		selector.selected = "cline"
		assert.False(t, selector.RequiresAPIKey())
	})

	t.Run("returns false when no selection", func(t *testing.T) {
		selector.selected = ""
		assert.False(t, selector.RequiresAPIKey())
	})
}

func TestProviderSelector_RequiresOAuth(t *testing.T) {
	selector := NewProviderSelector()

	t.Run("returns true for oauth provider", func(t *testing.T) {
		selector.selected = "cline"
		assert.True(t, selector.RequiresOAuth())
	})

	t.Run("returns false for apikey provider", func(t *testing.T) {
		selector.selected = "anthropic"
		assert.False(t, selector.RequiresOAuth())
	})

	t.Run("returns false when no selection", func(t *testing.T) {
		selector.selected = ""
		assert.False(t, selector.RequiresOAuth())
	})
}

func TestProviderSelector_Reset(t *testing.T) {
	selector := NewProviderSelector()
	selector.done = true
	selector.cancelled = true
	selector.cursor = 3

	selector.Reset()

	assert.False(t, selector.done)
	assert.False(t, selector.cancelled)
	assert.Equal(t, 0, selector.cursor)
}

func TestProviderSelector_SetProviders(t *testing.T) {
	selector := NewProviderSelector()

	newProviders := []ProviderOption{
		{ID: "p1", Name: "Provider 1"},
		{ID: "p2", Name: "Provider 2"},
	}

	selector.SetProviders(newProviders)

	assert.Equal(t, 2, len(selector.providers))
}

func TestProviderSelector_SetProviders_ResetsCursor(t *testing.T) {
	selector := NewProviderSelector()
	selector.cursor = 10 // Beyond new provider list

	newProviders := []ProviderOption{
		{ID: "p1", Name: "Provider 1"},
	}

	selector.SetProviders(newProviders)

	assert.Equal(t, 0, selector.cursor)
}

func TestProviderSelector_GetProviderCount(t *testing.T) {
	selector := NewProviderSelector()

	count := selector.GetProviderCount()

	assert.Greater(t, count, 0)
	assert.Equal(t, len(selector.providers), count)
}

func TestProviderSelector_GetProviderAt(t *testing.T) {
	selector := NewProviderSelector()

	t.Run("returns provider at valid index", func(t *testing.T) {
		provider, ok := selector.GetProviderAt(0)
		assert.True(t, ok)
		assert.Equal(t, "cline", provider.ID)
	})

	t.Run("returns false for invalid index", func(t *testing.T) {
		_, ok := selector.GetProviderAt(999)
		assert.False(t, ok)
	})
}

func TestProviderSelector_ToMsg(t *testing.T) {
	selector := NewProviderSelector()
	selector.selected = "anthropic"

	msg := selector.ToMsg()

	assert.Equal(t, "anthropic", msg.ProviderID)
	assert.Equal(t, "anthropic", msg.Provider.ID)
	assert.False(t, msg.Cancelled)
}

func TestValidateProvider(t *testing.T) {
	t.Run("returns true for valid provider", func(t *testing.T) {
		assert.True(t, ValidateProvider("anthropic"))
		assert.True(t, ValidateProvider("openai"))
	})

	t.Run("returns false for invalid provider", func(t *testing.T) {
		assert.False(t, ValidateProvider("invalid"))
	})

	t.Run("returns false for empty string", func(t *testing.T) {
		assert.False(t, ValidateProvider(""))
	})
}

func TestGetProviderByID(t *testing.T) {
	t.Run("returns provider for valid ID", func(t *testing.T) {
		provider, err := GetProviderByID("anthropic")

		assert.NoError(t, err)
		assert.Equal(t, "anthropic", provider.ID)
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		_, err := GetProviderByID("invalid")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider not found")
	})
}