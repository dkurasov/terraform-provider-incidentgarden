# Incident Garden Terraform Provider

- API contract: `/Users/d.kurasov/Downloads/openapi-incidentgarden.yaml` (development input; keep changes aligned with the supplied contract).
- Execution plan and decisions: `docs/exec-plans/terraform-provider-mvp.md`.
- HTTP, authentication, envelopes, retries: `internal/client`.
- Terraform schemas and lifecycle: `internal/provider`.
- Format/test: `make fmt`, `make test`, `make check`. The full check requires `golangci-lint` and `terraform` on `PATH`.
- Never log or commit JWTs, CSRF tokens, integration API keys, or response bodies containing secrets.
- Reads remove state only on a genuine HTTP 404; never reinterpret 403 as absence.
- Acceptance tests require `TF_ACC=1`, a disposable organization, and explicit environment configuration. The known-unsafe topology test additionally requires `INCIDENTGARDEN_RUN_TOPOLOGY_ACC=1`. Never point them at production.
