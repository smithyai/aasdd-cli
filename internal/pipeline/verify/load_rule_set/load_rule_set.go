// Package load_rule_set implements the LoadRuleSet ability.
package load_rule_set

import (
	"fmt"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// UnknownAASDDVersion is returned when the requested AASDD version is not
// recognised by the tool.
type UnknownAASDDVersion struct {
	Version string
}

func (e *UnknownAASDDVersion) Error() string {
	return fmt.Sprintf("unknown AASDD version: %q", e.Version)
}

const LatestVersion = "v1"

// VersionEntry describes a known AASDD methodology version.
type VersionEntry struct {
	Version string
	Summary string
}

// KnownVersions returns all recognised AASDD versions in order, oldest first.
func KnownVersions() []VersionEntry {
	return []VersionEntry{
		{Version: "v1", Summary: "Initial release — structural rules for spec.md, ability.md, concept.md, scenario.md, and decision.md."},
	}
}

// rulesByVersion maps known AASDD versions to their structural rule sets.
var rulesByVersion = map[string][]types.Rule{
	"v1": {
		{
			ID:          "directory.missing-spec",
			Description: "The root directory must contain a spec.md file.",
			AppliesTo:   "directory",
			Severity:    types.SeverityError,
		},
		{
			ID:          "directory.missing-abilities",
			Description: "The root directory must contain an abilities/ subdirectory.",
			AppliesTo:   "directory",
			Severity:    types.SeverityError,
		},
		{
			ID:          "directory.missing-concepts",
			Description: "The root directory must contain a concepts/ subdirectory.",
			AppliesTo:   "directory",
			Severity:    types.SeverityError,
		},
		{
			ID:          "abilities.empty",
			Description: "abilities/ contains no ability subdirectories.",
			AppliesTo:   "directory",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "ability.missing-ability-md",
			Description: "Each ability directory must contain an ability.md file.",
			AppliesTo:   "directory",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "concepts.empty",
			Description: "concepts/ contains no concept domain subdirectories.",
			AppliesTo:   "directory",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "concept.missing-concept-md",
			Description: "Each concept domain directory must contain a concept.md file.",
			AppliesTo:   "directory",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.missing-version",
			Description: "spec.md must declare **Version:**.",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.missing-summary",
			Description: "spec.md must declare **Summary:** with a one-sentence description of the spec.",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.missing-aasdd-version",
			Description: "spec.md must declare **AASDD:** to record the methodology version the spec conforms to.",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.invalid-version-format",
			Description: "spec.md **Version:** value must be a valid semver string (MAJOR.MINOR.PATCH) with no 'v' prefix.",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.invalid-aasdd-version",
			Description: "spec.md **AASDD:** value must be a recognised AASDD version (e.g. v1, v2).",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "ability.missing-purpose",
			Description: "ability.md must declare **Purpose:**.",
			AppliesTo:   "ability.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "ability.missing-inputs",
			Description: "ability.md must contain a ### Inputs section.",
			AppliesTo:   "ability.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "ability.missing-outputs",
			Description: "ability.md must contain a ### Outputs section.",
			AppliesTo:   "ability.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "concept.missing-type",
			Description: "concept.md must define at least one type (### heading).",
			AppliesTo:   "concept.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "scenario.missing-description",
			Description: "scenario.md must declare **Description:** with a summary of the scenario.",
			AppliesTo:   "scenario.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "scenario.missing-trace",
			Description: "scenario.md must contain a blockquote (>) execution trace.",
			AppliesTo:   "scenario.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "decision.missing-context",
			Description: "decision.md must contain a ## Context section.",
			AppliesTo:   "decision.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "decision.missing-requirement",
			Description: "decision.md must contain a ## Requirement section.",
			AppliesTo:   "decision.md",
			Severity:    types.SeverityError,
		},
		{
			ID:          "decision.missing-decision",
			Description: "decision.md must contain a ## Decision section.",
			AppliesTo:   "decision.md",
			Severity:    types.SeverityError,
		},
	},
}

// LoadRuleSet derives the complete rule set for the given AASDD version.
func LoadRuleSet(aasddVersion string) (types.RuleSet, error) {
	rules, ok := rulesByVersion[aasddVersion]
	if !ok {
		return types.RuleSet{}, &UnknownAASDDVersion{Version: aasddVersion}
	}

	ruleSet := types.RuleSet{
		AASDDVersion: aasddVersion,
		Rules:        rules,
	}

	// Invariant: rule_set.rules is non-empty.
	if len(ruleSet.Rules) == 0 {
		panic("LoadRuleSet: produced an empty rule set — this is a bug")
	}

	// Invariant: all IDs are unique.
	seen := make(map[string]struct{}, len(ruleSet.Rules))
	for _, r := range ruleSet.Rules {
		if _, dup := seen[r.ID]; dup {
			panic(fmt.Sprintf("LoadRuleSet: duplicate rule ID %q — this is a bug", r.ID))
		}
		seen[r.ID] = struct{}{}
	}

	return ruleSet, nil
}
