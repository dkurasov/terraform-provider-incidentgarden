# Testing

Unit and HTTP client tests are self-contained:

```sh
make test
make check
```

Acceptance tests are destructive and opt-in. Use a disposable local server and dedicated organization only:

```sh
export TF_ACC=1
export INCIDENTGARDEN_ENDPOINT=http://localhost:8080
export INCIDENTGARDEN_TOKEN='short-lived-access-token'
export INCIDENTGARDEN_CSRF_TOKEN='matching-session-csrf-token'
export INCIDENTGARDEN_TEST_ORGANIZATION='terraform-acceptance'
export TF_ACC_TERRAFORM_PATH='/absolute/path/to/terraform'
make testacc
```

The current API has no documented machine credential and its access JWT expires after 15 minutes. Refresh the test session immediately before a run. Never use a production organization.
