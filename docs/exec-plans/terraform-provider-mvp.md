# Incident Garden Terraform Provider MVP

## Resource and endpoint mapping

| Terraform resource | API collection / item | Identity | Relationships |
|---|---|---|---|
| `incidentgarden_team` | `/api/v1/orgs/{slug}/teams[/ {id}]` | UUID | Parent for the other MVP objects |
| `incidentgarden_team_members` | `/api/v1/orgs/{slug}/teams/{id}/members[/ {userId}]` | Team UUID | Authoritative complete team-member set |
| `incidentgarden_schedule` | `/api/v1/orgs/{slug}/schedules[/ {id}]` | UUID | Immutable `team_id` |
| `incidentgarden_rotation` | `/api/v1/orgs/{slug}/schedules/{id}/rotations[/ {rotationId}]` | Nested UUID | Immutable parent schedule; ordered participants |
| `incidentgarden_escalation_policy` | `/api/v1/orgs/{slug}/escalation-policies[/ {id}]` | UUID | Immutable `team_id`; ordered steps target a schedule, user, or team |
| `incidentgarden_integration` | `/api/v1/orgs/{slug}/integrations[/ {id}]` | UUID | Immutable `team_id` and `type`; mutable escalation policy |
| `incidentgarden_integration_policy` | `/api/v1/orgs/{slug}/policies[/ {id}]` | UUID | Immutable team/integration/all-integrations scope |

Most imports use `<organization-slug>/<resource-id>`. Team membership uses the team ID in that form, while rotations use `<organization-slug>/<schedule-id>/<rotation-id>`. Overrides remain future standalone resources.

## Authentication and transport

The contract documents 15-minute bearer access JWTs and six-hour refresh JWTs. Every authenticated mutation also requires a `csrf_token` cookie matching `X-CSRF-Token`. No machine/API token is documented. The provider accepts externally acquired bearer and CSRF values through a client credential-provider boundary; the configured implementation is static and does not persist rotating refresh tokens. This works for interactive/short automation but is not a durable unattended credential model. The product should add a revocable, scoped, long-lived machine token exempt from browser CSRF, or a documented non-interactive refresh grant.

The client owns URL construction, JSON envelopes, typed errors, bounded response reads, context cancellation, CSRF, and HTTP 429 retry for every method. The API guarantees that a 4xx response precedes mutation, making this specific mutation retry safe without a general idempotency key. `Retry-After` is honored with a 60-second per-attempt ceiling. Diagnostics retain HTTP status, code, and message without request/response logging.

## Terraform lifecycle decisions

- Organization is resource-address state and may default from the provider, preserving multi-organization use.
- All item reads reconstruct remote state. Genuine 404 removes state; 403 and other failures remain diagnostics.
- Team and schedule descriptions preserve API null separately from empty strings.
- Escalation steps use an ordered list with explicit target ID attributes. Descending offsets are rejected during configuration validation; stable equal offsets are allowed.
- Schedule rotations are standalone resources because they have independent identity and lifecycle; overrides remain unimplemented.
- Team membership is an authoritative collection resource because individually managed memberships would conflict with exclusive ownership of the remote set.
- Integration `api_key` is computed, sensitive, returned once, and retained from prior state on refresh. `api_key_prefix` is computed and non-secret.
- Integration severity mapping and integration-policy conditions/action use typed nested attributes. Type-dependent integration settings remain semantic JSON. Deprecated JSON fields preserve release-candidate state compatibility.
- Integration policy is a real independent CRUD object. Its API name is `/policies`, not `/integration-policies`; exactly one action is wrapped in the API array.
- Delete never cascades. API conflicts remain in state and include dependency-oriented context.

## Testing strategy

Unit tests cover import parsing, JSON canonicalization, and one-time secret preservation. Client tests cover auth/CSRF headers, envelopes, typed errors, 404 classification, retry timing parsing, and pagination. The team acceptance test covers create, update, import, and destroy against an explicitly selected test organization and passed against the hosted API on 2026-09-09. Drift and external-deletion steps require backend-side test helpers and remain pending because no local Incident Garden backend exists in this repository.

## Phases and progress

- [x] Inspect repository, full supplied OpenAPI paths/schemas, tooling, and local instructions.
- [x] Establish Plugin Framework v6 provider and reusable HTTP client.
- [x] Implement team vertical slice with CRUD, import, refresh, 404, tests, example, and docs.
- [x] Implement schedule CRUD without conflating rotations/overrides.
- [x] Implement ordered rotation CRUD and authoritative team membership.
- [x] Add privileged, paginated organization-member lookup.
- [x] Implement ordered escalation policy steps.
- [x] Implement integration lifecycle and one-time secret preservation.
- [x] Verify and implement independent integration policy lifecycle.
- [x] Add examples, unit/client tests, acceptance foundation, and developer commands.
- [x] Run the team lifecycle acceptance test against an explicitly authorized test organization.
- [ ] Complete live acceptance coverage for the remaining four resources (create/read/update reached the hosted API; import/destroy is blocked by the hidden integration reference defect).
- [ ] Add backend test helpers for out-of-band drift and deletion acceptance steps.
- [ ] Replace short-lived browser-session credentials with a supported machine credential when the API provides one.

## Unresolved API questions

1. There is no documented machine credential for unattended Terraform; access JWT lifetime and CSRF bind the provider to externally managed browser-session material.
2. The contract does not state defaults for integration severity mapping/settings, integration active state, policy conditions, or policy active state in request schemas; the provider treats returned values as authoritative computed defaults.
3. The supplied OpenAPI is now vendored, but its upstream commit was absent from both the artifact and API build metadata and remains recorded as unknown.
4. On the hosted API, deleting an integration removes it from GET/list immediately but can leave its escalation-policy reference active: deleting that policy then returns 409 `escalation policy is used by integrations`. This contradicts the documented observable deletion lifecycle and prevents deterministic Terraform destroy of the containing topology.

## Decision log

- 2026-09-09: Chose Plugin Framework and protocol v6; no legacy SDK constraints exist.
- 2026-09-09: Chose uniform two-segment import IDs because every MVP item endpoint needs only organization plus UUID.
- 2026-09-09: Added optional provider organization while retaining resource overrides.
- 2026-09-09: Exposed CSRF explicitly rather than implementing password/login workflows in resources.
- 2026-10-08: Extended 429 retries to every method after the API guaranteed 4xx responses occur before mutation; other mutation failures are never replayed.
- 2026-09-09: Kept rotations and overrides out of schedule and retained `/policies` as the integration-policy resource.
- 2026-09-09: Live team acceptance testing exposed and fixed computed IDs becoming unknown during update planning; create, update, composite import, and destroy now pass.
- 2026-09-09: Added a six-step full-topology acceptance test covering create, update, import, and destroy for schedule, escalation policy, integration, and integration policy. Live create/read/update reached every resource; import/destroy could not complete because integration deletion left a hidden reference blocking escalation-policy cleanup.
- 2026-09-09: Added `examples/live-test` as a credential-free, complete manual topology for the authorized `david-tech-org` organization.
- 2026-10-08: Added a standalone ordered rotation resource, privileged member lookup, and authoritative team-membership collection; rejected per-member resources because they cannot safely coexist with exclusive collection ownership.
- 2026-10-08: Extracted request-time credential injection behind an interface but deliberately left refresh-token rotation and persistence outside provider state.
- 2026-10-08: Rejected fixed resource-quota and account-wide rate-limit numbers from review feedback where the contract only defines operator/plan-configured limits; documentation keeps only contract-backed creation-rate defaults.
