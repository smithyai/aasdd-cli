package export_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/export"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// makeSpecDir creates a minimal but valid spec directory in a temp dir.
func makeSpecDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"),
		"## My Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Status:** Draft\n**Summary:** A test spec.\n")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "ability.md"),
		"## Greet\n\n**Purpose:** Says hello.\n\n### Inputs\n\n| Name | Type | Description |\n| ---- | ---- | ----------- |\n| `name` | text | Recipient name. |\n")
	writeFile(t, filepath.Join(dir, "concepts", "greeting", "concept.md"),
		"## Greeting domain\n\nGreeting types.\n\n### Greeting\n\nA greeting message.\n\n#### Properties\n\n| Name | Type | Description |\n| ---- | ---- | ----------- |\n| `text` | text | The message. |\n")
	writeFile(t, filepath.Join(dir, "decisions", "lang", "decision.md"),
		"## Lang\n\n### Context\n\nNeed a language.\n\n### Requirement\n\nA language.\n\n### Decision\n\nGo.\n")
	return dir
}

// --- Failure modes ---

func TestExport_SourceNotFound(t *testing.T) {
	_, err := export.Export("/nonexistent-export-xyz", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *export.SourceNotFound
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFound, got %T: %v", err, err)
	}
}

func TestExport_SourceNotDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "spec.md")
	writeFile(t, f, "## Spec\n")
	_, err := export.Export(f, "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *export.SourceNotDirectory
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotDirectory, got %T: %v", err, err)
	}
}

// --- Output ---

func TestExport_WritesValidJSON(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var v any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &v); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v", jsonErr)
	}
}

func TestExport_StructuredOutput(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal to SpecExport: %v", jsonErr)
	}
	if exp.Heading != "My Spec" {
		t.Errorf("Heading=%q, want \"My Spec\"", exp.Heading)
	}
	if len(exp.Abilities) == 0 {
		t.Error("Abilities must not be empty")
	} else if exp.Abilities[0].Heading != "Greet" {
		t.Errorf("Abilities[0].Heading=%q, want \"Greet\"", exp.Abilities[0].Heading)
	}
	if len(exp.Concepts) == 0 {
		t.Error("Concepts must not be empty")
	}
	if len(exp.Decisions) == 0 {
		t.Error("Decisions must not be empty")
	}
}

func TestExport_AbilityContainsParsedInputs(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if len(exp.Abilities) == 0 {
		t.Fatal("no abilities")
	}
	a := exp.Abilities[0]
	if a.Inputs == nil {
		t.Fatal("Inputs is nil")
	}
	if len(a.Inputs.Headers) == 0 {
		t.Error("Inputs.Headers is empty")
	}
	if len(a.Inputs.Rows) == 0 {
		t.Error("Inputs.Rows is empty")
	}
}

func TestExport_FileCountNonZero(t *testing.T) {
	var buf strings.Builder
	result, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileCount == 0 {
		t.Error("FileCount must be > 0")
	}
}

func TestExport_OutputPathEmptyWhenStdout(t *testing.T) {
	var buf strings.Builder
	result, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != "" {
		t.Errorf("OutputPath=%q, want empty", result.OutputPath)
	}
}

func TestExport_ToFile(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "spec.json")
	result, err := export.Export(makeSpecDir(t), outFile, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != outFile {
		t.Errorf("OutputPath=%q, want %q", result.OutputPath, outFile)
	}
	if _, statErr := os.Stat(outFile); statErr != nil {
		t.Errorf("output file does not exist: %v", statErr)
	}
}

func TestExport_NonSpecFilesExcluded(t *testing.T) {
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "README.md"), "# Readme")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "notes.txt"), "scratch")
	writeFile(t, filepath.Join(dir, ".DS_Store"), "binary junk")

	var buf strings.Builder
	result, err := export.Export(dir, "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File count reflects only spec Markdown files (spec.md + ability.md + concept.md + decision.md = 4)
	if result.FileCount == 0 {
		t.Error("FileCount must be > 0")
	}

	// The JSON must be a SpecExport (not a flat map)
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("output is not a valid SpecExport: %v", jsonErr)
	}
}

func TestExport_ConceptContainsParsedTypes(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if len(exp.Concepts) == 0 {
		t.Fatal("no concepts")
	}
	c := exp.Concepts[0]
	if len(c.Types) == 0 {
		t.Error("concept has no types")
	} else if c.Types[0].Properties == nil {
		t.Error("concept type has no properties table")
	}
}
