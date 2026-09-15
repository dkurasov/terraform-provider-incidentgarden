---
page_title: "Incident Garden Provider"
description: |-
  Manage Incident Garden teams and alert-routing topology with Terraform.
---

# Incident Garden Provider

The Incident Garden provider manages teams, schedules, escalation policies,
integrations, and integration policies through the Incident Garden API.

```terraform
provider "incidentgarden" {
  endpoint     = var.endpoint
  token        = var.token
  csrf_token   = var.csrf_token
  organization = var.organization
}
```

Configuration can instead use `INCIDENTGARDEN_ENDPOINT`,
`INCIDENTGARDEN_TOKEN`, `INCIDENTGARDEN_CSRF_TOKEN`, and
`INCIDENTGARDEN_ORGANIZATION`. A raw JWT or `Bearer <JWT>` is accepted. The API
currently uses short-lived browser-session credentials, so unattended runs need
an external renewal mechanism.

## Schema

### Optional

- `endpoint` (String) API base URL; defaults to `http://localhost:8080`.
- `token` (String, Sensitive) Access JWT; required here or by environment.
- `csrf_token` (String, Sensitive) CSRF value used for mutating requests.
- `organization` (String) Default organization slug for resources.
