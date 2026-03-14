## Direction Detection

## Context

The `parse` command accepts a `source` path that can point to either a spec directory or a JSON snapshot file. The command must determine which operation to perform — dir-to-JSON or JSON-to-dir — without requiring an explicit mode flag, since the source type already carries that information.

## Requirement

Direction must be determinable from the `source` path alone, without an additional flag or extra positional argument. The rule must be simple, unambiguous, and easy for users to predict.

## Decision

Inspect the entry at `source` using its filesystem type and, for files, its extension:

- If `source` is a directory → direction is **dir-to-JSON**.
- If `source` is a regular file with a `.json` extension (case-insensitive) → direction is **JSON-to-dir**.
- Any other case (file without `.json` extension, symlink to an unsupported type, etc.) → `SourceUnrecognized` failure.
