package openai

import (
	"bytes"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Transport is the low-level HTTP abstraction used by the OpenAI client.
// Swap the implementation for testing or alternate backends.
type Transport interface {
	Post(url, contentType string, payload []byte) (*http.Response, error)
}

// Retry constants.
const (
	retryBaseDelay = 1 * time.Second
	retryMaxDelay  = 30 * time.Second
	retryJitter    = 500 * time.Millisecond
)

// HTTPTransport is the production Transport: adds auth header and retries
type HTTPTransport struct {
	ApiKey  string
	Retries int
	client  *http.Client
}

func NewHTTPTransport(apiKey string) *HTTPTransport {
	return &HTTPTransport{
		ApiKey:  apiKey,
		Retries: 5,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// backoffDelay calculates exponential backoff with jitter:
//
//	delay = min(baseDelay * 2^attempt, maxDelay) + rand(0, jitter)
func backoffDelay(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(retryBaseDelay) * exp)
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}
	jitter := time.Duration(rand.Int63n(int64(retryJitter)))
	return delay + jitter
}

// parseRetryAfter reads the Retry-After header (seconds) from an HTTP response.
// Returns 0 if the header is absent or unparseable.
func parseRetryAfter(resp *http.Response) time.Duration {
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 0
	}
	secs, err := strconv.Atoi(val)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
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

		// Client errors (4xx except 429): our request is wrong, retrying won't help.
		// Return immediately with body intact so the caller can read the error.
		if err == nil && resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// 429 Too Many Requests: respect Retry-After header if present,
		// otherwise fall through to exponential backoff.
		if err == nil && resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter := parseRetryAfter(resp); retryAfter > 0 {
				log.Printf("[Transport] 429 rate limited, Retry-After: %v (attempt %d/%d)", retryAfter, i+1, t.Retries)
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				time.Sleep(retryAfter)
				continue
			}
		}

		// Server errors (5xx), rate limits without Retry-After, or transient
		// failures: drain body and retry with exponential backoff + jitter.
		if err == nil && resp != nil {
			log.Printf("[Transport] HTTP %d, retrying with backoff (attempt %d/%d)", resp.StatusCode, i+1, t.Retries)
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		} else if err != nil {
			log.Printf("[Transport] error: %v, retrying with backoff (attempt %d/%d)", err, i+1, t.Retries)
		}

		time.Sleep(backoffDelay(i))
	}

	// Retries exhausted: return whatever we got (caller is responsible for body).
	if err != nil {
		return nil, err
	}
	return resp, nil
}
