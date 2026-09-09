# Live manual test

This configuration creates one team, schedule, escalation policy, generic webhook integration, and integration policy in `david-tech-org`. It contains no credentials.

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
terraform plan
terraform apply
```

The access JWT lasts only 15 minutes. Refresh both session values immediately before `plan`/`apply` if the API reports `invalid or expired token`.

Inspect the one-time integration key with `terraform output integration_api_key`. Terraform marks it sensitive, but the command intentionally reveals it for this manual test.

Delete the complete topology in dependency-safe order with:

```sh
terraform destroy
```

Known hosted-API issue: on 2026-09-09, `DELETE /integrations/{id}` made the integration disappear from reads but retained its escalation-policy reference, so the subsequent policy delete returned HTTP 409. Until the backend lifecycle is corrected, a complete destroy may require backend/operator cleanup. Do not apply this example if leaving a temporary team, schedule, and escalation policy would be unacceptable.

Do not commit `.terraform/`, plan files, state files, variable files containing credentials, or shell history containing session values.
