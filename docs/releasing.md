# Releasing

Releases use GoReleaser and the Terraform Registry's protocol 6 manifest. The
release configuration creates one ZIP per supported OS/architecture, a renamed
registry manifest, SHA-256 checksums, and a binary detached GPG signature over
the checksum file. GitHub releases are drafts until explicitly reviewed.

## Publishing safety gate

Manual dispatch runs validation and an unsigned snapshot without publishing.
Pushing a SemVer-shaped `v*` tag runs the complete release checks and then waits
for approval on the protected `release` environment before creating a signed
draft GitHub release. Draft review remains mandatory before public publication.

## Signing key

Create a dedicated RSA signing key; the Terraform Registry does not accept the
default ECC key type. Keep the private key and passphrase outside Git. Export
the public key to the Terraform Registry namespace. In GitHub, create a
protected environment named `release`, require a reviewer, restrict deployment
refs to release tags when publishing is enabled, and configure these environment
secrets:

- `GPG_PRIVATE_KEY`: ASCII-armored private key.
- `PASSPHRASE`: private-key passphrase.

The workflow obtains `GPG_FINGERPRINT` from the imported key. Locally, export
the fingerprint without placing it in a committed file:

```shell
export GPG_FINGERPRINT='<full fingerprint>'
goreleaser release --clean
```

The expected signature is a binary detached file named
`terraform-provider-incidentgarden_<version>_SHA256SUMS.sig`. Verify it with:

```shell
gpg --verify dist/terraform-provider-incidentgarden_<version>_SHA256SUMS.sig \
  dist/terraform-provider-incidentgarden_<version>_SHA256SUMS
```

## Release candidate procedure

1. Finish and record every live lifecycle gate in the release plan.
2. Run `make check` and `make release-snapshot` from a clean checkout.
3. Inspect archives, binary names, registry manifest, and checksums in `dist/`.
4. Validate a local signature using the dedicated release key.
5. Create the annotated tag `v0.1.0-rc.1` on the reviewed commit.
6. Push the tag, approve the protected `release` environment, inspect the draft
   GitHub release, verify its signature and
   checksums, and only then publish the draft.

GoReleaser derives `0.1.0-rc.1` from the tag and injects it into `main.version`.
The existing Changie `0.1.0` notes remain the final-release notes; do not batch
the same fragments again for the RC.

## Final release

After RC validation, fix or explicitly accept every final-release blocker,
create `v0.1.0` from the selected commit, and repeat checksum/signature review.
Never replace assets for an already published tag; publish a new version.
