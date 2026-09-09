# Terraform Provider for Incident Garden

This provider manages the core Incident Garden alert-routing topology with Terraform Plugin Framework protocol v6.

```hcl
terraform {
  required_providers {
    incidentgarden = {
      source = "dkurasov/incidentgarden"
    }
  }
}

provider "incidentgarden" {
  endpoint     = var.endpoint
  token        = var.token
  csrf_token   = var.csrf_token
  organization = var.organization
}
```

Configuration can also be supplied through `INCIDENTGARDEN_ENDPOINT`, `INCIDENTGARDEN_TOKEN`, `INCIDENTGARDEN_CSRF_TOKEN`, and `INCIDENTGARDEN_ORGANIZATION`. JWT access tokens expire after 15 minutes according to the current API contract. Mutations additionally require the CSRF value as both cookie and header. The API currently documents no long-lived machine token, so unattended runs need an external session/token renewal mechanism; see the execution plan.

Supported resources are `incidentgarden_team`, `incidentgarden_schedule`, `incidentgarden_escalation_policy`, `incidentgarden_integration`, and `incidentgarden_integration_policy`. Every resource imports as `<organization-slug>/<resource-id>`.

Run `make test` for isolated tests and `make check` for tests plus static checks. Acceptance tests require a disposable Incident Garden instance and the variables described in [Testing](docs/testing.md).

User-visible changes require a Changie fragment. See the [changelog workflow](docs/changelog.md) for pull request and release instructions.

For a local build that works with `terraform init`, use the filesystem-mirror setup in [`examples/live-test`](examples/live-test/README.md). The provider accepts either a raw JWT or a value beginning with `Bearer `, though a raw JWT is preferred.
