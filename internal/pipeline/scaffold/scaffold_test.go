package scaffold_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Failure modes ---

func TestScaffold_TargetNotEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *scaffold.TargetNotEmpty
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotEmpty, got %T: %v", err, err)
	}
}

func TestScaffold_UnknownSpecVersion(t *testing.T) {
	dir := t.TempDir()
	sv := &types.SpecVersion{Value: "v99.0.0"}

	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, sv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *scaffold.UnknownSpecVersion
	if !errors.As(err, &target) {
		t.Fatalf("expected UnknownSpecVersion, got %T: %v", err, err)
	}
	if target.Version != "v99.0.0" {
		t.Errorf("unexpected version in error: %q", target.Version)
	}
}

// --- Invariants ---

func TestScaffold_FilesCreatedNonEmpty(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("files_created must be non-empty")
	}
}

func TestScaffold_AllCreatedFilesExistOnDisk(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, nil)
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

	r1, err := scaffold.Scaffold(types.SpecTarget{Path: dir1}, nil)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	r2, err := scaffold.Scaffold(types.SpecTarget{Path: dir2}, nil)
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
