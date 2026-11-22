package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
	// The API rate-limit window is one minute; never wait longer than that.
	maxRetryAfter = 60 * time.Second
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 && body != nil {
			data, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(data)
		}

		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "terraform-provider-nexdns")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				sleep(attempt)
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		if resp.StatusCode >= 500 && attempt < maxRetries-1 {
			lastErr = fmt.Errorf("server error: HTTP %d", resp.StatusCode)
			sleep(attempt)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries-1 {
			// Wait as long as the response asks: the window is a full minute.
			if wait := retryAfter(resp); wait > 0 {
				time.Sleep(wait)
			} else {
				sleep(attempt)
			}
			continue
		}

		if resp.StatusCode >= 400 {
			var errResp struct {
				Error *APIError `json:"error"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
				errResp.Error.StatusCode = resp.StatusCode
				return errResp.Error
			}
			return &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
		}

		if result == nil {
			return nil
		}

		var envelope APIResponse
		if err := json.Unmarshal(respBody, &envelope); err == nil && envelope.Status == "success" {
			if len(envelope.Data) > 0 {
				return json.Unmarshal(envelope.Data, result)
			}
			return nil
		}

		return json.Unmarshal(respBody, result)
	}
	return lastErr
}

func sleep(attempt int) {
	delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
	jitter := time.Duration(rand.Int64N(int64(500 * time.Millisecond)))
	time.Sleep(delay + jitter)
}

// retryAfter reads the wait the server asked for, from either the standard
// Retry-After header (seconds) or X-RateLimit-Reset (an absolute unix time).
// Returns 0 when neither is usable or the value is implausible.
func retryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
			return capWait(time.Duration(seconds) * time.Second)
		}
	}

	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if reset, err := strconv.ParseInt(v, 10, 64); err == nil {
			if wait := time.Until(time.Unix(reset, 0)); wait > 0 {
				return capWait(wait)
			}
		}
	}

	return 0
}

// capWait bounds the wait so a bad header cannot hang a plan indefinitely.
func capWait(d time.Duration) time.Duration {
	if d > maxRetryAfter {
		return maxRetryAfter
	}
	return d + time.Second // a second of slack for clock skew
}
