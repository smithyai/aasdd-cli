## Versions

**Purpose:** Lists all AASDD methodology versions known to the tool, with a short summary of each version's changes.

### Inputs

_None._

### Outputs

| Name       | Type                  | Description                                                    |
| ---------- | --------------------- | -------------------------------------------------------------- |
| `versions` | list of VersionEntry  | All known AASDD versions, oldest first.                        |

### VersionEntry

| Name      | Type | Description                                                          |
| --------- | ---- | -------------------------------------------------------------------- |
| `version` | text | The AASDD version string (e.g. `v1`).                                |
| `summary` | text | A one-sentence description of what was introduced in this version.   |

### Invariants

- `versions` is non-empty.
- The entry for the latest version is last in `versions`.
