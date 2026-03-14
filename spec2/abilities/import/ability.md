## Import

**Purpose:** Reconstructs a spec directory on disk from a previously exported snapshot.

### Inputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `source` | text | Path to an exported snapshot file to import. |
| `output` | text | Path to the directory to reconstruct. Must be empty or nonexistent. |

### Outputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `result` | [TransferResult](../../concepts/representation/concept.md#transferresult) | Summary of the import operation. |

### Invariants

- `output` must be an empty directory or a nonexistent path.
- Every parsed record in the export produces a Markdown file on disk at `output/<record.path>`.
- A round-trip (`export` followed immediately by `import` into an empty directory) produces a directory that is structurally equivalent to the original: all spec files are present at the same relative paths, and the semantic content is identical.
- `result.file_count` equals the number of files written to disk.

### Failure Modes

| Failure | Condition | Effect |
| ------- | --------- | ------ |
| `SourceNotFound` | `source` does not exist. | Error written to stderr; exit code non-zero. |
| `SourceNotFile` | `source` exists but is not a file. | Error written to stderr; exit code non-zero. |
| `OutputNotEmpty` | `output` exists and is non-empty, or exists as a non-directory. | Error written to stderr; exit code non-zero. |
| `ParseError` | `source` cannot be decoded as a valid snapshot. | Error written to stderr; exit code non-zero. |
| `WriteError` | A file cannot be written to the output directory. | Error written to stderr; exit code non-zero. |

### Notes

The `--verbose` and `--progress` flags inherited from the root command have no effect on `import`.
