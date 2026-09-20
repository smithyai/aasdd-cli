package collect_violations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/analysis/verify/load_rule_set"
	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// greeterV2 is a conformant v2 draft spec. Tables are written unpadded and
// padded on write so the fixture is in canonical form.
var greeterV2 = map[string]string{
	"spec.md": `## Greeter

**AASDD:** v2
**Version:** 0.1.0

A minimal service that produces a personalised greeting.

### Purpose

For callers that need a personalised greeting.

### Non-Goals

_None._

### Success Criteria

| Criterion | Abilities | Scenarios |
| --- | --- | --- |
| A name yields a greeting containing it. | ` + "`Greet`" + ` | ` + "`happy-path`" + ` |

### Invariants

- Every greeting contains the name it was produced for.
`,
	"abilities/greet/ability.md":           "## Greet\n\nProduces a greeting for a name.\n\n### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `name` | text | The name to greet. |\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `result` | [GreetingResult](../../concepts/greeting/concept.md#greetingresult) | The greeting. |\n\n### Invariants\n\n- `result.message` contains `name`.\n\n### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n| `EmptyName` | `name` is empty. | Error returned to caller. |\n",
	"concepts/greeting/concept.md":         "## Greeting domain\n\nTypes for greetings.\n\n### GreetingResult\n\nThe result of a greeting.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `message` | text | The greeting text. |\n",
	"decisions/output-channel/decision.md": "## OutputChannel\n\n### Context\n\n`Greet` returns its result to the caller.\n\n### Requirement\n\nA way to return the greeting.\n\n### Decision\n\nReturn value.\n",
	"scenarios/happy-path/scenario.md":     "## Happy Path\n\nA name produces a greeting.\n\n> `Greet`\n\n- `result.message` contains the name.\n",
}

func writeFixture(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(format.PadDocument(content)), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func withOverrides(base map[string]string, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

func v2Rules(t *testing.T) types.RuleSet {
	t.Helper()
	rs, err := load_rule_set.LoadRuleSet("v2")
	if err != nil {
		t.Fatalf("load v2 rules: %v", err)
	}
	return rs
}

func verifyFixture(t *testing.T, files map[string]string) types.VerificationResult {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, files)
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result
}

func ruleIDs(result types.VerificationResult) []string {
	var ids []string
	for _, v := range result.Violations {
		ids = append(ids, v.Rule+": "+v.Message)
	}
	return ids
}

func expectRule(t *testing.T, result types.VerificationResult, id string) {
	t.Helper()
	if !hasViolationID(result.Violations, id) {
		t.Errorf("expected %s; got %v", id, ruleIDs(result))
	}
}

func expectNoRule(t *testing.T, result types.VerificationResult, id string) {
	t.Helper()
	if hasViolationID(result.Violations, id) {
		t.Errorf("did not expect %s; got %v", id, ruleIDs(result))
	}
}

func ready(files map[string]string) map[string]string {
	return withOverrides(files, map[string]string{
		"spec.md": strings.Replace(files["spec.md"], "**Version:** 0.1.0", "**Version:** 1.0.0", 1),
	})
}

// --- conformant draft ---

func TestV2_ConformantDraft_NoViolations(t *testing.T) {
	result := verifyFixture(t, greeterV2)
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got %v", ruleIDs(result))
	}
	if !result.Passed {
		t.Error("expected passed")
	}
}

// --- readiness ---

func TestV2_Ready_RootFailureModeNeedsScenario(t *testing.T) {
	result := verifyFixture(t, ready(greeterV2))
	expectRule(t, result, "coverage.root-failure-mode")

	covered := withOverrides(ready(greeterV2), map[string]string{
		"scenarios/empty-name/scenario.md": "## Empty Name\n\nAn empty name is rejected.\n\n> `Greet` — EmptyName → `Greet`\n\n- `Greet` fails with `EmptyName`.\n",
	})
	result = verifyFixture(t, covered)
	expectNoRule(t, result, "coverage.root-failure-mode")
	expectNoRule(t, result, "scenario.unknown-condition")
}

