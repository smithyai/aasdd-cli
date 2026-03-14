## Scaffold

**Purpose:** Creates a spec directory populated with the correct structure and stub files for a new AASDD spec.

### Inputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `target` | [SpecTarget](../../concepts/cli/concept.md#spectarget) | The directory to create or populate. |
| `aasdd_version` | text | The AASDD version to scaffold for. Defaults to the latest version known to the tool. |
| `example` | boolean | When `true`, populates the directory with a worked example (one ability, one concept, one decision, one scenario) instead of minimal stubs. |

### Outputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `result` | [ScaffoldResult](../../concepts/scaffold/concept.md#scaffoldresult) | The directory created and the list of files written. |

### Invariants

- Every file in `result.files_created` exists on disk after the ability completes.
- `result.files_created` is non-empty.
- The generated `spec.md` records the AASDD version in its `**AASDD:**` label.

### Failure Modes

| Failure | Condition | Effect |
| ------- | --------- | ------ |
| `TargetIsFile` | `target.path` exists and is a file, not a directory. | Error written to stderr; exit code non-zero. No files are written. |
| `TargetNotEmpty` | `target.path` exists and already contains files. | Error written to stderr; exit code non-zero. No files are written. |
| `WriteError` | A file cannot be written due to a filesystem permission or I/O error. | Error written to stderr; exit code non-zero. Any files written before the error are left in place. |
| `UnknownAASDDVersion` | `aasdd_version` is not recognised by the tool. | Error written to stderr; exit code non-zero. No files are written. |
