package parse_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/parse"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// helpers

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func makeSpecDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "ability.md"), "## Greet\n\n**Purpose:** Says hello.\n\n### Inputs\n\n### Outputs\n")
	writeFile(t, filepath.Join(dir, "concepts", "greeting", "concept.md"), "## Greeting domain\n\n### Greeting\n\nA greeting.\n")
	return dir
}

// --- Failure modes ---

func TestParse_SourceNotFound(t *testing.T) {
	_, err := parse.Parse("/nonexistent-parse-xyz", "", false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.SourceNotFound
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFound, got %T: %v", err, err)
	}
}

func TestParse_SourceUnrecognized(t *testing.T) {
	f := filepath.Join(t.TempDir(), "data.txt")
	writeFile(t, f, "hello")
	_, err := parse.Parse(f, "", false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.SourceUnrecognized
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceUnrecognized, got %T: %v", err, err)
	}
}

func TestParse_OutputNotEmpty_NonEmptyDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"content"}`)
	dest := t.TempDir()
	writeFile(t, filepath.Join(dest, "existing.txt"), "x")
	_, err := parse.Parse(src, dest, false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestParse_OutputNotEmpty_ExistingFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"content"}`)
	dest := filepath.Join(t.TempDir(), "not-a-dir.txt")
	writeFile(t, dest, "x")
	_, err := parse.Parse(src, dest, false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestParse_ParseError_InvalidJSON(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `not json at all`)
	dest := t.TempDir()
	_, err := parse.Parse(src, dest, false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.ParseError
	if !errors.As(err, &e) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParse_ParseError_InvalidValue(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md": 42}`)
	dest := t.TempDir()
	_, err := parse.Parse(src, dest, false, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *parse.ParseError
	if !errors.As(err, &e) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
}

// --- Dir-to-JSON ---

func TestParse_DirToJSON_DirectionInResult(t *testing.T) {
	var buf strings.Builder
	result, err := parse.Parse(makeSpecDir(t), "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Direction != types.ParseDirToJSON {
		t.Errorf("direction=%q, want %q", result.Direction, types.ParseDirToJSON)
	}
}

func TestParse_DirToJSON_FileCountNonZero(t *testing.T) {
	var buf strings.Builder
	result, err := parse.Parse(makeSpecDir(t), "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileCount == 0 {
		t.Error("file_count must be > 0")
	}
}

func TestParse_DirToJSON_OutputPathEmptyWhenStdout(t *testing.T) {
	var buf strings.Builder
	result, err := parse.Parse(makeSpecDir(t), "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != "" {
		t.Errorf("output_path=%q, want empty", result.OutputPath)
	}
}

func TestParse_DirToJSON_WritesValidJSON(t *testing.T) {
	var buf strings.Builder
	_, err := parse.Parse(makeSpecDir(t), "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var v any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &v); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v", jsonErr)
	}
}

func TestParse_DirToJSON_ToFile(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "snap.json")
	result, err := parse.Parse(makeSpecDir(t), outFile, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != outFile {
		t.Errorf("output_path=%q, want %q", result.OutputPath, outFile)
	}
	if _, statErr := os.Stat(outFile); statErr != nil {
		t.Errorf("output file does not exist: %v", statErr)
	}
}

func TestParse_DirToJSON_Flat_AllValuesStrings(t *testing.T) {
	var buf strings.Builder
	_, err := parse.Parse(makeSpecDir(t), "", true, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &m); jsonErr != nil {
		t.Fatalf("not valid JSON: %v", jsonErr)
	}
	for k, v := range m {
		if _, ok := v.(string); !ok {
			t.Errorf("key %q has non-string value %T (expected flat format)", k, v)
		}
	}
}

func TestParse_DirToJSON_IgnoresHiddenFiles(t *testing.T) {
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, ".DS_Store"), "binary junk")
	writeFile(t, filepath.Join(dir, ".git", "config"), "[core]")

	var buf strings.Builder
	_, err := parse.Parse(dir, "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &m); jsonErr != nil {
		t.Fatalf("not valid JSON: %v", jsonErr)
	}
	if _, ok := m[".DS_Store"]; ok {
		t.Error(".DS_Store should be excluded from snapshot")
	}
	if _, ok := m[".git"]; ok {
		t.Error(".git directory should be excluded from snapshot")
	}
}

