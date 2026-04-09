// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
	"testing"
)

func TestMessageSerializer(t *testing.T) {
	serializer := NewMessageSerializer()

	t.Run("SerializeAndDeserialize", func(t *testing.T) {
		msg := &JSONMessage{
			Ts:      1234567890,
			Type:    string(ClineMessageTypeSay),
			Say:     string(ClineSayText),
			Text:    "Hello, World!",
			Partial: false,
		}

		data, err := serializer.Serialize(msg)
		if err != nil {
			t.Fatalf("Failed to serialize: %v", err)
		}

		deserialized, err := serializer.Deserialize(data)
		if err != nil {
			t.Fatalf("Failed to deserialize: %v", err)
		}

		if deserialized.Type != msg.Type {
			t.Errorf("Type mismatch: got %s, want %s", deserialized.Type, msg.Type)
		}

		if deserialized.Text != msg.Text {
			t.Errorf("Text mismatch: got %s, want %s", deserialized.Text, msg.Text)
		}
	})

	t.Run("SerializeStreaming", func(t *testing.T) {
		msg := &JSONMessage{
			Type: string(ClineMessageTypeSay),
			Say:  string(ClineSayText),
			Text: "Test",
		}

		data, err := serializer.SerializeStreaming(msg)
		if err != nil {
			t.Fatalf("Failed to serialize: %v", err)
		}

		// Should end with newline
		if data[len(data)-1] != '\n' {
			t.Error("Streaming output should end with newline")
		}
	})

	t.Run("AutoTimestamp", func(t *testing.T) {
		msg := &JSONMessage{
			Type: string(ClineMessageTypeSay),
			Say:  string(ClineSayText),
			Text: "Test",
		}

		// Should auto-generate timestamp
		if msg.Ts != 0 {
			t.Error("Initial timestamp should be 0")
		}

		data, err := serializer.Serialize(msg)
		if err != nil {
			t.Fatalf("Failed to serialize: %v", err)
		}

		var result JSONMessage
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.Ts == 0 {
			t.Error("Timestamp should be auto-generated")
		}
	})
}

func TestMessageRouter(t *testing.T) {
	router := NewMessageRouter()

	t.Run("RegisterAndRoute", func(t *testing.T) {
		handlerCalled := false
		router.RegisterHandlerFunc(ClineMessageTypeSay, func(msg *JSONMessage) error {
			handlerCalled = true
			return nil
		})

		msg := &JSONMessage{
			Type: string(ClineMessageTypeSay),
			Say:  string(ClineSayText),
			Text: "Test",
		}

		if err := router.Route(msg); err != nil {
			t.Fatalf("Failed to route: %v", err)
		}

		if !handlerCalled {
			t.Error("Handler should have been called")
		}
	})

	t.Run("MissingHandler", func(t *testing.T) {
		msg := &JSONMessage{
			Type: string(ClineMessageTypeAsk),
			Ask:  string(ClineAskFollowup),
			Text: "Test",
		}

		err := router.Route(msg)
		if err == nil {
			t.Error("Should return error for missing handler")
		}
	})

	t.Run("NilMessage", func(t *testing.T) {
		err := router.Route(nil)
		if err == nil {
			t.Error("Should return error for nil message")
		}
	})

	t.Run("HasHandler", func(t *testing.T) {
		if !router.HasHandler(ClineMessageTypeSay) {
			t.Error("Should have handler for Say type")
		}

		if router.HasHandler(ClineMessageTypeAsk) {
			t.Error("Should not have handler for Ask type")
		}
	})
}

