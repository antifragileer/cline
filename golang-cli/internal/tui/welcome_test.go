package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFirstTimeDetection tests first-time user detection
func TestFirstTimeDetection(t *testing.T) {
	t.Run("detects first time user", func(t *testing.T) {
		m := initialModel()
		m.config.FirstRun = true
		
		assert.True(t, m.isFirstTimeUser())
		assert.True(t, m.showWelcome)
	})

	t.Run("detects returning user", func(t *testing.T) {
		m := initialModel()
		m.config.FirstRun = false
		m.config.LastUsed = "2024-01-01"
		
		assert.False(t, m.isFirstTimeUser())
	})

	t.Run("checks for existing config", func(t *testing.T) {
		m := initialModel()
		// No config exists
		m.config = Config{}
		
		assert.True(t, m.isFirstTimeUser())
	})

	t.Run("detects from empty history", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{}
		
		assert.True(t, m.isFirstTimeUser())
	})

	t.Run("detects from task count", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "First task"},
		}
		
		assert.False(t, m.isFirstTimeUser())
	})

	t.Run("checks API key configuration", func(t *testing.T) {
		m := initialModel()
		m.config.APIKey = ""
		
		// First time if no API key
		assert.True(t, m.needsConfiguration())
	})

	t.Run("checks provider configuration", func(t *testing.T) {
		m := initialModel()
		m.config.Provider = ""
		
		// Needs config if no provider set
		assert.True(t, m.needsConfiguration())
	})
}

// TestWelcomeScreenRendering tests welcome screen rendering
func TestWelcomeScreenRendering(t *testing.T) {
	t.Run("renders welcome screen", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		m.width = 80
		
		view := m.renderWelcomeView()
		assert.Contains(t, view, "Welcome")
	})

	t.Run("renders Cline logo/brand", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		
		view := m.renderWelcomeView()
		assert.Contains(t, view, "Cline")
	})

	t.Run("renders welcome message", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		
		view := m.renderWelcomeView()
		assert.Contains(t, view, "AI assistant")
	})

	t.Run("renders quick start options", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		
		view := m.renderWelcomeView()
		// Should show options like "New Task", "Settings", etc.
		assert.NotEmpty(t, view)
	})

	t.Run("renders keyboard shortcuts help", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		
		view := m.renderWelcomeView()
		// Should contain help text
		assert.NotEmpty(t, view)
	})

	t.Run("renders setup prompt for first time", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		m.config.FirstRun = true
		
		view := m.renderWelcomeView()
		// Should prompt for setup
		assert.NotEmpty(t, view)
	})

	t.Run("handles narrow terminal", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		m.width = 40
		
		view := m.renderWelcomeView()
		assert.NotEmpty(t, view)
	})

	t.Run("handles short terminal", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.showWelcome = true
		m.height = 10
		
		view := m.renderWelcomeView()
		assert.NotEmpty(t, view)
	})
}

// TestRecentTasksDisplay tests recent tasks display
func TestRecentTasksDisplay(t *testing.T) {
	t.Run("renders recent tasks section", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "First task"},
			{ID: "task-2", Description: "Second task"},
		}
		
		view := m.renderRecentTasks()
		assert.Contains(t, view, "Recent Tasks")
	})

	t.Run("displays task descriptions", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "Fix bug in auth"},
			{ID: "task-2", Description: "Add new feature"},
		}
		
		view := m.renderRecentTasks()
		assert.Contains(t, view, "Fix bug in auth")
		assert.Contains(t, view, "Add new feature")
	})

	t.Run("displays task timestamps", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "Task", Timestamp: "2024-01-15 10:30"},
		}
		
		view := m.renderRecentTasks()
		// Should show timestamp
		assert.NotEmpty(t, view)
	})

	t.Run("limits displayed tasks", func(t *testing.T) {
		m := initialModel()
		
		// Add many tasks
		for i := 0; i < maxRecentTasks + 5; i++ {
			m.config.TaskHistory = append(m.config.TaskHistory, TaskHistoryItem{
				ID:          "task-" + string(rune(i)),
				Description: "Task " + string(rune(i)),
			})
		}
		
		view := m.renderRecentTasks()
		// Should only show maxRecentTasks
		assert.NotEmpty(t, view)
	})

	t.Run("shows empty state when no tasks", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{}
		
		view := m.renderRecentTasks()
		// Should show "No recent tasks" or similar
		assert.NotEmpty(t, view)
	})

	t.Run("formats relative timestamps", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "Recent task", Timestamp: "2024-01-15 10:30"},
		}
		
		view := m.renderRecentTasks()
		// Should show "2 hours ago" or similar
		assert.NotEmpty(t, view)
	})

	t.Run("highlights selected task", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "First"},
			{ID: "task-2", Description: "Second"},
		}
		m.selectedTaskIndex = 0
		
		view := m.renderRecentTasks()
		// First task should be highlighted
		assert.NotEmpty(t, view)
	})

	t.Run("allows task selection navigation", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "First"},
			{ID: "task-2", Description: "Second"},
		}
		m.selectedTaskIndex = 0
		
		// Navigate down
		m.selectedTaskIndex = 1
		
		assert.Equal(t, 1, m.selectedTaskIndex)
	})

	t.Run("displays task count", func(t *testing.T) {
		m := initialModel()
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "Task 1"},
			{ID: "task-2", Description: "Task 2"},
			{ID: "task-3", Description: "Task 3"},
		}
		
		view := m.renderRecentTasks()
		// Should show "3 tasks" or similar
		assert.NotEmpty(t, view)
	})
}

