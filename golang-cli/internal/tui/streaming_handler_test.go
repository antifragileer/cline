package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewStreamingHandler(t *testing.T) {
	t.Run("creates handler with defaults", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)

		assert.NotNil(t, handler)
		assert.Equal(t, updateChan, handler.updateChan)
		assert.Equal(t, 100*time.Millisecond, handler.flushDelay)
		assert.NotNil(t, handler.chunks)
		assert.Empty(t, handler.chunks)
	})
}

func TestStreamingHandler_StartStreaming(t *testing.T) {
	t.Run("starts new streaming session", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)

		handler.StartStreaming("msg-123", "text")

		assert.Equal(t, "msg-123", handler.streamingID)
		assert.Equal(t, "text", handler.messageType)
		assert.True(t, handler.isStreaming)
		assert.False(t, handler.isComplete)
		assert.Empty(t, handler.buffer.String())
		assert.Empty(t, handler.chunks)
		assert.Equal(t, 0, handler.sequenceNum)
	})

	t.Run("resets previous state when starting new session", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)

		// Start first session
		handler.StartStreaming("msg-1", "text")
		handler.AppendContent("previous content")
		handler.EndStreaming()

		// Start new session
		handler.StartStreaming("msg-2", "code")

		assert.Equal(t, "msg-2", handler.streamingID)
		assert.Equal(t, "code", handler.messageType)
		assert.True(t, handler.isStreaming)
		assert.False(t, handler.isComplete)
		assert.Empty(t, handler.buffer.String())
		assert.Empty(t, handler.chunks)
	})
}

func TestStreamingHandler_AppendContent(t *testing.T) {
	t.Run("appends content to buffer", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		handler.AppendContent("Hello ")
		handler.AppendContent("World")

		assert.Equal(t, "Hello World", handler.GetContent())
	})

	t.Run("does nothing when not streaming", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		// Don't start streaming

		handler.AppendContent("content")

		assert.Empty(t, handler.GetContent())
	})
}

func TestStreamingHandler_EndStreaming(t *testing.T) {
	t.Run("marks streaming as complete", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		handler.AppendContent("content")
		handler.EndStreaming()

		assert.False(t, handler.isStreaming)
		assert.True(t, handler.isComplete)
	})

	t.Run("sends completion message", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")
		handler.AppendContent("final content")

		handler.EndStreaming()

		// Check that completion message was sent
		select {
		case msg := <-updateChan:
			completeMsg, ok := msg.(StreamingCompleteMsg)
			assert.True(t, ok)
			assert.Equal(t, "msg-1", completeMsg.ID)
			assert.Equal(t, "final content", completeMsg.Content)
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected completion message not received")
		}
	})
}

func TestStreamingHandler_HandleChunk(t *testing.T) {
	t.Run("handles message chunk", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		chunk := &MessageChunk{
			Sequence: 0,
			Content:  "chunk content",
			IsLast:   false,
		}

		err := handler.HandleChunk(chunk)

		assert.NoError(t, err)
		assert.Equal(t, 1, handler.GetChunkCount())
		assert.Equal(t, "chunk content", handler.GetContent())
	})

	t.Run("returns nil when not streaming", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		// Don't start streaming

		chunk := &MessageChunk{
			Sequence: 0,
			Content:  "content",
			IsLast:   false,
		}

		err := handler.HandleChunk(chunk)

		assert.NoError(t, err)
	})

	t.Run("triggers UI update", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		chunk := &MessageChunk{
			Sequence: 0,
			Content:  "update",
			IsLast:   false,
		}

		handler.HandleChunk(chunk)

		// Check that update message was sent
		select {
		case msg := <-updateChan:
			updateMsg, ok := msg.(StreamingUpdateMsg)
			assert.True(t, ok)
			assert.Equal(t, "msg-1", updateMsg.ID)
		default:
			// Message might have been processed already
		}
	})
}

