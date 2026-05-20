package clinicaltrials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *Client) requestJSON(ctx context.Context, source string, method string, endpoint string, values url.Values, body any) (map[string]any, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		requestURL := endpoint
		if len(values) > 0 {
			requestURL += "?" + values.Encode()
		}
		var requestBody io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("encode %s request: %w", source, err)
			}
			requestBody = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
		if err != nil {
			return nil, fmt.Errorf("build %s request: %w", source, err)
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://euclinicaltrials.eu")
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 clinical-trials-mcp/0.1")

		resp, err := c.httpClient.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read %s response: %w", source, readErr)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close %s response: %w", source, closeErr)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = fmt.Errorf("%s returned HTTP %d for %s", source, resp.StatusCode, endpoint)
				if !retryableStatus(resp.StatusCode) {
					return nil, lastErr
				}
			} else {
				contentType := resp.Header.Get("content-type")
				if !strings.Contains(contentType, "json") && strings.HasPrefix(strings.TrimSpace(string(body)), "<") {
					return nil, fmt.Errorf("%s returned HTML instead of JSON for %s", source, endpoint)
				}
				var data map[string]any
				if err := json.Unmarshal(body, &data); err != nil {
					lastErr = err
				} else {
					return data, nil
				}
			}
		} else {
			lastErr = err
		}

		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(500*(1<<attempt)) * time.Millisecond):
			}
		}
	}

	return nil, fmt.Errorf("%s request failed for %s: %w", source, endpoint, lastErr)
}

func retryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
