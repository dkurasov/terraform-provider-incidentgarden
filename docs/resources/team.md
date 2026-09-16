---
page_title: "incidentgarden_team Resource - Incident Garden"
description: Manages an Incident Garden team.
---

# incidentgarden_team

```terraform
resource "incidentgarden_team" "platform" {
  name        = "Platform"
  description = "Platform operations"
}
```

## Schema

- `name` (String, Required) Team name.
- `description` (String, Optional) Team description.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `id` (String, Read-only) Server-generated identifier.
- `created_at` (String, Read-only) Creation timestamp.
- `updated_at` (String, Read-only) Update timestamp.

## Import

```shell
terraform import incidentgarden_team.platform organization-slug/resource-id
```
