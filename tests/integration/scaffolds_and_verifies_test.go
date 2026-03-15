package integration_test

import (
	"os"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/analysis/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScenario_ScaffoldsAndVerifies(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("expected files_created to be non-empty")
	}
	for _, f := range result.FilesCreated {
		if _, statErr := os.Stat(f); statErr != nil {
			t.Errorf("file %q does not exist: %v", f, statErr)
		}
	}
	vResult, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !vResult.Passed {
		t.Errorf("scaffolded spec not conformant: %v", vResult.Violations)
	}
}
