package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type Team struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type Schedule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	TeamID      string  `json:"team_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type EscalationStep struct {
	ID                      string  `json:"id,omitempty"`
	Position                int64   `json:"position,omitempty"`
	TargetType              string  `json:"target_type"`
	ScheduleID              *string `json:"schedule_id,omitempty"`
	UserID                  *string `json:"user_id,omitempty"`
	TeamID                  *string `json:"team_id,omitempty"`
	ActivationOffsetMinutes int64   `json:"activation_offset_minutes"`
}

type EscalationPolicy struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description"`
	TeamID      string           `json:"team_id"`
	Steps       []EscalationStep `json:"steps"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

type SeverityMapping struct {
	SourceField string              `json:"source_field,omitempty"`
	Mapping     map[string][]string `json:"mapping,omitempty"`
	Default     string              `json:"default,omitempty"`
}

type Integration struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Type               string          `json:"type"`
	TeamID             string          `json:"team_id"`
	EscalationPolicyID string          `json:"escalation_policy_id"`
	SeverityMapping    SeverityMapping `json:"severity_mapping"`
	Settings           json.RawMessage `json:"settings"`
	APIKeyPrefix       string          `json:"api_key_prefix"`
	APIKey             string          `json:"api_key,omitempty"`
	IsActive           bool            `json:"is_active"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

type IntegrationPolicy struct {
	ID            string          `json:"id"`
	TeamID        string          `json:"team_id"`
	IntegrationID *string         `json:"integration_id"`
	AppliesToAll  bool            `json:"applies_to_all"`
	Name          string          `json:"name"`
	Description   *string         `json:"description"`
	Position      int64           `json:"position"`
	IsActive      bool            `json:"is_active"`
	Conditions    json.RawMessage `json:"conditions"`
	Actions       json.RawMessage `json:"actions"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

type envelope[T any] struct {
	Data T `json:"data"`
}

func get[T any](ctx context.Context, c *Client, path string) (T, error) {
	var out envelope[T]
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Data, err
}

func mutate[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var out envelope[T]
	err := c.do(ctx, method, path, body, &out)
	return out.Data, err
}

func (c *Client) CreateTeam(ctx context.Context, org string, body any) (Team, error) {
	return mutate[Team](ctx, c, http.MethodPost, orgPath(org, "/teams"), body)
}
func (c *Client) GetTeam(ctx context.Context, org, id string) (Team, error) {
	return get[Team](ctx, c, orgPath(org, "/teams/"+url.PathEscape(id)))
}
func (c *Client) UpdateTeam(ctx context.Context, org, id string, body any) (Team, error) {
	return mutate[Team](ctx, c, http.MethodPatch, orgPath(org, "/teams/"+url.PathEscape(id)), body)
}
func (c *Client) DeleteTeam(ctx context.Context, org, id string) error {
	return c.do(ctx, http.MethodDelete, orgPath(org, "/teams/"+url.PathEscape(id)), nil, nil)
}

func (c *Client) CreateSchedule(ctx context.Context, org string, body any) (Schedule, error) {
	return mutate[Schedule](ctx, c, http.MethodPost, orgPath(org, "/schedules"), body)
}
func (c *Client) GetSchedule(ctx context.Context, org, id string) (Schedule, error) {
	return get[Schedule](ctx, c, orgPath(org, "/schedules/"+url.PathEscape(id)))
}
func (c *Client) UpdateSchedule(ctx context.Context, org, id string, body any) (Schedule, error) {
	return mutate[Schedule](ctx, c, http.MethodPatch, orgPath(org, "/schedules/"+url.PathEscape(id)), body)
}
func (c *Client) DeleteSchedule(ctx context.Context, org, id string) error {
	return c.do(ctx, http.MethodDelete, orgPath(org, "/schedules/"+url.PathEscape(id)), nil, nil)
}

func (c *Client) CreateEscalationPolicy(ctx context.Context, org string, body any) (EscalationPolicy, error) {
	return mutate[EscalationPolicy](ctx, c, http.MethodPost, orgPath(org, "/escalation-policies"), body)
}
func (c *Client) GetEscalationPolicy(ctx context.Context, org, id string) (EscalationPolicy, error) {
	return get[EscalationPolicy](ctx, c, orgPath(org, "/escalation-policies/"+url.PathEscape(id)))
}
func (c *Client) UpdateEscalationPolicy(ctx context.Context, org, id string, body any) (EscalationPolicy, error) {
	return mutate[EscalationPolicy](ctx, c, http.MethodPut, orgPath(org, "/escalation-policies/"+url.PathEscape(id)), body)
}
func (c *Client) DeleteEscalationPolicy(ctx context.Context, org, id string) error {
	return c.do(ctx, http.MethodDelete, orgPath(org, "/escalation-policies/"+url.PathEscape(id)), nil, nil)
}

func (c *Client) CreateIntegration(ctx context.Context, org string, body any) (Integration, error) {
	return mutate[Integration](ctx, c, http.MethodPost, orgPath(org, "/integrations"), body)
}
func (c *Client) GetIntegration(ctx context.Context, org, id string) (Integration, error) {
	return get[Integration](ctx, c, orgPath(org, "/integrations/"+url.PathEscape(id)))
}
func (c *Client) UpdateIntegration(ctx context.Context, org, id string, body any) (Integration, error) {
	return mutate[Integration](ctx, c, http.MethodPatch, orgPath(org, "/integrations/"+url.PathEscape(id)), body)
}
func (c *Client) DeleteIntegration(ctx context.Context, org, id string) error {
	return c.do(ctx, http.MethodDelete, orgPath(org, "/integrations/"+url.PathEscape(id)), nil, nil)
}

func (c *Client) CreateIntegrationPolicy(ctx context.Context, org string, body any) (IntegrationPolicy, error) {
	return mutate[IntegrationPolicy](ctx, c, http.MethodPost, orgPath(org, "/policies"), body)
}
func (c *Client) GetIntegrationPolicy(ctx context.Context, org, id string) (IntegrationPolicy, error) {
	return get[IntegrationPolicy](ctx, c, orgPath(org, "/policies/"+url.PathEscape(id)))
}
func (c *Client) UpdateIntegrationPolicy(ctx context.Context, org, id string, body any) (IntegrationPolicy, error) {
	return mutate[IntegrationPolicy](ctx, c, http.MethodPatch, orgPath(org, "/policies/"+url.PathEscape(id)), body)
}
func (c *Client) DeleteIntegrationPolicy(ctx context.Context, org, id string) error {
	return c.do(ctx, http.MethodDelete, orgPath(org, "/policies/"+url.PathEscape(id)), nil, nil)
}
