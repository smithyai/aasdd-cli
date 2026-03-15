## LoadRuleSet

Derives the complete set of structural rules for a given AASDD version.

### Inputs

| Name            | Type | Description                                         |
| --------------- | ---- | --------------------------------------------------- |
| `aasdd_version` | text | The AASDD version to load rules for (e.g., `"v1"`). |

### Outputs

| Name       | Type                                                         | Description                                           |
| ---------- | ------------------------------------------------------------ | ----------------------------------------------------- |
| `rule_set` | [RuleSet](../../../concepts/verification/concept.md#ruleset) | The complete rule set for the resolved AASDD version. |

### Invariants

- `rule_set.rules` is non-empty.
- Every rule in `rule_set.rules` has a unique `id`.
- `rule_set.aasdd_version` equals the input `aasdd_version`.

### Failure Modes

| Failure               | Condition                                      | Effect                      |
| --------------------- | ---------------------------------------------- | --------------------------- |
| `UnknownAASDDVersion` | `aasdd_version` is not recognised by the tool. | Error propagated to caller. |