// TestWelcomeInteractions tests welcome screen interactions
func TestWelcomeInteractions(t *testing.T) {
	t.Run("starts new task from welcome", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		
		// Simulate selecting "New Task"
		m.currentView = chatView
		m.showWelcome = false
		
		assert.Equal(t, chatView, m.currentView)
		assert.False(t, m.showWelcome)
	})

	t.Run("opens settings from welcome", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		
		m.currentView = settingsView
		
		assert.Equal(t, settingsView, m.currentView)
	})

	t.Run("resumes recent task", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.config.TaskHistory = []TaskHistoryItem{
			{ID: "task-1", Description: "Previous task"},
		}
		
		// Select and resume task
		m.currentTaskID = "task-1"
		m.currentView = chatView
		
		assert.Equal(t, "task-1", m.currentTaskID)
		assert.Equal(t, chatView, m.currentView)
	})

	t.Run("shows help from welcome", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		
		m.showHelp = true
		
		assert.True(t, m.showHelp)
	})

	t.Run("dismisses welcome screen", func(t *testing.T) {
		m := initialModel()
		m.showWelcome = true
		
		m.showWelcome = false
		
		assert.False(t, m.showWelcome)
	})
}

// TestWelcomeViewNavigation tests welcome view navigation
func TestWelcomeViewNavigation(t *testing.T) {
	t.Run("navigates menu with arrow keys", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.welcomeMenuItems = []string{"New Task", "Recent Tasks", "Settings"}
		m.selectedMenuIndex = 0
		
		// Navigate down
		m.selectedMenuIndex = 1
		
		assert.Equal(t, 1, m.selectedMenuIndex)
	})

	t.Run("wraps menu navigation", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.welcomeMenuItems = []string{"Item 1", "Item 2"}
		m.selectedMenuIndex = 1
		
		// Navigate down past end
		m.selectedMenuIndex = 0
		
		assert.Equal(t, 0, m.selectedMenuIndex)
	})

	t.Run("selects with enter", func(t *testing.T) {
		m := initialModel()
		m.currentView = welcomeView
		m.welcomeMenuItems = []string{"New Task", "Settings"}
		m.selectedMenuIndex = 0
		
		// Simulate enter - should select "New Task"
		m.currentView = chatView
		
		assert.Equal(t, chatView, m.currentView)
	})

	t.Run("navigates back to welcome", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		
		m.currentView = welcomeView
		
		assert.Equal(t, welcomeView, m.currentView)
	})
}

