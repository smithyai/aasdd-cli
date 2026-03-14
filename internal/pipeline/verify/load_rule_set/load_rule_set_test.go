package load_rule_set_test

import (
	"errors"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/load_rule_set"
)

// --- Failure modes ---

func TestLoadRuleSet_UnknownAASDDVersion(t *testing.T) {
	_, err := load_rule_set.LoadRuleSet("v99.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *load_rule_set.UnknownAASDDVersion
	if !errors.As(err, &target) {
		t.Fatalf("expected UnknownAASDDVersion, got %T: %v", err, err)
	}
	if target.Version != "v99.0.0" {
		t.Errorf("unexpected version in error: %q", target.Version)
	}
}

// --- Invariants ---

func TestLoadRuleSet_RulesNonEmpty(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rs.Rules) == 0 {
		t.Error("rule_set.rules must be non-empty")
	}
}

func TestLoadRuleSet_UniqueRuleIDs(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seen := make(map[string]struct{}, len(rs.Rules))
	for _, r := range rs.Rules {
		if _, dup := seen[r.ID]; dup {
			t.Errorf("duplicate rule ID: %q", r.ID)
		}
		seen[r.ID] = struct{}{}
	}
}

func TestLoadRuleSet_VersionMatches_Default(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.AASDDVersion == "" {
		t.Error("rule_set.aasdd_version must not be empty")
	}
}

func TestLoadRuleSet_VersionMatches_Explicit(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet("v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.AASDDVersion != "v1" {
		t.Errorf("aasdd_version mismatch: got %q, want %q", rs.AASDDVersion, "v1")
	}
}

func TestLoadRuleSet_ContainsAASDDVersionRule(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.ID == "spec.missing-aasdd-version" {
			return
		}
	}
	t.Error("rule set must contain spec.missing-aasdd-version")
}

func TestLoadRuleSet_ContainsVersionFormatRule(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.ID == "spec.invalid-version-format" {
			return
		}
	}
	t.Error("rule set must contain spec.invalid-version-format")
}

func TestLoadRuleSet_ContainsAASDDVersionFormatRule(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.ID == "spec.invalid-aasdd-version" {
			return
		}
	}
	t.Error("rule set must contain spec.invalid-aasdd-version")
}

func TestLoadRuleSet_ContainsSummaryRule(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.ID == "spec.missing-summary" {
			return
		}
	}
	t.Error("rule set must contain spec.missing-summary")
}

// --- Idempotency ---

func TestLoadRuleSet_Idempotent(t *testing.T) {
	rs1, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	rs2, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if rs1.AASDDVersion != rs2.AASDDVersion {
		t.Error("idempotency: aasdd_version differs between calls")
	}
	if len(rs1.Rules) != len(rs2.Rules) {
		t.Error("idempotency: rule count differs between calls")
	}
	for i := range rs1.Rules {
		if rs1.Rules[i].ID != rs2.Rules[i].ID {
			t.Errorf("idempotency: rule[%d].ID differs: %q vs %q", i, rs1.Rules[i].ID, rs2.Rules[i].ID)
		}
	}
}

// --- Rule completeness invariants ---

func TestLoadRuleSet_AllRulesHaveNonEmptyID(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, r := range rs.Rules {
		if r.ID == "" {
			t.Errorf("rule[%d] has empty ID", i)
		}
	}
}

func TestLoadRuleSet_AllRulesHaveNonEmptyAppliesTo(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.AppliesTo == "" {
			t.Errorf("rule %q has empty AppliesTo", r.ID)
		}
	}
}

func TestLoadRuleSet_AllRulesHaveNonEmptyDescription(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rs.Rules {
		if r.Description == "" {
			t.Errorf("rule %q has empty Description", r.ID)
		}
	}
}

// --- KnownVersions ---

func TestKnownVersions_NonEmpty(t *testing.T) {
	vs := load_rule_set.KnownVersions()
	if len(vs) == 0 {
		t.Error("KnownVersions must not be empty")
	}
}

func TestKnownVersions_LatestVersionPresent(t *testing.T) {
	for _, v := range load_rule_set.KnownVersions() {
		if v.Version == load_rule_set.LatestVersion {
			return
		}
	}
	t.Errorf("LatestVersion %q not found in KnownVersions()", load_rule_set.LatestVersion)
}

func TestKnownVersions_AllHaveNonEmptySummary(t *testing.T) {
	for _, v := range load_rule_set.KnownVersions() {
		if v.Summary == "" {
			t.Errorf("version %q has empty Summary", v.Version)
		}
	}
}
