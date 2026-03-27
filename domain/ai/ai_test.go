package ai

import (
	"encoding/base64"
	"testing"
)

// --- OutputText ---

func TestOutputText_ReturnsFirstMessageText(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "message", Content: []ResponseOutputContent{{Type: "output_text", Text: "hello"}}},
		},
	}
	if got := r.OutputText(); got != "hello" {
		t.Fatalf("OutputText() = %q, want %q", got, "hello")
	}
}

func TestOutputText_NilResponse(t *testing.T) {
	var r *Response
	if got := r.OutputText(); got != "" {
		t.Fatalf("OutputText() on nil = %q, want empty", got)
	}
}

func TestOutputText_NoMessageItems(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "function_call", Name: "fn"},
		},
	}
	if got := r.OutputText(); got != "" {
		t.Fatalf("OutputText() = %q, want empty", got)
	}
}

func TestOutputText_EmptyContent(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "message", Content: nil},
		},
	}
	if got := r.OutputText(); got != "" {
		t.Fatalf("OutputText() = %q, want empty", got)
	}
}

// --- ToolCalls ---

func TestToolCalls_ParsesFunctionCalls(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{
				Type:      "function_call",
				CallID:    "call-1",
				Name:      "get_weather",
				Arguments: `{"city":"Moscow"}`,
			},
		},
	}
	calls := r.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("len(ToolCalls()) = %d, want 1", len(calls))
	}
	if calls[0].ID != "call-1" || calls[0].Name != "get_weather" {
		t.Fatalf("unexpected call: %+v", calls[0])
	}
	if calls[0].Args["city"] != "Moscow" {
		t.Fatalf("Args[city] = %q, want Moscow", calls[0].Args["city"])
	}
}

func TestToolCalls_NilResponse(t *testing.T) {
	var r *Response
	if calls := r.ToolCalls(); calls != nil {
		t.Fatalf("ToolCalls() on nil = %v, want nil", calls)
	}
}

func TestToolCalls_InvalidJSON(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "function_call", CallID: "c1", Name: "fn", Arguments: "not-json"},
		},
	}
	calls := r.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("len = %d, want 1", len(calls))
	}
	if len(calls[0].Args) != 0 {
		t.Fatalf("Args should be empty on bad JSON, got %v", calls[0].Args)
	}
}

func TestToolCalls_SkipsNonFunctionItems(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "message", Content: []ResponseOutputContent{{Text: "hi"}}},
			{Type: "web_search_call"},
		},
	}
	if calls := r.ToolCalls(); len(calls) != 0 {
		t.Fatalf("expected 0 tool calls, got %d", len(calls))
	}
}

// --- ImageResults ---

func TestImageResults_DecodesBase64(t *testing.T) {
	raw := []byte("png-data")
	b64 := base64.StdEncoding.EncodeToString(raw)
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "image_generation_call", Result: b64},
		},
	}
	imgs := r.ImageResults()
	if len(imgs) != 1 {
		t.Fatalf("len = %d, want 1", len(imgs))
	}
	if string(imgs[0]) != "png-data" {
		t.Fatalf("decoded = %q, want %q", imgs[0], "png-data")
	}
}

func TestImageResults_NilResponse(t *testing.T) {
	var r *Response
	if imgs := r.ImageResults(); imgs != nil {
		t.Fatalf("ImageResults() on nil = %v, want nil", imgs)
	}
}

func TestImageResults_SkipsBadBase64(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "image_generation_call", Result: "%%%invalid%%%"},
		},
	}
	imgs := r.ImageResults()
	if len(imgs) != 0 {
		t.Fatalf("expected 0 images for bad base64, got %d", len(imgs))
	}
}

func TestImageResults_SkipsEmptyResult(t *testing.T) {
	r := &Response{
		Output: []ResponseOutputItem{
			{Type: "image_generation_call", Result: ""},
		},
	}
	if imgs := r.ImageResults(); len(imgs) != 0 {
		t.Fatalf("expected 0, got %d", len(imgs))
	}
}

// --- NewToolCallOutput ---

func TestNewToolCallOutput(t *testing.T) {
	out := NewToolCallOutput("call-42", "result-text")
	if out.Type != "function_call_output" {
		t.Fatalf("Type = %q, want function_call_output", out.Type)
	}
	if out.CallID != "call-42" {
		t.Fatalf("CallID = %q", out.CallID)
	}
	if out.Output != "result-text" {
		t.Fatalf("Output = %q", out.Output)
	}
}
