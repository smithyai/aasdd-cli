## Verification domain

Types representing the results of structural verification.

### Severity

Indicates whether a rule violation blocks conformance or is advisory only.

| Value     | Meaning                                                                                  |
| --------- | ---------------------------------------------------------------------------------------- |
| `Error`   | The spec is structurally unusable — an implementer cannot reliably translate it to code. |
| `Warning` | The spec is structurally sound but violates a convention or is missing tooling metadata. |

### Rule

A single structural check derived from the AASDD conventions and readiness conditions for a given methodology version.

#### Properties

| Name          | Type                  | Description                                                                                                                                                                                                 |
| ------------- | --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`          | text                  | Stable identifier for this rule (e.g., `"ability.missing-purpose"`).                                                                                                                                        |
| `description` | text                  | Human-readable explanation of what this rule checks.                                                                                                                                                        |
| `applies_to`  | text                  | The artifact filename this rule evaluates (e.g., `"ability.md"`, `"spec.md"`), `"directory"` for rules about the directory layout, or `"spec"` for rules evaluated once against the parsed spec as a whole. |
| `severity`    | [Severity](#severity) | Whether a violation of this rule is an error or a warning.                                                                                                                                                  |

### RuleSet

The complete set of structural rules for a given AASDD version.

#### Properties

| Name            | Type                  | Description                                                     |
| --------------- | --------------------- | --------------------------------------------------------------- |
| `aasdd_version` | text                  | The AASDD version these rules were derived from (e.g., `"v2"`). |
| `rules`         | list of [Rule](#rule) | All structural rules to be evaluated.                           |

### VerificationResult

The outcome of verifying a spec directory.

#### Properties

| Name              | Type                                       | Description                                                               |
| ----------------- | ------------------------------------------ | ------------------------------------------------------------------------- |
| `target`          | [SpecTarget](../cli/concept.md#spectarget) | The directory that was verified.                                          |
| `aasdd_version`   | text                                       | The AASDD version used during verification.                               |
| `rule_count`      | number                                     | The number of rules evaluated.                                            |
| `violations`      | list of [Violation](#violation)            | All structural violations found.                                          |
| `passed`          | boolean                                    | `true` when there are no `Error`-severity violations.                     |
| `spec_file_names` | list of text                               | Basenames of files that at least one rule applies to (e.g. `ability.md`). |

### Violation

A single structural rule that was not satisfied.

#### Properties

| Name          | Type                  | Description                                                       |
| ------------- | --------------------- | ----------------------------------------------------------------- |
| `rule`        | text                  | The `id` of the violated rule.                                    |
| `severity`    | [Severity](#severity) | The severity of this violation, copied from the rule.             |
| `description` | text                  | The human-readable description of the rule, copied from the rule. |
| `path`        | text                  | File or directory path where the violation was found.             |
| `message`     | text                  | Human-readable explanation of the violation.                      |