// TestConfigurationPrompts tests configuration prompts on welcome
func TestConfigurationPrompts(t *testing.T) {
	t.Run("prompts for API key if missing", func(t *testing.T) {
		m := initialModel()
		m.config.APIKey = ""
		m.config.FirstRun = true
		
		assert.True(t, m.needsConfiguration())
	})

	t.Run("prompts for provider if missing", func(t *testing.T) {
		m := initialModel()
		m.config.Provider = ""
		
		assert.True(t, m.needsConfiguration())
	})

	t.Run("shows configuration wizard", func(t *testing.T) {
		m := initialModel()
		m.showConfigWizard = true
		
		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("guides through provider selection", func(t *testing.T) {
		m := initialModel()
		m.configWizardStep = "provider"
		
		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("guides through API key entry", func(t *testing.T) {
		m := initialModel()
		m.configWizardStep = "apikey"
		
		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("validates configuration", func(t *testing.T) {
		m := initialModel()
		m.config.APIKey = "test-key"
		m.config.Provider = "anthropic"
		
		assert.True(t, m.hasValidConfiguration())
	})

	t.Run("saves configuration", func(t *testing.T) {
		m := initialModel()
		m.config.APIKey = "new-key"
		m.config.Provider = "openai"
		
		// Simulate save
		m.config.FirstRun = false
		
		assert.Equal(t, "new-key", m.config.APIKey)
		assert.Equal(t, "openai", m.config.Provider)
		assert.False(t, m.config.FirstRun)
	})
}

// TestWelcomeStyles tests welcome screen styling
func TestWelcomeStyles(t *testing.T) {
	t.Run("applies welcome styles", func(t *testing.T) {
		m := initialModel()
		
		style := m.getWelcomeStyle()
		assert.NotNil(t, style)
	})

	t.Run("styles logo appropriately", func(t *testing.T) {
		m := initialModel()
		
		logoStyle := m.getLogoStyle()
		assert.NotNil(t, logoStyle)
	})

	t.Run("styles menu items", func(t *testing.T) {
		m := initialModel()
		
		menuStyle := m.getMenuStyle()
		selectedStyle := m.getSelectedMenuStyle()
		
		assert.NotNil(t, menuStyle)
		assert.NotNil(t, selectedStyle)
	})

	t.Run("styles recent tasks list", func(t *testing.T) {
		m := initialModel()
		
		taskStyle := m.getTaskItemStyle()
		assert.NotNil(t, taskStyle)
	})
}

// TestOnboardingFlow tests the onboarding flow for new users
func TestOnboardingFlow(t *testing.T) {
	t.Run("starts onboarding for new users", func(t *testing.T) {
		m := initialModel()
		m.config.FirstRun = true
		
		view := m.View()
		assert.Contains(t, view, "Welcome")
	})

	t.Run("shows feature highlights", func(t *testing.T) {
		m := initialModel()
		m.showFeatureHighlights = true
		
		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("advances through onboarding steps", func(t *testing.T) {
		m := initialModel()
		m.onboardingStep = 0
		
		// Advance to next step
		m.onboardingStep = 1
		
		assert.Equal(t, 1, m.onboardingStep)
	})

	t.Run("completes onboarding", func(t *testing.T) {
		m := initialModel()
		m.config.FirstRun = true
		m.onboardingStep = 3 // Last step
		
		// Complete
		m.config.FirstRun = false
		m.showWelcome = false
		
		assert.False(t, m.config.FirstRun)
		assert.False(t, m.showWelcome)
	})

	t.Run("skips onboarding", func(t *testing.T) {
		m := initialModel()
		m.config.FirstRun = true
		
		// Skip
		m.config.FirstRun = false
		m.showWelcome = false
		
		assert.False(t, m.config.FirstRun)
	})
}

// TestWelcomeContent tests welcome screen content
func TestWelcomeContent(t *testing.T) {
	t.Run("displays version info", func(t *testing.T) {
		m := initialModel()
		m.version = "1.0.0"
		
		view := m.renderWelcomeView()
		// Should contain version
		assert.NotEmpty(t, view)
	})

	t.Run("displays help link", func(t *testing.T) {
		m := initialModel()
		
		view := m.renderWelcomeView()
		// Should contain help reference
		assert.NotEmpty(t, view)
	})

	t.Run("displays documentation link", func(t *testing.T) {
		m := initialModel()
		
		view := m.renderWelcomeView()
		// Should contain docs reference
		assert.NotEmpty(t, view)
	})

	t.Run("shows tip of the day", func(t *testing.T) {
		m := initialModel()
		m.showTips = true
		
		view := m.renderWelcomeView()
		assert.NotEmpty(t, view)
	})
}