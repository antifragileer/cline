package tui

import (
	"testing"
)

func TestGetModeColor(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"plan", PlanYellow},
		{"act", PrimaryBlue},
		{"", PrimaryBlue},
		{"unknown", PrimaryBlue},
		{"PLAN", PrimaryBlue}, // case sensitive
	}

	for _, test := range tests {
		result := GetModeColor(test.mode)
		if result != test.expected {
			t.Errorf("GetModeColor(%q) = %q, expected %q", test.mode, result, test.expected)
		}
	}
}

func TestGetModeSelectionColor(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{"plan", PlanYellow},
		{"act", SelectionBlue},
		{"", SelectionBlue},
		{"unknown", SelectionBlue},
		{"PLAN", SelectionBlue}, // case sensitive
	}

	for _, test := range tests {
		result := GetModeSelectionColor(test.mode)
		if result != test.expected {
			t.Errorf("GetModeSelectionColor(%q) = %q, expected %q", test.mode, result, test.expected)
		}
	}
}

func TestColorConstants(t *testing.T) {
	// Verify color constants are set
	if PrimaryBlue != "#B1B9F9" {
		t.Errorf("PrimaryBlue = %q, expected #B1B9F9", PrimaryBlue)
	}

	if SelectionBlue != "#00D9FF" {
		t.Errorf("SelectionBlue = %q, expected #00D9FF", SelectionBlue)
	}

	if PlanYellow != "#FFB000" {
		t.Errorf("PlanYellow = %q, expected #FFB000", PlanYellow)
	}

	if White != "#FFFFFF" {
		t.Errorf("White = %q, expected #FFFFFF", White)
	}

	if Gray != "#808080" {
		t.Errorf("Gray = %q, expected #808080", Gray)
	}

	if DimGray != "#505050" {
		t.Errorf("DimGray = %q, expected #505050", DimGray)
	}

	if SuccessGreen != "#00FF00" {
		t.Errorf("SuccessGreen = %q, expected #00FF00", SuccessGreen)
	}

	if ErrorRed != "#FF4444" {
		t.Errorf("ErrorRed = %q, expected #FF4444", ErrorRed)
	}

	if WarningAmber != "#FFB000" {
		t.Errorf("WarningAmber = %q, expected #FFB000", WarningAmber)
	}

	if DarkBackground != "#1a1a1a" {
		t.Errorf("DarkBackground = %q, expected #1a1a1a", DarkBackground)
	}
}