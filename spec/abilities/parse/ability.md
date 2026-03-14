## Parse

**Purpose:** Converts a spec directory into a portable JSON snapshot, or reconstructs a spec directory from a previously generated snapshot.

### Inputs

| Name     | Type    | Description                                                                                                                                                                                               |
| -------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `source` | path    | Path to a spec directory (dir-to-JSON) or a `.json` snapshot file (JSON-to-dir). Direction is inferred automatically — see the [direction-detection decision](decisions/direction-detection/decision.md). |
| `output` | path    | Destination for the result. For dir-to-JSON: a file path, or omit to write to stdout. For JSON-to-dir: the directory to write files into (must be empty or non-existent).                                 |
| `flat`   | boolean | When `true`, serialize as a flat `path → content` JSON object. When `false` (default), serialize as a nested directory-tree JSON object. Only meaningful for dir-to-JSON.                                 |

### Outputs

| Name     | Type                                                       | Description                     |
| -------- | ---------------------------------------------------------- | ------------------------------- |
| `result` | [ParseResult](../../concepts/parse/concept.md#parseresult) | Summary of the parse operation. |

### Invariants

- In dir-to-JSON, every spec file (`spec.md`, `ability.md`, `concept.md`, `scenario.md`, `decision.md`) under `source` appears in the snapshot. All other files and hidden entries are excluded.
- In JSON-to-dir, every entry in the snapshot produces a file on disk at `output/<entry-path>`.
- A round-trip (dir-to-JSON immediately followed by JSON-to-dir into an empty directory) produces a directory whose files are byte-for-byte identical to the originals.
- `result.file_count` equals the number of files in the snapshot.
- `result.output_path` is empty when output is written to stdout.

### Failure Modes

| Failure              | Condition                                                                                           | Effect                                       |
| -------------------- | --------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| `SourceNotFound`     | `source` does not exist.                                                                            | Error written to stderr; exit code non-zero. |
| `SourceUnrecognized` | `source` is a file but does not have a `.json` extension.                                           | Error written to stderr; exit code non-zero. |
| `OutputNotEmpty`     | JSON-to-dir and `output` exists and is non-empty, or exists as a file.                              | Error written to stderr; exit code non-zero. |
| `ParseError`         | JSON-to-dir and `source` is not valid JSON or contains values that are neither strings nor objects. | Error written to stderr; exit code non-zero. |
| `WriteError`         | An output file or directory cannot be created or written.                                           | Error written to stderr; exit code non-zero. |

### Notes

The `--verbose` and `--progress` flags inherited from the root command have no effect on `parse`.
