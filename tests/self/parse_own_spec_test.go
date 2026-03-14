package self_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/export"
	import_ "github.com/smithyai/aasdd-cli/internal/pipeline/import"
)

func TestExport_OwnSpec_RoundTrip(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	specDir := filepath.Join(repoRoot, "spec")

	// Export to a temp file
	tmpFile, err := os.CreateTemp("", "aasdd-export-*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.Close()
	snapshotPath := tmpFile.Name()
	defer os.Remove(snapshotPath)

	exportResult, err := export.Export(specDir, snapshotPath, nil)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if exportResult.FileCount == 0 {
		t.Fatal("export produced zero files")
	}
	if exportResult.OutputPath != snapshotPath {
		t.Errorf("OutputPath = %q, want %q", exportResult.OutputPath, snapshotPath)
	}

	// Import into a temp directory
	tmpDir, err := os.MkdirTemp("", "aasdd-import-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	importResult, err := import_.Import(snapshotPath, tmpDir)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if importResult.FileCount != exportResult.FileCount {
		t.Errorf("file count mismatch: exported %d, imported %d",
			exportResult.FileCount, importResult.FileCount)
	}

	// Verify key reconstructed files exist in the output directory
	expectedFiles := []string{
		"spec.md",
		"abilities/verify/ability.md",
		"abilities/verify/collect-violations/ability.md",
		"scenarios/scaffolds-and-verifies/scenario.md",
		"scenarios/scaffolds-example-and-verifies/scenario.md",
		"scenarios/exports-imports-and-verifies/scenario.md",
		"concepts/verification/concept.md",
		"decisions/invocation-channel/decision.md",
	}
	for _, rel := range expectedFiles {
		full := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if _, statErr := os.Stat(full); os.IsNotExist(statErr) {
			t.Errorf("expected file missing after import: %s", rel)
		}
	}
}
