// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(MessageTypeSay, "Hello")
	assert.Equal(t, MessageTypeSay, msg.Type)
	assert.Equal(t, "Hello", msg.Content)
	assert.NotEmpty(t, msg.ID)
	assert.False(t, msg.Partial)
	assert.NotZero(t, msg.Timestamp)
}

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("User message")
	assert.Equal(t, MessageTypeUser, msg.Type)
	assert.Equal(t, "User message", msg.Content)
}

func TestNewAIMessage(t *testing.T) {
	msg := NewAIMessage("AI message")
	assert.Equal(t, MessageTypeSay, msg.Type)
	assert.Equal(t, "AI message", msg.Content)
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("Error occurred")
	assert.Equal(t, MessageTypeError, msg.Type)
	assert.Equal(t, "Error occurred", msg.Content)
}

func TestNewToolUseMessage(t *testing.T) {
	input := map[string]interface{}{
		"file": "test.txt",
	}
	msg := NewToolUseMessage("read_file", input)
	assert.Equal(t, MessageTypeToolUse, msg.Type)
	assert.Equal(t, "read_file", msg.ToolName)
	assert.Equal(t, input, msg.ToolInput)
}

func TestNewToolResultMessage(t *testing.T) {
	msg := NewToolResultMessage("read_file", "file contents")
	assert.Equal(t, MessageTypeToolResult, msg.Type)
	assert.Equal(t, "read_file", msg.ToolName)
	assert.Equal(t, "file contents", msg.ToolResult)
}

func TestMessageSetters(t *testing.T) {
	msg := NewMessage(MessageTypeSay, "content")

	msg.SetPartial(true)
	assert.True(t, msg.Partial)

	msg.SetLanguage("go")
	assert.Equal(t, "go", msg.Language)

	msg.SetMetadata("key", "value")
	val, ok := msg.GetMetadata("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestMessageIsFromUser(t *testing.T) {
	userMsg := NewUserMessage("test")
	assert.True(t, userMsg.IsFromUser())
	assert.False(t, userMsg.IsFromAI())

	aiMsg := NewAIMessage("test")
	assert.False(t, aiMsg.IsFromUser())
	assert.True(t, aiMsg.IsFromAI())
}

func TestMessageIsToolRelated(t *testing.T) {
	toolUse := NewToolUseMessage("tool", nil)
	assert.True(t, toolUse.IsToolRelated())

	toolResult := NewToolResultMessage("tool", "result")
	assert.True(t, toolResult.IsToolRelated())

	text := NewMessage(MessageTypeSay, "hello")
	assert.False(t, text.IsToolRelated())
}

func TestMessageString(t *testing.T) {
	msg := NewMessage(MessageTypeSay, "hello")
	str := msg.String()
	assert.Contains(t, str, "say")
	assert.Contains(t, str, "hello")
}

func TestMessageStore(t *testing.T) {
	store := NewMessageStore()

	// Test empty store
	assert.Equal(t, 0, store.Len())
	_, ok := store.GetLast()
	assert.False(t, ok)

	// Add messages
	msg1 := NewUserMessage("first")
	msg2 := NewAIMessage("second")

	store.Add(msg1)
	store.Add(msg2)

	assert.Equal(t, 2, store.Len())

	// Get messages
	got1, ok := store.Get(0)
	assert.True(t, ok)
	assert.Equal(t, "first", got1.Content)

	got2, ok := store.Get(1)
	assert.True(t, ok)
	assert.Equal(t, "second", got2.Content)

	// Get out of bounds
	_, ok = store.Get(2)
	assert.False(t, ok)

	_, ok = store.Get(-1)
	assert.False(t, ok)

	// Get last
	last, ok := store.GetLast()
	assert.True(t, ok)
	assert.Equal(t, "second", last.Content)

	// Update last
	msg3 := NewAIMessage("updated")
	store.UpdateLast(msg3)

	last, _ = store.GetLast()
	assert.Equal(t, "updated", last.Content)

	// Get all
	all := store.GetAll()
	assert.Len(t, all, 2)

	// Filter
	userMsgs := store.Filter(MessageTypeUser)
	assert.Len(t, userMsgs, 1)

	// Clear
	store.Clear()
	assert.Equal(t, 0, store.Len())
}

func TestNewMarkdownRenderer(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)
	assert.NotNil(t, renderer)

	// Test with options
	renderer2, err := NewMarkdownRenderer(
		WithMarkdownStyle("dark"),
		WithMarkdownWidth(100),
	)
	require.NoError(t, err)
	assert.NotNil(t, renderer2)
}

func TestMarkdownRendererRender(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	// Basic render
	content := "Hello **world**"
	rendered, err := renderer.Render(content)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestMarkdownRendererRenderCode(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	code := `package main
func main() {
    println("hello")
}`
	rendered, err := renderer.RenderCode(code, "go")
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "package")
}

func TestMarkdownRendererRenderInlineCode(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	rendered := renderer.RenderInlineCode("code")
	assert.NotEmpty(t, rendered)
}

