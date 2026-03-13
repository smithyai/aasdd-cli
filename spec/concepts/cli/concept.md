## CLI domain

Types representing inputs and outputs at the command-line boundary.

### SpecTarget

The spec directory path supplied by the caller.

#### Properties

| Name | Type | Description |
| --- | --- | --- |
| `path` | text | Absolute or relative path to the directory to operate on. |

### SpecVersion

A pinned AASDD spec version to verify or scaffold against.

#### Properties

| Name | Type | Description |
| --- | --- | --- |
| `value` | text | Integer version string identifying the AASDD version (e.g., `"v1"`). |