func TestV2_PendingAllowedOnlyInDraft(t *testing.T) {
	pending := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": strings.Replace(greeterV2["abilities/greet/ability.md"],
			"### Invariants\n\n- `result.message` contains `name`.\n", "### Invariants\n\n_Pending._\n", 1),
	})
	result := verifyFixture(t, pending)
	expectNoRule(t, result, "pending.not-allowed")
	expectNoRule(t, result, "placeholder.misplaced")

	result = verifyFixture(t, ready(pending))
	expectRule(t, result, "pending.not-allowed")
}

func TestV2_PendingOnOptionalSectionIsMisplaced(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": greeterV2["abilities/greet/ability.md"] + "\n### Idempotency\n\n_Pending._\n",
	})
	result := verifyFixture(t, files)
	expectRule(t, result, "placeholder.misplaced")
}

func TestV2_OpenDecision(t *testing.T) {
	openNoOptions := withOverrides(greeterV2, map[string]string{
		"decisions/output-channel/decision.md": "## OutputChannel\n\n### Context\n\n`Greet` returns its result to the caller.\n\n### Requirement\n\nA way to return the greeting.\n\n### Decision\n\n_Open._\n",
	})
	result := verifyFixture(t, openNoOptions)
	expectRule(t, result, "decision.open-without-options")

	openWithOptions := withOverrides(greeterV2, map[string]string{
		"decisions/output-channel/decision.md": "## OutputChannel\n\n### Context\n\n`Greet` returns its result to the caller.\n\n### Requirement\n\nA way to return the greeting.\n\n### Options\n\n- Return value.\n- Write to stdout.\n\n### Decision\n\n_Open._\n",
	})
	result = verifyFixture(t, openWithOptions)
	expectNoRule(t, result, "decision.open-without-options")
	expectNoRule(t, result, "decision.open-not-allowed")

	result = verifyFixture(t, ready(openWithOptions))
	expectRule(t, result, "decision.open-not-allowed")
}

// --- composition ---

func TestV2_Composition(t *testing.T) {
	child := "## FormatName\n\nTrims and capitalises a name.\n\n### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `name` | text | The raw name. |\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `formatted` | text | The formatted name. |\n\n### Invariants\n\n- `formatted` has no leading or trailing spaces.\n\n### Failure Modes\n\n_None._\n"
	withChild := withOverrides(greeterV2, map[string]string{
		"abilities/greet/format-name/ability.md": child,
	})
	result := verifyFixture(t, withChild)
	expectNoRule(t, result, "ability.missing-composition")

	result = verifyFixture(t, ready(withChild))
	expectRule(t, result, "ability.missing-composition")

	bad := withOverrides(withChild, map[string]string{
		"abilities/greet/ability.md": greeterV2["abilities/greet/ability.md"] +
			"\n### Composition\n\n| Step | Ability | Consumes | Produces |\n| --- | --- | --- | --- |\n| 1 | `Nope` | `name` from parent | `formatted` |\n",
	})
	result = verifyFixture(t, bad)
	expectRule(t, result, "ability.composition-unknown-child")
	expectRule(t, result, "ability.composition-omits-child")

	good := withOverrides(withChild, map[string]string{
		"abilities/greet/ability.md": greeterV2["abilities/greet/ability.md"] +
			"\n### Composition\n\n| Step | Ability | Consumes | Produces |\n| --- | --- | --- | --- |\n| 1 | `FormatName` | `name` from parent | `formatted` |\n| 2 | — | `formatted` from step 1 | `result` |\n",
	})
	result = verifyFixture(t, ready(good))
	for _, id := range []string{"ability.missing-composition", "ability.composition-unknown-child", "ability.composition-omits-child",
		"ability.composition-unknown-source", "ability.composition-unknown-produce", "ability.composition-step-order"} {
		expectNoRule(t, result, id)
	}
}

// --- success criteria ---

