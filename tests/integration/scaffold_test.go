package integration_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScaffold_HappyPath(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("expected files_created to be non-empty")
	}
	for _, f := range result.FilesCreated {
		if _, statErr := os.Stat(f); statErr != nil {
			t.Errorf("file %q does not exist: %v", f, statErr)
		}
	}
	vResult, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !vResult.Passed {
		t.Errorf("scaffolded spec not conformant: %v", vResult.Violations)
	}
}

func TestScaffold_TargetNotEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var notEmpty *scaffold.TargetNotEmpty
	if !errors.As(err, &notEmpty) {
		t.Fatalf("expected TargetNotEmpty, got %T: %v", err, err)
	}
}

func TestScaffold_Example_HappyPath(t *testing.T) {
	dir := t.TempDir()
	result, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FilesCreated) == 0 {
		t.Error("expected files_created to be non-empty")
	}
	for _, f := range result.FilesCreated {
		if _, statErr := os.Stat(f); statErr != nil {
			t.Errorf("file %q does not exist: %v", f, statErr)
		}
	}
	vResult, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !vResult.Passed {
		t.Errorf("example spec not conformant: %v", vResult.Violations)
	}
}
