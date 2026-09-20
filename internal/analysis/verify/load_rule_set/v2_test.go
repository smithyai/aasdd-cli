package load_rule_set_test

import (
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify/load_rule_set"
)

func TestLoadRuleSet_LatestIsV2(t *testing.T) {
	if load_rule_set.LatestVersion != "v2" {
		t.Errorf("LatestVersion = %q, want v2", load_rule_set.LatestVersion)
	}
}

func TestLoadRuleSet_V2ContainsReadinessAndFormattingRules(t *testing.T) {
	rs, err := load_rule_set.LoadRuleSet("v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range rs.Rules {
		ids[r.ID] = true
	}
	for _, want := range []string{
		"spec.missing-purpose", "spec.criteria-missing-scenario", "pending.not-allowed", "decision.open-not-allowed",
		"ability.missing-composition", "ability.delegated-spec-not-found", "coverage.root-failure-mode", "coverage.transition",
		"folder.name-mismatch", "link.broken", "type.undefined", "format.crlf", "format.table-padding",
	} {
		if !ids[want] {
			t.Errorf("v2 rule set lacks %s", want)
		}
	}
}

func TestLoadRuleSet_V2IncludesEveryV1Rule(t *testing.T) {
	v1, err := load_rule_set.LoadRuleSet("v1")
	if err != nil {
		t.Fatalf("v1: %v", err)
	}
	v2, err := load_rule_set.LoadRuleSet("v2")
	if err != nil {
		t.Fatalf("v2: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range v2.Rules {
		ids[r.ID] = true
	}
	for _, r := range v1.Rules {
		if !ids[r.ID] {
			t.Errorf("v2 rule set lacks v1 rule %s", r.ID)
		}
	}
	if len(v2.Rules) <= len(v1.Rules) {
		t.Errorf("v2 has %d rules, v1 has %d; expected v2 to add rules", len(v2.Rules), len(v1.Rules))
	}
}

func TestKnownVersions_V2IsLast(t *testing.T) {
	vs := load_rule_set.KnownVersions()
	if vs[len(vs)-1].Version != "v2" {
		t.Errorf("last known version = %q, want v2", vs[len(vs)-1].Version)
	}
}
