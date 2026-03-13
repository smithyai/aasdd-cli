## CollectViolations

**Purpose:** Applies a rule set to a spec directory and returns all violations found.

### Inputs

| Name | Type | Description |
| --- | --- | --- |
| `target` | [SpecTarget](../../../concepts/cli/concept.md#spectarget) | The spec directory to evaluate. |
| `rule_set` | [RuleSet](../../../concepts/verification/concept.md#ruleset) | The rules to evaluate against the directory. |

### Outputs

| Name | Type | Description |
| --- | --- | --- |
| `result` | [VerificationResult](../../../concepts/verification/concept.md#verificationresult) | All violations found, or an empty list if the directory is fully conformant. |

### Invariants

- For every rule in `rule_set.rules`, every file under `target` whose filename matches `rule.applies_to` is evaluated against that rule.
- Every violation in `result.violations` has a `rule` value that matches the `id` of a rule in `rule_set.rules`.
- Every violation in `result.violations` has a `severity` copied from the corresponding rule.
- Every violation in `result.violations` references an existing path under `target`.
- `result.passed` is `true` if and only if `result.violations` contains no `Error`-severity violations.

### Failure Modes

| Failure | Condition | Effect |
| --- | --- | --- |
| `TargetNotFound` | `target.path` does not exist or is not a directory. | Error propagated to caller. |
