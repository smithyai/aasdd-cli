## WarningsOnly

**Description:** A spec directory with only warning-severity violations is still considered conformant.

> `LoadRuleSet` → `CollectViolations`

- All violations in `result.violations` have `severity` = `Warning`
- `result.passed` is `true`