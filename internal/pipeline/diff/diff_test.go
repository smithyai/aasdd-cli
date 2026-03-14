package diff_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/diff"
	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
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
