// Package collect_violations implements the CollectViolations ability.
package collect_violations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// TargetNotFound is returned when the target directory does not exist.
type TargetNotFound struct {
	Path string
}

func (e *TargetNotFound) Error() string {
	return fmt.Sprintf("target not found: %q", e.Path)
}

// TargetIsFile is returned when the target path exists but is a file, not a
// directory.
type TargetIsFile struct {
	Path string
}

func (e *TargetIsFile) Error() string {
	return fmt.Sprintf("target is a file, not a directory: %q", e.Path)
}

// ReadError is returned when a file under the target cannot be read due to a
// filesystem permission or I/O error during traversal.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("read error at %q: %v", e.Path, e.Err)
}

func (e *ReadError) Unwrap() error { return e.Err }

// CollectViolations applies a rule set to a spec directory and returns all
// violations found. When progress is true, "ok  <relpath>" is written to
// stderr for each validated path.
func CollectViolations(target types.SpecTarget, ruleSet types.RuleSet, progress bool) (types.VerificationResult, error) {
	info, err := os.Stat(target.Path)
	if err != nil {
		return types.VerificationResult{}, &TargetNotFound{Path: target.Path}
	}
	if !info.IsDir() {
		return types.VerificationResult{}, &TargetIsFile{Path: target.Path}
	}

	// Separate directory-level rules from file rules.
	var dirRules, fileRules []types.Rule
	for _, r := range ruleSet.Rules {
		if r.AppliesTo == "directory" {
			dirRules = append(dirRules, r)
		} else {
			fileRules = append(fileRules, r)
		}
	}

	var violations []types.Violation

	// Evaluate directory-level rules against the root.
	for _, r := range dirRules {
		vs := evaluateDirectoryRule(r, target.Path)
		violations = append(violations, vs...)
	}
	if progress {
		fmt.Fprintf(os.Stderr, "ok  .\n")
	}

	// Walk the tree and evaluate file rules against every matching filename.
	err = filepath.WalkDir(target.Path, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return &ReadError{Path: path, Err: walkErr}
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		matched := false
		for _, r := range fileRules {
			if name != r.AppliesTo {
				continue
			}
			matched = true
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return &ReadError{Path: path, Err: readErr}
			}
			vs := evaluateFileRule(r, path, string(content))
			violations = append(violations, vs...)
		}
		if matched && progress {
			rel, _ := filepath.Rel(target.Path, path)
			fmt.Fprintf(os.Stderr, "ok  %s\n", rel)
		}
		return nil
	})
	if err != nil {
		return types.VerificationResult{}, err
	}

	// Invariant: result.passed iff no Error-severity violations.
	passed := true
	for _, v := range violations {
		if v.Severity == types.SeverityError {
			passed = false
			break
		}
	}

	// Invariant: every violation has a rule ID present in the rule set and severity copied from the rule.
	rulesByID := make(map[string]types.Rule, len(ruleSet.Rules))
	for _, r := range ruleSet.Rules {
		rulesByID[r.ID] = r
	}
	for _, v := range violations {
		r, ok := rulesByID[v.Rule]
		if !ok {
			panic(fmt.Sprintf("CollectViolations: violation references unknown rule ID %q — this is a bug", v.Rule))
		}
		if v.Severity != r.Severity {
			panic(fmt.Sprintf("CollectViolations: violation severity for rule %q does not match rule severity — this is a bug", v.Rule))
		}
	}

	// Invariant: every violation references an existing path under target.
	for _, v := range violations {
		if _, statErr := os.Stat(v.Path); statErr != nil {
			panic(fmt.Sprintf("CollectViolations: violation path %q does not exist — this is a bug", v.Path))
		}
	}

	// Collect unique spec file basenames (from file rules).
	specFileSet := make(map[string]struct{}, len(fileRules))
	for _, r := range fileRules {
		specFileSet[r.AppliesTo] = struct{}{}
	}
	specFileNames := make([]string, 0, len(specFileSet))
	for name := range specFileSet {
		specFileNames = append(specFileNames, name)
	}
	sort.Strings(specFileNames)

	return types.VerificationResult{
		Target:        target,
		AASDDVersion:  ruleSet.AASDDVersion,
		RuleCount:     len(ruleSet.Rules),
		Violations:    violations,
		Passed:        passed,
		SpecFileNames: specFileNames,
	}, nil
}

