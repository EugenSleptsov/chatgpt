package openai

import (
	gpt "GPTBot/domain/ai"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// === Fakes ===

// fakeTransport is a stub Transport for tests.
type fakeTransport struct {
	handler func(url, contentType string, payload []byte) (*http.Response, error)
}

func (f *fakeTransport) Post(url, contentType string, payload []byte) (*http.Response, error) {
	return f.handler(url, contentType, payload)
}

// jsonResponse builds an *http.Response with the given status and JSON body.
func jsonResponse(status int, body interface{}) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

// textResponse builds an *http.Response with a plain-text body.
func textResponse(status int, text string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(text)),
	}
}

// fakeLog implements logger.Log, keeps log messages for assertions.
type fakeLog struct {
	messages []string
}

func (l *fakeLog) Log(message string) { l.messages = append(l.messages, message) }
func (l *fakeLog) Logf(format string, v ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf(format, v...))
}

// newTestClient creates a Client with a fake transport and logger.
func newTestClient(handler func(url, contentType string, payload []byte) (*http.Response, error)) (*Client, *fakeLog) {
	log := &fakeLog{}
	return &Client{
		Transport: &fakeTransport{handler: handler},
		Log:       log,
	}, log
}

// === CallGPT ===

func TestCallGPT_Success(t *testing.T) {
	want := gpt.Response{
		ID:     "resp-1",
		Object: "response",
		Output: []gpt.ResponseOutputItem{
			{Type: "message", Content: []gpt.ResponseOutputContent{{Type: "output_text", Text: "hello"}}},
		},
		Usage: gpt.ResponseUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
	}

	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		// Verify URL
		if url != responsesEndpoint {
			t.Errorf("unexpected URL: %s", url)
		}
		// Verify payload has correct model
		var req RequestResponsesPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if req.Model != ResolveModel("basic") {
			t.Errorf("model = %q, want %q", req.Model, ResolveModel("basic"))
		}
		if !req.Store {
			t.Error("Store should be true")
		}
		// No default tools: built-in tools (web_search, image_generation) are
		// passed by the caller when needed, not injected automatically.
		if len(req.Tools) != 0 {
			t.Errorf("expected 0 default tools, got %+v", req.Tools)
		}
		return jsonResponse(200, want), nil
	})

	msgs := []gpt.Message{{Role: "user", Content: "hi"}}
	resp, err := client.CallGPT(msgs, "basic", "you are helpful")
	if err != nil {
		t.Fatalf("CallGPT error: %v", err)
	}
	if resp.OutputText() != "hello" {
		t.Fatalf("OutputText = %q, want hello", resp.OutputText())
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

func TestCallGPT_MergesExtraTools(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		var req RequestResponsesPayload
		json.Unmarshal(payload, &req)
		// 0 default + 1 custom = 1
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		return jsonResponse(200, gpt.Response{ID: "r1"}), nil
	})

	customTool := gpt.Tool{Type: "function", Name: "get_weather", Description: "weather"}
	_, err := client.CallGPT(nil, "fast", "", customTool)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCallGPT_APIError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(429, `{"error":"rate_limit"}`), nil
	})

	_, err := client.CallGPT(nil, "basic", "")
	if err == nil {
		t.Fatal("expected error on 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("error should contain status code, got: %v", err)
	}
}

func TestCallGPT_TransportError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	})

	_, err := client.CallGPT(nil, "basic", "")
	if err == nil {
		t.Fatal("expected transport error")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCallGPT_BadJSONResponse(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(200, "not-json!!!"), nil
	})

	_, err := client.CallGPT(nil, "basic", "")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestCallGPT_ResolvesModelName(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		var req RequestResponsesPayload
		json.Unmarshal(payload, &req)
		expected := ResolveModel("premium")
		if req.Model != expected {
			t.Errorf("model = %q, want %q", req.Model, expected)
		}
		return jsonResponse(200, gpt.Response{ID: "r"}), nil
	})

	client.CallGPT(nil, "premium", "")
}

func TestCallGPT_UnknownModelFallsToDefault(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		var req RequestResponsesPayload
		json.Unmarshal(payload, &req)
		if req.Model != ResolveModel("some-unknown-model") {
			t.Errorf("model = %q, want default %q", req.Model, ResolveModel("some-unknown-model"))
		}
		return jsonResponse(200, gpt.Response{ID: "r"}), nil
	})

	client.CallGPT(nil, "some-unknown-model", "")
}

// === ContinueWithToolOutputs ===

func TestContinueWithToolOutputs_Success(t *testing.T) {
	want := gpt.Response{
		ID: "resp-cont",
		Output: []gpt.ResponseOutputItem{
			{Type: "message", Content: []gpt.ResponseOutputContent{{Text: "final answer"}}},
		},
	}

	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		var req ContinueResponsesPayload
		if err := json.Unmarshal(payload, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if req.PreviousResponseID != "prev-123" {
			t.Errorf("PreviousResponseID = %q", req.PreviousResponseID)
		}
		if len(req.Input) != 1 {
			t.Errorf("expected 1 tool output, got %d", len(req.Input))
		}
		if !req.Store {
			t.Error("Store should be true")
		}
		return jsonResponse(200, want), nil
	})

	outputs := []gpt.ToolCallOutput{gpt.NewToolCallOutput("call-1", "result")}
	resp, err := client.ContinueWithToolOutputs("prev-123", outputs, "basic", "instr")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp.OutputText() != "final answer" {
		t.Fatalf("OutputText = %q", resp.OutputText())
	}
}

