## Scaffold

**Purpose:** Creates a spec directory populated with the correct structure and stub files for a new AASDD spec.

### Inputs

| Name | Type | Description |
| --- | --- | --- |
| `target` | [SpecTarget](../../concepts/cli/concept.md#spectarget) | The directory to create or populate. |
| `spec_version` | optional [SpecVersion](../../concepts/cli/concept.md#specversion) | The AASDD version whose structure and templates to use. Defaults to the latest version known to the tool. |

### Outputs

| Name | Type | Description |
| --- | --- | --- |
| `result` | [ScaffoldResult](../../concepts/scaffold/concept.md#scaffoldresult) | The directory created and the list of files written. |

### Invariants

- Every file in `result.files_created` exists on disk after the ability completes.
- `result.files_created` is non-empty.

### Failure Modes

| Failure | Condition | Effect |
| --- | --- | --- |
| `TargetNotEmpty` | `target.path` exists and already contains files. | Error written to stderr; exit code non-zero. No files are written. |
| `UnknownSpecVersion` | `spec_version` is provided but not recognised by the tool. | Error written to stderr; exit code non-zero. No files are written. |
| `WriteError` | A file cannot be written due to a filesystem permission or I/O error. | Error written to stderr; exit code non-zero. Any files written before the error are left in place. |
