# OEFP CLI

Command-line tools for validating OEFP checkouts and generating submissions.

## Install

```sh
go install github.com/oefp/cli/cmd/oefp@latest
```

For local development, run `go run ./cmd/oefp help` and `go test ./...`.

## Lint a checkout

Run this from the root of an OEFP repository:

```sh
oefp lint
```

Use `--root PATH` to check another checkout. The linter validates canonical
eight-digit theorem and definition IDs, their sharded paths, and the YAML
metadata in each entry's first Lean module doc comment. It has no Python
runtime or package dependencies.

## Generate a submission

Commit all proposed changes, then generate an archive:

```sh
oefp generate
```

By default this bundles commits in `oefp/main..HEAD` and writes
`submission.oefp`. A checkout must have a clean working tree, the base must be
an ancestor of `HEAD`, and at least one commit must follow the base. Forks that
name the official remote differently can select any local commit or ref:

```sh
oefp generate --base upstream/main --output my-change.oefp
```

Use `--force` to replace an existing output file. The resulting uncompressed
tar contains exactly:

- `submission.yaml`, with `format_version: 1` and the bundle filename
- `submission.bundle`, containing the committed Git history after the base
