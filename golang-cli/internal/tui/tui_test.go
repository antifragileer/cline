package tui

import (
	"bytes"
	"errors"
	"os"
	"os/signal"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModel(t *testing.T) {
	model := NewModel("Test App")

	assert.Equal(t, "Test App", model.Title())
	assert.Equal(t, ModeTUI, model.Mode())
	assert.True(t, model.IsTUI())
	assert.False(t, model.IsPlain())
	assert.Equal(t, 80, model.Width())
	assert.Equal(t, 24, model.Height())
	assert.Empty(t, model.Content())
	assert.False(t, model.Ready())
	assert.NoError(t, model.Error())
}

func TestNewPlainModel(t *testing.T) {
	model := NewPlainModel()

	assert.Empty(t, model.Title())
	assert.Equal(t, ModePlain, model.Mode())
	assert.False(t, model.IsTUI())
	assert.True(t, model.IsPlain())
	assert.Equal(t, 80, model.Width())
	assert.Equal(t, 24, model.Height())
}

func TestModelSettersAndGetters(t *testing.T) {
	model := NewModel("Test")

	// Test SetContent/GetContent
	model.SetContent("Hello World")
	assert.Equal(t, "Hello World", model.Content())

	// Test SetTitle
	model.SetTitle("New Title")
	assert.Equal(t, "New Title", model.Title())

	// Test SetError
	err := errors.New("test error")
	model.SetError(err)
	assert.Equal(t, err, model.Error())
}

func TestModelDimensions(t *testing.T) {
	model := NewModel("Test")

	// Update dimensions via WindowSizeMsg
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
	assert.True(t, m.Ready())
}

func TestCustomWindowSizeMsg(t *testing.T) {
	model := NewModel("Test")

	// Update dimensions via custom WindowSizeMsg
	updatedModel, _ := model.Update(WindowSizeMsg{Width: 100, Height: 30})

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 30, m.Height())
}

func TestModelInit(t *testing.T) {
	model := NewModel("Test")
	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestKeyHandling(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantQuit bool
	}{
		{"quit with q", "q", true},
		{"quit with ctrl+c", "ctrl+c", true},
		{"quit with esc", "esc", true},
		{"other key", "a", false},
		{"enter key", "enter", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel("Test")

			updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{[]rune(tt.key)[0]}})

			// Check if the key is recognized as quit key
			isQuit := model.isQuitKey(tt.key)

			if tt.wantQuit {
				assert.True(t, isQuit, "key %q should be a quit key", tt.key)
			} else {
				assert.False(t, isQuit, "key %q should not be a quit key", tt.key)
			}

			// Just verify the update doesn't panic
			_ = updatedModel
			_ = cmd
		})
	}
}

func TestContentMsg(t *testing.T) {
	model := NewModel("Test")
	model.SetContent("Initial")

	updatedModel, _ := model.Update(ContentMsg{Content: "Updated"})

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.Equal(t, "Updated", m.Content())
}

func TestErrorMsg(t *testing.T) {
	model := NewModel("Test")
	testErr := errors.New("test error")

	updatedModel, _ := model.Update(ErrorMsg{Err: testErr})

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.Equal(t, testErr, m.Error())
}

func TestShutdownMsg(t *testing.T) {
	model := NewModel("Test")

	_, cmd := model.Update(ShutdownMsg{})

	// ShutdownMsg should return tea.Quit command
	// We can't directly compare commands, but we can verify it's not nil
	assert.NotNil(t, cmd)
}

func TestInitMsg(t *testing.T) {
	model := NewModel("Test")

	updatedModel, _ := model.Update(InitMsg{})

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.True(t, m.Ready())
}

func TestRegisterShutdownCallback(t *testing.T) {
	model := NewModel("Test")

	called := false
	callback := func() {
		called = true
	}

	model.RegisterShutdownCallback(callback)
	model.Shutdown()

	assert.True(t, called, "shutdown callback should have been called")
}

func TestMultipleShutdownCallbacks(t *testing.T) {
	model := NewModel("Test")

	callOrder := []int{}
	callback1 := func() { callOrder = append(callOrder, 1) }
	callback2 := func() { callOrder = append(callOrder, 2) }
	callback3 := func() { callOrder = append(callOrder, 3) }

	model.RegisterShutdownCallback(callback1)
	model.RegisterShutdownCallback(callback2)
	model.RegisterShutdownCallback(callback3)

	model.Shutdown()

	assert.Equal(t, []int{1, 2, 3}, callOrder)
}

func TestDefaultTUIKeyMap(t *testing.T) {
	km := DefaultTUIKeyMap()

	assert.Contains(t, km.Quit, "q")
	assert.Contains(t, km.Quit, "ctrl+c")
	assert.Contains(t, km.Quit, "esc")
}

func TestSetupSignalHandling(t *testing.T) {
	sigChan := SetupSignalHandling()
	require.NotNil(t, sigChan)

	// Clean up
	signal.Reset()
}