func TestV2_Criteria(t *testing.T) {
	unknown := withOverrides(greeterV2, map[string]string{
		"spec.md": strings.Replace(greeterV2["spec.md"], "| `Greet` | `happy-path` |", "| `Nope` | `missing` |", 1),
	})
	result := verifyFixture(t, unknown)
	expectRule(t, result, "spec.criteria-unknown-ability")
	expectRule(t, result, "spec.criteria-unserved-root")
	expectRule(t, result, "spec.criteria-unknown-scenario")

	noScenario := withOverrides(greeterV2, map[string]string{
		"spec.md": strings.Replace(greeterV2["spec.md"], "| `Greet` | `happy-path` |", "| `Greet` | — |", 1),
	})
	result = verifyFixture(t, noScenario)
	expectNoRule(t, result, "spec.criteria-missing-scenario")
	result = verifyFixture(t, ready(noScenario))
	expectRule(t, result, "spec.criteria-missing-scenario")
}

// --- formatting ---

func TestV2_CRLF_ReportedNotMisparsed(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, greeterV2)
	specPath := filepath.Join(dir, "spec.md")
	data, _ := os.ReadFile(specPath)
	crlf := strings.ReplaceAll(string(data), "\n", "\r\n")
	if err := os.WriteFile(specPath, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectRule(t, result, "format.crlf")
	expectNoRule(t, result, "spec.missing-purpose")
	expectNoRule(t, result, "spec.section-order")
}

func TestV2_TablePaddingWarning(t *testing.T) {
	dir := t.TempDir()
	for rel, content := range greeterV2 {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil { // unpadded
			t.Fatal(err)
		}
	}
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectRule(t, result, "format.table-padding")
	if !result.Passed {
		t.Errorf("padding is a warning; expected passed, got %v", ruleIDs(result))
	}
}

// --- naming, links, types ---

func TestV2_FolderNameMismatch(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"scenarios/happypath/scenario.md": greeterV2["scenarios/happy-path/scenario.md"],
	})
	delete(files, "scenarios/happy-path/scenario.md")
	result := verifyFixture(t, files)
	expectRule(t, result, "folder.name-mismatch")
}

func TestV2_BrokenLinkAndUndefinedType(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": strings.Replace(greeterV2["abilities/greet/ability.md"],
			"#greetingresult", "#nope", 1),
		"concepts/greeting/concept.md": strings.Replace(greeterV2["concepts/greeting/concept.md"],
			"| `message` | text |", "| `message` | integer |", 1),
	})
	result := verifyFixture(t, files)
	expectRule(t, result, "link.broken")
	expectRule(t, result, "type.undefined")
}

// --- delegation ---

func TestV2_DelegatedAbility(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "greeter"), greeterV2)

	delegatedSpec := withOverrides(greeterV2, map[string]string{
		"spec.md":                                strings.Replace(greeterV2["spec.md"], "## Greeter", "## Front Desk", 1),
		"abilities/greet/ability.md":             "## Greet\n\nGreets a visitor by delegating to the greeter spec.\n\n**Spec:** ../../../greeter\n**Version:** 0.1.0\n",
		"abilities/greet/format-name/ability.md": "",
	})
	delete(delegatedSpec, "abilities/greet/format-name/ability.md")
	writeFixture(t, filepath.Join(root, "front-desk"), delegatedSpec)

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: filepath.Join(root, "front-desk")}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, id := range []string{"ability.missing-inputs", "ability.missing-outputs", "ability.missing-invariants", "ability.missing-failure-modes",
		"ability.delegated-spec-not-found", "ability.delegated-version-mismatch", "ability.delegated-multiple-roots"} {
		expectNoRule(t, result, id)
	}

	broken := withOverrides(delegatedSpec, map[string]string{
		"abilities/greet/ability.md": "## Greet\n\nGreets a visitor by delegating to the greeter spec.\n\n**Spec:** ../../../greeter\n**Version:** 9.9.9\n\n### Inputs\n\n_None._\n",
	})
	brokenDir := filepath.Join(root, "front-desk-broken")
	writeFixture(t, brokenDir, broken)
	result, err = collect_violations.CollectViolations(types.SpecTarget{Path: brokenDir}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectRule(t, result, "ability.delegated-extra-sections")
	expectRule(t, result, "ability.delegated-version-mismatch")
}
