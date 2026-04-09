package parity

import (
	"fmt"
	"testing"
)

func TestCountScenarios(t *testing.T) {
	registry := NewExtendedRegistry()
	count := registry.Count()
	
	fmt.Printf("Total scenarios: %d\n", count)
	
	// Verify we have at least 50 scenarios as required by Phase 5
	if count < 50 {
		t.Errorf("Expected at least 50 scenarios, got %d", count)
	}
	
	// Print breakdown by category
	categories := make(map[Category]int)
	for _, s := range registry.GetAll() {
		categories[s.Category]++
	}
	
	fmt.Printf("Scenarios by category:\n")
	for cat, cnt := range categories {
		fmt.Printf("  - %s: %d\n", cat, cnt)
	}
}