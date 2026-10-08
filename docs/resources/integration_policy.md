---
page_title: "incidentgarden_integration_policy Resource - Incident Garden"
description: Manages a conditional Incident Garden integration policy.
---

# incidentgarden_integration_policy

```terraform
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

## Schema

- `team_id` (String, Required, Forces replacement) Owning team identifier.
- `applies_to_all` (Boolean, Required, Forces replacement) All-integration scope.
- `name` (String, Required) Policy name.
- `position` (Number, Required) Evaluation position.
- `action` (Object, Required) Typed single action discriminated by `type`.
- `integration_id` (String, Optional, Forces replacement) Single-integration scope identifier.
- `description` (String, Optional) Policy description.
- `is_active` (Boolean, Optional, Computed) Whether the policy is active.
- `conditions` (Object, Optional, Computed) Typed matchers, severities, and time window.
- `action_json` and `conditions_json` are deprecated compatibility fields for release-candidate state.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `id`, `created_at`, `updated_at` (String, Read-only) Server metadata.

`conditions` supports `matchers` (`key`, `op`, optional `value`), `severity_in`,
and `time_window` (`start`, `end`, `days_of_week`). `action.type` is one of
`drop`, `suppress`, `override_severity`, `add_labels`, or
`hold_outside_window`; its optional `value` object follows the API's
discriminated shape.

Set `integration_id` when `applies_to_all` is false and omit it when true.

## Import

```shell
terraform import incidentgarden_integration_policy.production organization-slug/resource-id
```
