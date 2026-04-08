// Package host provides gRPC client functionality for communicating with the Cline extension.
// This file contains tests for the message converter.
package host

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/stretchr/testify/assert"
)

func TestNewMessageConverter(t *testing.T) {
	converter := NewMessageConverter()
	assert.NotNil(t, converter)
}

func TestMessageConverter_ProtoToInternal(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("converts basic message", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TEXT,
			Text: "Hello, World!",
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.Equal(t, int64(1234567890), internal.Ts)
		assert.Equal(t, ClineMessageType_SAY, internal.Type)
		assert.Equal(t, ClineSay_TEXT, internal.Say)
		assert.Equal(t, "Hello, World!", internal.Text)
	})

	t.Run("converts nil message", func(t *testing.T) {
		internal := converter.ProtoToInternal(nil)
		assert.Nil(t, internal)
	})

	t.Run("converts message with tool", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TOOL_SAY,
			Text: "Tool executed",
			SayTool: &cline.ClineSayTool{
				Tool: cline.ClineSayToolType_READ_FILE,
				Path: "/test/file.txt",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.NotNil(t, internal.SayTool)
		assert.Equal(t, ClineSayToolType_READ_FILE, internal.SayTool.Tool)
		assert.Equal(t, "/test/file.txt", internal.SayTool.Path)
	})

	t.Run("converts message with browser action", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_BROWSER_ACTION,
			SayBrowserAction: &cline.ClineSayBrowserAction{
				Action:     cline.BrowserAction_LAUNCH,
				Coordinate: "100,200",
				Text:       "https://example.com",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.NotNil(t, internal.SayBrowserAction)
		assert.Equal(t, BrowserAction_LAUNCH, internal.SayBrowserAction.Action)
		assert.Equal(t, "100,200", internal.SayBrowserAction.Coordinate)
	})

	t.Run("converts message with API request info", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_API_REQ_FINISHED,
			ApiReqInfo: &cline.ClineApiReqInfo{
				TokensIn:  100,
				TokensOut: 200,
				Cost:      0.01,
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.NotNil(t, internal.ApiReqInfo)
		assert.Equal(t, int32(100), internal.ApiReqInfo.TokensIn)
		assert.Equal(t, int32(200), internal.ApiReqInfo.TokensOut)
		assert.Equal(t, 0.01, internal.ApiReqInfo.Cost)
	})

	t.Run("converts message with ask question", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_ASK,
			Ask:  cline.ClineAsk_FOLLOWUP,
			AskQuestion: &cline.ClineAskQuestion{
				Question: "What would you like to do?",
				Options:  []string{"Option 1", "Option 2"},
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal)
		assert.NotNil(t, internal.AskQuestion)
		assert.Equal(t, "What would you like to do?", internal.AskQuestion.Question)
		assert.Equal(t, []string{"Option 1", "Option 2"}, internal.AskQuestion.Options)
	})
}

func TestMessageConverter_InternalToProto(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("converts basic message", func(t *testing.T) {
		internal := &ClineMessage{
			Ts:   1234567890,
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "Hello, World!",
		}

		proto := converter.InternalToProto(internal)

		assert.NotNil(t, proto)
		assert.Equal(t, int64(1234567890), proto.Ts)
		assert.Equal(t, cline.ClineMessageType_SAY, proto.Type)
		assert.Equal(t, cline.ClineSay_TEXT, proto.Say)
		assert.Equal(t, "Hello, World!", proto.Text)
	})

	t.Run("converts nil message", func(t *testing.T) {
		proto := converter.InternalToProto(nil)
		assert.Nil(t, proto)
	})

	t.Run("round-trip conversion preserves data", func(t *testing.T) {
		original := &ClineMessage{
			Ts:                          1234567890,
			Type:                        ClineMessageType_ASK,
			Ask:                         ClineAsk_COMMAND,
			Text:                        "Run this command",
			Reasoning:                   "Testing round-trip",
			Images:                      []string{"img1.png", "img2.png"},
			Files:                       []string{"file1.txt"},
			Partial:                     false,
			LastCheckpointHash:          "abc123",
			IsCheckpointCheckedOut:      true,
			IsOperationOutsideWorkspace: false,
			ConversationHistoryIndex:    5,
		}

		proto := converter.InternalToProto(original)
		converted := converter.ProtoToInternal(proto)

		assert.Equal(t, original.Ts, converted.Ts)
		assert.Equal(t, original.Type, converted.Type)
		assert.Equal(t, original.Ask, converted.Ask)
		assert.Equal(t, original.Text, converted.Text)
		assert.Equal(t, original.Reasoning, converted.Reasoning)
		assert.Equal(t, original.Images, converted.Images)
		assert.Equal(t, original.Files, converted.Files)
		assert.Equal(t, original.Partial, converted.Partial)
		assert.Equal(t, original.LastCheckpointHash, converted.LastCheckpointHash)
		assert.Equal(t, original.IsCheckpointCheckedOut, converted.IsCheckpointCheckedOut)
		assert.Equal(t, original.IsOperationOutsideWorkspace, converted.IsOperationOutsideWorkspace)
		assert.Equal(t, original.ConversationHistoryIndex, converted.ConversationHistoryIndex)
	})
}