func TestParse_DirToJSON_IgnoresNonSpecFiles(t *testing.T) {
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "README.md"), "# Readme")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "notes.txt"), "scratch")

	var buf strings.Builder
	_, err := parse.Parse(dir, "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Parse with flat to get a simple path→string map for easy inspection.
	var bufFlat strings.Builder
	_, err = parse.Parse(dir, "", true, &bufFlat)
	if err != nil {
		t.Fatalf("unexpected error (flat): %v", err)
	}
	var m map[string]string
	if jsonErr := json.Unmarshal([]byte(bufFlat.String()), &m); jsonErr != nil {
		t.Fatalf("not valid JSON: %v", jsonErr)
	}
	for key := range m {
		base := filepath.Base(key)
		if base == "README.md" || base == "notes.txt" {
			t.Errorf("non-spec file %q should be excluded from snapshot", key)
		}
	}
}

func TestParse_DirToJSON_Nested_HasDirectoryKeys(t *testing.T) {
	var buf strings.Builder
	_, err := parse.Parse(makeSpecDir(t), "", false, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &m); jsonErr != nil {
		t.Fatalf("not valid JSON: %v", jsonErr)
	}
	// "abilities" and "concepts" should be objects in nested format.
	if _, ok := m["abilities"].(map[string]any); !ok {
		t.Error("nested JSON should have 'abilities' as an object")
	}
	if _, ok := m["concepts"].(map[string]any); !ok {
		t.Error("nested JSON should have 'concepts' as an object")
	}
}

// --- JSON-to-dir ---

func TestParse_JSONToDir_DirectionInResult(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"## My Spec\n"}`)
	dest := t.TempDir()
	result, err := parse.Parse(src, dest, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Direction != types.ParseJSONToDir {
		t.Errorf("direction=%q, want %q", result.Direction, types.ParseJSONToDir)
	}
}

func TestParse_JSONToDir_FilesExistOnDisk(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"content","abilities/greet/ability.md":"# Greet"}`)
	dest := t.TempDir()
	_, err := parse.Parse(src, dest, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rel := range []string{"spec.md", "abilities/greet/ability.md"} {
		full := filepath.Join(dest, filepath.FromSlash(rel))
		if _, statErr := os.Stat(full); statErr != nil {
			t.Errorf("expected file %q to exist: %v", rel, statErr)
		}
	}
}

func TestParse_JSONToDir_ContentPreserved(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"hello world"}`)
	dest := t.TempDir()
	_, err := parse.Parse(src, dest, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(filepath.Join(dest, "spec.md"))
	if readErr != nil {
		t.Fatalf("read spec.md: %v", readErr)
	}
	if string(data) != "hello world" {
		t.Errorf("content=%q, want %q", string(data), "hello world")
	}
}

func TestParse_JSONToDir_FileCountInResult(t *testing.T) {
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, `{"spec.md":"a","abilities/greet/ability.md":"b","concepts/x/concept.md":"c"}`)
	dest := t.TempDir()
	result, err := parse.Parse(src, dest, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileCount != 3 {
		t.Errorf("file_count=%d, want 3", result.FileCount)
	}
}

func TestParse_JSONToDir_AcceptsNestedFormat(t *testing.T) {
	nested := `{"abilities":{"greet":{"ability.md":"# Greet"}},"spec.md":"# Spec"}`
	src := filepath.Join(t.TempDir(), "snap.json")
	writeFile(t, src, nested)
	dest := t.TempDir()
	result, err := parse.Parse(src, dest, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileCount != 2 {
		t.Errorf("file_count=%d, want 2", result.FileCount)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "abilities", "greet", "ability.md")); statErr != nil {
		t.Errorf("nested path not reconstructed: %v", statErr)
	}
}

// --- Round-trip ---

func TestParse_RoundTrip_DirToJSONToDir(t *testing.T) {
	src := makeSpecDir(t)

	// Step 1: dir → JSON
	snapFile := filepath.Join(t.TempDir(), "snap.json")
	_, err := parse.Parse(src, snapFile, false, nil)
	if err != nil {
		t.Fatalf("dir-to-JSON: %v", err)
	}

	// Step 2: JSON → dir
	dest := t.TempDir()
	_, err = parse.Parse(snapFile, dest, false, nil)
	if err != nil {
		t.Fatalf("JSON-to-dir: %v", err)
	}

	// Verify every file from src exists in dest with identical content.
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		srcData, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		destData, readErr := os.ReadFile(filepath.Join(dest, rel))
		if readErr != nil {
			t.Errorf("file %q missing in round-trip dest: %v", rel, readErr)
			return nil
		}
		if string(srcData) != string(destData) {
			t.Errorf("file %q content differs after round-trip", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
