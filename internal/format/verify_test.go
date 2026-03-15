package format_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- ViolationLabel ---

func TestViolationLabel_Warning(t *testing.T) {
	if got := format.ViolationLabel(types.SeverityWarning); got != "warning" {
		t.Errorf("expected %q, got %q", "warning", got)
	}
}

func TestViolationLabel_Error(t *testing.T) {
	if got := format.ViolationLabel(types.SeverityError); got != "error" {
		t.Errorf("expected %q, got %q", "error", got)
	}
}

// --- WriteViolations counts ---

func makeResult(violations ...types.Violation) types.VerificationResult {
	passed := true
	for _, v := range violations {
		if v.Severity == types.SeverityError {
			passed = false
			break
		}
	}
	return types.VerificationResult{Violations: violations, Passed: passed}
}

func TestWriteViolations_EmptyResult(t *testing.T) {
	var sb strings.Builder
	errCount, warnCount := format.WriteViolations(&sb, makeResult(), false)
	if errCount != 0 || warnCount != 0 {
		t.Errorf("expected 0/0, got errCount=%d warnCount=%d", errCount, warnCount)
	}
	if sb.Len() != 0 {
		t.Errorf("expected no output for empty result, got: %q", sb.String())
	}
}

func TestWriteViolations_CountsErrors(t *testing.T) {
	v1 := types.Violation{Rule: "r1", Severity: types.SeverityError, Path: ".", Message: "m1"}
	v2 := types.Violation{Rule: "r2", Severity: types.SeverityError, Path: ".", Message: "m2"}
	var sb strings.Builder
	errCount, warnCount := format.WriteViolations(&sb, makeResult(v1, v2), false)
	if errCount != 2 {
		t.Errorf("expected errCount=2, got %d", errCount)
	}
	if warnCount != 0 {
		t.Errorf("expected warnCount=0, got %d", warnCount)
	}
}

func TestWriteViolations_CountsWarnings(t *testing.T) {
	v1 := types.Violation{Rule: "r1", Severity: types.SeverityWarning, Path: ".", Message: "m1"}
	v2 := types.Violation{Rule: "r2", Severity: types.SeverityWarning, Path: ".", Message: "m2"}
	var sb strings.Builder
	errCount, warnCount := format.WriteViolations(&sb, makeResult(v1, v2), false)
	if warnCount != 2 {
		t.Errorf("expected warnCount=2, got %d", warnCount)
	}
	if errCount != 0 {
		t.Errorf("expected errCount=0, got %d", errCount)
	}
}

func TestWriteViolations_MixedCounts(t *testing.T) {
	vs := []types.Violation{
		{Rule: "r1", Severity: types.SeverityError, Path: ".", Message: "m"},
		{Rule: "r2", Severity: types.SeverityWarning, Path: ".", Message: "m"},
		{Rule: "r3", Severity: types.SeverityWarning, Path: ".", Message: "m"},
	}
	var sb strings.Builder
	errCount, warnCount := format.WriteViolations(&sb, makeResult(vs...), false)
	if errCount != 1 || warnCount != 2 {
		t.Errorf("expected 1 error + 2 warnings, got errCount=%d warnCount=%d", errCount, warnCount)
	}
}

// --- WriteViolations output format ---

func TestWriteViolations_ContainsRuleID(t *testing.T) {
	v := types.Violation{Rule: "spec.missing-version", Severity: types.SeverityWarning, Path: "spec.md", Message: "missing"}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), false)
	if !strings.Contains(sb.String(), "spec.missing-version") {
		t.Errorf("output missing rule ID, got: %q", sb.String())
	}
}

func TestWriteViolations_ContainsPath(t *testing.T) {
	v := types.Violation{Rule: "r", Severity: types.SeverityError, Path: "abilities/foo/ability.md", Message: "m"}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), false)
	if !strings.Contains(sb.String(), "abilities/foo/ability.md") {
		t.Errorf("output missing path, got: %q", sb.String())
	}
}

func TestWriteViolations_ContainsMessage(t *testing.T) {
	v := types.Violation{Rule: "r", Severity: types.SeverityError, Path: ".", Message: "required field missing"}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), false)
	if !strings.Contains(sb.String(), "required field missing") {
		t.Errorf("output missing message, got: %q", sb.String())
	}
}

