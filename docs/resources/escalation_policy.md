---
page_title: "incidentgarden_escalation_policy Resource - Incident Garden"
description: Manages an ordered Incident Garden escalation policy.
---

# incidentgarden_escalation_policy

```terraform
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

## Schema

- `name` (String, Required) Policy name.
- `team_id` (String, Required, Forces replacement) Owning team identifier.
- `step` (List of Object, Required) Steps ordered by increasing activation offset.
  - `target_type` (String, Required) Target discriminator.
  - `activation_offset_minutes` (Number, Required) Minutes before activation.
  - `schedule_id` (String, Optional) Schedule target.
  - `user_id` (String, Optional) User target.
  - `team_id` (String, Optional) Team target.
  - `id` (String, Read-only) Server-generated step identifier.
  - `position` (Number, Read-only) Server-calculated position.
- `description` (String, Optional) Policy description.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `id`, `created_at`, `updated_at` (String, Read-only) Server metadata.

Configure exactly the target identifier matching `target_type`. The API sorts
steps by activation offset, so configure offsets in ascending order.

## Import

```shell
terraform import incidentgarden_escalation_policy.critical organization-slug/resource-id
```