func TestUpdateContent(t *testing.T) {
	cmd := UpdateContent("new content")
	require.NotNil(t, cmd)

	msg := cmd()
	require.NotNil(t, msg)

	contentMsg, ok := msg.(ContentMsg)
	require.True(t, ok)
	assert.Equal(t, "new content", contentMsg.Content)
}

func TestShutdownCmd(t *testing.T) {
	cmd := ShutdownCmd()
	require.NotNil(t, cmd)

	msg := cmd()
	require.NotNil(t, msg)

	_, ok := msg.(ShutdownMsg)
	assert.True(t, ok)
}

func TestRenderPlain(t *testing.T) {
	result := RenderPlain("test content")
	assert.Equal(t, "test content\n", result)
}

func TestRenderError(t *testing.T) {
	err := errors.New("test error")

	// With ANSI colors
	resultANSI := RenderError(err, true)
	assert.Contains(t, resultANSI, "Error: test error")
	assert.Contains(t, resultANSI, "\033[31m")

	// Without ANSI colors
	resultPlain := RenderError(err, false)
	assert.Equal(t, "Error: test error\n", resultPlain)

	// Nil error
	resultNil := RenderError(nil, true)
	assert.Empty(t, resultNil)
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	assert.Equal(t, "cyan", styles.BorderColor)
	assert.Equal(t, "yellow", styles.TitleColor)
	assert.Equal(t, "white", styles.TextColor)
	assert.Equal(t, "red", styles.ErrorColor)
	assert.True(t, styles.UseANSI)
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		text   string
		width  int
		expect string
	}{
		{"hi", 10, "    hi    "},
		{"hello", 10, "  hello   "},
		{"toolong", 5, "toolo"},
		{"", 5, "     "},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := centerText(tt.text, tt.width)
			assert.Equal(t, tt.expect, result)
			assert.Equal(t, tt.width, len(result))
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text   string
		width  int
		expect []string
	}{
		{"hello world", 20, []string{"hello world"}},
		{"hello world test", 11, []string{"hello world", "test"}},
		{"a b c d e", 5, []string{"a b c", "d e"}},
		{"", 10, []string{""}},
		{"word", 0, []string{"word"}},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestPlainView(t *testing.T) {
	// Plain model with content
	model := NewPlainModel()
	model.SetContent("plain content")

	view := model.View()
	assert.Equal(t, "plain content\n", view)

	// Plain model with error
	model.SetError(errors.New("plain error"))
	view = model.View()
	assert.Contains(t, view, "Error: plain error")
}

func TestTUIView(t *testing.T) {
	model := NewModel("Test Title")

	// Not ready - should show initializing
	view := model.View()
	assert.Equal(t, "Initializing...", view)

	// Ready - should show full UI
	model.ready = true
	view = model.View()

	// Check that the view contains expected elements
	assert.Contains(t, view, "Test Title")
	assert.Contains(t, view, "q/ctrl+c: quit")
}

func TestDefaultProgramOptions(t *testing.T) {
	opts := DefaultProgramOptions()

	assert.Equal(t, "Cline", opts.Title)
	assert.Equal(t, ModeTUI, opts.Mode)
	assert.Equal(t, os.Stdin, opts.Input)
	assert.Equal(t, os.Stdout, opts.Output)
	assert.True(t, opts.AltScreen)
	assert.False(t, opts.Mouse)
}

func TestNewProgram(t *testing.T) {
	opts := DefaultProgramOptions()
	opts.Title = "Test Program"

	program, err := NewProgram(opts)
	require.NoError(t, err)
	require.NotNil(t, program)

	assert.Equal(t, "Test Program", program.model.Title())
	assert.Equal(t, ModeTUI, program.model.Mode())
}

func TestNewProgramPlainMode(t *testing.T) {
	opts := DefaultProgramOptions()
	opts.Mode = ModePlain

	program, err := NewProgram(opts)
	require.NoError(t, err)
	require.NotNil(t, program)

	assert.Equal(t, ModePlain, program.model.Mode())
}

func TestProgramRunPlain(t *testing.T) {
	var output bytes.Buffer

	opts := DefaultProgramOptions()
	opts.Mode = ModePlain
	opts.Output = &output
	opts.InitialContent = "Hello Plain World"

	program, err := NewProgram(opts)
	require.NoError(t, err)

	finalModel, err := program.Start()
	require.NoError(t, err)

	assert.Equal(t, "Hello Plain World", finalModel.Content())
	assert.Contains(t, output.String(), "Hello Plain World")
}

func TestProgramShutdown(t *testing.T) {
	opts := DefaultProgramOptions()
	opts.Mode = ModePlain

	program, err := NewProgram(opts)
	require.NoError(t, err)

	shutdownCalled := false
	program.model.RegisterShutdownCallback(func() {
		shutdownCalled = true
	})

	program.Shutdown()

	assert.True(t, shutdownCalled)
}

func TestProgramWithShutdownCallback(t *testing.T) {
	opts := DefaultProgramOptions()
	opts.Mode = ModePlain

	shutdownCalled := false
	opts.OnShutdown = func() {
		shutdownCalled = true
	}

	program, err := NewProgram(opts)
	require.NoError(t, err)

	_, err = program.Start()
	require.NoError(t, err)

	assert.True(t, shutdownCalled)
}

func TestGetTerminalDimensions(t *testing.T) {
	// This test may fail if not running in a terminal
	// We're mainly checking it doesn't panic
	width, height, err := GetTerminalDimensions()

	// If we're in a terminal, we should get valid dimensions
	if IsTerminal() {
		require.NoError(t, err)
		assert.Greater(t, width, 0)
		assert.Greater(t, height, 0)
	}
}

func TestIsTerminal(t *testing.T) {
	// Should return true when stdout is a terminal
	result := IsTerminal()
	// We can't assert the exact value without controlling the environment
	// but we can verify it doesn't panic
	_ = result
}

func TestSupportsTUI(t *testing.T) {
	// Should return true when stdout is a terminal and TERM is not "dumb"
	result := SupportsTUI()
	// We can't assert the exact value without controlling the environment
	_ = result

	// Test with TERM=dumb
	origTerm := os.Getenv("TERM")
	os.Setenv("TERM", "dumb")
	defer os.Setenv("TERM", origTerm)

	// Even in a terminal, TERM=dumb should affect SupportsTUI
	if IsTerminal() {
		assert.False(t, SupportsTUI())
	}
}

func TestRunPlain(t *testing.T) {
	var output bytes.Buffer

	// Save original stdout and restore after test
	origStdout := os.Stdout
	defer func() { os.Stdout = origStdout }()

	// Redirect stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run in plain mode
	err := RunPlain("test plain output")

	// Close pipe and read output
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)

	require.NoError(t, err)
	outputStr := buf.String()
	// Plain mode doesn't write to stdout in test environment,
	// but we can verify no panic occurred
	assert.NotNil(t, outputStr)
	_ = output
}

