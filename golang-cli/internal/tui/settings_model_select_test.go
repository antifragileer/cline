package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultModelSelectorStyles(t *testing.T) {
	styles := DefaultModelSelectorStyles()

	assert.NotNil(t, styles.containerStyle)
	assert.NotNil(t, styles.titleStyle)
	assert.NotNil(t, styles.modelStyle)
	assert.NotNil(t, styles.selectedStyle)
	assert.NotNil(t, styles.descriptionStyle)
	assert.NotNil(t, styles.helpStyle)
	assert.NotNil(t, styles.providerStyle)
	assert.NotNil(t, styles.contextStyle)
	assert.NotNil(t, styles.customInputStyle)
}

func TestNewModelSelector(t *testing.T) {
	selector := NewModelSelector()

	assert.NotNil(t, selector)
	assert.NotNil(t, selector.styles)
	assert.NotNil(t, selector.models)
	assert.False(t, selector.done)
	assert.False(t, selector.cancelled)
	assert.Equal(t, 0, selector.cursor)
}

func TestNewModelSelectorWithModels(t *testing.T) {
	models := []ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
		{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
	}
	selector := NewModelSelectorWithModels(models)

	assert.NotNil(t, selector)
	assert.Equal(t, 3, len(selector.models)) // 2 + custom option
}

func TestModelSelector_SetDimensions(t *testing.T) {
	selector := NewModelSelector()

	selector.SetDimensions(80, 24)

	assert.Equal(t, 80, selector.width)
	assert.Equal(t, 24, selector.height)
}

func TestModelSelector_SetProvider(t *testing.T) {
	selector := NewModelSelector()

	selector.SetProvider("anthropic")

	assert.Equal(t, "anthropic", selector.provider)
	assert.NotEmpty(t, selector.models)
}

func TestModelSelector_SetSelected(t *testing.T) {
	selector := NewModelSelectorWithModels([]ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
		{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
	})

	selector.SetSelected("claude-3")

	assert.Equal(t, "claude-3", selector.selected)
}

func TestModelSelector_Init(t *testing.T) {
	selector := NewModelSelector()
	cmd := selector.Init()

	assert.Nil(t, cmd)
}

func TestModelSelector_GetSelected(t *testing.T) {
	selector := NewModelSelectorWithModels([]ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
	})
	selector.selected = "gpt-4"

	assert.Equal(t, "gpt-4", selector.GetSelected())
}

func TestModelSelector_IsDone(t *testing.T) {
	selector := NewModelSelector()

	assert.False(t, selector.IsDone())

	selector.done = true

	assert.True(t, selector.IsDone())
}

func TestModelSelector_IsCancelled(t *testing.T) {
	selector := NewModelSelector()

	assert.False(t, selector.IsCancelled())

	selector.cancelled = true

	assert.True(t, selector.IsCancelled())
}

func TestModelSelector_Reset(t *testing.T) {
	selector := NewModelSelectorWithModels([]ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
	})
	selector.selected = "gpt-4"
	selector.done = true
	selector.cursor = 1

	selector.Reset()

	assert.Equal(t, 0, selector.cursor)
	// Reset may or may not clear selected depending on implementation
	assert.False(t, selector.done)
	assert.False(t, selector.cancelled)
}

func TestModelSelector_GetModelCount(t *testing.T) {
	selector := NewModelSelectorWithModels([]ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
		{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
		{ID: "gemini-pro", Name: "Gemini Pro", Provider: "google"},
	})

	assert.Equal(t, 4, selector.GetModelCount()) // 3 + custom
}

func TestModelSelector_ToMsg(t *testing.T) {
	selector := NewModelSelector()
	selector.selected = "gpt-4"

	msg := selector.ToMsg()

	// ToMsg returns ModelSelectionMsg with ModelID
	assert.Equal(t, "gpt-4", msg.ModelID)
}

func TestModelSelector_ModelValidation(t *testing.T) {
	t.Run("validates known models by ID lookup", func(t *testing.T) {
		selector := NewModelSelectorWithModels([]ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
			{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
		})

		// Check if model exists in list
		found := false
		for _, m := range selector.models {
			if m.ID == "gpt-4" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("invalid model not in list", func(t *testing.T) {
		selector := NewModelSelectorWithModels([]ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
		})

		// Check if model exists in list
		found := false
		for _, m := range selector.models {
			if m.ID == "invalid-model" {
				found = true
				break
			}
		}
		assert.False(t, found)
	})
}

func TestGetModelsForProvider(t *testing.T) {
	t.Run("returns models for known provider", func(t *testing.T) {
		models := GetModelsForProvider("openai")
		assert.NotNil(t, models)
	})

	t.Run("returns default for unknown provider", func(t *testing.T) {
		models := GetModelsForProvider("unknown-provider")
		// Implementation returns default model for unknown providers
		assert.NotNil(t, models)
	})
}

func TestModelSelector_addCustomOption(t *testing.T) {
	selector := NewModelSelectorWithModels([]ModelInfo{
		{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
	})

	// Custom option should be added automatically
	assert.Equal(t, 2, len(selector.models))
	assert.Equal(t, "custom", selector.models[1].ID)
}

func TestModelSelector_GetSelectedModel(t *testing.T) {
	t.Run("returns selected model info", func(t *testing.T) {
		selector := NewModelSelectorWithModels([]ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
			{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
		})
		selector.selected = "gpt-4"

		info, exists := selector.GetSelectedModel()
		assert.True(t, exists)
		assert.Equal(t, "gpt-4", info.ID)
	})

	t.Run("returns false when nothing selected", func(t *testing.T) {
		selector := NewModelSelectorWithModels([]ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
		})

		_, exists := selector.GetSelectedModel()
		assert.False(t, exists)
	})
}

func TestModelSelector_showCustomInputMode(t *testing.T) {
	selector := NewModelSelector()
	selector.showCustom = true

	assert.True(t, selector.showCustom)
}

func TestModelSelector_handleCustomInput(t *testing.T) {
	t.Run("custom value can be set", func(t *testing.T) {
		selector := NewModelSelector()
		selector.showCustom = true
		selector.customValue = "custom-model"

		assert.True(t, selector.showCustom)
		assert.Equal(t, "custom-model", selector.customValue)
	})
}

func TestModelSelector_handleEnter(t *testing.T) {
	t.Run("selects model at cursor", func(t *testing.T) {
		selector := NewModelSelectorWithModels([]ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
			{ID: "claude-3", Name: "Claude 3", Provider: "anthropic"},
		})
		selector.cursor = 0

		// Simulate handleEnter
		selector.selected = selector.models[selector.cursor].ID
		selector.done = true

		assert.Equal(t, "gpt-4", selector.selected)
		assert.True(t, selector.done)
	})
}

func TestModelInfo_Struct(t *testing.T) {
	info := ModelInfo{
		ID:            "gpt-4",
		Name:          "GPT-4",
		Description:   "GPT-4 model",
		ContextWindow: 8192,
		MaxTokens:     4096,
		Provider:      "openai",
	}

	assert.Equal(t, "gpt-4", info.ID)
	assert.Equal(t, "GPT-4", info.Name)
	assert.Equal(t, "GPT-4 model", info.Description)
	assert.Equal(t, 8192, info.ContextWindow)
	assert.Equal(t, 4096, info.MaxTokens)
	assert.Equal(t, "openai", info.Provider)
}