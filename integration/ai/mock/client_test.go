package mock

import (
	"GPTBot/domain/ai"
	"strings"
	"testing"
)

func TestCallGPT(t *testing.T) {
	c := NewClient()
	resp, err := c.CallGPT(nil, "test-model", "instructions")
	if err != nil {
		t.Fatalf("CallGPT error: %v", err)
	}
	text := resp.OutputText()
	if !strings.Contains(text, "mock") {
		t.Fatalf("expected mock marker, got %q", text)
	}
	if !strings.Contains(text, "test-model") {
		t.Fatalf("expected model name in response, got %q", text)
	}
}

func TestContinueWithToolOutputs(t *testing.T) {
	c := NewClient()
	outputs := []ai.ToolCallOutput{
		ai.NewToolCallOutput("c1", "result1"),
		ai.NewToolCallOutput("c2", "result2"),
	}
	resp, err := c.ContinueWithToolOutputs("prev-id", outputs, "m", "instr")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	text := resp.OutputText()
	if !strings.Contains(text, "2") {
		t.Fatalf("expected count of outputs in text, got %q", text)
	}
}

func TestGenerateImage(t *testing.T) {
	c := NewClient()
	data, err := c.GenerateImage("a cat", ai.ImageSize256)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected image bytes, got empty")
	}
}

func TestGenerateVoice(t *testing.T) {
	c := NewClient()
	audio, err := c.GenerateVoice("hello", ai.VoiceModelHD, ai.VoiceOnyx)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(audio) == 0 {
		t.Fatal("expected non-empty audio bytes")
	}
}

func TestTranscribeAudio(t *testing.T) {
	c := NewClient()
	text, err := c.TranscribeAudio([]byte("fake-audio"))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(text, "mock") {
		t.Fatalf("expected mock marker, got %q", text)
	}
}
