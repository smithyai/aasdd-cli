# aasdd-cli

[![Latest release](https://img.shields.io/github/v/release/smithyai/aasdd-cli)](https://github.com/smithyai/aasdd-cli/releases/latest)
[![Release](https://github.com/smithyai/aasdd-cli/actions/workflows/release.yml/badge.svg)](https://github.com/smithyai/aasdd-cli/actions/workflows/release.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/smithyai/aasdd-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Packaging status](https://repology.org/badge/tiny-repos/aasdd.svg)](https://repology.org/project/aasdd/versions)

The official CLI for [Ability-Anchored Spec-Driven Development (AASDD)](https://github.com/smithyai/aasdd).

`aasdd` provides commands for verifying, scaffolding, exporting, importing, diffing, and graphing AASDD specs. It is also self-hosted: this repository's own spec lives in [`spec/`](spec/) and is verified against the tool on every commit.

## Installation

### Homebrew (macOS / Linux)

```sh
brew install smithyai/tap/aasdd
```

### Docker

```sh
docker pull ghcr.io/smithyai/aasdd-cli:latest
docker run --rm -v "$PWD:/work" ghcr.io/smithyai/aasdd-cli:latest verify /work/spec
```

### apt / deb

```sh
curl -fsSL https://github.com/smithyai/aasdd-cli/releases/latest/download/aasdd_linux_amd64.deb -o aasdd.deb
sudo dpkg -i aasdd.deb
```

### rpm / apk

Download the appropriate package from the [latest release](https://github.com/smithyai/aasdd-cli/releases/latest).

### Scoop (Windows)

```powershell
scoop bucket add smithyai https://github.com/smithyai/scoop-bucket
scoop install aasdd
```

### Go install

```sh
go install github.com/smithyai/aasdd-cli@latest
```

## Commands

### `verify`

Check a spec directory for conformance with AASDD structural conventions. Reports all violations with severity and location.

```sh
aasdd verify ./spec
```

### `scaffold`

Create a new spec directory populated with the correct structure and stub files.

```sh
# Minimal stubs
aasdd scaffold ./my-spec

# With a worked example (one ability, concept, decision, and scenario)
aasdd scaffold --example ./my-spec

# Pin to a specific AASDD version
aasdd scaffold --aasdd-version v1 ./my-spec
```

### `diff`

Compare two spec directories and report all structural differences.

```sh
aasdd diff ./spec-v1 ./spec-v2
```

### `export`

Serialize a spec directory into a portable structured file.

```sh
aasdd export ./spec -o spec.json
```

### `import`

Reconstruct a spec directory on disk from a previously exported snapshot.

```sh
aasdd import spec.json -o ./spec
```

### `graph`

Generate a dependency graph showing how abilities, concepts, decisions, and scenarios relate to each other.

```sh
# Default: Mermaid
aasdd graph ./spec

# Output to file
aasdd graph ./spec -o graph.mmd
```

### `versions`

List all AASDD methodology versions known to the tool.

```sh
aasdd versions
```

## Usage in CI

The Docker image is suitable for use in GitHub Actions and other CI systems without installing anything:

```yaml
- name: Verify spec
  run: docker run --rm -v "${{ github.workspace }}:/work" ghcr.io/smithyai/aasdd-cli:latest verify /work/spec
```

Or use the [`smithyai/aasdd-release`](https://github.com/smithyai/aasdd-release) action, which parses and validates the spec version as part of your release workflow:

```yaml
- uses: smithyai/aasdd-release@v1
  with:
    spec-path: spec/spec.md
    enforce-tag-match: true
```

## Development

**Prerequisites:** Go 1.26+, [goreleaser](https://goreleaser.com), [act](https://nektosact.com) (optional, for local workflow simulation), Docker with buildx.

```sh
make help        # list all available commands

make build       # build ./aasdd
make test        # run tests
make verify      # verify this repo's own spec using the local build

make release-build   # build all release artifacts locally (binaries + Docker images)
make release-test    # simulate the CI release workflow via act

make clean       # remove build artifacts and locally built Docker images
```

Releases are triggered by pushing a `vX.Y.Z` tag. The [GitHub Actions workflow](.github/workflows/release.yml) handles all publishing — Homebrew, Scoop, AUR, deb/rpm/apk, and ghcr.io.

## Methodology

This repository follows AASDD end-to-end. The spec in [`spec/`](spec/) defines the CLI's abilities, concepts, decisions, and scenarios. Implementation tracks the spec — if behavior exists in the code but not in the spec, the spec is incomplete. See the [AASDD methodology](https://github.com/smithyai/aasdd) for the full rule set.

## License

[MIT](LICENSE)
