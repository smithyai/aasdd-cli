package self_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	import_ "github.com/smithyai/aasdd-cli/internal/transfer/import"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestExport_OwnSpec_RoundTrip(t *testing.T) {
	specDir := filepath.Join(repoRoot(t), "spec")

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
	// TODO: Replace this hardcoded file list with a Diff check once Diff is implemented.
	// Diff provides a complete structural equivalence check with no maintenance burden.
	expectedFiles := []string{
		"spec.md",
		"abilities/verify/ability.md",
		"abilities/verify/collect-violations/ability.md",
		"scenarios/scaffolds-and-verifies/scenario.md",
		"scenarios/scaffolds-example-and-verifies/scenario.md",
		"scenarios/exports-imports-verifies-and-diffs/scenario.md",
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

// TestExport_OwnSpec_CanonicalFormat verifies that the CLI's own spec files are
// already in the canonical format that renderTable and the import renderers
// produce. This is a self-consistency check for the files in this repository —
// it does not assert that the tool preserves arbitrary author formatting in
// user specs. The general round-trip contract (all data preserved) is covered
// by TestExport_OwnSpec_RoundTrip and the JSON-level equivalence tests.
func TestExport_OwnSpec_CanonicalFormat(t *testing.T) {
	specDir := filepath.Join(repoRoot(t), "spec")

	// Export
	tmpFile, err := os.CreateTemp("", "aasdd-export-*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.Close()
	snapshotPath := tmpFile.Name()
	defer os.Remove(snapshotPath)

	if _, err := export.Export(specDir, snapshotPath, nil); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Import
	tmpDir, err := os.MkdirTemp("", "aasdd-import-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if _, err := import_.Import(snapshotPath, tmpDir); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Walk original spec and compare every file against the reconstructed output.
	// Any difference means a spec file has drifted from canonical format.
	err = filepath.Walk(specDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		rel, err := filepath.Rel(specDir, path)
		if err != nil {
			return err
		}

		original, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read original %s: %v", rel, err)
			return nil
		}

		reconstructed, err := os.ReadFile(filepath.Join(tmpDir, rel))
		if err != nil {
			t.Errorf("reconstructed file missing: %s", rel)
			return nil
		}

		if string(original) != string(reconstructed) {
			// Find first differing line for a useful error message
			origLines := splitLines(string(original))
			reconLines := splitLines(string(reconstructed))
			for i := 0; i < len(origLines) || i < len(reconLines); i++ {
				var ol, rl string
				if i < len(origLines) {
					ol = origLines[i]
				}
				if i < len(reconLines) {
					rl = reconLines[i]
				}
				if ol != rl {
					t.Errorf("%s: first difference at line %d\n  original:      %q\n  reconstructed: %q", rel, i+1, ol, rl)
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
