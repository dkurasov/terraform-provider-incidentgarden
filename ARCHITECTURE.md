# Architecture

This document records the provider's current structure and the decisions that
future changes should preserve. It describes the implementation in this
repository; the OpenAPI contract remains the authority for the remote API.

## Scope

The provider manages the core Incident Garden alert-routing topology:

```text
team
├── authoritative membership
├── schedule
│   └── ordered rotations → user, team, or empty position
├── escalation policy
│   └── ordered steps → schedule, user, or team
└── integration → escalation policy
    └── integration policy
```

Schedule rotations have separate remote identities and standalone Terraform
resources; schedule overrides remain outside the provider. The organization
members data source provides the privileged email-to-user-ID lookup needed to
configure memberships and rotations.

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

The client obtains credentials through a small `CredentialProvider` interface
on every request attempt, then normalizes both raw JWTs and values prefixed with
`Bearer`. Mutating requests additionally send the CSRF token in both
`X-CSRF-Token` and the `csrf_token` cookie. Provider configuration currently
installs a static implementation; the boundary permits a future external
credential source without coupling refresh persistence to Terraform resources.

## State and lifecycle rules

- Every resource supports create, read, update, delete, and import.
- Most imports use `<organization-slug>/<resource-id>`. Rotation imports also
  carry the parent schedule ID.
- A genuine HTTP 404 during read removes the object from Terraform state. Other
  failures, including 403, remain errors.
- `organization` and remote relationship fields marked immutable require
  replacement instead of in-place updates.
- Reads reconstruct state from the API so out-of-band changes appear as drift.
- `incidentgarden_team_members` has exclusive, authoritative ownership of a
  team's whole member set. It adds and updates desired members before removing
  unmanaged members; destroy removes the complete set.
- Rotation participant lists preserve order. Rotation reads use the parent
  schedule detail because the API exposes no rotation item GET endpoint.
- Static integration and policy structures use typed nested attributes. Dynamic
  integration settings remain JSON and use semantic equality, so whitespace,
  key ordering, and equivalent number spellings do not create drift.
- Integration `api_key` is sensitive, returned only on creation, and preserved
  from prior state during subsequent reads. `api_key_prefix` is non-sensitive.
- Deletes do not cascade; dependency conflicts are returned as diagnostics.

## Reliability and security

The HTTP client uses request contexts, a 30-second default timeout, bounded
response reads, typed API errors, and URL escaping. All methods retry an HTTP
429 response up to three attempts because the API guarantees that 4xx responses
precede mutation. `Retry-After` is honored with a 60-second per-attempt ceiling.

Credentials and secret response bodies must never be logged or committed. The
current browser-session JWT and CSRF model is suitable for short interactive
runs but not durable unattended automation. A scoped machine credential is an
upstream API requirement.

## Testing and delivery

Unit tests cover state helpers, JSON normalization, imports, secret retention,
credential retrieval, authentication, envelopes, errors, retries, and pagination. Acceptance tests
exercise real Terraform lifecycle operations against an explicitly configured
disposable organization. The full topology test remains opt-in because of the
documented hosted-API integration deletion issue.

Pull requests run formatting, linting, tests, build validation, and changelog
checks. Pushes to `main` build the provider. Publishing is intentionally present
only as a disabled CI placeholder.

Release artifacts are produced by GoReleaser from `.goreleaser.yml`. Builds are
static, version-injected protocol-6 binaries packaged per platform with a
Terraform Registry manifest, SHA-256 checksums, and a detached GPG checksum
signature. The release workflow currently exposes only manual unsigned snapshot
validation; its GitHub publishing job is hard-disabled and configured to create
draft releases when eventually enabled.

## Known boundaries

- No machine authentication or automatic JWT refresh is implemented.
- Schedule overrides are not managed.
- Dynamic integration settings remain canonical JSON; stable severity,
  condition, and policy-action structures are typed.
- The hosted API can retain a hidden escalation-policy reference after an
  integration is deleted, preventing deterministic teardown of some topologies.
