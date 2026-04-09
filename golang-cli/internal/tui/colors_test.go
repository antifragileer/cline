package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetModeColor(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"act mode", "act", PrimaryBlue},
		{"plan mode", "plan", PlanYellow},
		{"unknown mode", "unknown", PrimaryBlue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetModeColor(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetModeSelectionColor(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"act mode", "act", SelectionBlue},
		{"plan mode", "plan", PlanYellow},
		{"unknown mode", "unknown", SelectionBlue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetModeSelectionColor(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColorConstants(t *testing.T) {
	t.Run("color constants are defined", func(t *testing.T) {
		// Verify all color constants are defined and valid
		assert.NotEmpty(t, PrimaryBlue)
		assert.NotEmpty(t, SelectionBlue)
		assert.NotEmpty(t, PlanYellow)
		assert.NotEmpty(t, White)
		assert.NotEmpty(t, Gray)
		assert.NotEmpty(t, DimGray)
		assert.NotEmpty(t, SuccessGreen)
		assert.NotEmpty(t, ErrorRed)
		assert.NotEmpty(t, WarningAmber)
		assert.NotEmpty(t, DarkBackground)
	})

	t.Run("mode colors are different", func(t *testing.T) {
		actColor := GetModeColor("act")
		planColor := GetModeColor("plan")

		assert.NotEqual(t, actColor, planColor)
	})

	t.Run("selection colors match mode colors", func(t *testing.T) {
		// Plan mode uses same color for mode and selection
		assert.Equal(t, PlanYellow, GetModeSelectionColor("plan"))
		// Act mode uses different colors
		assert.Equal(t, SelectionBlue, GetModeSelectionColor("act"))
	})
}