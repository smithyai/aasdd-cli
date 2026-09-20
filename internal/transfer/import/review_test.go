package import__test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/format"
)

// Tests for the findings of the pre-push review: constructs the v2 fixture did
// not cover must also round-trip byte for byte.

func writeFiles(t *testing.T, dir string, files map[string]string) {
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

func assertByteIdentical(t *testing.T, files map[string]string) {
	t.Helper()
	srcDir := t.TempDir()
	writeFiles(t, srcDir, files)
	_, dstDir := roundTrip(t, srcDir)
	for rel := range files {
		original, err := os.ReadFile(filepath.Join(srcDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		reconstructed, err := os.ReadFile(filepath.Join(dstDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("reconstructed file missing: %s", rel)
			continue
		}
		if string(original) != string(reconstructed) {
			t.Errorf("%s differs after round-trip:\n--- original ---\n%s\n--- reconstructed ---\n%s", rel, original, reconstructed)
		}
	}
}

func TestReview_StateMachineCustomSectionsAndPlaceholdersRoundTrip(t *testing.T) {
	assertByteIdentical(t, map[string]string{
		"spec.md":          "## Draft\n\n**AASDD:** v2\n**Version:** 0.1.0\n\nA draft.\n\n### Purpose\n\n_Pending._\n\n### Non-Goals\n\n_None._\n\n### Success Criteria\n\n_Pending._\n\n### Invariants\n\n_Pending._\n",
		"state-machine.md": "## State Machine\n\nCoordinates a draft.\n\n```mermaid\nstateDiagram-v2\n    [*] --> Working\n```\n\n### Orchestrator\n\n_Pending._\n\n### States\n\n_Pending._\n\n### Transitions\n\n_Pending._\n\n### Notes\n\nA custom section on the state machine.\n",
	})
	assertByteIdentical(t, map[string]string{
		"spec.md":          "## Draft\n\n**AASDD:** v2\n**Version:** 0.1.0\n\nA draft.\n\n### Purpose\n\n_Pending._\n\n### Non-Goals\n\n_None._\n\n### Success Criteria\n\n_Pending._\n\n### Invariants\n\n_Pending._\n",
		"state-machine.md": "## State Machine\n\nCoordinates.\n\n```mermaid\nstateDiagram-v2\n    [*] --> Working\n```\n\n### Orchestrator\n\n`Work` owns it.\n\n### States\n\n| State | Ability | Description |\n| --- | --- | --- |\n| `Working` | `Work` | Works. |\n\n### Transitions\n\n| From | To | Trigger | Data Passed Forward |\n| --- | --- | --- | --- |\n| `Working` | `Working` | `Again` — again | — |\n\n### Transition Rules\n\n_Pending._\n\n### Exceptional Flows\n\n_Pending._\n\n### References\n\nSee elsewhere.\n",
	})
}

func TestReview_V1SpecWithV2NamedCustomSectionsIsPreserved(t *testing.T) {
	assertByteIdentical(t, map[string]string{
		"spec.md":                       "## Legacy\n\n**AASDD:** v1\n**Version:** 1.0.0\n\nA v1 spec.\n\n### Invariants\n\n- Something holds.\n\n### Failure Modes\n\nProse, not a table, as v1 allowed for a custom section.\n\n### Purpose\n\nA custom section that happens to share a v2 name.\n",
		"abilities/do/ability.md":       "## Do\n\nDoes it.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `done` | boolean | Whether it was done. |\n\n### Invariants\n\n- `done` is `true`.\n\n### Failure Modes\n\n_None._\n\n### Composition\n\nProse under a heading that v2 recognizes and v1 does not.\n",
		"decisions/choice/decision.md":  "## Choice\n\n### Context\n\n`Do` needs a choice.\n\n### Requirement\n\nA choice.\n\n### Decision\n\nMade.\n\n### Options\n\nProse options, preserved in place for a v1 spec.\n",
		"scenarios/does-it/scenario.md": "## DoesIt\n\nIt does it.\n\n> `Do`\n\n- `done` is `true`\n\n### Example\n\nAn example.\n\n### Notes\n\nA custom section.\n",
	})
}

// A v2 section whose body is neither its expected form nor a placeholder is
// not conformant, so byte identity is not promised; its content must survive.
func TestReview_MalformedV2OptionalSectionsArePreserved(t *testing.T) {
	files := map[string]string{
		"spec.md":                 "## Odd\n\n**AASDD:** v2\n**Version:** 0.1.0\n\nAn odd draft.\n\n### Purpose\n\nFor testing.\n\n### Non-Goals\n\nNot a bullet list.\n\n### Success Criteria\n\nNot a table either.\n\n### Invariants\n\n- Something.\n",
		"abilities/do/ability.md": "## Do\n\nDoes it.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `done` | boolean | Whether it was done. |\n\n### Invariants\n\n- `done` is `true`.\n\n### Failure Modes\n\n_None._\n\n### Composition\n\nProse where a table belongs.\n",
	}
	srcDir := t.TempDir()
	writeFiles(t, srcDir, files)
	_, dstDir := roundTrip(t, srcDir)
	for rel, want := range map[string][]string{
		"spec.md":                 {"### Non-Goals\n\nNot a bullet list.", "### Success Criteria\n\nNot a table either."},
		"abilities/do/ability.md": {"### Composition\n\nProse where a table belongs."},
	} {
		data, err := os.ReadFile(filepath.Join(dstDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reconstructed file missing: %s", rel)
		}
		for _, w := range want {
			if !strings.Contains(string(data), w) {
				t.Errorf("%s lost %q after round-trip:\n%s", rel, w, data)
			}
		}
	}
}

func TestReview_ConceptCustomSectionRoundTrips(t *testing.T) {
	assertByteIdentical(t, map[string]string{
		"spec.md":                      "## Tiny\n\n**AASDD:** v2\n**Version:** 0.1.0\n\nTiny.\n\n### Purpose\n\nFor testing.\n\n### Non-Goals\n\n_None._\n\n### Success Criteria\n\n_Pending._\n\n### Invariants\n\n_Pending._\n",
		"concepts/greeting/concept.md": "## Greeting domain\n\nTypes for greetings.\n\n### GreetingResult\n\nThe result.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `message` | text | The message. |\n\n### Notes\n\nThis domain is deliberately tiny.\n",
	})
}
