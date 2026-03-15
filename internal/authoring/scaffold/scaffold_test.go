package scaffold_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Failure modes ---

func TestScaffold_TargetNotEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *scaffold.TargetNotEmpty
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotEmpty, got %T: %v", err, err)
	}
}

func TestScaffold_TargetIsFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "a-file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := scaffold.Scaffold(types.SpecTarget{Path: f}, "v1", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var isFile *scaffold.TargetIsFile
	if !errors.As(err, &isFile) {
		t.Fatalf("expected TargetIsFile, got %T: %v", err, err)
	}
}

func TestScaffold_UnknownAASDDVersion(t *testing.T) {
	dir := t.TempDir()
	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v99", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var unknown *scaffold.UnknownAASDDVersion
	if !errors.As(err, &unknown) {
		t.Fatalf("expected UnknownAASDDVersion, got %T: %v", err, err)
	}
}

// --- Invariants ---

func TestScaffold_FilesCreatedNonEmpty(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("files_created must be non-empty")
	}
}

func TestScaffold_AllCreatedFilesExistOnDisk(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range result.FilesCreated {
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("created file %q does not exist: %v", p, statErr)
		}
	}
}

// --- Idempotency ---

func TestScaffold_Idempotent(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	r1, err := scaffold.Scaffold(types.SpecTarget{Path: dir1}, "v1", false)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	r2, err := scaffold.Scaffold(types.SpecTarget{Path: dir2}, "v1", false)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if len(r1.FilesCreated) != len(r2.FilesCreated) {
		t.Errorf("idempotency: file count differs: %d vs %d", len(r1.FilesCreated), len(r2.FilesCreated))
	}

	rel := func(base, full string) string {
		r, _ := filepath.Rel(base, full)
		return r
	}
	for i := range r1.FilesCreated {
		if rel(dir1, r1.FilesCreated[i]) != rel(dir2, r2.FilesCreated[i]) {
			t.Errorf("idempotency: file[%d] relative path differs: %q vs %q",
				i, rel(dir1, r1.FilesCreated[i]), rel(dir2, r2.FilesCreated[i]))
		}
	}
}

// --- Example mode ---

func TestScaffold_Example_FilesCreatedNonEmpty(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("files_created must be non-empty")
	}
}

func TestScaffold_Example_AllCreatedFilesExistOnDisk(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range result.FilesCreated {
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("created file %q does not exist: %v", p, statErr)
		}
	}
}

// --- Default version ---

func TestScaffold_DefaultVersion_Succeeds(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "", false)
	if err != nil {
		t.Fatalf("Scaffold with empty version failed: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("files_created must be non-empty when version is empty (defaults to latest)")
	}
}

// --- WriteError ---

func TestScaffold_WriteError(t *testing.T) {
	dir := t.TempDir()
	// Make directory read-only so file creation fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err == nil {
		t.Fatal("expected WriteError, got nil")
	}
	var writeErr *scaffold.WriteError
	if !errors.As(err, &writeErr) {
		t.Fatalf("expected *scaffold.WriteError, got %T: %v", err, err)
	}
}
