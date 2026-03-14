## ListVersions

**Purpose:** Lists all AASDD methodology versions known to the tool, with a short summary of each version's changes.

### Outputs

| Name | Type | Description |
| ---- | ---- | ----------- |
| `versions` | list of [VersionEntry](../../concepts/versions/concept.md#versionentry) | All known AASDD versions, oldest first. |

### Invariants

- `versions` is non-empty.
- The entry for the latest version is last in `versions`.
