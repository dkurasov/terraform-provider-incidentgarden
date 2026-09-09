# Changelog workflow

This project uses [Changie](https://changie.dev/) to collect one small changelog
fragment per pull request and assemble those fragments into `CHANGELOG.md` at
release time. Keeping unreleased changes as separate files avoids frequent merge
conflicts in the main changelog.

## Pull requests

Install Changie, then create a fragment:

```shell
go install github.com/miniscruff/changie@v1.25.0
make changelog-new
```

Select the category that best describes the user-visible change and enter the
pull request or issue number. Write the entry as `subsystem: Description`, for
example `resource/team: Preserve member order during refresh.` Commit the new
YAML file under `.changes/unreleased` with the rest of the change.

Use `BREAKING CHANGES` for incompatible behavior, `FEATURES` for new
capabilities, `ENHANCEMENTS` for improvements, `BUG FIXES` for corrections, and
`NOTES` for important operational or upgrade information. Internal refactoring,
tests, and documentation corrections do not need a fragment; add the
`skip-changelog` label to those pull requests.

Run `make changelog-check` before opening the pull request. CI requires a valid
fragment unless the pull request has the `skip-changelog` label.

## Releases

After choosing a semantic version, batch the fragments and merge the generated
version file into the changelog:

```shell
make changelog-release VERSION=0.1.0
```

Review and commit `CHANGELOG.md`, the new `.changes/<version>.md` file, and the
removal of the consumed fragments as part of the release pull request. The
release tag must use the same version prefixed with `v`, for example `v0.1.0`.
