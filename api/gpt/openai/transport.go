package openai

import (
	"bytes"
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
}

func NewHTTPTransport(apiKey string) *HTTPTransport {
	return &HTTPTransport{ApiKey: apiKey, Retries: 3}
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

		resp, err = (&http.Client{}).Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return resp, err
}
