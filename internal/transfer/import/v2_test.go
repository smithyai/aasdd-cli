package import__test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	import_ "github.com/smithyai/aasdd-cli/internal/transfer/import"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// v2Fixture exercises every v2 construct: vision sections, placeholders, an
// open decision with options, composition, idempotency, a delegated
// sub-ability, a scenario example, and custom sections.
var v2Fixture = map[string]string{
	"spec.md":                           "## Notes App\n\n**AASDD:** v2\n**Version:** 0.3.0\n\nCaptures and finds notes.\n\n### Purpose\n\nFor people who lose notes.\n\n### Non-Goals\n\n- Does not sync.\n\n### Success Criteria\n\n| Criterion | Abilities | Scenarios |\n| --- | --- | --- |\n| A captured note is retrievable. | `CaptureNote` | `capture-and-find` |\n| A note can be found by tag. | `CaptureNote` | — |\n\n### Invariants\n\n- A note's id never changes.\n\n### References\n\nSee the product brief.\n",
	"abilities/capture-note/ability.md": "## CaptureNote\n\nStores a note.\n\n### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `draft` | [NoteDraft](../../concepts/note/concept.md#notedraft) | The draft. |\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `note` | [Note](../../concepts/note/concept.md#note) | The stored note. |\n\n### Invariants\n\n- `note.title` equals `draft.title`.\n\n### Failure Modes\n\n_Pending._\n\n### Idempotency\n\nInvoking twice with the same `draft` stores one note.\n\n### Composition\n\n| Step | Ability | Consumes | Produces |\n| --- | --- | --- | --- |\n| 1 | `ValidateDraft` | `draft` from parent | `clean` |\n| 2 | — | `clean` from step 1 | `note` |\n",
	"abilities/capture-note/validate-draft/ability.md": "## ValidateDraft\n\nRejects empty drafts.\n\n### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `draft` | [NoteDraft](../../../concepts/note/concept.md#notedraft) | The draft. |\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `clean` | [NoteDraft](../../../concepts/note/concept.md#notedraft) | The validated draft. |\n\n### Invariants\n\n_Pending._\n\n### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n| `EmptyNote` | Title and body are empty. | Error propagated to caller. |\n",
	"abilities/capture-note/check-links/ability.md":    "## CheckLinks\n\nChecks the links in a note.\n\n**Spec:** ../../../link-checker\n**Version:** 1.0.0\n",
	"concepts/note/concept.md":                         "## Note domain\n\nNotes and drafts.\n\n### NoteDraft\n\nA draft.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `title` | text | The title. |\n\n### Note\n\nA stored note.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `id` | text | The id. |\n| `title` | text | The title. |\n",
	"decisions/note-storage/decision.md":               "## NoteStorage\n\n### Context\n\n`CaptureNote` stores notes somewhere.\n\n### Requirement\n\nDurable storage.\n\n### Options\n\n- An embedded database.\n- One file per note.\n\n### Decision\n\n_Open._\n",
	"scenarios/capture-and-find/scenario.md":           "## Capture and Find\n\nA note is captured and found again.\n\n> `CaptureNote` → `ValidateDraft`\n\n- `CaptureNote` returns the note.\n\n### Example\n\nThe user writes a note titled Standup and finds it by searching for roadmap.\n",
}

func writeV2Fixture(t *testing.T, dir string) {
	t.Helper()
	for rel, content := range v2Fixture {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(format.PadDocument(content)), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func roundTrip(t *testing.T, srcDir string) (string, string) {
	t.Helper()
	snapshot := filepath.Join(t.TempDir(), "spec.json")
	if _, err := export.Export(srcDir, snapshot, nil); err != nil {
		t.Fatalf("export: %v", err)
	}
	dstDir := filepath.Join(t.TempDir(), "out")
	if _, err := import_.Import(snapshot, dstDir); err != nil {
		t.Fatalf("import: %v", err)
	}
	return snapshot, dstDir
}

func TestV2_RoundTrip_ByteIdentical(t *testing.T) {
	srcDir := t.TempDir()
	writeV2Fixture(t, srcDir)
	_, dstDir := roundTrip(t, srcDir)

	for rel := range v2Fixture {
		original, err := os.ReadFile(filepath.Join(srcDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read original %s: %v", rel, err)
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

func TestV2_Export_RepresentsConstructs(t *testing.T) {
	srcDir := t.TempDir()
	writeV2Fixture(t, srcDir)
	snapshot, _ := roundTrip(t, srcDir)

	data, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var exp types.SpecExport
	if err := json.Unmarshal(data, &exp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if exp.Purpose == "" || len(exp.NonGoals) != 1 || exp.SuccessCriteria == nil || len(exp.SuccessCriteria.Rows) != 2 {
		t.Errorf("vision sections not represented: %+v", exp.ParsedSpecFile)
	}
	if len(exp.CustomSections) != 1 || exp.CustomSections[0].Heading != "References" {
		t.Errorf("custom section not preserved: %+v", exp.CustomSections)
	}
	if len(exp.Abilities) != 1 {
		t.Fatalf("expected 1 root ability, got %d", len(exp.Abilities))
	}
	root := exp.Abilities[0]
	if root.Composition == nil || len(root.Composition.Rows) != 2 {
		t.Errorf("composition not represented: %+v", root.Composition)
	}
	if root.Idempotency == "" {
		t.Error("idempotency not represented")
	}
	if root.Placeholders["Failure Modes"] != types.PlaceholderPending {
		t.Errorf("pending placeholder not recorded: %v", root.Placeholders)
	}
	if len(root.SubAbilities) != 2 {
		t.Fatalf("expected 2 sub-abilities, got %d", len(root.SubAbilities))
	}
	var delegated *types.ParsedAbility
	for i := range root.SubAbilities {
		if root.SubAbilities[i].Delegated() {
			delegated = &root.SubAbilities[i]
		}
	}
	if delegated == nil || delegated.Spec != "../../../link-checker" || delegated.SpecVersion != "1.0.0" {
		t.Errorf("delegated ability not represented: %+v", delegated)
	}
	if len(exp.Decisions) != 1 || !exp.Decisions[0].Open() || len(exp.Decisions[0].Options) != 2 {
		t.Errorf("open decision not represented: %+v", exp.Decisions)
	}
	if len(exp.Scenarios) != 1 || !strings.Contains(exp.Scenarios[0].Example, "Standup") {
		t.Errorf("scenario example not represented: %+v", exp.Scenarios)
	}
	if strings.Contains(string(data), "\\r") {
		t.Error("export contains carriage returns")
	}
}

func TestV2_Import_FolderNamesFromTitleCaseHeadings(t *testing.T) {
	srcDir := t.TempDir()
	writeV2Fixture(t, srcDir)
	_, dstDir := roundTrip(t, srcDir)
	if _, err := os.Stat(filepath.Join(dstDir, "scenarios", "capture-and-find", "scenario.md")); err != nil {
		t.Errorf("scenario folder not derived from Title Case heading: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "abilities", "capture-note", "check-links", "ability.md")); err != nil {
		t.Errorf("delegated sub-ability not written: %v", err)
	}
}

func TestV2_Export_NormalizesCRLF(t *testing.T) {
	srcDir := t.TempDir()
	writeV2Fixture(t, srcDir)
	specPath := filepath.Join(srcDir, "spec.md")
	data, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, []byte(strings.ReplaceAll(string(data), "\n", "\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	exp, _, err := export.LoadSpec(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	if exp.Heading != "Notes App" || exp.Purpose == "" {
		t.Errorf("CRLF spec misparsed: heading=%q purpose=%q", exp.Heading, exp.Purpose)
	}
}
