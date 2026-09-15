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
  conditions_json = jsonencode({
    matchers = [{ key = "env", op = "neq", value = "production" }]
  })
  action_json = jsonencode({ type = "drop" })
}
```

## Schema

- `team_id` (String, Required, Forces replacement) Owning team identifier.
- `applies_to_all` (Boolean, Required, Forces replacement) All-integration scope.
- `name` (String, Required) Policy name.
- `position` (Number, Required) Evaluation position.
- `action_json` (String, Required) Canonical JSON for the single action.
- `integration_id` (String, Optional, Forces replacement) Single-integration scope identifier.
- `description` (String, Optional) Policy description.
- `is_active` (Boolean, Optional, Computed) Whether the policy is active.
- `conditions_json` (String, Optional, Computed) Canonical JSON conditions.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `id`, `created_at`, `updated_at` (String, Read-only) Server metadata.

Set `integration_id` when `applies_to_all` is false and omit it when true.

## Import

```shell
terraform import incidentgarden_integration_policy.production organization-slug/resource-id
```