// --- WriteViolations verbose ---

func TestWriteViolations_Verbose_IncludesDescription(t *testing.T) {
	v := types.Violation{Rule: "r", Severity: types.SeverityError, Path: ".", Message: "m", Description: "Checks that the thing is present."}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), true)
	if !strings.Contains(sb.String(), "Checks that the thing is present.") {
		t.Errorf("verbose output missing description, got: %q", sb.String())
	}
}

func TestWriteViolations_NonVerbose_OmitsDescription(t *testing.T) {
	v := types.Violation{Rule: "r", Severity: types.SeverityError, Path: ".", Message: "m", Description: "Should not appear."}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), false)
	if strings.Contains(sb.String(), "Should not appear.") {
		t.Errorf("non-verbose output must not include description, got: %q", sb.String())
	}
}

func TestWriteViolations_Verbose_EmptyDescription_NoExtraLine(t *testing.T) {
	v := types.Violation{Rule: "r", Severity: types.SeverityError, Path: ".", Message: "m", Description: ""}
	var sb strings.Builder
	format.WriteViolations(&sb, makeResult(v), true)
	// Exactly two lines: the violation line and the message line (no third line for description).
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines for empty description, got %d: %q", len(lines), sb.String())
	}
}

// --- WriteFileTree ---

func makeVerificationTarget(t *testing.T) (abilityAbsPath string, result types.VerificationResult) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte("spec"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "abilities", "greet"), 0o755); err != nil {
		t.Fatal(err)
	}
	absAbility := filepath.Join(dir, "abilities", "greet", "ability.md")
	if err := os.WriteFile(absAbility, []byte("ability"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := types.VerificationResult{
		Target:        types.SpecTarget{Path: dir},
		SpecFileNames: []string{"spec.md", "ability.md"},
	}
	return absAbility, res
}

func TestWriteFileTree_ListsFiles(t *testing.T) {
	_, result := makeVerificationTarget(t)
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	output := sb.String()
	if !strings.Contains(output, "spec.md") {
		t.Errorf("output missing spec.md: %q", output)
	}
	if !strings.Contains(output, "ability.md") {
		t.Errorf("output missing ability.md: %q", output)
	}
}

func TestWriteFileTree_OKForSpecFileNoViolations(t *testing.T) {
	_, result := makeVerificationTarget(t)
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	if !strings.Contains(sb.String(), "ok  ") {
		t.Errorf("expected 'ok  ' prefix for unviolated spec file, got: %q", sb.String())
	}
}

func TestWriteFileTree_ErrForViolatedFile(t *testing.T) {
	abilityPath, result := makeVerificationTarget(t)
	result.Violations = []types.Violation{
		{Rule: "r", Severity: types.SeverityError, Path: abilityPath, Message: "missing heading"},
	}
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	if !strings.Contains(sb.String(), "err ") {
		t.Errorf("expected 'err ' prefix for error-violated file, got: %q", sb.String())
	}
}

func TestWriteFileTree_WarnForWarnedFile(t *testing.T) {
	abilityPath, result := makeVerificationTarget(t)
	result.Violations = []types.Violation{
		{Rule: "r", Severity: types.SeverityWarning, Path: abilityPath, Message: "style warning"},
	}
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	if !strings.Contains(sb.String(), "warn") {
		t.Errorf("expected 'warn' prefix for warning-violated file, got: %q", sb.String())
	}
}

func TestWriteFileTree_InfoForUnknownFile(t *testing.T) {
	_, result := makeVerificationTarget(t)
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	if !strings.Contains(sb.String(), "info") {
		t.Errorf("expected 'info' prefix for non-spec file other.txt, got: %q", sb.String())
	}
}

func TestWriteFileTree_DirectoryListed(t *testing.T) {
	_, result := makeVerificationTarget(t)
	var sb strings.Builder
	format.WriteFileTree(&sb, result)
	if !strings.Contains(sb.String(), "abilities/") {
		t.Errorf("expected directory entry in tree output, got: %q", sb.String())
	}
}
