## CollectViolations

**Purpose:** Applies a rule set to a spec directory and returns all violations found.

### Inputs

| Name       | Type                                                         | Description                                                                                                                                                                                                              |
| ---------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `target`   | [SpecTarget](../../../concepts/cli/concept.md#spectarget)    | The spec directory to evaluate.                                                                                                                                                                                          |
| `rule_set` | [RuleSet](../../../concepts/verification/concept.md#ruleset) | The rules to evaluate against the directory.                                                                                                                                                                             |
| `progress` | optional boolean                                             | When `true`, `ok  <relpath>` is written to the error channel for each validated item: `.` after directory-level rules, and the file's path relative to `target` after each file matched by at least one file-level rule. |

### Outputs

| Name     | Type                                                                               | Description                                                                  |
| -------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `result` | [VerificationResult](../../../concepts/verification/concept.md#verificationresult) | All violations found, or an empty list if the directory is fully conformant. |

`result.spec_file_names` is the deduplicated, sorted list of basenames that at least one file-level rule in `rule_set` applies to.

### Invariants

- For every rule in `rule_set.rules`, every file under `target` whose filename matches `rule.applies_to` is evaluated against that rule.
- Every violation in `result.violations` has a `rule` value that matches the `id` of a rule in `rule_set.rules`.
- Every violation in `result.violations` has a `severity` copied from the corresponding rule.
- Every violation in `result.violations` has a `description` copied from the corresponding rule.
- Every violation in `result.violations` references an existing path under `target`.
- `result.passed` is `true` if and only if `result.violations` contains no `Error`-severity violations.
- `result.aasdd_version` equals `rule_set.aasdd_version`.
- `result.rule_count` equals the number of rules in `rule_set.rules`.
- `result.spec_file_names` contains exactly the set of `applies_to` values from file-level rules (rules where `applies_to` ≠ `"directory"`), deduplicated and sorted.
- When `progress` is `true`, exactly one `ok  .` line is written after directory-level rules, and exactly one `ok  <relpath>` line is written per file matched by at least one file-level rule.

### Failure Modes

| Failure          | Condition                                                                                          | Effect                                                 |
| ---------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| `TargetNotFound` | `target.path` does not exist.                                                                      | Error propagated to caller.                            |
| `TargetIsFile`   | `target.path` exists but is a file, not a directory.                                               | Error propagated to caller.                            |
| `ReadError`      | A file under `target` cannot be read due to a filesystem permission or I/O error during traversal. | Error propagated to caller; partial results discarded. |
