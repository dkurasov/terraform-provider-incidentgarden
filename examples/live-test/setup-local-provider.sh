#!/bin/sh
set -eu

example_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_dir=$(CDPATH= cd -- "$example_dir/../.." && pwd)
provider_version=0.1.0
provider_os=$(go env GOOS)
provider_arch=$(go env GOARCH)
mirror_root="$example_dir/.terraform-provider-mirror"
package_dir="$mirror_root/registry.terraform.io/incidentgarden/incidentgarden/$provider_version/${provider_os}_${provider_arch}"

rm -rf "$example_dir/.terraform" "$mirror_root"
rm -f "$example_dir/.terraform.lock.hcl" "$example_dir/.terraformrc"
mkdir -p "$package_dir"
go build -o "$package_dir/terraform-provider-incidentgarden_v$provider_version" "$repository_dir"

sed "s|MIRROR_PATH|$mirror_root|g" "$example_dir/terraformrc.template" > "$example_dir/.terraformrc"

printf '%s\n' "Local provider built for ${provider_os}_${provider_arch}."
printf '%s\n' "Run: export TF_CLI_CONFIG_FILE='$example_dir/.terraformrc'"
printf '%s\n' "Then run: terraform init"
