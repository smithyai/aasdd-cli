## Export

**Purpose:** Serializes a spec directory into a portable structured file that can be stored, transferred, or used to reconstruct the original directory.

### Inputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `source` | text | Path to a spec directory to export. |
| `output` | text | Destination file path for the exported output. Omit to write to the output channel. |

### Outputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `result` | [TransferResult](../../concepts/representation/concept.md#transferresult) | Summary of the export operation. |

The data written to `output` (or the output channel) is a serialized [SpecExport](../../concepts/representation/concept.md#specexport) — a structured, semantic representation of the spec directory. The serialization format is determined by the implementation — see `decisions/` for the rationale.

### Invariants

- `source` must be a directory.
- Every spec file (`spec.md`, `ability.md`, `concept.md`, `scenario.md`, `decision.md`) under `source` is represented in the export. All other files and hidden entries are excluded.
- Abilities, scenarios, sub-abilities, concepts, and decisions are each parsed into their respective structured types.
- `result.output_path` is empty when output is written to the output channel.

### Failure Modes

| Failure | Condition | Effect |
| ------- | --------- | ------ |
| `SourceNotFound` | `source` does not exist. | Error written to stderr; exit code non-zero. |
| `SourceNotDirectory` | `source` exists but is not a directory. | Error written to stderr; exit code non-zero. |
| `WriteError` | An output file cannot be written. | Error written to stderr; exit code non-zero. |

### Notes

The `--verbose` and `--progress` flags inherited from the root command have no effect on `export`.
