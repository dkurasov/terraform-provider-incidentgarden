# External dogfooding workflow

Use a separate directory or repository so Terraform consumes the provider as an
external plugin. Never commit credentials, `.terraform`, plans, or state.

## Local or snapshot build

Run `make release-snapshot`, select the archive matching the test machine, and
unzip its provider binary into a local filesystem mirror matching:

```text
registry.terraform.io/dkurasov/incidentgarden/<version>/<os>_<arch>/
```

Point a private Terraform CLI configuration at that mirror as demonstrated by
`examples/live-test/terraformrc.template`. For an RC published to the Registry,
set the exact constraint `version = "0.1.0-rc.1"`; Terraform does not select a
prerelease implicitly.

## Lifecycle script

From a credential-free copy of `examples/live-test`:

```shell
terraform init
terraform apply
terraform plan -detailed-exitcode
terraform refresh
terraform plan -detailed-exitcode
terraform destroy
terraform plan -detailed-exitcode
```

Both clean-plan commands must exit 0. Repeat from zero state. Then perform the
documented test cases one at a time: mutate a mutable field through the API,
confirm Terraform detects and reconciles drift; import each object into matching
configuration; delete each object through the API, confirm refresh removes it
from state and plan recreates it; finally destroy and verify zero remote objects.

Record timestamps, resource IDs, command exit codes, and sanitized HTTP status
and error codes in the release plan. Never record access tokens, CSRF tokens,
integration API keys, raw state, or response bodies containing secrets.
