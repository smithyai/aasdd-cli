package diff_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/diff"
	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Failure mode tests ---

func TestDiff_LeftNotFound(t *testing.T) {
	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	_, err := diff.Diff(
		types.SpecTarget{Path: "/nonexistent/left"},
		types.SpecTarget{Path: right},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *diff.LeftNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected LeftNotFound, got %T: %v", err, err)
	}
}

func TestDiff_LeftNotDirectory(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	os.WriteFile(file, []byte("x"), 0o644)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	_, err := diff.Diff(
		types.SpecTarget{Path: file},
		types.SpecTarget{Path: right},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *diff.LeftNotDirectory
	if !errors.As(err, &target) {
		t.Fatalf("expected LeftNotDirectory, got %T: %v", err, err)
	}
}

func TestDiff_RightNotFound(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	_, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: "/nonexistent/right"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *diff.RightNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected RightNotFound, got %T: %v", err, err)
	}
}

func TestDiff_RightNotDirectory(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	os.WriteFile(file, []byte("x"), 0o644)

	_, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: file},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *diff.RightNotDirectory
	if !errors.As(err, &target) {
		t.Fatalf("expected RightNotDirectory, got %T: %v", err, err)
	}
}

// --- Happy path tests ---

func TestDiff_IdenticalSpecs_NoDifferences(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Errorf("expected changed=false, got true")
	}
	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d: %v", len(result.Entries), result.Entries)
	}
}

func TestDiff_MinimalVsExample_DetectsAdded(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", true)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected changed=true, got false")
	}

	// The example scaffold adds an ability, concept, decision, and scenario
	// that the minimal scaffold doesn't have.
	constructs := map[string]bool{"ability": false, "concept": false, "decision": false, "scenario": false}
	for _, entry := range result.Entries {
		if entry.Kind == types.DiffAdded {
			constructs[entry.Construct] = true
		}
	}
	for construct, found := range constructs {
		if !found {
			t.Errorf("expected Added entry for construct %q, not found", construct)
		}
	}
}

func TestDiff_ExampleVsMinimal_DetectsRemoved(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", true)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected changed=true, got false")
	}

	constructs := map[string]bool{"ability": false, "concept": false, "decision": false, "scenario": false}
	for _, entry := range result.Entries {
		if entry.Kind == types.DiffRemoved {
			constructs[entry.Construct] = true
		}
	}
	for construct, found := range constructs {
		if !found {
			t.Errorf("expected Removed entry for construct %q, not found", construct)
		}
	}
}

func TestDiff_ModifiedSpec_DetectsChanged(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	// Modify the summary in right's spec.md
	specPath := filepath.Join(right, "spec.md")
	modified := []byte("## Modified Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** A completely different summary.\n")
	os.WriteFile(specPath, modified, 0o644)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected changed=true, got false")
	}

	foundSpec := false
	for _, entry := range result.Entries {
		if entry.Kind == types.DiffChanged && entry.Construct == "spec" {
			foundSpec = true
			break
		}
	}
	if !foundSpec {
		t.Errorf("expected Changed entry for construct 'spec', not found in: %v", result.Entries)
	}
}

func TestDiff_EntriesInPathOrder(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", true)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 1; i < len(result.Entries); i++ {
		if result.Entries[i].Path < result.Entries[i-1].Path {
			t.Errorf("entries not in path order: %q came after %q",
				result.Entries[i].Path, result.Entries[i-1].Path)
		}
	}
}

func TestDiff_ResultTargetsMatch(t *testing.T) {
	left := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: left}, "v1", false)

	right := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: right}, "v1", false)

	result, err := diff.Diff(
		types.SpecTarget{Path: left},
		types.SpecTarget{Path: right},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Left.Path != left {
		t.Errorf("Left.Path = %q, want %q", result.Left.Path, left)
	}
	if result.Right.Path != right {
		t.Errorf("Right.Path = %q, want %q", result.Right.Path, right)
	}
}

// --- Per-construct change detection ---

func makeExampleSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true)
	return dir
}

func hasChangedEntry(entries []types.DiffEntry, construct string) bool {
	for _, e := range entries {
		if e.Kind == types.DiffChanged && e.Construct == construct {
			return true
		}
	}
	return false
}

func TestDiff_ChangedAbility(t *testing.T) {
	left := makeExampleSpec(t)
	right := makeExampleSpec(t)
	abilityPath := filepath.Join(right, "abilities", "greet", "ability.md")
	os.WriteFile(abilityPath,
		[]byte("## Greet\n\nProduces a completely different greeting.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n_None._\n\n### Invariants\n\n_None._\n\n### Failure Modes\n\n_None._\n"),
		0o644)

	result, err := diff.Diff(types.SpecTarget{Path: left}, types.SpecTarget{Path: right})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasChangedEntry(result.Entries, "ability") {
		t.Errorf("expected Changed ability entry, got: %v", result.Entries)
	}
}

func TestDiff_ChangedConcept(t *testing.T) {
	left := makeExampleSpec(t)
	right := makeExampleSpec(t)
	conceptPath := filepath.Join(right, "concepts", "greeting", "concept.md")
	os.WriteFile(conceptPath,
		[]byte("## Greeting domain\n\nModified intro.\n\n### GreetingResult\n\nDifferent description.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `message` | text | Changed. |\n"),
		0o644)

	result, err := diff.Diff(types.SpecTarget{Path: left}, types.SpecTarget{Path: right})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasChangedEntry(result.Entries, "concept") {
		t.Errorf("expected Changed concept entry, got: %v", result.Entries)
	}
}

func TestDiff_ChangedDecision(t *testing.T) {
	left := makeExampleSpec(t)
	right := makeExampleSpec(t)
	decisionPath := filepath.Join(right, "decisions", "output-channel", "decision.md")
	os.WriteFile(decisionPath,
		[]byte("## OutputChannel\n\n### Context\n\nDifferent context.\n\n### Requirement\n\nDifferent req.\n\n### Decision\n\nDifferent decision.\n"),
		0o644)

	result, err := diff.Diff(types.SpecTarget{Path: left}, types.SpecTarget{Path: right})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasChangedEntry(result.Entries, "decision") {
		t.Errorf("expected Changed decision entry, got: %v", result.Entries)
	}
}

func TestDiff_ChangedScenario(t *testing.T) {
	left := makeExampleSpec(t)
	right := makeExampleSpec(t)
	scenarioPath := filepath.Join(right, "scenarios", "happy-path", "scenario.md")
	os.WriteFile(scenarioPath,
		[]byte("## HappyPath\n\nModified description.\n\n> `Greet`\n\n- different assertion\n"),
		0o644)

	result, err := diff.Diff(types.SpecTarget{Path: left}, types.SpecTarget{Path: right})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasChangedEntry(result.Entries, "scenario") {
		t.Errorf("expected Changed scenario entry, got: %v", result.Entries)
	}
}
