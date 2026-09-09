package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	baseURL *url.URL
	token   string
	csrf    string
	http    *http.Client
}

func New(endpoint, token, csrf string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid Incident Garden endpoint %q", endpoint)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	token = strings.TrimSpace(token)
	if len(token) >= len("Bearer ") && strings.EqualFold(token[:len("Bearer ")], "Bearer ") {
		token = strings.TrimSpace(token[len("Bearer "):])
	}
	return &Client{baseURL: u, token: token, csrf: csrf, http: httpClient}, nil
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    any
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Incident Garden API returned HTTP %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("Incident Garden API returned HTTP %d: %s", e.StatusCode, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details"`
	} `json:"error"`
}

func decodeAPIError(resp *http.Response, body []byte) error {
	e := &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	var envelope errorEnvelope
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
		e.Code, e.Message, e.Details = envelope.Error.Code, envelope.Error.Message, envelope.Error.Details
	}
	if e.Message == "" {
		e.Message = http.StatusText(resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		e.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	}
	return e
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return 0
}

func (c *Client) do(ctx context.Context, method, path string, request, response any) error {
	var payload []byte
	var err error
	if request != nil {
		payload, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	maxAttempts := 1
	if method == http.MethodGet {
		maxAttempts = 3
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL.ResolveReference(&url.URL{Path: path}).String(), bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if request != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		if method != http.MethodGet && method != http.MethodHead && c.csrf != "" {
			req.Header.Set("X-CSRF-Token", c.csrf)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: c.csrf})
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if response == nil || len(body) == 0 {
				return nil
			}
			if err := json.Unmarshal(body, response); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
			return nil
		}

		apiErr := decodeAPIError(resp, body)
		var typed *APIError
		if attempt+1 < maxAttempts && errors.As(apiErr, &typed) && typed.StatusCode == http.StatusTooManyRequests {
			delay := typed.RetryAfter
			if delay <= 0 {
				delay = time.Duration(attempt+1) * 100 * time.Millisecond
			}
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				continue
			}
		}
		return apiErr
	}
	return errors.New("request attempts exhausted")
}

func orgPath(org, suffix string) string {
	return "/api/v1/orgs/" + url.PathEscape(org) + suffix
}

type pageEnvelope[T any] struct {
	Data struct {
		Items  []T `json:"items"`
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"data"`
}

func Paginate[T any](ctx context.Context, fetch func(context.Context, int, int) ([]T, int, error), limit int) ([]T, error) {
	if limit <= 0 {
		limit = 100
	}
	var all []T
	for offset := 0; ; offset += limit {
		items, total, err := fetch(ctx, limit, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(all) >= total || len(items) == 0 {
			return all, nil
		}
	}
}