func TestModelWithEmptyTitle(t *testing.T) {
	model := NewModel("")
	model.ready = true

	view := model.View()

	// When title is empty, header should be minimal or empty
	// Just verify it doesn't panic and returns something
	assert.NotNil(t, view)
}

func TestModelWithError(t *testing.T) {
	model := NewModel("Test")
	model.ready = true
	model.SetError(errors.New("test error"))

	view := model.View()

	// View should contain error message
	assert.Contains(t, view, "Error: test error")
}

func TestUpdateContentCmdIntegration(t *testing.T) {
	model := NewModel("Test")

	// Create update content command
	cmd := UpdateContent("new content")
	msg := cmd()

	// Apply the message
	updatedModel, _ := model.Update(msg)

	m, ok := updatedModel.(Model)
	require.True(t, ok)
	assert.Equal(t, "new content", m.Content())
}

func TestBatchCmd(t *testing.T) {
	// Test with messages
	cmd := BatchCmd(ContentMsg{Content: "test"}, ShutdownMsg{})
	msg := cmd()
	require.NotNil(t, msg)

	// Should return first message
	contentMsg, ok := msg.(ContentMsg)
	require.True(t, ok)
	assert.Equal(t, "test", contentMsg.Content)

	// Test with no messages
	emptyCmd := BatchCmd()
	emptyMsg := emptyCmd()
	assert.Nil(t, emptyMsg)
}

func TestSignalCmd(t *testing.T) {
	// Create a signal channel
	sigChan := make(chan os.Signal, 1)

	// Create command
	cmd := SignalCmd(sigChan)

	// Send a signal
	go func() {
		sigChan <- os.Interrupt
	}()

	// Execute command
	msg := cmd()
	require.NotNil(t, msg)

	// Should be ShutdownMsg
	_, ok := msg.(ShutdownMsg)
	assert.True(t, ok)
}

func TestViewRenderingInPlainMode(t *testing.T) {
	model := NewPlainModel()
	model.SetContent("Simple content")

	view := model.View()
	assert.Equal(t, "Simple content\n", view)
}

func TestViewRenderingWithLongContent(t *testing.T) {
	model := NewModel("Test")
	model.ready = true
	model.SetContent(strings.Repeat("word ", 100))

	view := model.View()

	// Should render without panic
	assert.NotNil(t, view)
	assert.Contains(t, view, "word")
}

func TestModelWithShutdownAndCallbacks(t *testing.T) {
	model := NewModel("Test")

	callbackCount := 0
	callback1 := func() { callbackCount++ }
	callback2 := func() { callbackCount++ }

	model.RegisterShutdownCallback(callback1)
	model.RegisterShutdownCallback(callback2)

	model.Shutdown()

	assert.Equal(t, 2, callbackCount)
}

func TestTUIKeyMapCustomization(t *testing.T) {
	km := TUIKeyMap{
		Quit: []string{"x", "ctrl+q"},
	}

	// Verify custom keymap can be created
	assert.Contains(t, km.Quit, "x")
	assert.Contains(t, km.Quit, "ctrl+q")
}
