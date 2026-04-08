// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchableListItem_FilterValue(t *testing.T) {
	item := SearchableListItem{
		ID:          "test-1",
		Title:       "Test Item",
		Description: "A test item",
	}
	assert.Equal(t, "Test Item", item.FilterValue())
}

func TestNewSearchableListModel(t *testing.T) {
	items := []SearchableListItem{
		{ID: "1", Title: "Item 1", Description: "First item"},
		{ID: "2", Title: "Item 2", Description: "Second item"},
	}

	model := NewSearchableListModel("Test List", items)

	assert.NotNil(t, model)
	assert.Equal(t, items, model.items)
	assert.NotNil(t, model.list)
	assert.Equal(t, "Test List", model.list.Title)
	assert.True(t, model.list.FilteringEnabled())
}

func TestSearchableListModel_SetDimensions(t *testing.T) {
	items := []SearchableListItem{
		{ID: "1", Title: "Item 1"},
	}

	model := NewSearchableListModel("Test", items)
	model.SetDimensions(80, 24)

	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
}

func TestSearchableListModel_GetSelected(t *testing.T) {
	items := []SearchableListItem{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
	}

	model := NewSearchableListModel("Test", items)
	assert.Nil(t, model.GetSelected())

	// Simulate selection
	selectedItem := &items[0]
	model.selected = selectedItem
	assert.Equal(t, selectedItem, model.GetSelected())
}

func TestSearchableListModel_IsDone(t *testing.T) {
	items := []SearchableListItem{{ID: "1", Title: "Item 1"}}
	model := NewSearchableListModel("Test", items)

	assert.False(t, model.IsDone())

	model.done = true
	assert.True(t, model.IsDone())
}

func TestSearchableListModel_SetItems(t *testing.T) {
	initialItems := []SearchableListItem{
		{ID: "1", Title: "Item 1"},
	}

	model := NewSearchableListModel("Test", initialItems)
	assert.Equal(t, 1, len(model.items))

	newItems := []SearchableListItem{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}

	model.SetItems(newItems)
	assert.Equal(t, 3, len(model.items))
	assert.Equal(t, 3, len(model.list.Items()))
}

func TestConvertStringsToSearchableItems(t *testing.T) {
	strings := []string{"Apple", "Banana", "Cherry"}
	items := ConvertStringsToSearchableItems(strings)

	assert.Equal(t, 3, len(items))
	assert.Equal(t, "Apple", items[0].Title)
	assert.Equal(t, "item-0", items[0].ID)
	assert.Equal(t, "Banana", items[1].Title)
	assert.Equal(t, "Cherry", items[2].Title)
}

func TestConvertMapToSearchableItems(t *testing.T) {
	input := map[string]string{
		"1": "Apple|A red fruit",
		"2": "Banana",
		"3": "Cherry|A small red fruit",
	}

	items := ConvertMapToSearchableItems(input)
	assert.Equal(t, 3, len(items))

	// Check that items have correct structure
	for _, item := range items {
		switch item.ID {
		case "1":
			assert.Equal(t, "Apple", item.Title)
			assert.Equal(t, "A red fruit", item.Description)
		case "2":
			assert.Equal(t, "Banana", item.Title)
			assert.Equal(t, "", item.Description)
		case "3":
			assert.Equal(t, "Cherry", item.Title)
			assert.Equal(t, "A small red fruit", item.Description)
		}
	}
}

func TestFilterItems(t *testing.T) {
	items := []SearchableListItem{
		{ID: "1", Title: "Apple Pie", Description: "A delicious dessert"},
		{ID: "2", Title: "Banana Bread", Description: "Healthy snack"},
		{ID: "3", Title: "Cherry Tart", Description: "Sweet and sour"},
	}

	// Filter by title
	result := FilterItems(items, "apple")
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "Apple Pie", result[0].Title)

	// Filter by description
	result = FilterItems(items, "healthy")
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "Banana Bread", result[0].Title)

	// Filter by ID
	result = FilterItems(items, "3")
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "Cherry Tart", result[0].Title)

	// Case insensitive
	result = FilterItems(items, "CHERRY")
	assert.Equal(t, 1, len(result))

	// Empty query returns all
	result = FilterItems(items, "")
	assert.Equal(t, 3, len(result))

	// No match
	result = FilterItems(items, "xyz")
	assert.Equal(t, 0, len(result))
}

func TestSortItemsByTitle(t *testing.T) {
	items := []SearchableListItem{
		{ID: "3", Title: "Cherry"},
		{ID: "1", Title: "Apple"},
		{ID: "2", Title: "Banana"},
	}

	sorted := SortItemsByTitle(items)

	assert.Equal(t, "Apple", sorted[0].Title)
	assert.Equal(t, "Banana", sorted[1].Title)
	assert.Equal(t, "Cherry", sorted[2].Title)
}

func TestSortItemsByTitle_CaseInsensitive(t *testing.T) {
	items := []SearchableListItem{
		{ID: "1", Title: "zebra"},
		{ID: "2", Title: "Apple"},
		{ID: "3", Title: "banana"},
	}

	sorted := SortItemsByTitle(items)

	assert.Equal(t, "Apple", sorted[0].Title)
	assert.Equal(t, "banana", sorted[1].Title)
	assert.Equal(t, "zebra", sorted[2].Title)
}

func TestDefaultSearchableListStyles(t *testing.T) {
	styles := DefaultSearchableListStyles()

	assert.NotNil(t, styles.TitleStyle)
	assert.NotNil(t, styles.SelectedStyle)
	assert.NotNil(t, styles.FilterStyle)
	assert.NotNil(t, styles.HelpStyle)
	assert.NotNil(t, styles.DescriptionStyle)
}

func TestSearchableListModel_Init(t *testing.T) {
	items := []SearchableListItem{{ID: "1", Title: "Item 1"}}
	model := NewSearchableListModel("Test", items)

	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestSearchableListResult(t *testing.T) {
	// Test with selected item
	item := &SearchableListItem{ID: "1", Title: "Test"}
	result := SearchableListResult{
		Selected: item,
		Canceled: false,
		Error:    nil,
	}
	assert.Equal(t, item, result.Selected)
	assert.False(t, result.Canceled)

	// Test canceled result
	result = SearchableListResult{
		Selected: nil,
		Canceled: true,
	}
	assert.True(t, result.Canceled)
}
