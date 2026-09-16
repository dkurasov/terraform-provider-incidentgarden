package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMutationAuthenticationAndEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs/acme/teams" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("X-CSRF-Token"); got != "csrf" {
			t.Errorf("csrf header = %q", got)
		}
		cookie, err := r.Cookie("csrf_token")
		if err != nil || cookie.Value != "csrf" {
			t.Errorf("csrf cookie missing")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"team-1","name":"Platform","description":null,"created_at":"a","updated_at":"b"}}`))
	}))
	defer server.Close()
	c, err := New(server.URL, "access", "csrf", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	team, err := c.CreateTeam(context.Background(), "acme", map[string]any{"name": "Platform"})
	if err != nil {
		t.Fatal(err)
	}
	if team.ID != "team-1" || team.Description != nil {
		t.Fatalf("unexpected team: %#v", team)
	}
}

func TestBearerPrefixIsNormalized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access" {
			t.Errorf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"id":"team-1","name":"Platform"}}`))
	}))
	defer server.Close()

	c, err := New(server.URL, "Bearer access", "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetTeam(context.Background(), "acme", "team-1"); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"in_use","message":"still referenced","details":{"count":1}}}`))
	}))
	defer server.Close()
	c, _ := New(server.URL, "token", "", server.Client())
	_, err := c.GetTeam(context.Background(), "acme", "id")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Code != "in_use" || apiErr.StatusCode != 409 {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
}

func TestNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	defer server.Close()
	c, _ := New(server.URL, "token", "", server.Client())
	_, err := c.GetTeam(context.Background(), "acme", "gone")
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestOnlyNotFoundStatusRemovesState(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		err := &APIError{StatusCode: status, Code: "test", Message: "request failed"}
		if IsNotFound(err) {
			t.Errorf("HTTP %d must not be classified as not found", status)
		}
	}
}

func TestAPIErrorDoesNotFormatDetails(t *testing.T) {
	err := (&APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "invalid_request",
		Message:    "request rejected",
		Details:    map[string]any{"api_key": "must-not-appear"},
	}).Error()
	if strings.Contains(err, "must-not-appear") {
		t.Fatalf("API error exposed details: %s", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := parseRetryAfter("3", now); got != 3*time.Second {
		t.Fatalf("got %v", got)
	}
	if got := parseRetryAfter(now.Add(2*time.Second).Format(http.TimeFormat), now); got != 2*time.Second {
		t.Fatalf("got %v", got)
	}
}

func TestPaginate(t *testing.T) {
	calls := 0
	got, err := Paginate(context.Background(), func(_ context.Context, limit, offset int) ([]int, int, error) {
		calls++
		all := []int{1, 2, 3, 4, 5}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		return all[offset:end], len(all), nil
	}, 2)
	if err != nil || len(got) != 5 || calls != 3 {
		t.Fatalf("got=%v calls=%d err=%v", got, calls, err)
	}
}
