## InvocationChannel

### Context

Both `Verify` and `Scaffold` accept a `SpecTarget` as their root-level input. That path must arrive from outside the tool, which requires choosing how callers invoke the tool and pass arguments.

### Requirement

A mechanism for a caller to invoke the tool, supply a target path and optional spec version, and receive output.

### Decision

CLI — the tool is invoked as `aasdd <command> <path> [--spec-version <version>]`. The target path is a positional argument. Results are written to stdout in human-readable form; errors and diagnostics are written to stderr. Exit code signals success or failure.
