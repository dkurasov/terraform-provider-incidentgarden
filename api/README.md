# Vendored API contract

`openapi.yaml` is the contract used to implement and review this provider.
`openapi.meta` records its checksum and the upstream revision provenance that
was actually available. The initial supplied artifact and live API build
metadata did not identify a commit, so the value is explicitly `unknown`.

Refresh it from an upstream artifact with:

```sh
make api-sync OPENAPI_SOURCE=/path/to/openapi.yaml OPENAPI_COMMIT=<commit>
```

`make api-check` verifies the recorded checksum. When `OPENAPI_SOURCE` is set,
it also verifies that the vendored copy exactly matches that source.
