## CollectViolations

Applies a rule set to a spec directory and returns all violations found.

### Inputs

| Name       | Type                                                         | Description                                                                                                                                                                                                     |
| ---------- | ------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `target`   | [SpecTarget](../../../concepts/cli/concept.md#spectarget)    | The spec directory to evaluate.                                                                                                                                                                                 |
| `rule_set` | [RuleSet](../../../concepts/verification/concept.md#ruleset) | The rules to evaluate against the directory.                                                                                                                                                                    |
| `progress` | optional boolean                                             | When `true`, `ok  <relpath>` is written to the error channel for each validated item: `.` after directory-level rules, and the path of each file matched by at least one file-level rule, relative to `target`. |

### Outputs

| Name     | Type                                                                               | Description                                                                  |
| -------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `result` | [VerificationResult](../../../concepts/verification/concept.md#verificationresult) | All violations found, or an empty list if the directory is fully conformant. |

### Invariants

- For every rule in `rule_set.rules` whose `applies_to` is a filename, every file under `target` with that filename is evaluated against that rule.
- Every rule in `rule_set.rules` whose `applies_to` is `spec` is evaluated once against the parsed spec as a whole, after all directory-level and file-level rules.
- The spec version is read from the `**Version:**` label in `spec.md`; rules that apply only at `1.0.0` and above produce no violations while the version is below `1.0.0` or absent.
- Line endings are normalized before any rule reads a file, so a file with CRLF line endings is reported by the line-ending rule rather than misread by every other rule.
- Every violation in `result.violations` has a `rule` value that matches the `id` of a rule in `rule_set.rules`.
- Every violation in `result.violations` has a `severity` copied from the corresponding rule.
- Every violation in `result.violations` has a `description` copied from the corresponding rule.
- Every violation in `result.violations` references an existing path under `target`.
- `result.passed` is `true` if and only if `result.violations` contains no `Error`-severity violations.
- `result.aasdd_version` equals `rule_set.aasdd_version`.
- `result.rule_count` equals the number of rules in `rule_set.rules`.
- `result.spec_file_names` contains exactly the set of `applies_to` values of file-level rules, deduplicated and sorted.
- When `progress` is `true`, exactly one `ok  .` line is written after directory-level rules, and exactly one `ok  <relpath>` line is written per file matched by at least one file-level rule.

### Failure Modes

| Failure          | Condition                                                                                          | Effect                                                 |
| ---------------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| `TargetNotFound` | `target.path` does not exist.                                                                      | Error propagated to caller.                            |
| `TargetIsFile`   | `target.path` exists but is a file, not a directory.                                               | Error propagated to caller.                            |
| `ReadError`      | A file under `target` cannot be read due to a filesystem permission or I/O error during traversal. | Error propagated to caller; partial results discarded. |
