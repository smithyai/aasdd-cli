// Package collect_violations implements the CollectViolations ability.
package collect_violations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
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

// specContext carries spec-wide facts that individual rules depend on.
type specContext struct {
	version string // the **Version:** label of spec.md, or "" when absent
	ready   bool   // true when the version is 1.0.0 or above
}

// readSpecContext reads the spec version from spec.md. A missing or unparseable
// version yields a draft context, so readiness rules stay silent.
func readSpecContext(dir string) specContext {
	data, err := os.ReadFile(filepath.Join(dir, "spec.md"))
	if err != nil {
		return specContext{}
	}
	m := versionLineRe.FindStringSubmatch(export.NormalizeNewlines(string(data)))
	if m == nil {
		return specContext{}
	}
	return specContext{version: m[1], ready: isReadyVersion(m[1])}
}

// isReadyVersion reports whether a semver string is 1.0.0 or above.
func isReadyVersion(v string) bool {
	m := semverRe.FindStringSubmatch(v)
	return m != nil && m[1] != "0"
}

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

	// Separate directory-level rules, spec-wide rules, and file rules.
	var dirRules, fileRules, specRules []types.Rule
	for _, r := range ruleSet.Rules {
		switch r.AppliesTo {
		case "directory":
			dirRules = append(dirRules, r)
		case "spec":
			specRules = append(specRules, r)
		default:
			fileRules = append(fileRules, r)
		}
	}

	ctx := readSpecContext(target.Path)

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
		content := ""
		loaded := false
		for _, r := range fileRules {
			if name != r.AppliesTo {
				continue
			}
			matched = true
			if !loaded {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return &ReadError{Path: path, Err: readErr}
				}
				content = export.NormalizeNewlines(string(data))
				loaded = true
			}
			vs := evaluateFileRule(r, path, content, ctx)
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

	// Evaluate spec-wide rules against the parsed spec.
	if len(specRules) > 0 {
		exp, _, loadErr := export.LoadSpec(target.Path)
		if loadErr != nil {
			return types.VerificationResult{}, &ReadError{Path: target.Path, Err: loadErr}
		}
		violations = append(violations, evaluateSpecRules(specRules, target.Path, exp, ctx)...)
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
			return []types.Violation{newViolation(rule, dir, "spec.md not found in root directory")}
		}
	case "directory.missing-abilities":
		subdir := filepath.Join(dir, "abilities")
		info, err := os.Stat(subdir)
		if err != nil {
			return []types.Violation{newViolation(rule, dir, "abilities/ directory not found in root")}
		}
		if !info.IsDir() {
			return []types.Violation{newViolation(rule, subdir, "abilities exists but is not a directory")}
		}
	case "directory.missing-concepts":
		subdir := filepath.Join(dir, "concepts")
		info, err := os.Stat(subdir)
		if err != nil {
			return []types.Violation{newViolation(rule, dir, "concepts/ directory not found in root")}
		}
		if !info.IsDir() {
			return []types.Violation{newViolation(rule, subdir, "concepts exists but is not a directory")}
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
		return []types.Violation{newViolation(rule, abilitiesDir, "abilities/ contains no ability subdirectories")}
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
				vs = append(vs, newViolation(rule, abilityPath, "ability directory is missing ability.md"))
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
		return []types.Violation{newViolation(rule, conceptsDir, "concepts/ contains no concept domain subdirectories")}
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
				vs = append(vs, newViolation(rule, conceptPath, "concept domain directory is missing concept.md"))
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

// versionLineRe extracts the value after **Version:** on the same line; an
// empty value captures "" rather than the next line's first word.
var versionLineRe = regexp.MustCompile(`(?m)^\*\*Version:\*\*[ \t]*(\S*)`)

// aasddVersionLineRe extracts the value after **AASDD:** on the same line.
var aasddVersionLineRe = regexp.MustCompile(`(?m)^\*\*AASDD:\*\*[ \t]*(\S*)`)

// aasddVersionRe matches a valid AASDD version: v followed by a positive integer with no leading zeros.
var aasddVersionRe = regexp.MustCompile(`^v[1-9][0-9]*$`)

// evaluateFileRule checks content-based rules for a single file. Content has
// normalized line endings.
func evaluateFileRule(rule types.Rule, path, content string, ctx specContext) []types.Violation {
	if delegatedExempt[rule.ID] && isDelegated(content) {
		return nil
	}

	checks := map[string]string{
		"spec.missing-version":         "**Version:**",
		"spec.missing-aasdd-version":   "**AASDD:**",
		"ability.missing-inputs":       "### Inputs",
		"ability.missing-outputs":      "### Outputs",
		"concept.missing-type":         "### ",
		"scenario.missing-trace":       ">",
		"decision.missing-context":     "## Context",
		"decision.missing-requirement": "## Requirement",
		"decision.missing-decision":    "## Decision",
	}

	if needle, ok := checks[rule.ID]; ok {
		if !strings.Contains(content, needle) {
			return []types.Violation{newViolation(rule, path, fmt.Sprintf("required content missing: %s", needle))}
		}
		return nil
	}

	// Check for preamble paragraph (text between heading and first ### section).
	switch rule.ID {
	case "spec.missing-summary":
		if !hasPreambleParagraph(content, "**Version:**") {
			return []types.Violation{newViolation(rule, path, "spec.md must have a summary paragraph after the version metadata")}
		}
		return nil
	case "ability.missing-purpose":
		if !hasPreambleParagraph(content, "## ") {
			return []types.Violation{newViolation(rule, path, "ability.md must have a purpose paragraph after the heading")}
		}
		return nil
	case "scenario.missing-description":
		if !hasPreambleParagraph(content, "## ") {
			return []types.Violation{newViolation(rule, path, "scenario.md must have a description paragraph after the heading")}
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
			return []types.Violation{newViolation(rule, path,
				fmt.Sprintf("**Version:** value %q is not valid semver (expected MAJOR.MINOR.PATCH, no 'v' prefix)", value))}
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
			return []types.Violation{newViolation(rule, path,
				fmt.Sprintf("**AASDD:** value %q is not a valid AASDD version (expected v1, …)", value))}
		}
		return nil
	}

	return evaluateFileRuleV2(rule, path, content, ctx)
}

// hasPreambleParagraph checks whether non-empty text exists between the line
// containing anchor and the first ### section. For ability and scenario files
// the anchor is "## "; for spec files it is "**Version:**".
func hasPreambleParagraph(content, anchor string) bool {
	lines := strings.Split(content, "\n")
	pastAnchor := false
	for _, line := range lines {
		if strings.HasPrefix(line, anchor) {
			pastAnchor = true
			continue
		}
		if !pastAnchor {
			continue
		}
		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "> ") {
			return false
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "**") {
			return true
		}
	}
	return false
}

// newViolation builds a violation for rule at path.
func newViolation(rule types.Rule, path, message string) types.Violation {
	return types.Violation{
		Rule:        rule.ID,
		Severity:    rule.Severity,
		Description: rule.Description,
		Path:        path,
		Message:     message,
	}
}
