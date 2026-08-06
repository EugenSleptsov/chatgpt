package openai

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// A slow call must be abandoned after ONE attempt. Retrying re-runs the same
// long model call from scratch, so N retries multiply both the wall-clock time
// the caller's goroutine is blocked and the amount OpenAI bills.
func TestPost_TimeoutIsNotRetried(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &HTTPTransport{
		ApiKey:  "test",
		Retries: 5,
		client:  &http.Client{Timeout: 50 * time.Millisecond},
	}

	start := time.Now()
	_, err := transport.Post(srv.URL, "application/json", []byte(`{}`))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("server hit %d times, want 1 (timeouts must not be retried)", got)
	}
	if elapsed > 250*time.Millisecond {
		t.Errorf("Post took %v, want ~one timeout (no backoff sleeps)", elapsed)
	}
}

// Server errors are still retried — only client-side timeouts are exempt.
func TestPost_ServerErrorIsRetried(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &HTTPTransport{
		ApiKey:  "test",
		Retries: 5,
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	resp, err := transport.Post(srv.URL, "application/json", []byte(`{}`))
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("server hit %d times, want 3", got)
	}
}
