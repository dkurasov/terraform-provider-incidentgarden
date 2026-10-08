#!/bin/sh
set -eu

: "${OPENAPI_SOURCE:?Set OPENAPI_SOURCE to the upstream OpenAPI file}"

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination="$repository_root/api/openapi.yaml"
metadata="$repository_root/api/openapi.meta"
upstream_commit=${OPENAPI_COMMIT:-unknown}

cp "$OPENAPI_SOURCE" "$destination"
checksum=$(shasum -a 256 "$destination" | awk '{print $1}')
vendored_at=$(date -u +%Y-%m-%d)

{
  printf 'source=%s\n' "$OPENAPI_SOURCE"
  printf 'upstream_commit=%s\n' "$upstream_commit"
  printf 'sha256=%s\n' "$checksum"
  printf 'vendored_at=%s\n' "$vendored_at"
} >"$metadata"

printf 'Vendored %s (%s)\n' "$destination" "$checksum"
