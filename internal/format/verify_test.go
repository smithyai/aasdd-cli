package format_test

import (
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