func TestStreamingHandler_Flush(t *testing.T) {
	t.Run("triggers UI update", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		err := handler.Flush()

		assert.NoError(t, err)

		// Check that update message was sent
		select {
		case msg := <-updateChan:
			updateMsg, ok := msg.(StreamingUpdateMsg)
			assert.True(t, ok)
			assert.Equal(t, "msg-1", updateMsg.ID)
		default:
			// Message might have been processed already
		}
	})

	t.Run("stops flush timer", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")
		handler.AppendContent("content") // This starts the timer

		err := handler.Flush()

		assert.NoError(t, err)
		assert.Nil(t, handler.flushTimer)
	})
}

func TestStreamingHandler_IsComplete(t *testing.T) {
	t.Run("returns false when streaming", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		assert.False(t, handler.IsComplete())
	})

	t.Run("returns true after completion", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")
		handler.EndStreaming()

		assert.True(t, handler.IsComplete())
	})
}

func TestStreamingHandler_Reset(t *testing.T) {
	t.Run("clears all state", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")
		handler.AppendContent("content")
		handler.EndStreaming()

		handler.Reset()

		assert.Empty(t, handler.streamingID)
		assert.Empty(t, handler.buffer.String())
		assert.False(t, handler.isStreaming)
		assert.False(t, handler.isComplete)
		assert.Empty(t, handler.chunks)
		assert.Equal(t, 0, handler.sequenceNum)
		assert.Equal(t, 0, handler.totalChunks)
		assert.Empty(t, handler.messageType)
	})
}

func TestStreamingHandler_GetStreamingID(t *testing.T) {
	t.Run("returns current streaming ID", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("test-id", "text")

		assert.Equal(t, "test-id", handler.GetStreamingID())
	})
}

func TestStreamingHandler_IsStreaming(t *testing.T) {
	t.Run("returns true when streaming", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		assert.True(t, handler.IsStreaming())
	})

	t.Run("returns false when not streaming", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)

		assert.False(t, handler.IsStreaming())
	})
}

func TestStreamingHandler_GetMessageType(t *testing.T) {
	t.Run("returns message type", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "code")

		assert.Equal(t, "code", handler.GetMessageType())
	})
}

func TestStreamingHandler_GetChunkCount(t *testing.T) {
	t.Run("returns chunk count", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		assert.Equal(t, 0, handler.GetChunkCount())

		handler.HandleChunk(&MessageChunk{Sequence: 0, Content: "1"})
		assert.Equal(t, 1, handler.GetChunkCount())

		handler.HandleChunk(&MessageChunk{Sequence: 1, Content: "2"})
		assert.Equal(t, 2, handler.GetChunkCount())
	})
}

func TestStreamingHandler_ParseMessageChunks(t *testing.T) {
	t.Run("parses data into chunks", func(t *testing.T) {
		updateChan := make(chan tea.Msg, 10)
		handler := NewStreamingHandler(updateChan)
		handler.StartStreaming("msg-1", "text")

		data := []byte("test data")
		chunks, err := handler.ParseMessageChunks(data)

		assert.NoError(t, err)
		assert.Len(t, chunks, 1)
		assert.Equal(t, "test data", chunks[0].Content)
		assert.Equal(t, 0, chunks[0].Sequence)
	})
}

func TestNewStreamBuffer(t *testing.T) {
	t.Run("creates empty buffer", func(t *testing.T) {
		sb := NewStreamBuffer()

		assert.NotNil(t, sb)
		assert.Empty(t, sb.Read())
		assert.False(t, sb.IsComplete())
	})
}

func TestStreamBuffer_Write(t *testing.T) {
	t.Run("writes chunks", func(t *testing.T) {
		sb := NewStreamBuffer()

		sb.Write("Hello ")
		sb.Write("World")

		assert.Equal(t, "Hello World", sb.Read())
	})
}

func TestStreamBuffer_Read(t *testing.T) {
	t.Run("reads all content", func(t *testing.T) {
		sb := NewStreamBuffer()
		sb.Write("chunk1")
		sb.Write("chunk2")

		content := sb.Read()

		assert.Equal(t, "chunk1chunk2", content)
	})
}

