# Architecture

This document records the provider's current structure and the decisions that
future changes should preserve. It describes the implementation in this
repository; the OpenAPI contract remains the authority for the remote API.

## Scope

The provider manages the core Incident Garden alert-routing topology:

```text
team
├── schedule
├── escalation policy
│   └── ordered steps → schedule, user, or team
└── integration → escalation policy
    └── integration policy
```

Schedule rotations and overrides have separate remote identities and are not
part of the schedule resource. The provider currently exposes no data sources.

## Components

- `main.go` starts the Terraform provider server using protocol version 6.
- `internal/provider` defines provider configuration, Terraform schemas,
  lifecycle methods, state conversion, imports, and diagnostics.
- `internal/client` owns HTTP transport, authentication, endpoint paths, API
  envelopes, typed errors, pagination, and retry behavior.
- `examples` contains a reusable complete configuration and a local live-test
  topology.
- `docs` contains contributor, testing, changelog, and resource documentation.

Terraform-facing code depends on the client package. The client has no
Terraform dependencies, which keeps transport behavior independently testable.

## Configuration and request flow

Provider configuration is read from explicit Terraform attributes first, then
from `INCIDENTGARDEN_*` environment variables. The endpoint defaults to
`http://localhost:8080`; an access token is mandatory. A resource-level
`organization` overrides the provider default.

```text
Terraform configuration
  → provider schema and validation
  → resource expand function
  → authenticated API client
  → Incident Garden JSON envelope
  → resource flatten/apply function
  → Terraform state
```

The client normalizes both raw JWTs and values prefixed with `Bearer`. It sends
the token in the `Authorization` header. Mutating requests additionally send
the CSRF token in both `X-CSRF-Token` and the `csrf_token` cookie.

## State and lifecycle rules

- Every resource supports create, read, update, delete, and import.
- Imports use `<organization-slug>/<resource-id>` because item endpoints require
  both values.
- A genuine HTTP 404 during read removes the object from Terraform state. Other
  failures, including 403, remain errors.
- `organization` and remote relationship fields marked immutable require
  replacement instead of in-place updates.
- Reads reconstruct state from the API so out-of-band changes appear as drift.
- Optional JSON attributes are canonicalized before storage to prevent diffs
  caused only by whitespace or object-key formatting.
- Integration `api_key` is sensitive, returned only on creation, and preserved
  from prior state during subsequent reads. `api_key_prefix` is non-sensitive.
- Deletes do not cascade; dependency conflicts are returned as diagnostics.

## Reliability and security

The HTTP client uses request contexts, a 30-second default timeout, bounded
response reads, typed API errors, and URL escaping. Only GET requests retry HTTP
429 responses, up to three attempts, because the API does not document
idempotency keys for mutations. `Retry-After` is honored with a five-second
per-attempt ceiling.

Credentials and secret response bodies must never be logged or committed. The
current browser-session JWT and CSRF model is suitable for short interactive
runs but not durable unattended automation. A scoped machine credential is an
upstream API requirement.

## Testing and delivery

Unit tests cover state helpers, JSON normalization, imports, secret retention,
authentication, envelopes, errors, retries, and pagination. Acceptance tests
exercise real Terraform lifecycle operations against an explicitly configured
disposable organization. The full topology test remains opt-in because of the
documented hosted-API integration deletion issue.

Pull requests run formatting, linting, tests, build validation, and changelog
checks. Pushes to `main` build the provider. Publishing is intentionally present
only as a disabled CI placeholder.

## Known boundaries

- No machine authentication or automatic JWT refresh is implemented.
- Schedule rotations and overrides are future standalone resources.
- Integration settings, severity mappings, and policy conditions/actions use
  canonical JSON until stable typed schemas are introduced.
- The hosted API can retain a hidden escalation-policy reference after an
  integration is deleted, preventing deterministic teardown of some topologies.