// evaluateDirectoryRule checks directory-level rules.
func evaluateDirectoryRule(rule types.Rule, dir string) []types.Violation {
	switch rule.ID {
	case "directory.missing-spec":
		specPath := filepath.Join(dir, "spec.md")
		if _, err := os.Stat(specPath); err != nil {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        dir,
				Message:     "spec.md not found in root directory",
			}}
		}
	case "directory.missing-abilities":
		subdir := filepath.Join(dir, "abilities")
		info, err := os.Stat(subdir)
		if err != nil {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        dir,
				Message:     "abilities/ directory not found in root",
			}}
		}
		if !info.IsDir() {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        subdir,
				Message:     "abilities exists but is not a directory",
			}}
		}
	case "directory.missing-concepts":
		subdir := filepath.Join(dir, "concepts")
		info, err := os.Stat(subdir)
		if err != nil {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        dir,
				Message:     "concepts/ directory not found in root",
			}}
		}
		if !info.IsDir() {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        subdir,
				Message:     "concepts exists but is not a directory",
			}}
		}
	case "abilities.empty":
		abilitiesDir := filepath.Join(dir, "abilities")
		entries, err := os.ReadDir(abilitiesDir)
		if err != nil {
			return nil // covered by directory.missing-abilities
		}
		for _, e := range entries {
			if e.IsDir() {
				return nil
			}
		}
		return []types.Violation{{
			Rule:        rule.ID,
			Severity:    rule.Severity,
			Description: rule.Description,
			Path:        abilitiesDir,
			Message:     "abilities/ contains no ability subdirectories",
		}}
	case "ability.missing-ability-md":
		abilitiesDir := filepath.Join(dir, "abilities")
		entries, err := os.ReadDir(abilitiesDir)
		if err != nil {
			return nil
		}
		var vs []types.Violation
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			abilityPath := filepath.Join(abilitiesDir, e.Name())
			if _, statErr := os.Stat(filepath.Join(abilityPath, "ability.md")); statErr != nil {
				vs = append(vs, types.Violation{
					Rule:        rule.ID,
					Severity:    rule.Severity,
					Description: rule.Description,
					Path:        abilityPath,
					Message:     "ability directory is missing ability.md",
				})
			}
		}
		return vs
	case "concepts.empty":
		conceptsDir := filepath.Join(dir, "concepts")
		entries, err := os.ReadDir(conceptsDir)
		if err != nil {
			return nil // covered by directory.missing-concepts
		}
		for _, e := range entries {
			if e.IsDir() {
				return nil
			}
		}
		return []types.Violation{{
			Rule:        rule.ID,
			Severity:    rule.Severity,
			Description: rule.Description,
			Path:        conceptsDir,
			Message:     "concepts/ contains no concept domain subdirectories",
		}}
	case "concept.missing-concept-md":
		conceptsDir := filepath.Join(dir, "concepts")
		entries, err := os.ReadDir(conceptsDir)
		if err != nil {
			return nil
		}
		var vs []types.Violation
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			conceptPath := filepath.Join(conceptsDir, e.Name())
			if _, statErr := os.Stat(filepath.Join(conceptPath, "concept.md")); statErr != nil {
				vs = append(vs, types.Violation{
					Rule:        rule.ID,
					Severity:    rule.Severity,
					Description: rule.Description,
					Path:        conceptPath,
					Message:     "concept domain directory is missing concept.md",
				})
			}
		}
		return vs
	}
	return nil
}

// semverRe matches a valid semver string with no v prefix: MAJOR.MINOR.PATCH
// with optional pre-release and build metadata.
var semverRe = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// versionLineRe extracts the value after **Version:**
var versionLineRe = regexp.MustCompile(`(?m)^\*\*Version:\*\*\s*(\S+)`)

// aasddVersionLineRe extracts the value after **AASDD:**
var aasddVersionLineRe = regexp.MustCompile(`(?m)^\*\*AASDD:\*\*\s*(\S+)`)

// aasddVersionRe matches a valid AASDD version: v followed by a positive integer with no leading zeros.
var aasddVersionRe = regexp.MustCompile(`^v[1-9][0-9]*$`)

// evaluateFileRule checks content-based rules for a single file.
func evaluateFileRule(rule types.Rule, path, content string) []types.Violation {
	checks := map[string]string{
		"spec.missing-version":         "**Version:**",
		"spec.missing-summary":         "**Summary:**",
		"spec.missing-aasdd-version":   "**AASDD:**",
		"ability.missing-purpose":      "**Purpose:**",
		"ability.missing-inputs":       "### Inputs",
		"ability.missing-outputs":      "### Outputs",
		"concept.missing-type":         "### ",
		"scenario.missing-description": "**Description:**",
		"scenario.missing-trace":       ">",
		"decision.missing-context":     "## Context",
		"decision.missing-requirement": "## Requirement",
		"decision.missing-decision":    "## Decision",
	}

	if needle, ok := checks[rule.ID]; ok {
		if !strings.Contains(content, needle) {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        path,
				Message:     fmt.Sprintf("required content missing: %s", needle),
			}}
		}
		return nil
	}

	if rule.ID == "spec.invalid-version-format" {
		m := versionLineRe.FindStringSubmatch(content)
		// If no value follows **Version:** at all, the presence rule covers the
		// missing field; treat an empty/whitespace-only value as invalid format.
		value := ""
		if m != nil {
			value = m[1]
		}
		if !semverRe.MatchString(value) || strings.HasPrefix(value, "v") {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        path,
				Message:     fmt.Sprintf("**Version:** value %q is not valid semver (expected MAJOR.MINOR.PATCH, no 'v' prefix)", value),
			}}
		}
		return nil
	}

	if rule.ID == "spec.invalid-aasdd-version" {
		m := aasddVersionLineRe.FindStringSubmatch(content)
		value := ""
		if m != nil {
			value = m[1]
		}
		if !aasddVersionRe.MatchString(value) {
			return []types.Violation{{
				Rule:        rule.ID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Path:        path,
				Message:     fmt.Sprintf("**AASDD:** value %q is not a valid AASDD version (expected v1, v2, \u2026)", value),
			}}
		}
		return nil
	}

	return nil
}
