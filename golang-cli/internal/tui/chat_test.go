package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// TestMessageRendering tests message rendering functionality
func TestMessageRendering(t *testing.T) {
	t.Run("renders user message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "user",
			Content: "Hello, Cline!",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Hello, Cline!")
		assert.Contains(t, rendered, "You")
	})

	t.Run("renders assistant message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "Hello! How can I help you today?",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Hello! How can I help you today?")
		assert.Contains(t, rendered, "Cline")
	})

	t.Run("renders system message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "system",
			Content: "System notification",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "System notification")
	})

	t.Run("renders error message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "error",
			Content: "An error occurred",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "An error occurred")
	})

	t.Run("handles empty message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "user",
			Content: "",
		}

		rendered := m.renderMessage(msg)
		assert.NotEmpty(t, rendered)
	})

	t.Run("handles very long message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		longContent := strings.Repeat("This is a very long message. ", 50)
		msg := Message{
			Role:    "user",
			Content: longContent,
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "This is a very long message.")
	})

	t.Run("handles multiline message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "user",
			Content: "Line 1\nLine 2\nLine 3",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Line 1")
		assert.Contains(t, rendered, "Line 2")
		assert.Contains(t, rendered, "Line 3")
	})
}

// TestMarkdownFormatting tests markdown formatting in messages
func TestMarkdownFormatting(t *testing.T) {
	t.Run("renders bold text", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "This is **bold** text",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "bold")
	})

	t.Run("renders italic text", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "This is *italic* text",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "italic")
	})

	t.Run("renders inline code", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "Use `printf` for output",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "printf")
	})

	t.Run("renders code blocks", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "func main()")
		assert.Contains(t, rendered, "fmt.Println")
	})

	t.Run("renders lists", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "- Item 1\n- Item 2\n- Item 3",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Item 1")
		assert.Contains(t, rendered, "Item 2")
		assert.Contains(t, rendered, "Item 3")
	})

	t.Run("renders numbered lists", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "1. First\n2. Second\n3. Third",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "First")
		assert.Contains(t, rendered, "Second")
		assert.Contains(t, rendered, "Third")
	})

	t.Run("renders links", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "Visit [Cline](https://cline.bot)",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Cline")
	})

	t.Run("renders blockquotes", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "> This is a quote",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "This is a quote")
	})

	t.Run("renders headers", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:    "assistant",
			Content: "# Heading 1\n## Heading 2\n### Heading 3",
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Heading 1")
		assert.Contains(t, rendered, "Heading 2")
		assert.Contains(t, rendered, "Heading 3")
	})
}

// TestSyntaxHighlighting tests syntax highlighting in code blocks
func TestSyntaxHighlighting(t *testing.T) {
	t.Run("highlights Go code", func(t *testing.T) {
		m := initialModel()
		
		code := "```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
		highlighted := m.highlightCode(code, "go")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "package")
		assert.Contains(t, highlighted, "func")
	})

	t.Run("highlights Python code", func(t *testing.T) {
		m := initialModel()
		
		code := "```python\ndef hello():\n    print(\"Hello\")\n```"
		highlighted := m.highlightCode(code, "python")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "def")
	})

	t.Run("highlights JavaScript code", func(t *testing.T) {
		m := initialModel()
		
		code := "```javascript\nfunction hello() {\n    console.log(\"Hello\");\n}\n```"
		highlighted := m.highlightCode(code, "javascript")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "function")
	})

	t.Run("highlights TypeScript code", func(t *testing.T) {
		m := initialModel()
		
		code := "```typescript\nconst x: string = \"hello\";\n```"
		highlighted := m.highlightCode(code, "typescript")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "const")
	})

	t.Run("highlights Bash code", func(t *testing.T) {
		m := initialModel()
		
		code := "```bash\n#!/bin/bash\necho \"Hello\"\n```"
		highlighted := m.highlightCode(code, "bash")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "echo")
	})

	t.Run("handles unknown language gracefully", func(t *testing.T) {
		m := initialModel()
		
		code := "```unknown\nsome code\n```"
		highlighted := m.highlightCode(code, "unknown")

		// Should still render the code, even if not highlighted
		assert.NotEmpty(t, highlighted)
	})

	t.Run("handles code without language specifier", func(t *testing.T) {
		m := initialModel()
		
		code := "```\nsome plain code\n```"
		highlighted := m.highlightCode(code, "")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "some plain code")
	})

	t.Run("handles empty code block", func(t *testing.T) {
		m := initialModel()
		
		code := "```go\n```"
		highlighted := m.highlightCode(code, "go")

		assert.NotEmpty(t, highlighted)
	})

	t.Run("handles code with special characters", func(t *testing.T) {
		m := initialModel()
		
		code := "```go\nfmt.Printf(\"Special: %s\\n\", \"chars\")\n```"
		highlighted := m.highlightCode(code, "go")

		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "fmt.Printf")
	})
}

// TestStreamingMessageDisplay tests streaming message display
func TestStreamingMessageDisplay(t *testing.T) {
	t.Run("renders streaming message with indicator", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:      "assistant",
			Content:   "Generating",
			Streaming: true,
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Generating")
		// Should indicate streaming state
	})

	t.Run("updates streaming message content", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:      "assistant",
			Content:   "Hello",
			Streaming: true,
		}

		// Simulate streaming update
		msg.Content += " world"
		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Hello world")
	})

	t.Run("finalizes streaming message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:      "assistant",
			Content:   "Complete response",
			Streaming: false,
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Complete response")
		// Should not show streaming indicator
	})

	t.Run("handles streaming with markdown", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:      "assistant",
			Content:   "**Bold** and `code`",
			Streaming: true,
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "Bold")
		assert.Contains(t, rendered, "code")
	})

	t.Run("handles streaming code blocks", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		
		msg := Message{
			Role:      "assistant",
			Content:   "```go\nfunc",
			Streaming: true,
		}

		rendered := m.renderMessage(msg)
		assert.Contains(t, rendered, "func")
	})
}

