package integration_test

import (
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/diff"
	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScenario_DiffsTwoScaffolds(t *testing.T) {
	// Scaffold a minimal spec (no example).
	minimalDir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: minimalDir}, "v1", false); err != nil {
		t.Fatalf("scaffold minimal: %v", err)
	}

	// Scaffold an example spec.
	exampleDir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: exampleDir}, "v1", true); err != nil {
		t.Fatalf("scaffold example: %v", err)
	}

	// Diff minimal (left) vs example (right).
	result, err := diff.Diff(types.SpecTarget{Path: minimalDir}, types.SpecTarget{Path: exampleDir})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	if !result.Changed {
		t.Fatal("expected differences between minimal and example scaffolds")
	}

	// The scenario requires at least one Added entry per construct type.
	constructs := map[string]bool{
		"ability":  false,
		"concept":  false,
		"decision": false,
		"scenario": false,
	}
	for _, entry := range result.Entries {
		if entry.Kind == types.DiffAdded {
			if _, tracked := constructs[entry.Construct]; tracked {
				constructs[entry.Construct] = true
			}
		}
	}
	for construct, found := range constructs {
		if !found {
			t.Errorf("expected at least one Added entry for %q", construct)
		}
	}
}
