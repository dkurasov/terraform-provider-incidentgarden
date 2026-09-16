---
page_title: "incidentgarden_schedule Resource - Incident Garden"
description: Manages an Incident Garden on-call schedule container.
---

# incidentgarden_schedule

Rotations and overrides have independent lifecycles and are not managed here.

```terraform
resource "incidentgarden_schedule" "primary" {
  team_id = incidentgarden_team.platform.id
  name    = "Primary"
}
```

## Schema

- `name` (String, Required) Schedule name.
- `team_id` (String, Required, Forces replacement) Owning team identifier.
- `description` (String, Optional) Schedule description.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `id` (String, Read-only) Server-generated identifier.
- `created_at` (String, Read-only) Creation timestamp.
- `updated_at` (String, Read-only) Update timestamp.

## Import

```shell
terraform import incidentgarden_schedule.primary organization-slug/resource-id
```