func TestMarkdownRendererRenderBlockquote(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	rendered := renderer.RenderBlockquote("This is a quote")
	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, ">")
}

func TestMarkdownRendererRenderList(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	items := []string{"item1", "item2", "item3"}
	rendered := renderer.RenderList(items, false)
	assert.Contains(t, rendered, "item1")
	assert.Contains(t, rendered, "item2")

	// Ordered
	rendered = renderer.RenderList(items, true)
	assert.Contains(t, rendered, "1.")
	assert.Contains(t, rendered, "2.")
}

func TestMarkdownRendererRenderTable(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
	}
	rendered := renderer.RenderTable(headers, rows)
	assert.Contains(t, rendered, "Name")
	assert.Contains(t, rendered, "key1")
}

func TestMarkdownRendererRenderHeading(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	for i := 1; i <= 6; i++ {
		rendered := renderer.RenderHeading(i, "Heading")
		assert.Contains(t, rendered, "Heading")
		assert.Contains(t, rendered, strings.Repeat("#", i))
	}
}

func TestMarkdownRendererRenderBold(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	rendered := renderer.RenderBold("bold text")
	assert.NotEmpty(t, rendered)
}

func TestMarkdownRendererRenderItalic(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	rendered := renderer.RenderItalic("italic text")
	assert.NotEmpty(t, rendered)
}

func TestMarkdownRendererSetWidth(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	renderer.SetWidth(120)
}

