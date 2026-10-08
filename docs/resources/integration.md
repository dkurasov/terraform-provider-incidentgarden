---
page_title: "incidentgarden_integration Resource - Incident Garden"
description: Manages an Incident Garden alert-ingestion integration.
---

# incidentgarden_integration

```terraform
resource "incidentgarden_integration" "alertmanager" {
  name                 = "Alertmanager"
  type                 = "alertmanager"
  team_id              = incidentgarden_team.platform.id
  escalation_policy_id = incidentgarden_escalation_policy.critical.id
}
```

## Schema

- `name` (String, Required) Integration name.
- `type` (String, Required, Forces replacement) Integration type.
- `team_id` (String, Required, Forces replacement) Owning team identifier.
- `escalation_policy_id` (String, Required) Escalation policy identifier.
- `severity_mapping` (Object, Optional, Computed) Typed severity mapping; an empty `source_field` is preserved.
- `settings_json` (String, Optional, Computed) Type-specific settings compared as semantic JSON.
- `severity_mapping_json` (String, Deprecated) Release-candidate compatibility field.
- `is_active` (Boolean, Optional, Computed) Whether the integration is active.
- `organization` (String, Optional, Forces replacement) Organization slug; defaults to the provider.
- `api_key_prefix` (String, Read-only) Non-secret generated-key prefix.
- `api_key` (String, Sensitive, Read-only) Create-only key retained in state.
- `id`, `created_at`, `updated_at` (String, Read-only) Server metadata.

`severity_mapping` contains optional `source_field`, `default`, and `mapping`.
`mapping` contains optional `critical` and `warning` string lists. An empty
`source_field` is meaningful: it disables extraction and uses `default`.

Protect Terraform state because it contains `api_key`. The key cannot be
recovered when importing an existing integration.

## Import

```shell
terraform import incidentgarden_integration.alertmanager organization-slug/resource-id
```
