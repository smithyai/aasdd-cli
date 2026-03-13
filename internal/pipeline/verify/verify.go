// Package verify implements the Verify ability.
package verify

import (
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/load_rule_set"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// Verify checks a spec directory for conformance with AASDD structural
// conventions and reports all violations.
//
// When specVersion is nil, the latest known version is used.
// Returns TargetNotFound or UnknownSpecVersion on failure.
func Verify(target types.SpecTarget, specVersion *types.SpecVersion) (types.VerificationResult, error) {
	ruleSet, err := load_rule_set.LoadRuleSet(specVersion)
	if err != nil {
		return types.VerificationResult{}, err
	}

	result, err := collect_violations.CollectViolations(target, ruleSet)
	if err != nil {
		return types.VerificationResult{}, err
	}

	return result, nil
}
