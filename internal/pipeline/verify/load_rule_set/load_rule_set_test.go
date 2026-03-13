package load_rule_set_test

import (
	"errors"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/load_rule_set"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Failure modes ---

func TestLoadRuleSet_UnknownSpecVersion(t *testing.T) {
	sv := &types.SpecVersion{Value: "v99.0.0"}
	_, err := load_rule_set.LoadRuleSet(sv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *load_rule_set.UnknownSpecVersion
	if !errors.As(err, &target) {
		t.Fatalf("expected UnknownSpecVersion, got %T: %v", err, err)
	}
	if target.Version != "v99.0.0" {
		t.Errorf("unexpected version in error: %q", target.Version)
	}
}

// --- Invariants ---

func TestLoadRuleSet_RulesNonEmpty(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rs.Rules) == 0 {
		t.Error("rule_set.rules must be non-empty")
	}
}

func TestLoadRuleSet_UniqueRuleIDs(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(nil)
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
	rs, err := load_rule_set.LoadRuleSet(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.SpecVersion.Value == "" {
		t.Error("rule_set.spec_version must not be empty when using default")
	}
}

func TestLoadRuleSet_VersionMatches_Explicit(t *testing.T) {
	sv := &types.SpecVersion{Value: "v1"}
	rs, err := load_rule_set.LoadRuleSet(sv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.SpecVersion.Value != sv.Value {
		t.Errorf("spec_version mismatch: got %q, want %q", rs.SpecVersion.Value, sv.Value)
	}
}

func TestLoadRuleSet_ContainsAASDDVersionRule(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet(nil)
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
	rs, err := load_rule_set.LoadRuleSet(nil)
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
	rs, err := load_rule_set.LoadRuleSet(nil)
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

// --- Idempotency ---

func TestLoadRuleSet_Idempotent(t *testing.T) {
	rs1, err := load_rule_set.LoadRuleSet(nil)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	rs2, err := load_rule_set.LoadRuleSet(nil)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if rs1.SpecVersion != rs2.SpecVersion {
		t.Error("idempotency: spec_version differs between calls")
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
