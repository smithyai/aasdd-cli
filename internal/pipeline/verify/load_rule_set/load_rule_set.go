// Package load_rule_set implements the LoadRuleSet ability.
package load_rule_set

import (
	"fmt"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// UnknownSpecVersion is returned when the requested spec version is not
// recognised by the tool.
type UnknownSpecVersion struct {
	Version string
}

func (e *UnknownSpecVersion) Error() string {
	return fmt.Sprintf("unknown spec version: %q", e.Version)
}

const latestVersion = "v1"

// rulesByVersion maps known AASDD spec versions to their structural rule sets.
var rulesByVersion = map[string][]types.Rule{
	"v1": {
		{
			ID:          "directory.missing-spec",
			Description: "The root directory must contain a spec.md file.",
			AppliesTo:   "directory",
			Severity:    types.SeverityError,
		},
		{
			ID:          "spec.missing-version",
			Description: "spec.md must declare **Version:**.",
			AppliesTo:   "spec.md",
			Severity:    types.SeverityWarning,
		},
		{
			ID:          "spec.missing-status",
			Description: "spec.md must declare **Status:**.",
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
	},
}

// LoadRuleSet derives the complete rule set for the given AASDD spec version.
// When specVersion is nil, the latest known version is used.
func LoadRuleSet(specVersion *types.SpecVersion) (types.RuleSet, error) {
	resolved := latestVersion
	if specVersion != nil {
		resolved = specVersion.Value
	}

	rules, ok := rulesByVersion[resolved]
	if !ok {
		return types.RuleSet{}, &UnknownSpecVersion{Version: resolved}
	}

	ruleSet := types.RuleSet{
		SpecVersion: types.SpecVersion{Value: resolved},
		Rules:       rules,
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
