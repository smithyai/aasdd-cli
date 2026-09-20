## Diff

Compares two spec directories and reports all structural differences between them.

### Inputs

| Name    | Type                                                   | Description                                        |
| ------- | ------------------------------------------------------ | -------------------------------------------------- |
| `left`  | [SpecTarget](../../concepts/cli/concept.md#spectarget) | The first spec directory (baseline).               |
| `right` | [SpecTarget](../../concepts/cli/concept.md#spectarget) | The second spec directory (compared against left). |

### Outputs

| Name     | Type                                                    | Description                       |
| -------- | ------------------------------------------------------- | --------------------------------- |
| `result` | [DiffResult](../../concepts/diff/concept.md#diffresult) | All structural differences found. |

### Invariants

- Both `left` and `right` are parsed into their structured representations before comparison.
- Every field of the structured representation is compared, including the vision sections of `spec.md`, Composition, Idempotency, Options, Example, delegation fields, placeholders, and custom sections.
- `result.changed` is `true` if and only if `result.entries` is non-empty.
- Every entry in `result.entries` references a path that exists in at least one of the two specs.
- Comparison is structural, not textual — whitespace and formatting differences that do not change parsed content are ignored.

### Failure Modes

| Failure             | Condition                                     | Effect                                       |
| ------------------- | --------------------------------------------- | -------------------------------------------- |
| `LeftNotFound`      | `left.path` does not exist.                   | Error written to stderr; exit code non-zero. |
| `LeftNotDirectory`  | `left.path` is a file, not a directory.       | Error written to stderr; exit code non-zero. |
| `RightNotFound`     | `right.path` does not exist.                  | Error written to stderr; exit code non-zero. |
| `RightNotDirectory` | `right.path` is a file, not a directory.      | Error written to stderr; exit code non-zero. |
| `ReadError`         | A file under either directory cannot be read. | Error written to stderr; exit code non-zero. |

### Notes

`--verbose` enriches each entry with the full detail of what changed within the construct. Without it, only the path, kind, and construct type are shown.