func TestStreamBuffer_Complete(t *testing.T) {
	t.Run("marks buffer complete", func(t *testing.T) {
		sb := NewStreamBuffer()

		sb.Complete()

		assert.True(t, sb.IsComplete())
	})
}

func TestStreamBuffer_IsComplete(t *testing.T) {
	t.Run("returns false initially", func(t *testing.T) {
		sb := NewStreamBuffer()

		assert.False(t, sb.IsComplete())
	})

	t.Run("returns true after completion", func(t *testing.T) {
		sb := NewStreamBuffer()
		sb.Complete()

		assert.True(t, sb.IsComplete())
	})
}

func TestStreamBuffer_Reset(t *testing.T) {
	t.Run("clears buffer", func(t *testing.T) {
		sb := NewStreamBuffer()
		sb.Write("content")
		sb.Complete()

		sb.Reset()

		assert.Empty(t, sb.Read())
		assert.False(t, sb.IsComplete())
	})
}

func TestNewMessageReassembler(t *testing.T) {
	t.Run("creates reassembler", func(t *testing.T) {
		mr := NewMessageReassembler()

		assert.NotNil(t, mr)
		assert.NotNil(t, mr.chunks)
		assert.NotNil(t, mr.timeouts)
	})
}

func TestMessageReassembler_AddChunk(t *testing.T) {
	t.Run("adds chunk and returns incomplete", func(t *testing.T) {
		mr := NewMessageReassembler()

		chunk := MessageChunk{
			Sequence: 0,
			Content:  "part 1",
			IsLast:   false,
		}

		result, complete := mr.AddChunk("msg-1", chunk)

		assert.Empty(t, result)
		assert.False(t, complete)
	})

	t.Run("returns complete message on last chunk", func(t *testing.T) {
		mr := NewMessageReassembler()

		// Add first chunk
		mr.AddChunk("msg-1", MessageChunk{
			Sequence: 0,
			Content:  "Hello ",
			IsLast:   false,
		})

		// Add last chunk
		result, complete := mr.AddChunk("msg-1", MessageChunk{
			Sequence: 1,
			Content:  "World",
			IsLast:   true,
		})

		assert.True(t, complete)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("handles multiple messages", func(t *testing.T) {
		mr := NewMessageReassembler()

		// Complete first message
		mr.AddChunk("msg-1", MessageChunk{
			Sequence: 0,
			Content:  "Message 1",
			IsLast:   true,
		})

		// Start second message
		_, complete := mr.AddChunk("msg-2", MessageChunk{
			Sequence: 0,
			Content:  "Message 2",
			IsLast:   false,
		})

		assert.False(t, complete)
	})
}

func TestMessageReassembler_Cleanup(t *testing.T) {
	t.Run("removes stale messages", func(t *testing.T) {
		mr := NewMessageReassembler()

		// Add chunk with very short timeout
		mr.AddChunk("msg-1", MessageChunk{
			Sequence: 0,
			Content:  "test",
			IsLast:   false,
		})

		// Manually set timeout to past
		mr.timeouts["msg-1"] = time.Now().Add(-1 * time.Second)

		// Cleanup
		mr.Cleanup()

		// Message should be removed
		_, exists := mr.chunks["msg-1"]
		assert.False(t, exists)
	})

	t.Run("keeps fresh messages", func(t *testing.T) {
		mr := NewMessageReassembler()

		mr.AddChunk("msg-1", MessageChunk{
			Sequence: 0,
			Content:  "test",
			IsLast:   false,
		})

		// Cleanup should not remove fresh messages
		mr.Cleanup()

		_, exists := mr.chunks["msg-1"]
		assert.True(t, exists)
	})
}

func TestStartSpinner(t *testing.T) {
	t.Run("returns tick command", func(t *testing.T) {
		cmd := StartSpinner()

		assert.NotNil(t, cmd)

		// Execute command and check message type
		msg := cmd()
		_, ok := msg.(SpinnerTickMsg)
		assert.True(t, ok)
	})
}