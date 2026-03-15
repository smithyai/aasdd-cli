package verify_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify"
	"github.com/smithyai/aasdd-cli/internal/analysis/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/types"
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

// --- Failure modes ---

func TestVerify_TargetNotFound(t *testing.T) {
	_, err := verify.Verify(types.SpecTarget{Path: "/nonexistent-verify-unit-xyz"}, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *collect_violations.TargetNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotFound, got %T: %v", err, err)
	}
}

func TestVerify_TargetIsFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "not-a-dir.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := verify.Verify(types.SpecTarget{Path: f}, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var isFile *collect_violations.TargetIsFile
	if !errors.As(err, &isFile) {
		t.Fatalf("expected TargetIsFile, got %T: %v", err, err)
	}
}

// --- Result fields ---

func TestVerify_ResultAASDDVersionPopulated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** Test.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AASDDVersion == "" {
		t.Error("result.aasdd_version must not be empty")
	}
}

func TestVerify_ResultRuleCountPopulated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** Test.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RuleCount == 0 {
		t.Error("result.rule_count must be non-zero")
	}
}

func TestVerify_ResultTargetMatches(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** Test.\n")
	target := types.SpecTarget{Path: dir}
	result, err := verify.Verify(target, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Target.Path != target.Path {
		t.Errorf("result.target.path mismatch: got %q, want %q", result.Target.Path, target.Path)
	}
}

// --- AASDD version resolution ---

func TestVerify_UnknownAASDDVersionInSpec_FallsBackToLatest(t *testing.T) {
	dir := t.TempDir()
	// spec.md declares an unknown AASDD version — Verify should silently fall back.
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v999\n**Version:** 0.1.0\n**Summary:** Test.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if result.AASDDVersion == "" {
		t.Error("result.aasdd_version must not be empty after fallback")
	}
}

func TestVerify_MissingAASDDLabelInSpec_UsesLatest(t *testing.T) {
	dir := t.TempDir()
	// spec.md has no **AASDD:** label — must fall back to latest.
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n**Summary:** Test.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AASDDVersion == "" {
		t.Error("result.aasdd_version must not be empty when AASDD label absent")
	}
}

func TestVerify_MissingSpecMd_UsesLatest(t *testing.T) {
	dir := t.TempDir()
	// No spec.md at all — readAASDDVersion should fall back gracefully.
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AASDDVersion == "" {
		t.Error("result.aasdd_version must not be empty when spec.md missing")
	}
}

// --- Invariants ---

func TestVerify_PassedIffNoErrorViolations_Conformant(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** Test.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hasError := false
	for _, v := range result.Violations {
		if v.Severity == types.SeverityError {
			hasError = true
		}
	}
	if result.Passed == hasError {
		t.Errorf("passed=%v but hasError=%v — invariant violated", result.Passed, hasError)
	}
}

func TestVerify_AllViolationsReferenceExistingPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if _, statErr := os.Stat(v.Path); statErr != nil {
			t.Errorf("violation path %q does not exist: %v", v.Path, statErr)
		}
	}
}