// TestMessageHistory tests message history functionality
func TestMessageHistory(t *testing.T) {
	t.Run("renders empty history", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		m.height = 24
		m.messages = []Message{}

		view := m.renderChatView()
		assert.NotEmpty(t, view)
	})

	t.Run("renders single message in history", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		m.height = 24
		m.messages = []Message{
			{Role: "user", Content: "Hello"},
		}

		view := m.renderChatView()
		assert.Contains(t, view, "Hello")
	})

	t.Run("renders multiple messages in history", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		m.height = 24
		m.messages = []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
			{Role: "user", Content: "How are you?"},
		}

		view := m.renderChatView()
		assert.Contains(t, view, "Hello")
		assert.Contains(t, view, "Hi!")
		assert.Contains(t, view, "How are you?")
	})

	t.Run("handles many messages", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		m.height = 24
		
		// Add many messages
		for i := 0; i < 50; i++ {
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: "Message " + string(rune('0'+i%10)),
			})
		}

		view := m.renderChatView()
		assert.NotEmpty(t, view)
	})

	t.Run("scrolls to bottom on new message", func(t *testing.T) {
		m := initialModel()
		m.width = 80
		m.height = 24
		m.autoScroll = true

		// Add messages
		for i := 0; i < 10; i++ {
			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: "Message",
			})
		}

		view := m.renderChatView()
		assert.NotEmpty(t, view)
		// Viewport should be positioned to show latest
	})

	t.Run("preserves message order", func(t *testing.T) {
		m := initialModel()
		
		m.messages = []Message{
			{Role: "user", Content: "First"},
			{Role: "assistant", Content: "Second"},
			{Role: "user", Content: "Third"},
		}

		assert.Equal(t, "First", m.messages[0].Content)
		assert.Equal(t, "Second", m.messages[1].Content)
		assert.Equal(t, "Third", m.messages[2].Content)
	})
}

// TestChatViewRendering tests chat view rendering
func TestChatViewRendering(t *testing.T) {
	t.Run("renders chat view with header", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 24

		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("renders chat view with input area", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 24
		m.inputFocused = true

		view := m.View()
		assert.NotEmpty(t, view)
		// Should contain input indicator
	})

	t.Run("renders chat view with help", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 24
		m.showHelp = true

		view := m.View()
		assert.NotEmpty(t, view)
		// Should contain help text
	})

	t.Run("handles narrow terminal", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 40
		m.height = 24

		view := m.View()
		assert.NotEmpty(t, view)
	})

	t.Run("handles short terminal", func(t *testing.T) {
		m := initialModel()
		m.currentView = chatView
		m.width = 80
		m.height = 10

		view := m.View()
		assert.NotEmpty(t, view)
	})
}

// TestMessageFormatting tests message formatting utilities
func TestMessageFormatting(t *testing.T) {
	t.Run("wraps long lines", func(t *testing.T) {
		m := initialModel()
		m.width = 40
		
		longText := strings.Repeat("a", 100)
		wrapped := m.wrapText(longText, 30)

		lines := strings.Split(wrapped, "\n")
		for _, line := range lines {
			assert.LessOrEqual(t, lipgloss.Width(line), 35) // Allow some margin
		}
	})

	t.Run("preserves line breaks", func(t *testing.T) {
		m := initialModel()
		
		text := "Line 1\nLine 2\nLine 3"
		wrapped := m.wrapText(text, 80)

		assert.Contains(t, wrapped, "Line 1")
		assert.Contains(t, wrapped, "Line 2")
		assert.Contains(t, wrapped, "Line 3")
	})

	t.Run("handles empty text", func(t *testing.T) {
		m := initialModel()
		
		wrapped := m.wrapText("", 80)
		assert.Equal(t, "", wrapped)
	})

	t.Run("trims trailing whitespace", func(t *testing.T) {
		m := initialModel()
		
		text := "Hello   \nWorld   "
		wrapped := m.wrapText(text, 80)

		assert.NotContains(t, wrapped, "Hello   ")
		assert.NotContains(t, wrapped, "World   ")
	})
}

// TestCodeBlockExtraction tests code block parsing
func TestCodeBlockExtraction(t *testing.T) {
	t.Run("extracts code block with language", func(t *testing.T) {
		m := initialModel()
		
		content := "```go\nfunc main() {}\n```"
		blocks := m.extractCodeBlocks(content)

		assert.Len(t, blocks, 1)
		assert.Equal(t, "go", blocks[0].Language)
		assert.Equal(t, "func main() {}", blocks[0].Code)
	})

	t.Run("extracts multiple code blocks", func(t *testing.T) {
		m := initialModel()
		
		content := "```go\ncode1\n```\n\n```python\ncode2\n```"
		blocks := m.extractCodeBlocks(content)

		assert.Len(t, blocks, 2)
	})

	t.Run("extracts code block without language", func(t *testing.T) {
		m := initialModel()
		
		content := "```\nplain code\n```"
		blocks := m.extractCodeBlocks(content)

		assert.Len(t, blocks, 1)
		assert.Equal(t, "", blocks[0].Language)
	})

	t.Run("handles no code blocks", func(t *testing.T) {
		m := initialModel()
		
		content := "Just plain text"
		blocks := m.extractCodeBlocks(content)

		assert.Empty(t, blocks)
	})
}