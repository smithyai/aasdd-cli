// Package verify implements the Verify ability.
package verify

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/analysis/verify/load_rule_set"
	"github.com/smithyai/aasdd-cli/internal/types"
)

var aasddVersionRe = regexp.MustCompile(`(?m)^\*\*AASDD:\*\*[ \t]*(\S*)`)

// readAASDDVersion reads the **AASDD:** value from spec.md in the target
// directory. Returns the latest version if spec.md is missing or unreadable,
// or if the label is absent.
func readAASDDVersion(target types.SpecTarget) string {
	data, err := os.ReadFile(filepath.Join(target.Path, "spec.md"))
	if err != nil {
		return load_rule_set.LatestVersion
	}
	m := aasddVersionRe.FindSubmatch(data)
	if m == nil || len(m[1]) == 0 {
		return load_rule_set.LatestVersion
	}
	return string(m[1])
}

// Verify checks a spec directory for conformance with AASDD structural
// conventions and reports all violations.
//
// The AASDD version is read from spec.md. If missing, the latest version is used.
// When progress is true, "ok  <path>" is written to stderr for each validated item.
func Verify(target types.SpecTarget, progress bool) (types.VerificationResult, error) {
	aasddVersion := readAASDDVersion(target)

	ruleSet, err := load_rule_set.LoadRuleSet(aasddVersion)
	if err != nil {
		// Unknown version in spec.md — fall back to latest.
		ruleSet, err = load_rule_set.LoadRuleSet(load_rule_set.LatestVersion)
		if err != nil {
			return types.VerificationResult{}, err
		}
	}

	result, err := collect_violations.CollectViolations(target, ruleSet, progress)
	if err != nil {
		return types.VerificationResult{}, err
	}

	return result, nil
}
