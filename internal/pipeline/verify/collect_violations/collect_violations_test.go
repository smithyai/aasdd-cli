package collect_violations_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func minimalRuleSet() types.RuleSet {
	return types.RuleSet{
		AASDDVersion: "v1",
		Rules: []types.Rule{
			{ID: "directory.missing-spec", Description: "root must have spec.md", AppliesTo: "directory", Severity: types.SeverityError},
			{ID: "spec.missing-version", Description: "spec.md must declare **Version:**", AppliesTo: "spec.md", Severity: types.SeverityWarning},
			{ID: "ability.missing-purpose", Description: "ability.md must declare **Purpose:**", AppliesTo: "ability.md", Severity: types.SeverityError},
		},
	}
}

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

func TestCollectViolations_TargetNotFound(t *testing.T) {
	_, err := collect_violations.CollectViolations(
		types.SpecTarget{Path: "/nonexistent-path-xyz"},
		minimalRuleSet(),
		false,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *collect_violations.TargetNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotFound, got %T: %v", err, err)
	}
}

func TestCollectViolations_TargetIsFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "not-a-dir.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := collect_violations.CollectViolations(
		types.SpecTarget{Path: f},
		minimalRuleSet(),
		false,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *collect_violations.TargetIsFile
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetIsFile, got %T: %v", err, err)
	}
}

func TestCollectViolations_ReadError(t *testing.T) {
	dir := t.TempDir()
	// Create spec.md so directory rule passes, then create an unreadable ability.md.
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	abilityPath := filepath.Join(dir, "abilities", "x", "ability.md")
	writeFile(t, abilityPath, "content")
	// Remove read permission to trigger ReadError.
	if err := os.Chmod(abilityPath, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(abilityPath, 0o644) })

	_, err := collect_violations.CollectViolations(
		types.SpecTarget{Path: dir},
		minimalRuleSet(),
		false,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var readErr *collect_violations.ReadError
	if !errors.As(err, &readErr) {
		t.Fatalf("expected ReadError, got %T: %v", err, err)
	}
}

// --- Invariants ---

// TestCollectViolations_PassedIffNoErrors_Conformant — no violations at all.
func TestCollectViolations_PassedIffNoErrors_Conformant(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Status:** Draft\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected passed=true for conformant spec, violations: %v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got %d", len(result.Violations))
	}
}

// TestCollectViolations_PassedIffNoErrors_WarningsOnly — warnings don't block passed.
func TestCollectViolations_PassedIffNoErrors_WarningsOnly(t *testing.T) {
	dir := t.TempDir()
	// spec.md exists but is missing **Version:** (warning-severity rule in minimalRuleSet)
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected passed=true when only warning violations exist")
	}
	if len(result.Violations) == 0 {
		t.Errorf("expected warning violations to be present")
	}
}

// TestCollectViolations_PassedIffNoErrors_ErrorViolation — error-severity violation blocks passed.
func TestCollectViolations_PassedIffNoErrors_ErrorViolation(t *testing.T) {
	dir := t.TempDir()
	// No spec.md at all — directory.missing-spec fires with SeverityError

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected passed=false when error-severity violations exist")
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one error violation")
	}
}

func TestCollectViolations_AllViolationsReferenceValidRuleIDs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")

	rs := minimalRuleSet()
	ruleIDs := make(map[string]struct{})
	for _, r := range rs.Rules {
		ruleIDs[r.ID] = struct{}{}
	}

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if _, ok := ruleIDs[v.Rule]; !ok {
			t.Errorf("violation references unknown rule ID %q", v.Rule)
		}
	}
}

func TestCollectViolations_AllViolationsReferenceExistingPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if _, statErr := os.Stat(v.Path); statErr != nil {
			t.Errorf("violation path %q does not exist: %v", v.Path, statErr)
		}
	}
}

func TestCollectViolations_EveryMatchingFileEvaluated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** v0.1.0\n")
	writeFile(t, filepath.Join(dir, "abilities", "foo", "ability.md"), "no purpose\n### Inputs\n### Outputs\n")
	writeFile(t, filepath.Join(dir, "abilities", "bar", "ability.md"), "no purpose\n### Inputs\n### Outputs\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for _, v := range result.Violations {
		if v.Rule == "ability.missing-purpose" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 ability.missing-purpose violations (one per file), got %d", count)
	}
}

// --- Idempotency ---

func TestCollectViolations_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")

	target := types.SpecTarget{Path: dir}
	rs := minimalRuleSet()

	r1, err := collect_violations.CollectViolations(target, rs, false)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	r2, err := collect_violations.CollectViolations(target, rs, false)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if r1.Passed != r2.Passed {
		t.Error("idempotency: passed differs between calls")
	}
	if len(r1.Violations) != len(r2.Violations) {
		t.Errorf("idempotency: violation count differs: %d vs %d", len(r1.Violations), len(r2.Violations))
	}
}