func TestMessageValidator(t *testing.T) {
	validator := NewMessageValidator()

	t.Run("ValidSayMessage", func(t *testing.T) {
		msg := &JSONMessage{
			Ts:   1234567890,
			Type: string(ClineMessageTypeSay),
			Say:  string(ClineSayText),
			Text: "Test",
		}

		if err := validator.Validate(msg); err != nil {
			t.Errorf("Should be valid: %v", err)
		}
	})

	t.Run("MissingTimestamp", func(t *testing.T) {
		msg := &JSONMessage{
			Type: string(ClineMessageTypeSay),
			Say:  string(ClineSayText),
			Text: "Test",
		}

		if err := validator.Validate(msg); err == nil {
			t.Error("Should fail validation without timestamp")
		}
	})

	t.Run("MissingType", func(t *testing.T) {
		msg := &JSONMessage{
			Ts:  1234567890,
			Say: string(ClineSayText),
		}

		if err := validator.Validate(msg); err == nil {
			t.Error("Should fail validation without type")
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		msg := &JSONMessage{
			Ts:   1234567890,
			Type: "invalid_type",
			Text: "Test",
		}

		if err := validator.Validate(msg); err == nil {
			t.Error("Should fail validation with invalid type")
		}
	})

	t.Run("MissingSayType", func(t *testing.T) {
		msg := &JSONMessage{
			Ts:   1234567890,
			Type: string(ClineMessageTypeSay),
			Text: "Test",
		}

		if err := validator.Validate(msg); err == nil {
			t.Error("Should fail validation without say type")
		}
	})

	t.Run("NilMessage", func(t *testing.T) {
		if err := validator.Validate(nil); err == nil {
			t.Error("Should fail validation for nil message")
		}
	})

	t.Run("ValidateBatch", func(t *testing.T) {
		messages := []*JSONMessage{
			{
				Ts:   1234567890,
				Type: string(ClineMessageTypeSay),
				Say:  string(ClineSayText),
				Text: "Valid",
			},
			{
				Ts:   1234567891,
				Type: string(ClineMessageTypeAsk),
				Ask:  string(ClineAskFollowup),
				Text: "Valid",
			},
			{
				// Invalid - missing timestamp
				Type: string(ClineMessageTypeSay),
				Say:  string(ClineSayText),
			},
		}

		errors := validator.ValidateBatch(messages)
		if len(errors) != 1 {
			t.Errorf("Expected 1 validation error, got %d", len(errors))
		}
	})
}

func TestJSONMessageTypes(t *testing.T) {
	t.Run("MessageTypes", func(t *testing.T) {
		// Verify all message types are defined
		types := []ClineMessageType{
			ClineMessageTypeSay,
			ClineMessageTypeAsk,
			ClineMessageTypeToolUse,
			ClineMessageTypeToolResult,
			ClineMessageTypeCommand,
			ClineMessageTypeCommandOutput,
			ClineMessageTypeCheckpoint,
			ClineMessageTypeBrowserAction,
			ClineMessageTypeMCPRequest,
			ClineMessageTypeError,
			ClineMessageTypeSystem,
		}

		for _, mt := range types {
			if string(mt) == "" {
				t.Error("Message type should not be empty")
			}
		}
	})

	t.Run("SayTypes", func(t *testing.T) {
		sayTypes := []ClineSay{
			ClineSayTask,
			ClineSayError,
			ClineSayAPIReqStarted,
			ClineSayAPIReqFinished,
			ClineSayText,
			ClineSayReasoning,
			ClineSayCompletionResult,
			ClineSayCommand,
			ClineSayCommandOutput,
			ClineSayTool,
			ClineSayToolUse,
			ClineSayToolResult,
			ClineSayBrowserAction,
			ClineSayBrowserActionResult,
			ClineSayMCPServerRequestStarted,
			ClineSayMCPServerResponse,
			ClineSayCheckpointCreated,
			ClineSayInfo,
		}

		for _, st := range sayTypes {
			if string(st) == "" {
				t.Error("Say type should not be empty")
			}
		}
	})

	t.Run("AskTypes", func(t *testing.T) {
		askTypes := []ClineAsk{
			ClineAskFollowup,
			ClineAskPlanModeRespond,
			ClineAskActModeRespond,
			ClineAskCommand,
			ClineAskCompletionResult,
			ClineAskTool,
			ClineAskResumeTask,
			ClineAskNewTask,
		}

		for _, at := range askTypes {
			if string(at) == "" {
				t.Error("Ask type should not be empty")
			}
		}
	})
}