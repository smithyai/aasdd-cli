package collect_violations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// Tests for the findings of the pre-push review.

const draftStateMachine = "## State Machine\n\nCoordinates greeting.\n\n```mermaid\nstateDiagram-v2\n    [*] --> Greeting\n    Greeting --> [*]\n```\n\n### Orchestrator\n\n_Pending._\n\n### States\n\n_Pending._\n\n### Transitions\n\n_Pending._\n"

func TestReview_StateMachinePendingIsDraftOnlyAndCounted(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{"state-machine.md": draftStateMachine})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "placeholder.misplaced")
	expectNoRule(t, result, "pending.not-allowed")

	result = verifyFixture(t, ready(files))
	expectRule(t, result, "pending.not-allowed")
}

func TestReview_StateMachineCustomSectionAllowed(t *testing.T) {
	sm := "## State Machine\n\nCoordinates greeting.\n\n```mermaid\nstateDiagram-v2\n    [*] --> Greeting\n    Greeting --> [*]\n```\n\n### Orchestrator\n\n`Greet` owns the single transition.\n\n### States\n\n| State | Ability | Description |\n| --- | --- | --- |\n| `Greeting` | `Greet` | Produces the greeting. |\n\n### Transitions\n\n| From | To | Trigger | Data Passed Forward |\n| --- | --- | --- | --- |\n| `Greeting` | `Greeting` | `Repeat` — the caller greets again | — |\n\n### Notes\n\nA custom section on the state machine.\n"
	files := withOverrides(greeterV2, map[string]string{
		"state-machine.md":                 sm,
		"scenarios/happy-path/scenario.md": "## Happy Path\n\nA name produces a greeting.\n\n> `Greeting`\n\n- `result.message` contains the name.\n",
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "state-machine.section-order")
	expectNoRule(t, result, "placeholder.misplaced")
}

func TestReview_ConceptCustomSectionAfterTypesAllowed(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"concepts/greeting/concept.md": greeterV2["concepts/greeting/concept.md"] + "\n### Notes\n\nThis domain is deliberately tiny.\n",
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "concept.type-table")

	// A section without a table before a type is still a type without a table.
	files = withOverrides(greeterV2, map[string]string{
		"concepts/greeting/concept.md": strings.Replace(greeterV2["concepts/greeting/concept.md"],
			"### GreetingResult", "### Untabled\n\nNo table here.\n\n### GreetingResult", 1),
	})
	result = verifyFixture(t, files)
	expectRule(t, result, "concept.type-table")
}

func TestReview_PendingContextIsNotAMissingAbility(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"decisions/output-channel/decision.md": "## OutputChannel\n\n### Context\n\n_Pending._\n\n### Requirement\n\nA way to return the greeting.\n\n### Decision\n\n_Pending._\n",
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "decision.context-names-no-ability")
	expectNoRule(t, result, "placeholder.misplaced")
	result = verifyFixture(t, ready(files))
	expectRule(t, result, "pending.not-allowed")
}

func TestReview_TrailingConditionIsRejected(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"scenarios/happy-path/scenario.md": "## Happy Path\n\nA name produces a greeting.\n\n> `Greet` — EmptyName\n\n- `result.message` contains the name.\n",
	})
	result := verifyFixture(t, files)
	expectRule(t, result, "scenario.trace-format")
}

func TestReview_TableColumnsChecked(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": strings.Replace(greeterV2["abilities/greet/ability.md"],
			"| Failure | Condition | Effect |", "| Error | When | Then |", 1),
	})
	result := verifyFixture(t, files)
	expectRule(t, result, "ability.table-columns")

	files = withOverrides(greeterV2, map[string]string{
		"concepts/greeting/concept.md": strings.Replace(greeterV2["concepts/greeting/concept.md"],
			"| Name | Type | Description |", "| Field | Kind | Notes |", 1),
	})
	result = verifyFixture(t, files)
	expectRule(t, result, "concept.table-columns")
}

func TestReview_DuplicateSectionIsAnOrderViolation(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": greeterV2["abilities/greet/ability.md"] + "\n### Inputs\n\n_None._\n",
	})
	result := verifyFixture(t, files)
	expectRule(t, result, "ability.section-order")
}

func TestReview_ExternalTypeNamedBySpec(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": strings.Replace(greeterV2["abilities/greet/ability.md"],
			"| `name` | text | The name to greet. |",
			"| `name` | text | The name to greet. |\n| `report` | LinkReport | The report produced by the Link Checker spec. |", 1),
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "type.undefined")
}

func TestReview_FencedTablesAreNotTables(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": greeterV2["abilities/greet/ability.md"] + "\n### Notes\n\n```text\n| a | b |\n| - | - |\n| x \\| y | z |\n```\n",
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "format.table-padding")
	expectNoRule(t, result, "format.pipe-in-cell")
}

func TestReview_SpecFailureModesNoneAllowed(t *testing.T) {
	files := withOverrides(greeterV2, map[string]string{
		"spec.md": greeterV2["spec.md"] + "\n### Failure Modes\n\n_None._\n",
	})
	result := verifyFixture(t, files)
	expectNoRule(t, result, "placeholder.misplaced")
}

func TestReview_CaseMismatchedLinkIsBroken(t *testing.T) {
	dir := t.TempDir()
	files := withOverrides(greeterV2, map[string]string{
		"abilities/greet/ability.md": strings.Replace(greeterV2["abilities/greet/ability.md"],
			"../../concepts/greeting/concept.md#greetingresult", "../../Concepts/Greeting/concept.md#greetingresult", 1),
	})
	writeFixture(t, dir, files)
	if _, err := os.Stat(filepath.Join(dir, "Concepts", "Greeting", "concept.md")); err != nil {
		t.Skip("filesystem is case-sensitive; the link is simply broken there")
	}
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, v2Rules(t), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectRule(t, result, "link.broken")
}
