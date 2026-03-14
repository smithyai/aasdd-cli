package integration_test

import (
	"os"
	"path/filepath"
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

	// Every spec file from the original should exist at the same relative path.
	specFiles := map[string]bool{
		"spec.md": true, "ability.md": true, "concept.md": true,
		"scenario.md": true, "decision.md": true,
	}
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !specFiles[info.Name()] {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		dst := filepath.Join(dstDir, rel)
		if _, statErr := os.Stat(dst); os.IsNotExist(statErr) {
			t.Errorf("file missing after import: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
