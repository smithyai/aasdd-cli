package integration_test

import (
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestVerify_ConformantSpec(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Status:** Draft\n**Summary:** A test spec.\n")
	writeFile(t, filepath.Join(dir, "abilities", "do-thing", "ability.md"), "## DoThing\n\n**Purpose:** Does a thing.\n\n### Inputs\n\n### Outputs\n")
	writeFile(t, filepath.Join(dir, "concepts", "thing", "concept.md"), "## Thing domain\n\n### Thing\n\nA thing.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected passed=true, violations: %v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got %d", len(result.Violations))
	}
}

func TestVerify_WarningsOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n\n**Status:** Draft\n**Summary:** A test spec.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected passed=true when all violations are warnings")
	}
	for _, v := range result.Violations {
		if v.Severity == types.SeverityError {
			t.Errorf("expected only warnings, got error: %s", v.Rule)
		}
	}
}

func TestVerify_ErrorViolations(t *testing.T) {
	dir := t.TempDir()
	result, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected passed=false")
	}
	hasError := false
	for _, v := range result.Violations {
		if v.Severity == types.SeverityError {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected at least one error-severity violation")
	}
}

func TestVerify_TargetNotFound(t *testing.T) {
	_, err := verify.Verify(types.SpecTarget{Path: "/nonexistent-xyz-integration"}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
