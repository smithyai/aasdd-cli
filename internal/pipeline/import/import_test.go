package import__test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	import_ "github.com/smithyai/aasdd-cli/internal/pipeline/import"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// makeSnapshot creates a minimal SpecExport JSON file in a temp dir and returns its path.
func makeSnapshot(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{
			Heading:      "Test Spec",
			AASDDVersion: "v1",
			Version:      "0.1.0",
			Status:       "Draft",
			Summary:      "A test spec.",
		},
		Abilities: []types.ParsedAbility{
			{
				Path:    "abilities/greet/ability.md",
				Heading: "Greet",
				Purpose: "Says hello.",
				Inputs: &types.Table{
					Headers: []string{"Name", "Type", "Description"},
					Rows: []map[string]string{
						{"Name": "`name`", "Type": "text", "Description": "Recipient name."},
					},
				},
			},
		},
		Scenarios: []types.ParsedScenario{
			{
				Path:        "scenarios/hello/scenario.md",
				Heading:     "Hello",
				Description: "A greeting is produced.",
				Trace:       "Greet",
				Assertions:  []string{"result is non-empty"},
			},
		},
		Concepts: []types.ParsedConcept{
			{
				Path:    "concepts/greeting/concept.md",
				Heading: "Greeting domain",
			},
		},
		Decisions: []types.ParsedDecision{
			{
				Path:        "decisions/lang/decision.md",
				Heading:     "Lang",
				Context:     "Need a language.",
				Requirement: "A language.",
				Decision:    "Go.",
			},
		},
	}
	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	tmp := t.TempDir()
	snapshotPath := filepath.Join(tmp, "snap.json")
	if err := os.WriteFile(snapshotPath, data, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	return snapshotPath
}

// --- Failure modes ---

func TestImport_SourceNotFound(t *testing.T) {
	_, err := import_.Import("/nonexistent-import-xyz.json", t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.SourceNotFound
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFound, got %T: %v", err, err)
	}
}

func TestImport_SourceNotFile(t *testing.T) {
	_, err := import_.Import(t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.SourceNotFile
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFile, got %T: %v", err, err)
	}
}

func TestImport_OutputNotEmpty_NonEmptyDir(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "existing.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(makeSnapshot(t), out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestImport_OutputNotEmpty_ExistingFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "output")
	if err := os.WriteFile(outPath, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(makeSnapshot(t), outPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestImport_ParseError_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(badFile, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(badFile, t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.ParseError
	if !errors.As(err, &e) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
}

// --- Success ---

func TestImport_FilesExistOnDisk(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPaths := []string{
		"spec.md",
		"abilities/greet/ability.md",
		"scenarios/hello/scenario.md",
		"concepts/greeting/concept.md",
		"decisions/lang/decision.md",
	}
	for _, rel := range expectedPaths {
		full := filepath.Join(out, filepath.FromSlash(rel))
		if _, statErr := os.Stat(full); os.IsNotExist(statErr) {
			t.Errorf("expected file missing: %s", rel)
		}
	}
}

func TestImport_AbilityContentPreserved(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(filepath.Join(out, "abilities", "greet", "ability.md"))
	if readErr != nil {
		t.Fatalf("read ability.md: %v", readErr)
	}
	content := string(data)
	if !contains(content, "Greet") {
		t.Error("ability.md should contain heading \"Greet\"")
	}
	if !contains(content, "Says hello.") {
		t.Error("ability.md should contain purpose text")
	}
}

func TestImport_FileCountInResult(t *testing.T) {
	out := t.TempDir()
	result, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// spec.md + ability.md + scenario.md + concept.md + decision.md = 5
	if result.FileCount != 5 {
		t.Errorf("FileCount=%d, want 5", result.FileCount)
	}
}

func TestImport_OutputPathInResult(t *testing.T) {
	out := t.TempDir()
	result, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != out {
		t.Errorf("OutputPath=%q, want %q", result.OutputPath, out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