func TestMessageConverter_CreateMessageFromJSON(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("creates message from valid JSON", func(t *testing.T) {
		jsonData := `{"ts":1234567890,"type":1,"say":4,"text":"Hello"}`

		msg, err := converter.CreateMessageFromJSON([]byte(jsonData))

		assert.NoError(t, err)
		assert.NotNil(t, msg)
		assert.Equal(t, int64(1234567890), msg.Ts)
		assert.Equal(t, ClineMessageType_SAY, msg.Type)
		assert.Equal(t, ClineSay_TEXT, msg.Say)
		assert.Equal(t, "Hello", msg.Text)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		jsonData := `{"ts":invalid,"type":1}`

		msg, err := converter.CreateMessageFromJSON([]byte(jsonData))

		assert.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("returns error for empty JSON", func(t *testing.T) {
		msg, err := converter.CreateMessageFromJSON([]byte{})

		assert.Error(t, err)
		assert.Nil(t, msg)
	})
}

func TestMessageConverter_ConvertTimestamp(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("converts milliseconds to time", func(t *testing.T) {
		// 1234567890000 milliseconds = Feb 13, 2009
		timestamp := int64(1234567890000)
		result := converter.ConvertTimestamp(timestamp)

		expected := time.Unix(0, timestamp*int64(time.Millisecond))
		assert.Equal(t, expected, result)
	})

	t.Run("converts zero timestamp", func(t *testing.T) {
		result := converter.ConvertTimestamp(0)
		assert.Equal(t, time.Unix(0, 0), result)
	})
}

func TestMessageConverter_CreateTimestamp(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("creates valid timestamp", func(t *testing.T) {
		before := time.Now().UnixMilli()
		timestamp := converter.CreateTimestamp()
		after := time.Now().UnixMilli()

		assert.GreaterOrEqual(t, timestamp, before)
		assert.LessOrEqual(t, timestamp, after)
	})
}

func TestMessageConverter_IsCompleteMessage(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("returns true for non-partial message", func(t *testing.T) {
		msg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Partial: false,
			},
		}
		assert.True(t, converter.IsCompleteMessage(msg))
	})

	t.Run("returns false for partial message", func(t *testing.T) {
		msg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Partial: true,
			},
		}
		assert.False(t, converter.IsCompleteMessage(msg))
	})

	t.Run("returns false for nil message", func(t *testing.T) {
		assert.False(t, converter.IsCompleteMessage(nil))
	})

	t.Run("returns false for message with nil ClineMessage", func(t *testing.T) {
		msg := &ClineMessageProto{}
		assert.False(t, converter.IsCompleteMessage(msg))
	})
}

func TestMessageConverter_ExtractMessageType(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("extracts ASK type", func(t *testing.T) {
		msg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Type: ClineMessageType_ASK,
				Ask:  ClineAsk_COMMAND,
			},
		}
		assert.Equal(t, "ask:2", converter.ExtractMessageType(msg))
	})

	t.Run("extracts SAY type", func(t *testing.T) {
		msg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Type: ClineMessageType_SAY,
				Say:  ClineSay_TEXT,
			},
		}
		assert.Equal(t, "say:4", converter.ExtractMessageType(msg))
	})

	t.Run("returns unknown for nil message", func(t *testing.T) {
		assert.Equal(t, "unknown", converter.ExtractMessageType(nil))
	})

	t.Run("returns unknown for invalid type", func(t *testing.T) {
		msg := &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Type: ClineMessageType(999),
			},
		}
		assert.Equal(t, "unknown", converter.ExtractMessageType(msg))
	})
}

