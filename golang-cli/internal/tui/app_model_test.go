package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewAppModel(t *testing.T) {
	t.Run("creates app model with defaults", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		assert.NotNil(t, model)
		assert.NotNil(t, model.welcome)
		assert.NotNil(t, model.history)
		assert.NotNil(t, model.settings)
		assert.NotNil(t, model.chat)
		assert.Equal(t, AppStateWelcome, model.state)
		assert.Equal(t, "act", model.mode)
		assert.False(t, model.yolo)
		assert.False(t, model.quitting)
	})

	t.Run("initializes all sub-models", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		cmd := model.Init()

		// Init should return a batch command
		assert.NotNil(t, cmd)
	})
}

func TestAppModel_SetMode(t *testing.T) {
	t.Run("sets mode correctly", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		model.SetMode("plan")
		assert.Equal(t, "plan", model.mode)

		model.SetMode("act")
		assert.Equal(t, "act", model.mode)
	})
}

func TestAppModel_SetYolo(t *testing.T) {
	t.Run("sets yolo mode correctly", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		model.SetYolo(true)
		assert.True(t, model.yolo)

		model.SetYolo(false)
		assert.False(t, model.yolo)
	})
}

func TestAppModel_GetState(t *testing.T) {
	t.Run("returns current state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		assert.Equal(t, AppStateWelcome, model.GetState())

		model.SetState(AppStateChat)
		assert.Equal(t, AppStateChat, model.GetState())
	})
}

func TestAppModel_IsQuitting(t *testing.T) {
	t.Run("returns quitting state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		assert.False(t, model.IsQuitting())

		// Simulate quitting via key press
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		model.Update(msg)

		assert.True(t, model.IsQuitting())
	})
}

func TestAppModel_Update_WindowSize(t *testing.T) {
	t.Run("updates dimensions on window resize", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		updated, cmd := model.Update(msg)

		assert.Nil(t, cmd)
		assert.Equal(t, 100, model.width)
		assert.Equal(t, 50, model.height)
		assert.Equal(t, updated, model)
	})
}

func TestAppModel_Update_WelcomeResult(t *testing.T) {
	tests := []struct {
		name           string
		action         WelcomeAction
		expectedState  AppState
		expectedQuitting bool
	}{
		{
			name:          "new task transitions to chat",
			action:        ActionNewTask,
			expectedState: AppStateChat,
		},
		{
			name:          "continue task transitions to chat",
			action:        ActionContinueTask,
			expectedState: AppStateChat,
		},
		{
			name:          "history transitions to history",
			action:        ActionHistory,
			expectedState: AppStateHistory,
		},
		{
			name:          "settings transitions to settings",
			action:        ActionSettings,
			expectedState: AppStateSettings,
		},
		{
			name:             "quit sets quitting flag",
			action:           ActionQuit,
			expectedState:    AppStateWelcome,
			expectedQuitting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewAppModel(nil, nil, nil)
			msg := WelcomeResultMsg{Action: tt.action}

			_, cmd := model.Update(msg)

			if tt.expectedQuitting {
				assert.True(t, model.IsQuitting())
				assert.NotNil(t, cmd)
			} else {
				assert.Equal(t, tt.expectedState, model.GetState())
			}
		})
	}
}

func TestAppModel_Update_WelcomeResultWithInput(t *testing.T) {
	t.Run("new task with input sets chat input", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		msg := WelcomeResultMsg{
			Action:      ActionNewTask,
			InputValue:  "test task input",
		}
		model.Update(msg)

		assert.Equal(t, AppStateChat, model.GetState())
	})
}

func TestAppModel_GetSubModels(t *testing.T) {
	model := NewAppModel(nil, nil, nil)

	t.Run("GetChatModel returns chat model", func(t *testing.T) {
		assert.Equal(t, model.chat, model.GetChatModel())
	})

	t.Run("GetWelcomeModel returns welcome model", func(t *testing.T) {
		assert.Equal(t, model.welcome, model.GetWelcomeModel())
	})

	t.Run("GetHistoryModel returns history model", func(t *testing.T) {
		assert.Equal(t, model.history, model.GetHistoryModel())
	})

	t.Run("GetSettingsModel returns settings model", func(t *testing.T) {
		assert.Equal(t, model.settings, model.GetSettingsModel())
	})
}

func TestAppModel_View(t *testing.T) {
	t.Run("returns welcome view in welcome state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		view := model.View()
		assert.NotEmpty(t, view)
	})

	t.Run("returns chat view in chat state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		model.SetState(AppStateChat)
		view := model.View()
		assert.NotEmpty(t, view)
	})

	t.Run("returns history view in history state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		model.SetState(AppStateHistory)
		view := model.View()
		assert.NotEmpty(t, view)
	})

	t.Run("returns settings view in settings state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		model.SetState(AppStateSettings)
		view := model.View()
		assert.NotEmpty(t, view)
	})

	t.Run("returns unknown for invalid state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)
		model.SetState(AppState(999))
		view := model.View()
		assert.Contains(t, view, "Unknown")
	})
}

func TestAppModel_ShowDiff(t *testing.T) {
	t.Run("shows diff and transitions to diff state", func(t *testing.T) {
		model := NewAppModel(nil, nil, nil)

		model.ShowDiff("test.go", "+added line\n-removed line")

		assert.Equal(t, AppStateDiff, model.GetState())
	})
}

func TestDefaultAppStyles(t *testing.T) {
	t.Run("returns valid styles", func(t *testing.T) {
		styles := DefaultAppStyles()

		assert.NotZero(t, styles.containerStyle)
		assert.NotZero(t, styles.titleStyle)
	})
}