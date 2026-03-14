## Diff domain

Types representing the structural differences between two specs.

### DiffKind

Classifies how a construct differs between the two specs.

| Value | Meaning |
| ----- | ------- |
| `Added` | The construct exists in the right spec but not the left. |
| `Removed` | The construct exists in the left spec but not the right. |
| `Changed` | The construct exists in both specs but differs structurally. |

### DiffEntry

A single structural difference between two specs.

#### Properties

| Name | Type | Description |
| ---- | ---- | ----------- |
| `path` | text | Spec-relative path of the affected file. |
| `kind` | [DiffKind](#diffkind) | Whether the construct was added, removed, or changed. |
| `construct` | text | The type of construct (e.g. `ability`, `concept`, `decision`, `scenario`, `spec`). |
| `detail` | text | Human-readable description of what changed. |

### DiffResult

The outcome of comparing two spec directories.

#### Properties

| Name | Type | Description |
| ---- | ---- | ----------- |
| `left` | [SpecTarget](../cli/concept.md#spectarget) | The baseline spec directory. |
| `right` | [SpecTarget](../cli/concept.md#spectarget) | The spec directory compared against the baseline. |
| `entries` | list of [DiffEntry](#diffentry) | All structural differences found, in path order. |
| `changed` | boolean | `true` when `entries` is non-empty. |
