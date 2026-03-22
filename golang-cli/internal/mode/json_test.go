package mode

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewJSONFormatter(t *testing.T) {
	tests := []struct {
		name     string
		opts     []JSONOption
		wantPretty bool
	}{
		{
			name:     "default formatter",
			opts:     nil,
			wantPretty: false,
		},
		{
			name:     "pretty formatter",
			opts:     []JSONOption{WithPretty(true)},
			wantPretty: true,
		},
		{
			name:     "non-pretty formatter",
			opts:     []JSONOption{WithPretty(false)},
			wantPretty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter(tt.opts...)
			if f.pretty != tt.wantPretty {
				t.Errorf("NewJSONFormatter() pretty = %v, want %v", f.pretty, tt.wantPretty)
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		output  *JSONOutput
		pretty  bool
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "basic text output",
			output: &JSONOutput{
				Type:      OutputTypeText,
				Text:      "Hello, World!",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Check deterministic key ordering: ask, files, images, partial, reasoning, say, text, ts, type
				want := `{"text":"Hello, World!","ts":1234567890000,"type":"text"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "say output with all optional fields",
			output: &JSONOutput{
				Type:      OutputTypeSay,
				Text:      "I will help you",
				Say:       "assistant_response",
				Reasoning: "Let me think about this",
				Partial:   true,
				Images:    []string{"image1.png", "image2.png"},
				Files:     []string{"file1.go", "file2.go"},
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Keys should be sorted alphabetically
				want := `{"files":["file1.go","file2.go"],"images":["image1.png","image2.png"],"partial":true,"reasoning":"Let me think about this","say":"assistant_response","text":"I will help you","ts":1234567890000,"type":"say"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "ask output",
			output: &JSONOutput{
				Type:      OutputTypeAsk,
				Text:      "What would you like to do?",
				Ask:       "followup",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				want := `{"ask":"followup","text":"What would you like to do?","ts":1234567890000,"type":"ask"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "error output",
			output: &JSONOutput{
				Type:      OutputTypeError,
				Text:      "Something went wrong",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				want := `{"text":"Something went wrong","ts":1234567890000,"type":"error"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "tool use output",
			output: &JSONOutput{
				Type:      OutputTypeToolUse,
				Text:      "Using tool: read_file",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				want := `{"text":"Using tool: read_file","ts":1234567890000,"type":"tool_use"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "tool result output",
			output: &JSONOutput{
				Type:      OutputTypeToolResult,
				Text:      "File contents here",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				want := `{"text":"File contents here","ts":1234567890000,"type":"tool_result"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name: "reasoning output",
			output: &JSONOutput{
				Type:      OutputTypeReasoning,
				Text:      "Thinking through this problem",
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				want := `{"text":"Thinking through this problem","ts":1234567890000,"type":"reasoning"}`
				if string(data) != want {
					t.Errorf("Format() = %s, want %s", string(data), want)
				}
			},
		},
		{
			name:    "nil output",
			output:  nil,
			pretty:  false,
			wantErr: true,
		},
		{
			name: "pretty print output",
			output: &JSONOutput{
				Type:      OutputTypeText,
				Text:      "Pretty please",
				Timestamp: 1234567890000,
			},
			pretty:  true,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Should contain newlines and indentation
				if !bytes.Contains(data, []byte("\n")) {
					t.Error("Pretty output should contain newlines")
				}
				if !bytes.Contains(data, []byte("  ")) {
					t.Error("Pretty output should contain indentation")
				}
			},
		},
		{
			name: "auto-timestamp when zero",
			output: &JSONOutput{
				Type: OutputTypeText,
				Text: "Auto timestamp",
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Should have a non-zero timestamp
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("Failed to unmarshal: %v", err)
				}
				if result["ts"].(float64) == 0 {
					t.Error("Expected auto-generated timestamp")
				}
			},
		},
		{
			name: "empty slices not included",
			output: &JSONOutput{
				Type:      OutputTypeText,
				Text:      "No empty slices",
				Images:    []string{},
				Files:     []string{},
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Empty slices should not appear in output
				if bytes.Contains(data, []byte("images")) {
					t.Error("Empty images slice should not be included")
				}
				if bytes.Contains(data, []byte("files")) {
					t.Error("Empty files slice should not be included")
				}
			},
		},
		{
			name: "partial false not included",
			output: &JSONOutput{
				Type:      OutputTypeText,
				Text:      "Not partial",
				Partial:   false,
				Timestamp: 1234567890000,
			},
			pretty:  false,
			wantErr: false,
			check: func(t *testing.T, data []byte) {
				// Partial=false should not appear in output (check for "partial":false or "partial": true)
				if bytes.Contains(data, []byte(`"partial":false`)) || bytes.Contains(data, []byte(`"partial":true`)) {
					t.Error("Partial=false should not be included in deterministic output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter(WithPretty(tt.pretty))
			data, err := f.Format(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

func TestJSONFormatter_FormatToWriter(t *testing.T) {
	tests := []struct {
		name    string
		output  *JSONOutput
		wantErr bool
	}{
		{
			name: "write to buffer",
			output: &JSONOutput{
				Type:      OutputTypeText,
				Text:      "Hello, Writer!",
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewJSONFormatter(WithWriter(&buf))

			err := f.FormatToWriter(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("FormatToWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check output ends with newline
				output := buf.String()
				if !strings.HasSuffix(output, "\n") {
					t.Error("Output should end with newline")
				}

				// Check it contains valid JSON
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
					t.Errorf("Output is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestJSONFormatter_FormatToWriter_NoWriter(t *testing.T) {
	f := NewJSONFormatter()
	output := &JSONOutput{
		Type:      OutputTypeText,
		Text:      "test",
		Timestamp: 1234567890000,
	}

	err := f.FormatToWriter(output)
	if err == nil {
		t.Error("Expected error when no writer configured")
	}
}

func TestJSONFormatter_FormatError(t *testing.T) {
	f := NewJSONFormatter()
	data, err := f.FormatError("test error message")

	if err != nil {
		t.Fatalf("FormatError() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal error output: %v", err)
	}

	if result["type"] != "error" {
		t.Errorf("Expected type=error, got %v", result["type"])
	}
	if result["text"] != "test error message" {
		t.Errorf("Expected text='test error message', got %v", result["text"])
	}
	if result["ts"].(float64) == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestJSONFormatter_FormatErrorToWriter(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(WithWriter(&buf))

	err := f.FormatErrorToWriter("test error")
	if err != nil {
		t.Fatalf("FormatErrorToWriter() error = %v", err)
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	if result["type"] != "error" {
		t.Errorf("Expected type=error, got %v", result["type"])
	}
}

func TestJSONFormatter_FormatErrorToWriter_NoWriter(t *testing.T) {
	f := NewJSONFormatter()
	err := f.FormatErrorToWriter("test error")
	if err == nil {
		t.Error("Expected error when no writer configured")
	}
}

func TestNewJSONOutput(t *testing.T) {
	output := NewJSONOutput(OutputTypeText, "test content")

	if output.Type != OutputTypeText {
		t.Errorf("Expected type=text, got %v", output.Type)
	}
	if output.Text != "test content" {
		t.Errorf("Expected text='test content', got %v", output.Text)
	}
	if output.Timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestNewSayOutput(t *testing.T) {
	output := NewSayOutput("Hello", "greeting")

	if output.Type != OutputTypeSay {
		t.Errorf("Expected type=say, got %v", output.Type)
	}
	if output.Text != "Hello" {
		t.Errorf("Expected text='Hello', got %v", output.Text)
	}
	if output.Say != "greeting" {
		t.Errorf("Expected say='greeting', got %v", output.Say)
	}
}

func TestNewAskOutput(t *testing.T) {
	output := NewAskOutput("Question?", "followup")

	if output.Type != OutputTypeAsk {
		t.Errorf("Expected type=ask, got %v", output.Type)
	}
	if output.Text != "Question?" {
		t.Errorf("Expected text='Question?', got %v", output.Text)
	}
	if output.Ask != "followup" {
		t.Errorf("Expected ask='followup', got %v", output.Ask)
	}
}

func TestNewTextOutput(t *testing.T) {
	output := NewTextOutput("plain text")

	if output.Type != OutputTypeText {
		t.Errorf("Expected type=text, got %v", output.Type)
	}
	if output.Text != "plain text" {
		t.Errorf("Expected text='plain text', got %v", output.Text)
	}
}

func TestNewErrorOutput(t *testing.T) {
	output := NewErrorOutput("error message")

	if output.Type != OutputTypeError {
		t.Errorf("Expected type=error, got %v", output.Type)
	}
	if output.Text != "error message" {
		t.Errorf("Expected text='error message', got %v", output.Text)
	}
}

func TestNewReasoningOutput(t *testing.T) {
	output := NewReasoningOutput("reasoning text")

	if output.Type != OutputTypeReasoning {
		t.Errorf("Expected type=reasoning, got %v", output.Type)
	}
	if output.Text != "reasoning text" {
		t.Errorf("Expected text='reasoning text', got %v", output.Text)
	}
}

func TestJSONOutput_Chaining(t *testing.T) {
	output := NewJSONOutput(OutputTypeText, "base").
		SetReasoning("reasoning").
		SetSay("say_type").
		SetAsk("ask_type").
		SetPartial(true).
		SetImages([]string{"img1.png"}).
		SetFiles([]string{"file1.go"}).
		SetTimestamp(1234567890000)

	if output.Reasoning != "reasoning" {
		t.Errorf("Expected reasoning='reasoning', got %v", output.Reasoning)
	}
	if output.Say != "say_type" {
		t.Errorf("Expected say='say_type', got %v", output.Say)
	}
	if output.Ask != "ask_type" {
		t.Errorf("Expected ask='ask_type', got %v", output.Ask)
	}
	if !output.Partial {
		t.Error("Expected partial=true")
	}
	if len(output.Images) != 1 || output.Images[0] != "img1.png" {
		t.Errorf("Expected images=['img1.png'], got %v", output.Images)
	}
	if len(output.Files) != 1 || output.Files[0] != "file1.go" {
		t.Errorf("Expected files=['file1.go'], got %v", output.Files)
	}
	if output.Timestamp != 1234567890000 {
		t.Errorf("Expected timestamp=1234567890000, got %v", output.Timestamp)
	}
}

func TestMarshalSorted(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want string
	}{
		{
			name: "empty map",
			m:    map[string]interface{}{},
			want: "{}",
		},
		{
			name: "single key",
			m:    map[string]interface{}{"a": 1},
			want: `{"a":1}`,
		},
		{
			name: "sorted keys",
			m:    map[string]interface{}{"z": 1, "a": 2, "m": 3},
			want: `{"a":2,"m":3,"z":1}`,
		},
		{
			name: "various types",
			m: map[string]interface{}{
				"string":  "value",
				"number":  42,
				"bool":    true,
				"array":   []string{"a", "b"},
				"null":    nil,
			},
			want: `{"array":["a","b"],"bool":true,"null":null,"number":42,"string":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalSorted(tt.m)
			if err != nil {
				t.Errorf("marshalSorted() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("marshalSorted() = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestMarshalSorted_Escaping(t *testing.T) {
	// Test that keys and values are properly escaped
	m := map[string]interface{}{
		`key"with"quotes`: `value"with"quotes`,
		`key\nwith\nnewlines`: "value\nwith\nnewlines",
	}

	data, err := marshalSorted(m)
	if err != nil {
		t.Fatalf("marshalSorted() error = %v", err)
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
}

func TestDeterministicOutput(t *testing.T) {
	// Run multiple times and verify same output
	output := &JSONOutput{
		Type:      OutputTypeSay,
		Text:      "Hello",
		Say:       "greeting",
		Reasoning: "Thinking...",
		Partial:   true,
		Images:    []string{"a.png", "b.png"},
		Files:     []string{"x.go", "y.go"},
		Timestamp: 1234567890000,
	}

	f := NewJSONFormatter()

	var firstResult []byte
	for i := 0; i < 5; i++ {
		data, err := f.Format(output)
		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		if i == 0 {
			firstResult = data
		} else {
			if string(data) != string(firstResult) {
				t.Errorf("Non-deterministic output: iteration %d differs from first", i)
			}
		}
	}
}

func TestAllOutputTypes(t *testing.T) {
	types := []struct {
		name     string
		typeVal  OutputType
		expected string
	}{
		{"say", OutputTypeSay, "say"},
		{"ask", OutputTypeAsk, "ask"},
		{"text", OutputTypeText, "text"},
		{"tool_use", OutputTypeToolUse, "tool_use"},
		{"tool_result", OutputTypeToolResult, "tool_result"},
		{"error", OutputTypeError, "error"},
		{"user", OutputTypeUser, "user"},
		{"reasoning", OutputTypeReasoning, "reasoning"},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			output := &JSONOutput{
				Type:      tt.typeVal,
				Text:      "test",
				Timestamp: 1234567890000,
			}

			f := NewJSONFormatter()
			data, err := f.Format(output)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if result["type"] != tt.expected {
				t.Errorf("Expected type=%s, got %v", tt.expected, result["type"])
			}
		})
	}
}

func TestJSONOutput_UnicodeAndSpecialChars(t *testing.T) {
	output := &JSONOutput{
		Type:      OutputTypeText,
		Text:      "Hello 世界! 🌍 <script>alert('xss')</script>",
		Timestamp: 1234567890000,
	}

	f := NewJSONFormatter()
	data, err := f.Format(output)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["text"] != output.Text {
		t.Errorf("Unicode text not preserved: got %v, want %v", result["text"], output.Text)
	}
}

func BenchmarkFormat(b *testing.B) {
	output := &JSONOutput{
		Type:      OutputTypeSay,
		Text:      "This is a test message with some content",
		Say:       "test_type",
		Reasoning: "Some reasoning here",
		Partial:   false,
		Images:    []string{"img1.png", "img2.png"},
		Files:     []string{"file1.go", "file2.go"},
		Timestamp: 1234567890000,
	}

	f := NewJSONFormatter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := f.Format(output)
		if err != nil {
			b.Fatalf("Format() error = %v", err)
		}
	}
}

func BenchmarkFormatPretty(b *testing.B) {
	output := &JSONOutput{
		Type:      OutputTypeSay,
		Text:      "This is a test message with some content",
		Say:       "test_type",
		Reasoning: "Some reasoning here",
		Partial:   false,
		Images:    []string{"img1.png", "img2.png"},
		Files:     []string{"file1.go", "file2.go"},
		Timestamp: 1234567890000,
	}

	f := NewJSONFormatter(WithPretty(true))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := f.Format(output)
		if err != nil {
			b.Fatalf("Format() error = %v", err)
		}
	}
}