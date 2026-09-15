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

## Machine authentication

The API currently requires a short-lived browser-session JWT and a matching
CSRF token for mutations. This requires external renewal, couples automation to
a browser security mechanism, and is unsuitable for unattended Terraform runs.

The backend workstream should add named automation identities with revocable,
organization- and permission-scoped bearer tokens. Tokens should support an
optional expiry, last-used metadata, audit events, one-time display, hashed
server-side storage, and mutation access without browser CSRF.