func TestHighlightSyntax(t *testing.T) {
	code := `package main
func main() {
    println("hello")
}`

	rendered, err := HighlightSyntax(code, "go", "monokai")
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestHighlightSyntaxFallback(t *testing.T) {
	code := "some generic code"
	rendered, err := HighlightSyntax(code, "unknown", "")
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestAvailableStyles(t *testing.T) {
	styles := AvailableStyles()
	assert.NotEmpty(t, styles)
	assert.Contains(t, styles, "dark")
	assert.Contains(t, styles, "light")
}

func TestDefaultBubbleStyle(t *testing.T) {
	style := DefaultBubbleStyle()
	assert.NotNil(t, style.UserBubble)
	assert.NotNil(t, style.AIBubble)
	assert.NotNil(t, style.ErrorBubble)
	assert.NotNil(t, style.ToolBubble)
	assert.NotNil(t, style.SystemBubble)
	assert.Equal(t, 70, style.Width)
	assert.Equal(t, 1, style.Margin)
}

func TestNewChatRenderer(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)
	assert.NotNil(t, renderer)
	assert.NotNil(t, renderer.markdown)
	assert.NotNil(t, renderer.store)
}

func TestNewChatRendererWithOptions(t *testing.T) {
	style := DefaultBubbleStyle()
	renderer, err := NewChatRenderer(
		WithStyle(style),
		WithWidth(120),
		WithTimestamps(true),
		WithAvatars(false),
	)
	require.NoError(t, err)
	assert.NotNil(t, renderer)
	assert.Equal(t, 120, renderer.width)
	assert.True(t, renderer.showTimestamps)
	assert.False(t, renderer.showAvatars)
}

func TestChatRendererRenderMessage(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	// User message
	userMsg := NewUserMessage("Hello")
	rendered, err := renderer.RenderMessage(userMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)

	// AI message
	aiMsg := NewAIMessage("Hi there!")
	rendered, err = renderer.RenderMessage(aiMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)

	// Error message
	errMsg := NewErrorMessage("Something went wrong")
	rendered, err = renderer.RenderMessage(errMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)

	// Tool use message
	toolMsg := NewToolUseMessage("read_file", map[string]interface{}{
		"file": "test.txt",
	})
	rendered, err = renderer.RenderMessage(toolMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "read_file")

	// Tool result message
	toolResult := NewToolResultMessage("read_file", "file contents")
	rendered, err = renderer.RenderMessage(toolResult)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestChatRendererAddMessage(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	msg := NewUserMessage("test")
	renderer.AddMessage(msg)

	assert.Equal(t, 1, renderer.GetMessageCount())
}

func TestChatRendererStreaming(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	// Start streaming
	assert.False(t, renderer.IsStreaming())

	streamingMsg := renderer.StartStreaming(MessageTypeSay)
	assert.NotNil(t, streamingMsg)
	assert.True(t, renderer.IsStreaming())
	assert.True(t, streamingMsg.Partial)

	// Update streaming
	renderer.UpdateStreaming("Hello")
	renderer.UpdateStreaming(" World")

	content := renderer.GetStreamingContent()
	assert.Equal(t, "Hello World", content)

	// End streaming
	final := renderer.EndStreaming()
	assert.NotNil(t, final)
	assert.False(t, renderer.IsStreaming())
	assert.False(t, final.Partial)
	assert.Equal(t, "Hello World", final.Content)
}

func TestChatRendererStreamingMultipleUpdates(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	renderer.StartStreaming(MessageTypeSay)

	updates := []string{"One", "Two", "Three", "Four"}
	for _, update := range updates {
		renderer.UpdateStreaming(update)
	}

	assert.Equal(t, "OneTwoThreeFour", renderer.GetStreamingContent())

	renderer.EndStreaming()
}

func TestChatRendererClear(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	renderer.AddMessage(NewUserMessage("test"))
	renderer.StartStreaming(MessageTypeSay)
	renderer.Clear()

	assert.Equal(t, 0, renderer.GetMessageCount())
	assert.False(t, renderer.IsStreaming())
	assert.Empty(t, renderer.GetStreamingContent())
}

func TestChatRendererGetLastMessage(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	// Empty store
	_, ok := renderer.GetLastMessage()
	assert.False(t, ok)

	// Add messages
	renderer.AddMessage(NewUserMessage("first"))
	renderer.AddMessage(NewAIMessage("second"))

	last, ok := renderer.GetLastMessage()
	assert.True(t, ok)
	assert.Equal(t, "second", last.Content)
}

func TestChatRendererRenderChat(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	renderer.AddMessage(NewUserMessage("Hello"))
	renderer.AddMessage(NewAIMessage("Hi!"))

	rendered, err := renderer.RenderChat()
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestChatRendererRenderMessageType(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	renderer.AddMessage(NewUserMessage("user message"))
	renderer.AddMessage(NewAIMessage("ai message"))
	renderer.AddMessage(NewUserMessage("another user"))

	rendered, err := renderer.RenderMessageType(MessageTypeUser)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestChatRendererSetWidth(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	renderer.SetWidth(100)
	assert.Equal(t, 100, renderer.width)
}

func TestStreamingRenderer(t *testing.T) {
	chatRenderer, err := NewChatRenderer()
	require.NoError(t, err)

	onUpdate := func(s string) {
		_ = true
	}

	streamer := NewStreamingRenderer(chatRenderer, onUpdate)
	require.NotNil(t, streamer)

	// Not running initially
	assert.False(t, streamer.IsRunning())

	// Start
	streamer.Start()
	assert.True(t, streamer.IsRunning())

	// Write some content
	n, err := streamer.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// Stop
	streamer.Stop()
	assert.False(t, streamer.IsRunning())
}

func TestStreamingRendererStopNotRunning(t *testing.T) {
	chatRenderer, err := NewChatRenderer()
	require.NoError(t, err)

	streamer := NewStreamingRenderer(chatRenderer, nil)
	// Should not panic
	streamer.Stop()
}

func TestStreamingRendererStartAlreadyRunning(t *testing.T) {
	chatRenderer, err := NewChatRenderer()
	require.NoError(t, err)

	streamer := NewStreamingRenderer(chatRenderer, nil)
	streamer.Start()
	streamer.Start() // Should not panic or create extra goroutines

	assert.True(t, streamer.IsRunning())
	streamer.Stop()
}

func TestMessageTimestamp(t *testing.T) {
	before := time.Now()
	msg := NewMessage(MessageTypeSay, "test")
	after := time.Now()

	assert.True(t, msg.Timestamp.Equal(before) || msg.Timestamp.After(before))
	assert.True(t, msg.Timestamp.Equal(after) || msg.Timestamp.Before(after))
}

func TestMessageStoreUpdateLastEmpty(t *testing.T) {
	store := NewMessageStore()
	msg := NewUserMessage("test")
	ok := store.UpdateLast(msg)
	assert.False(t, ok)
}

func TestRenderMarkdownEmptyContent(t *testing.T) {
	renderer, err := NewMarkdownRenderer()
	require.NoError(t, err)

	rendered := renderer.RenderBasic("")
	assert.Empty(t, rendered)
}

func TestRenderToolUseNoInput(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	msg := NewToolUseMessage("simple_tool", nil)
	rendered, err := renderer.renderToolUse(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestRenderToolResultNoLanguage(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	msg := NewToolResultMessage("tool", "plain text result")
	rendered, err := renderer.renderToolResult(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestRenderToolResultWithLanguage(t *testing.T) {
	renderer, err := NewChatRenderer()
	require.NoError(t, err)

	msg := NewToolResultMessage("tool", "package main")
	msg.SetLanguage("go")
	rendered, err := renderer.renderToolResult(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestStreamingRendererWriteMultiple(t *testing.T) {
	chatRenderer, err := NewChatRenderer()
	require.NoError(t, err)

	received := make([]string, 0)
	onUpdate := func(s string) {
		received = append(received, s)
	}

	streamer := NewStreamingRenderer(chatRenderer, onUpdate)
	streamer.Start()

	// Write multiple times
	streamer.Write([]byte("chunk1"))
	streamer.Write([]byte("chunk2"))
	streamer.Write([]byte("chunk3"))

	// Give goroutine time to process
	time.Sleep(100 * time.Millisecond)

	streamer.Stop()

	// Should have received updates
	assert.True(t, len(received) > 0)
}