func TestContinueWithToolOutputs_APIError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(500, "internal server error"), nil
	})

	_, err := client.ContinueWithToolOutputs("prev", nil, "basic", "")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

// === GenerateImage ===

func TestGenerateImage_Success(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		var req RequestImagePayload
		json.Unmarshal(payload, &req)
		if req.Model != ModelDalle3 {
			t.Errorf("model = %q, want %q", req.Model, ModelDalle3)
		}
		if req.Prompt != "a cat" {
			t.Errorf("prompt = %q", req.Prompt)
		}
		body := ResponseImagePayload{Data: []struct {
			URL string `json:"url"`
		}{{URL: "https://example.com/cat.png"}}}
		return jsonResponse(200, body), nil
	})

	url, err := client.GenerateImage("a cat", gpt.ImageSize1024)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if url != "https://example.com/cat.png" {
		t.Fatalf("url = %q", url)
	}
}

func TestGenerateImage_APIError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(400, `{"error":"bad request"}`), nil
	})

	_, err := client.GenerateImage("x", gpt.ImageSize256)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error should contain 400: %v", err)
	}
}

func TestGenerateImage_EmptyData(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		body := ResponseImagePayload{}
		return jsonResponse(200, body), nil
	})

	_, err := client.GenerateImage("x", gpt.ImageSize256)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
	if !strings.Contains(err.Error(), "empty data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateImage_ContentPolicyViolation(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		body := map[string]interface{}{
			"data": []interface{}{},
			"error": map[string]string{
				"code":    "content_policy_violation",
				"message": "blocked",
				"type":    "invalid_request_error",
			},
		}
		return jsonResponse(200, body), nil
	})

	_, err := client.GenerateImage("bad prompt", gpt.ImageSize1024)
	if err == nil {
		t.Fatal("expected content policy error")
	}
	if !strings.Contains(err.Error(), "content policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateImage_SizeMapping(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{gpt.ImageSize256, gpt.ImageSize256},
		{gpt.ImageSize512, gpt.ImageSize512},
		{gpt.ImageSize1024, gpt.ImageSize1024},
		{"invalid", gpt.ImageSize1024}, // default fallback
		{"", gpt.ImageSize1024},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
				var req RequestImagePayload
				json.Unmarshal(payload, &req)
				if req.Size != tc.want {
					t.Errorf("size = %q, want %q", req.Size, tc.want)
				}
				body := ResponseImagePayload{Data: []struct {
					URL string `json:"url"`
				}{{URL: "https://x.com/img.png"}}}
				return jsonResponse(200, body), nil
			})

			client.GenerateImage("test", tc.input)
		})
	}
}

// === TranscribeAudio ===

func TestTranscribeAudio_Success(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		if url != audioEndpoint {
			t.Errorf("url = %q, want %q", url, audioEndpoint)
		}
		if !strings.Contains(contentType, "multipart/form-data") {
			t.Errorf("expected multipart content-type, got %q", contentType)
		}
		// Verify payload contains audio data and model field
		body := string(payload)
		if !strings.Contains(body, "audio.ogg") {
			t.Error("payload should contain filename audio.ogg")
		}
		if !strings.Contains(body, "whisper-1") {
			t.Error("payload should contain model whisper-1")
		}
		resp := TranscriptionResponse{Text: "transcribed text"}
		return jsonResponse(200, resp), nil
	})

	text, err := client.TranscribeAudio([]byte("fake-audio"))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if text != "transcribed text" {
		t.Fatalf("text = %q", text)
	}
}

func TestTranscribeAudio_TransportError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return nil, fmt.Errorf("timeout")
	})

	_, err := client.TranscribeAudio([]byte("audio"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranscribeAudio_BadJSON(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(200, "not-json"), nil
	})

	_, err := client.TranscribeAudio([]byte("audio"))
	if err == nil {
		t.Fatal("expected decode error")
	}
}

// === GenerateVoice ===

func TestGenerateVoice_Success(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		if url != voiceEndpoint {
			t.Errorf("url = %q, want %q", url, voiceEndpoint)
		}
		var req struct {
			Model string `json:"model"`
			Voice string `json:"voice"`
			Input string `json:"input"`
		}
		json.Unmarshal(payload, &req)
		if req.Model != gpt.VoiceModelHD {
			t.Errorf("model = %q", req.Model)
		}
		if req.Voice != gpt.VoiceOnyx {
			t.Errorf("voice = %q", req.Voice)
		}
		if req.Input != "hello world" {
			t.Errorf("input = %q", req.Input)
		}
		return textResponse(200, "mp3-bytes"), nil
	})

	audio, err := client.GenerateVoice("hello world", gpt.VoiceModelHD, gpt.VoiceOnyx)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if string(audio) != "mp3-bytes" {
		t.Fatalf("audio = %q", audio)
	}
}

func TestGenerateVoice_APIError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return textResponse(500, "server error"), nil
	})

	_, err := client.GenerateVoice("hi", "model", "voice")
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error should contain status: %v", err)
	}
}

func TestGenerateVoice_TransportError(t *testing.T) {
	client, _ := newTestClient(func(url, contentType string, payload []byte) (*http.Response, error) {
		return nil, fmt.Errorf("dns failure")
	})

	_, err := client.GenerateVoice("hi", "model", "voice")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dns failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}
