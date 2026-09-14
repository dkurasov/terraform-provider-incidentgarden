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

Every resource imports with `<organization-slug>/<resource-id>`.

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

## `incidentgarden_schedule`

Manages an on-call schedule container. Rotations and overrides are not managed
by this resource.

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
| `severity_mapping_json` | Optional, computed | JSON `SeverityMapping` object; stored canonically. |
| `settings_json` | Optional, computed | Type-specific JSON settings object or `null`; stored canonically. |
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
| `conditions_json` | Optional, computed | JSON `PolicyConditions` object; stored canonically. |
| `action_json` | Required | JSON for the policy's single discriminated action object. |

```hcl
resource "incidentgarden_integration_policy" "production" {
  team_id        = incidentgarden_team.platform.id
  integration_id = incidentgarden_integration.alertmanager.id
  applies_to_all = false
  name           = "Drop non-production alerts"
  position       = 0

  conditions_json = jsonencode({
    matchers = [{ key = "env", op = "neq", value = "production" }]
  })
  action_json = jsonencode({ type = "drop" })
}
```

The provider sends `action_json` as the API's single-element `actions` array.
The valid JSON shapes and discriminator values follow the Incident Garden API
contract and can vary by integration type.
