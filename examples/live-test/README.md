# Live manual test

This configuration creates one team, its authoritative membership, a schedule with a real rotation, an escalation policy, a generic webhook integration, and an integration policy in `david-tech-org`. It contains no credentials. Set `TF_VAR_on_call_email` to an existing organization member; listing members requires an organization admin or owner.

Build a local filesystem mirror and tell Terraform to use it instead of querying the public registry:

```sh
sh setup-local-provider.sh
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"
terraform init
```

The setup script removes only generated initialization metadata (`.terraform`, the local mirror, CLI configuration, and dependency lock) so stale provider addresses or checksums from an earlier build cannot interfere. It never removes Terraform state.

Then provide the short-lived session values without writing them to disk. The token may be the raw JWT or include the `Bearer ` prefix; the provider normalizes both forms and sends exactly one prefix:

```sh
export INCIDENTGARDEN_TOKEN='eyJ...'
export INCIDENTGARDEN_CSRF_TOKEN='...'
export TF_VAR_on_call_email='operator@example.com'
terraform plan
terraform apply
```

The access JWT lasts only 15 minutes. Refresh both session values immediately before `plan`/`apply` if the API reports `invalid or expired token`.

Inspect the one-time integration key with `terraform output integration_api_key`. Terraform marks it sensitive, but the command intentionally reveals it for this manual test.

Delete the complete topology in dependency-safe order with:

```sh
terraform destroy
```

The membership resource owns the complete member set of the newly created team. Removing it destroys all team memberships, so do not retarget it at an existing team. A known hosted-API issue can also leave a deleted integration's escalation-policy reference behind, causing the subsequent policy delete to return HTTP 409. Until the backend lifecycle is corrected, a complete destroy may require backend/operator cleanup. Do not apply this example if leaving temporary objects would be unacceptable.

Do not commit `.terraform/`, plan files, state files, variable files containing credentials, or shell history containing session values.
