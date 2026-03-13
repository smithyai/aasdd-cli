## LoadRuleSet

**Purpose:** Derives the complete set of structural rules for a given AASDD spec version.

### Inputs

| Name | Type | Description |
| --- | --- | --- |
| `spec_version` | optional [SpecVersion](../../../concepts/cli/concept.md#specversion) | The AASDD version to load rules for. Defaults to the latest version known to the tool. |

### Outputs

| Name | Type | Description |
| --- | --- | --- |
| `rule_set` | [RuleSet](../../../concepts/verification/concept.md#ruleset) | The complete rule set for the resolved spec version. |

### Invariants

- `rule_set.rules` is non-empty.
- Every rule in `rule_set.rules` has a unique `id`.
- `rule_set.spec_version` equals the resolved version (the requested version, or the latest if none was requested).

### Failure Modes

| Failure | Condition | Effect |
| --- | --- | --- |
| `UnknownSpecVersion` | `spec_version` is provided but not recognised by the tool. | Error propagated to caller. |
