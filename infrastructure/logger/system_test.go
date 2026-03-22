package logger

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// captureLog redirects the standard logger output and runs fn,
// returning everything that was printed.
func captureLog(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0) // drop timestamps for stable assertions
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()
	fn()
	return buf.String()
}

// --- SystemLog.Log ---

func TestSystemLog_Log(t *testing.T) {
	l := NewSystem()
	out := captureLog(func() { l.Log("hello world") })
	if !strings.Contains(out, "hello world") {
		t.Fatalf("Log output = %q, want 'hello world'", out)
	}
}

// --- SystemLog.Logf ---

func TestSystemLog_Logf(t *testing.T) {
	l := NewSystem()
	out := captureLog(func() { l.Logf("count=%d name=%s", 42, "bot") })
	if !strings.Contains(out, "count=42 name=bot") {
		t.Fatalf("Logf output = %q", out)
	}
}

// --- SystemLog.LogToFile ---

func TestSystemLog_LogToFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"

	l := NewSystem()
	l.LogToFile(tmpFile, []string{"line1", "line2"})

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "line1") || !strings.Contains(content, "line2") {
		t.Fatalf("file content = %q", content)
	}
}

func TestSystemLog_LogToFile_Append(t *testing.T) {
	tmpFile := t.TempDir() + "/test.log"

	l := NewSystem()
	l.LogToFile(tmpFile, []string{"first"})
	l.LogToFile(tmpFile, []string{"second"})

	data, _ := os.ReadFile(tmpFile)
	content := string(data)
	if !strings.Contains(content, "first") || !strings.Contains(content, "second") {
		t.Fatalf("expected both lines, got %q", content)
	}
}

func TestSystemLog_LogToFile_InvalidPath(t *testing.T) {
	l := NewSystem()
	// Should not panic on bad path — just silently return.
	l.LogToFile("/nonexistent/dir/file.log", []string{"nope"})
}

// --- Interface compliance ---

func TestSystemLog_ImplementsLog(t *testing.T) {
	var _ Log = (*SystemLog)(nil)
}

func TestSystemLog_ImplementsFileLog(t *testing.T) {
	var _ FileLog = (*SystemLog)(nil)
}
