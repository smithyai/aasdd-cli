## InvocationChannel

### Context

All root-level abilities (`Verify`, `Scaffold`, `Export`, `Import`, `ListVersions`, `Diff`, `Graph`) accept inputs that must arrive from outside the tool, which requires choosing how callers invoke the tool and pass arguments.

### Requirement

A mechanism for a caller to invoke the tool, supply a target path, and receive output.

### Decision

CLI — the tool is invoked as `aasdd <command> <path> [flags]`. The target path is a positional argument. Results are written to stdout in human-readable form; errors and diagnostics are written to stderr. Exit code signals success or failure.

`Diff` takes two positional arguments (`aasdd diff <left> <right>`) instead of one.

`Graph` accepts an optional `--format` / `-f` flag (`mermaid` or `dot`). Defaults to `mermaid`.

The AASDD version is not supplied as a flag. For `verify`, it is read from the `**AASDD:**` label in `spec.md` (falling back to the latest version if absent). For `scaffold`, the latest version is always used unless `--aasdd-version` / `-v` overrides it.

Two optional diagnostic flags are available:

- `--progress` / `-p`: streams `ok  <path>` to stderr for each validated path as work happens — the root directory after directory-level rules, and each file matched by at least one file-level rule.
- `--verbose`: always prints `AASDD: <version> (<N> rules)` to stderr, and enriches each violation line with the rule's human-readable description.
- `--aasdd-version` / `-a` (scaffold only): the AASDD version to scaffold for (e.g. `--aasdd-version v1`). Defaults to the latest version known to the tool.

`--version` / `-V` is available on the root command only. It prints the tool version derived from the Go module build info (e.g. `v0.1.0`), or `(devel)` for local untagged builds.
