package collect_violations_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func minimalRuleSet() types.RuleSet {
	return types.RuleSet{
		SpecVersion: types.SpecVersion{Value: "v0.1.0"},
		Rules: []types.Rule{
			{ID: "directory.missing-spec", Description: "root must have spec.md", AppliesTo: "directory"},
			{ID: "spec.missing-version", Description: "spec.md must declare **Version:**", AppliesTo: "spec.md"},
			{ID: "ability.missing-purpose", Description: "ability.md must declare **Purpose:**", AppliesTo: "ability.md"},
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
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *collect_violations.TargetNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotFound, got %T: %v", err, err)
	}
}

// --- Invariants ---

func TestCollectViolations_PassedIffViolationsEmpty_Conformant(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "**AASDD:** v1\n**Version:** 0.1.0\n**Status:** Draft\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet())
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

func TestCollectViolations_PassedIffViolationsEmpty_NonConformant(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "no version here\n")

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected passed=false when violations exist")
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation")
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

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, rs)
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

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet())
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

	result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, minimalRuleSet())
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

	r1, err := collect_violations.CollectViolations(target, rs)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	r2, err := collect_violations.CollectViolations(target, rs)
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
		SpecVersion: types.SpecVersion{Value: "v1"},
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
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, versionFormatRuleSet())
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
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, versionFormatRuleSet())
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
		SpecVersion: types.SpecVersion{Value: "v1"},
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
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, aasddVersionRuleSet())
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
			result, err := collect_violations.CollectViolations(types.SpecTarget{Path: dir}, aasddVersionRuleSet())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Passed {
				t.Errorf("expected violation for invalid AASDD version %q", ver)
			}
		})
	}
}
