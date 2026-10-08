# Provider and resource reference

The provider source address is `dkurasov/incidentgarden`. Attribute modes below
use **Required**, **Optional**, and **Computed** as Terraform defines them.
**Replace** means changing the field destroys and recreates the resource.

## Provider configuration

| Field | Mode | Environment variable | Description |
|---|---|---|---|
| `endpoint` | Optional | `INCIDENTGARDEN_ENDPOINT` | API base URL. Defaults to `http://localhost:8080`. |
| `token` | Optional, sensitive | `INCIDENTGARDEN_TOKEN` | Required access JWT. A raw JWT or `Bearer <JWT>` is accepted. |
| `csrf_token` | Optional, sensitive | `INCIDENTGARDEN_CSRF_TOKEN` | Required by API mutations and sent as both header and cookie. |
| `organization` | Optional | `INCIDENTGARDEN_ORGANIZATION` | Default organization slug for resources. |

All resources share these computed fields:

| Field | Mode | Description |
|---|---|---|
| `id` | Computed | Server-generated resource identifier. |
| `organization` | Optional, computed, replace | Organization slug; defaults to provider configuration. |
| `created_at` | Computed | API creation timestamp. |
| `updated_at` | Computed | API update timestamp. |

Most resources import with `<organization-slug>/<resource-id>`. Exceptions are
documented with their resources.

## `incidentgarden_members` data source

Lists all organization members across every API page and exposes an exact-email
lookup map. The API permits this lookup only for organization administrators and
owners.

| Field | Mode | Description |
|---|---|---|
| `organization` | Optional, computed | Organization slug; defaults to provider configuration. |
| `by_email` | Computed map | Exact member email to user ID. |

```hcl
data "incidentgarden_members" "organization" {}

locals {
  on_call_user_id = data.incidentgarden_members.organization.by_email[var.on_call_email]
}
```

## `incidentgarden_team`

Manages an Incident Garden team.

| Field | Mode | Description |
|---|---|---|
| `name` | Required | Team name. |
| `description` | Optional | Team description. Null and an empty string remain distinct. |

```hcl
resource "incidentgarden_team" "platform" {
  name        = "Platform"
  description = "Platform operations"
}
```

## `incidentgarden_team_members`

Authoritatively manages the complete membership of one team. On apply, the
provider adds or updates configured members and then removes every remote member
not in `member`. On destroy it removes every team member. Use it only when this
Terraform resource has exclusive ownership; importing or pointing it at a
shared team can remove people and can revoke the credentials running Terraform.

| Field | Mode | Description |
|---|---|---|
| `organization` | Optional, computed, replace | Organization slug; defaults to provider configuration. |
| `team_id` | Required, replace | Team whose complete membership is owned. |
| `member` | Required set | Complete desired set of users and roles. |
| `member.user_id` | Required | Organization-member user ID. |
| `member.team_role` | Required | `team_admin` or `team_member`. |

Imports use `<organization-slug>/<team-id>`.

```hcl
resource "incidentgarden_team_members" "platform" {
  team_id = incidentgarden_team.platform.id
  member = [{
    user_id   = data.incidentgarden_members.organization.by_email[var.on_call_email]
    team_role = "team_admin"
  }]
}
```

## `incidentgarden_schedule`

Manages an on-call schedule container. Rotations are separate
`incidentgarden_rotation` resources; overrides are not currently managed.

| Field | Mode | Description |
|---|---|---|
| `name` | Required | Schedule name. |
| `description` | Optional | Schedule description. Null and an empty string remain distinct. |
| `team_id` | Required, replace | Owning team identifier. |

```hcl
resource "incidentgarden_schedule" "primary" {
  team_id = incidentgarden_team.platform.id
  name    = "Primary"
}
```

## `incidentgarden_rotation`

Manages an ordered on-call rotation within a schedule. The participant list is
positional: reordering entries changes the rotation order.

| Field | Mode | Description |
|---|---|---|
| `schedule_id` | Required, replace | Parent schedule identifier. |
| `name` | Required | Rotation name. |
| `type` | Required | `daily`, `weekly`, or `hourly`. |
| `length` | Optional, computed | Shift length in rotation-type units, from 1 through 365. |
| `start_date` | Required | RFC 3339 anchor in calendar years 2020–2099. |
| `end_date` | Optional | Exclusive RFC 3339 end after `start_date`; removing it clears the remote value. |
| `time_restriction` | Optional, computed | `weekday_and_time` restriction; not valid for hourly rotations. Removing it replaces the rotation because the hosted API currently ignores an in-place null clear. |
| `participant` | Required list | One through 50 ordered rotation positions. |

Each participant uses exactly one shape: `user` with only `user_id`, `team`
with only `team_id`, or `none` with neither. Participant `id` and `position` are
computed. A time-restriction window requires `start_hour`, `start_min`,
`end_hour`, `end_min`, `start_day`, and `end_day`.

