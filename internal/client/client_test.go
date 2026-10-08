package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type changingCredentials struct{ calls int }

func (provider *changingCredentials) Credentials(context.Context) (Credentials, error) {
	provider.calls++
	return Credentials{AccessToken: fmt.Sprintf("access-%d", provider.calls), CSRFToken: "csrf"}, nil
}

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

func TestMutationRetriesTooManyRequests(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < maxAttempts {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"team-1","name":"Platform"}}`))
	}))
	defer server.Close()

	c, _ := New(server.URL, "token", "csrf", server.Client())
	if _, err := c.CreateTeam(context.Background(), "acme", map[string]any{"name": "Platform"}); err != nil {
		t.Fatal(err)
	}
	if attempts != maxAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, maxAttempts)
	}
}

func TestCredentialProviderIsConsultedForEveryAttempt(t *testing.T) {
	provider := &changingCredentials{}
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if got := r.Header.Get("Authorization"); got != fmt.Sprintf("Bearer access-%d", attempts) {
			t.Errorf("authorization = %q on attempt %d", got, attempts)
		}
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"id":"team-1","name":"Platform"}}`))
	}))
	defer server.Close()

	c, err := NewWithCredentials(server.URL, provider, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetTeam(context.Background(), "acme", "team-1"); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("credential calls = %d, want 2", provider.calls)
	}
}

func TestRetryDelay(t *testing.T) {
	if got := retryDelay(90*time.Second, 0); got != maxRetryAfterWait {
		t.Fatalf("retry delay = %v, want %v", got, maxRetryAfterWait)
	}
	if got := retryDelay(0, 1); got != 200*time.Millisecond {
		t.Fatalf("fallback retry delay = %v", got)
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

func TestOrganizationMemberPaginationQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs/acme/members" || r.URL.Query().Get("limit") != "25" || r.URL.Query().Get("offset") != "50" {
			t.Errorf("unexpected request URL %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"data":{"items":[{"id":"user-1","email":"on-call@example.com"}],"total":51,"limit":25,"offset":50}}`))
	}))
	defer server.Close()

	c, err := New(server.URL, "access", "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	members, total, err := c.ListOrganizationMembersPage(context.Background(), "acme", 25, 50)
	if err != nil || total != 51 || len(members) != 1 || members[0].ID != "user-1" {
		t.Fatalf("members=%#v total=%d err=%v", members, total, err)
	}
}
