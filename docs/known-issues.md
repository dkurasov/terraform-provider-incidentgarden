# Known release issues

## Retained integration reference after deletion

The hosted API has returned HTTP 409 when deleting an escalation policy after
the integration referencing it was successfully deleted. The deleted
integration subsequently disappears from both item GET and collection results,
but the escalation-policy reference check still considers it active.

Minimal reproduction:

1. Create an escalation policy and record its ID.
2. Create an integration whose `escalation_policy_id` is that policy ID.
3. Delete the integration and verify its item GET returns 404.
4. Delete the escalation policy.
5. Observe HTTP 409 with the message `escalation policy is used by integrations`.

Expected result: step 4 succeeds because no live integration references the
policy. Actual result: the API rejects deletion based on an object that is no
longer observable. Current evidence assigns this to the backend reference/data
model, potentially a soft-deleted row included by the reference query. Cache,
transaction, and persistence timing still require backend instrumentation.

Minimum backend fix: dependency checks must exclude deleted integrations (or
remove their policy reference atomically during deletion), with a regression
test for the sequence above.

The provider intentionally does not sleep, cascade, retry destructive requests,
or discard state. It reports the API failure and retains the policy in state.

## Destruction is not always a reversible topology operation

The retained integration reference above is not the only reason a full destroy
can stop. Alerts may retain escalation-policy references for up to the product's
90-day alert-retention period, so a policy can remain undeletable even after
its live routing objects are gone. Team and schedule deletion then remain
blocked by that policy. Terraform correctly retains each rejected object in
state; it cannot safely reinterpret a dependency conflict as successful
deletion.

Plan production ownership around these constraints: use dedicated teams, keep
state after a partial destroy, and arrange backend/operator cleanup when a 409
cannot be resolved through visible live resources. Do not automate blind state
removal. The authoritative `incidentgarden_team_members` resource is a separate
destructive boundary: apply removes unconfigured members and destroy removes
all members, potentially revoking the caller's own team access.

## Machine authentication

The API currently requires a short-lived browser-session JWT and a matching
CSRF token for mutations. This requires external renewal, couples automation to
a browser security mechanism, and is unsuitable for unattended Terraform runs.

The backend workstream should add named automation identities with revocable,
organization- and permission-scoped bearer tokens. Tokens should support an
optional expiry, last-used metadata, audit events, one-time display, hashed
server-side storage, and mutation access without browser CSRF.

## Rotation time restriction cannot be cleared in place

The OpenAPI update schema permits `time_restriction: null`, but the hosted API
retained the existing restriction when Terraform sent that clear on 2026-10-08.
The provider therefore replaces a rotation when configuration removes a stored
time restriction. Adding or changing a restriction remains an in-place update.
