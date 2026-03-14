package integration_test

import (
	"os"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/export"
	import_ "github.com/smithyai/aasdd-cli/internal/pipeline/import"
	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScenario_ExportsImportsAndVerifies(t *testing.T) {
	// Scaffold a conformant spec to use as input.
	srcDir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: srcDir}, "v1", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	// Export to a temp file.
	tmpFile, err := os.CreateTemp(t.TempDir(), "export-*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.Close()
	snapshotPath := tmpFile.Name()

	exportResult, err := export.Export(srcDir, snapshotPath, nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if exportResult.FileCount == 0 {
		t.Fatal("export produced zero files")
	}

	// Import into a new directory.
	dstDir := t.TempDir()
	importResult, err := import_.Import(snapshotPath, dstDir)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if importResult.FileCount != exportResult.FileCount {
		t.Errorf("file count mismatch: exported %d, imported %d",
			exportResult.FileCount, importResult.FileCount)
	}

	// Verify the imported directory is conformant.
	vResult, err := verify.Verify(types.SpecTarget{Path: dstDir}, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !vResult.Passed {
		t.Errorf("imported spec not conformant: %v", vResult.Violations)
	}

	// TODO: Extend this test with Diff assertions once Diff is implemented,
	// completing the exports-imports-verifies-and-diffs scenario.
}