func TestMessageConverter_ComplexConversions(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("converts message with MCP server request", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_ASK,
			Ask:  cline.ClineAsk_USE_MCP_SERVER,
			AskUseMcpServer: &cline.ClineAskUseMcpServer{
				ServerName: "test-server",
				Type:       cline.McpServerRequestType_USE_MCP_TOOL,
				ToolName:   "test-tool",
				Arguments:  `{"arg1": "value1"}`,
				Uri:        "test://resource",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.AskUseMcpServer)
		assert.Equal(t, "test-server", internal.AskUseMcpServer.ServerName)
		assert.Equal(t, McpServerRequestType_USE_MCP_TOOL, internal.AskUseMcpServer.Type)
		assert.Equal(t, "test-tool", internal.AskUseMcpServer.ToolName)
	})

	t.Run("converts message with plan mode response", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_TEXT,
			PlanModeResponse: &cline.ClinePlanModeResponse{
				Response: "Plan created",
				Options:  []string{"Execute", "Modify", "Cancel"},
				Selected: "Execute",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.PlanModeResponse)
		assert.Equal(t, "Plan created", internal.PlanModeResponse.Response)
		assert.Equal(t, []string{"Execute", "Modify", "Cancel"}, internal.PlanModeResponse.Options)
		assert.Equal(t, "Execute", internal.PlanModeResponse.Selected)
	})

	t.Run("converts message with model info", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_API_REQ_STARTED,
			ModelInfo: &cline.ClineModelInfo{
				ProviderId: "anthropic",
				ModelId:    "claude-3-opus",
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.ModelInfo)
		assert.Equal(t, "anthropic", internal.ModelInfo.ProviderId)
		assert.Equal(t, "claude-3-opus", internal.ModelInfo.ModelId)
	})

	t.Run("converts message with retry status", func(t *testing.T) {
		proto := &cline.ClineMessage{
			Ts:   1234567890,
			Type: cline.ClineMessageType_SAY,
			Say:  cline.ClineSay_API_REQ_RETRIED,
			ApiReqInfo: &cline.ClineApiReqInfo{
				RetryStatus: &cline.ApiReqRetryStatus{
					Attempt:      2,
					MaxAttempts:  5,
					DelaySec:     10,
					ErrorSnippet: "rate limit exceeded",
				},
			},
		}

		internal := converter.ProtoToInternal(proto)

		assert.NotNil(t, internal.ApiReqInfo)
		assert.NotNil(t, internal.ApiReqInfo.RetryStatus)
		assert.Equal(t, int32(2), internal.ApiReqInfo.RetryStatus.Attempt)
		assert.Equal(t, int32(5), internal.ApiReqInfo.RetryStatus.MaxAttempts)
		assert.Equal(t, int32(10), internal.ApiReqInfo.RetryStatus.DelaySec)
		assert.Equal(t, "rate limit exceeded", internal.ApiReqInfo.RetryStatus.ErrorSnippet)
	})
}

func TestMessageConverter_JSONSerialization(t *testing.T) {
	converter := NewMessageConverter()

	t.Run("serializes and deserializes message", func(t *testing.T) {
		original := &ClineMessage{
			Ts:   1234567890,
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "Test message",
		}

		// Convert to proto then to JSON
		proto := converter.InternalToProto(original)
		jsonData, err := json.Marshal(proto)
		assert.NoError(t, err)

		// Parse JSON back
		var parsedProto cline.ClineMessage
		err = json.Unmarshal(jsonData, &parsedProto)
		assert.NoError(t, err)

		// Convert back to internal
		converted := converter.ProtoToInternal(&parsedProto)

		assert.Equal(t, original.Ts, converted.Ts)
		assert.Equal(t, original.Text, converted.Text)
	})
}

func BenchmarkMessageConverter_ProtoToInternal(b *testing.B) {
	converter := NewMessageConverter()
	proto := &cline.ClineMessage{
		Ts:   1234567890,
		Type: cline.ClineMessageType_SAY,
		Say:  cline.ClineSay_TEXT,
		Text: "Benchmark message",
		SayTool: &cline.ClineSayTool{
			Tool: cline.ClineSayToolType_READ_FILE,
			Path: "/test/file.txt",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = converter.ProtoToInternal(proto)
	}
}

func BenchmarkMessageConverter_InternalToProto(b *testing.B) {
	converter := NewMessageConverter()
	internal := &ClineMessage{
		Ts:   1234567890,
		Type: ClineMessageType_SAY,
		Say:  ClineSay_TEXT,
		Text: "Benchmark message",
		SayTool: &ClineSayTool{
			Tool: ClineSayToolType_READ_FILE,
			Path: "/test/file.txt",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = converter.InternalToProto(internal)
	}
}