// --- Version format validation ---

func versionFormatRuleSet() types.RuleSet {
	return types.RuleSet{
		AASDDVersion: "v1",
		Rules: []types.Rule{
			{ID: "spec.invalid-version-format", Description: "version must be semver without v prefix", AppliesTo: "spec.md"},
		},
	}
}

func TestCollectViolations_VersionFormat_Valid(t *testing.T) {
	cases := []string{"1.0.0", "0.1.0", "2.3.14", "1.0.0-alpha.1", "1.0.0+build.1"}
	for _, ver := range cases {
		t.Run(ver, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** "+ver+"\n")
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, versionFormatRuleSet(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Passed {
				t.Errorf("expected no violation for valid version %q, got: %v", ver, result.Violations)
			}
		})
	}
}

func TestCollectViolations_VersionFormat_Invalid(t *testing.T) {
	cases := []string{"v1.0.0", "v0.1.0", "1.0", "1", "not-a-version", ""}
	for _, ver := range cases {
		t.Run("invalid:"+ver, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** "+ver+"\n")
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, versionFormatRuleSet(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Passed {
				t.Errorf("expected violation for invalid version %q", ver)
			}
		})
	}
}

func aasddVersionRuleSet() types.RuleSet {
	return types.RuleSet{
		AASDDVersion: "v1",
		Rules: []types.Rule{
			{ID: "spec.invalid-aasdd-version", Description: "AASDD version must be v1, v2, …", AppliesTo: "spec.md"},
		},
	}
}

func TestCollectViolations_AASDDVersion_Valid(t *testing.T) {
	cases := []string{"v1", "v2", "v10", "v99"}
	for _, ver := range cases {
		t.Run(ver, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** "+ver+"\n")
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, aasddVersionRuleSet(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Passed {
				t.Errorf("expected no violation for valid AASDD version %q, got: %v", ver, result.Violations)
			}
		})
	}
}

func TestCollectViolations_AASDDVersion_Invalid(t *testing.T) {
	cases := []string{"1", "v0", "v01", "v1.0", "v1.0.0", "latest", ""}
	for _, ver := range cases {
		t.Run("invalid:"+ver, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** "+ver+"\n")
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, aasddVersionRuleSet(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Passed {
				t.Errorf("expected violation for invalid AASDD version %q", ver)
			}
		})
	}
}

// --- Result field invariants ---

func TestCollectViolations_AASDDVersionMatchesRuleSet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	rs := minimalRuleSet()
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AASDDVersion != rs.AASDDVersion {
		t.Errorf("result.aasdd_version=%q, want %q", result.AASDDVersion, rs.AASDDVersion)
	}
}

func TestCollectViolations_RuleCountMatchesRuleSet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	rs := minimalRuleSet()
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RuleCount != len(rs.Rules) {
		t.Errorf("result.rule_count=%d, want %d", result.RuleCount, len(rs.Rules))
	}
}

func TestCollectViolations_ViolationSeverityCopiedFromRule(t *testing.T) {
	dir := t.TempDir()
	// No spec.md → directory.missing-spec fires (SeverityError in minimalRuleSet).
	rs := minimalRuleSet()
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ruleMap := make(map[string]types.Severity)
	for _, r := range rs.Rules {
		ruleMap[r.ID] = r.Severity
	}
	for _, v := range result.Violations {
		want, ok := ruleMap[v.Rule]
		if !ok {
			t.Errorf("violation references unknown rule ID %q", v.Rule)
			continue
		}
		if v.Severity != want {
			t.Errorf("violation %q severity=%v, rule severity=%v", v.Rule, v.Severity, want)
		}
	}
}

func TestCollectViolations_ViolationDescriptionCopiedFromRule(t *testing.T) {
	dir := t.TempDir()
	rs := minimalRuleSet()
	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	descMap := make(map[string]string)
	for _, r := range rs.Rules {
		descMap[r.ID] = r.Description
	}
	for _, v := range result.Violations {
		want, ok := descMap[v.Rule]
		if !ok {
			continue
		}
		if v.Description != want {
			t.Errorf("violation %q description=%q, rule description=%q", v.Rule, v.Description, want)
		}
	}
}

// --- Progress output ---

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = oldStderr
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestCollectViolations_Progress_RootMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	output := captureStderr(t, func() {
		_, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "ok  .") {
		t.Errorf("expected progress to contain 'ok  .', got: %q", output)
	}
}

func TestCollectViolations_Progress_FileMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	output := captureStderr(t, func() {
		_, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "spec.md") {
		t.Errorf("expected progress to contain 'spec.md', got: %q", output)
	}
}

func TestCollectViolations_Progress_FalseNoOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**Version:** 0.1.0\n")
	output := captureStderr(t, func() {
		_, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if output != "" {
		t.Errorf("expected no stderr output when progress=false, got: %q", output)
	}
}
