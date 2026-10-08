#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
contract="$repository_root/api/openapi.yaml"
metadata="$repository_root/api/openapi.meta"

expected=$(awk -F= '$1 == "sha256" { print $2 }' "$metadata")
actual=$(shasum -a 256 "$contract" | awk '{print $1}')
if [ -z "$expected" ] || [ "$actual" != "$expected" ]; then
  printf 'OpenAPI checksum mismatch: expected %s, got %s\n' "$expected" "$actual" >&2
  exit 1
fi

if [ -n "${OPENAPI_SOURCE:-}" ] && ! cmp -s "$OPENAPI_SOURCE" "$contract"; then
  printf 'Vendored OpenAPI differs from OPENAPI_SOURCE=%s\n' "$OPENAPI_SOURCE" >&2
  exit 1
fi