Imports use `<organization-slug>/<schedule-id>/<rotation-id>`.

```hcl
resource "incidentgarden_rotation" "primary" {
  schedule_id = incidentgarden_schedule.primary.id
  name        = "Primary daily rotation"
  type        = "daily"
  length      = 1
  start_date  = "2026-01-01T09:00:00Z"

  participant = [{
    participant_type = "user"
    user_id          = data.incidentgarden_members.organization.by_email[var.on_call_email]
  }]
}
```

## `incidentgarden_escalation_policy`

Manages an escalation policy and its ordered steps.

| Field | Mode | Description |
|---|---|---|
| `name` | Required | Policy name. |
| `description` | Optional | Policy description. |
| `team_id` | Required, replace | Owning team identifier. |
| `step` | Required list | Ordered escalation steps, configured by increasing activation offset. |

Each `step` supports:

| Field | Mode | Description |
|---|---|---|
| `id` | Computed | Server-generated step identifier. |
| `position` | Computed | Server-calculated position. |
| `target_type` | Required | API target discriminator, such as `on_call_schedule`, `user`, or `team`. |
| `schedule_id` | Optional | Target schedule identifier when the target type is a schedule. |
| `user_id` | Optional | Target user identifier when the target type is a user. |
| `team_id` | Optional | Target team identifier when the target type is a team. |
| `activation_offset_minutes` | Required | Minutes after alert creation when the step activates. |

Exactly the target identifier matching `target_type` should be configured. The
API sorts steps by `activation_offset_minutes`, so offsets should be ascending.

```hcl
resource "incidentgarden_escalation_policy" "critical" {
  team_id = incidentgarden_team.platform.id
  name    = "Critical alerts"

  step = [{
    target_type               = "on_call_schedule"
    schedule_id               = incidentgarden_schedule.primary.id
    activation_offset_minutes = 0
  }]
}
```

## `incidentgarden_integration`

Manages an alert-ingestion integration.

| Field | Mode | Description |
|---|---|---|
| `name` | Required | Integration name. |
| `type` | Required, replace | Integration type understood by the API. |
| `team_id` | Required, replace | Owning team identifier. |
| `escalation_policy_id` | Required | Policy used for incoming alerts. |
| `severity_mapping` | Optional, computed | Typed source field, critical/warning mappings, and default severity. |
| `settings_json` | Optional, computed | Type-specific JSON settings object or `null`; compared semantically. |
| `is_active` | Optional, computed | Whether the integration is active; API default is retained when omitted. |
| `api_key_prefix` | Computed | Non-secret prefix identifying the generated API key. |
| `api_key` | Computed, sensitive | API key returned at creation and retained in Terraform state. |

```hcl
resource "incidentgarden_integration" "alertmanager" {
  name                 = "Alertmanager"
  type                 = "alertmanager"
  team_id              = incidentgarden_team.platform.id
  escalation_policy_id = incidentgarden_escalation_policy.critical.id
}
```

Terraform state contains the sensitive `api_key`; use an encrypted remote state
backend with appropriately restricted access.

## `incidentgarden_integration_policy`

Manages a conditional policy exposed by the API's `/policies` endpoints.

| Field | Mode | Description |
|---|---|---|
| `team_id` | Required, replace | Owning team identifier. |
| `integration_id` | Optional, replace | Integration identifier; required when `applies_to_all` is false and omitted when true. |
| `applies_to_all` | Required, replace | Applies the policy to all integrations owned by the team. |
| `name` | Required | Policy name. |
| `description` | Optional | Policy description. |
| `position` | Required | Evaluation order within the policy scope. |
| `is_active` | Optional, computed | Whether the policy is active; API default is retained when omitted. |
| `conditions` | Optional, computed | Typed matchers, severity filter, and time window. |
| `action` | Optional, computed | Typed single action discriminated by `type`; required in configuration. |

```hcl
resource "incidentgarden_integration_policy" "production" {
  team_id        = incidentgarden_team.platform.id
  integration_id = incidentgarden_integration.alertmanager.id
  applies_to_all = false
  name           = "Drop non-production alerts"
  position       = 0

  conditions = {
    matchers = [{ key = "env", op = "neq", value = "production" }]
  }
  action = { type = "drop" }
}
```

The provider sends `action` as the API's single-element `actions` array.
The deprecated `severity_mapping_json`, `conditions_json`, and `action_json`
attributes remain temporarily readable and configurable for release-candidate
state compatibility; do not configure a typed field and its JSON predecessor.

## Hosted API limits

Resource quotas are backend- and plan-configured rather than fixed by the API
contract. Query the API usage response and quota headers for the active limits;
do not assume example/default plan values. The contract does specify default
creation rate limits of five teams per minute and ten schedules, integrations,
or policies per minute. Rotation creation is instead bounded by its configured
per-schedule quota. The provider retries HTTP 429 responses at most three
attempts and does not turn remote quotas into Terraform-side admission control.
