package openai

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// Transport is the low-level HTTP abstraction used by the OpenAI client.
// Swap the implementation for testing or alternate backends.
type Transport interface {
	Post(url, contentType string, payload []byte) (*http.Response, error)
}

// HTTPTransport is the production Transport: adds auth header and retries.
type HTTPTransport struct {
	ApiKey  string
	Retries int
	client  *http.Client
}

func NewHTTPTransport(apiKey string) *HTTPTransport {
	return &HTTPTransport{
		ApiKey:  apiKey,
		Retries: 3,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (t *HTTPTransport) Post(url, contentType string, payload []byte) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for i := 0; i < t.Retries; i++ {
		req, reqErr := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if reqErr != nil {
			return nil, reqErr
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+t.ApiKey)

		resp, err = t.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		// Client errors (4xx): our request is wrong, retrying won't help.
		// Return immediately with body intact so the caller can read the error.
		if err == nil && resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, nil
		}

		// Server errors (5xx) or transient failures: drain body and retry.
		if err == nil && resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		time.Sleep(time.Duration(i+1) * time.Second)
	}

	// Retries exhausted: return whatever we got (caller is responsible for body).
	if err != nil {
		return nil, err
	}
	return resp, nil
}
