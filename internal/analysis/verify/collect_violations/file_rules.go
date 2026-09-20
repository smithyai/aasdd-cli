package collect_violations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// mdSection is one ### section of a spec file.
type mdSection struct {
	heading string
	lines   []string
}

// splitSections splits content into preamble lines and ### sections.
func splitSections(content string) (preamble []string, sections []mdSection) {
	var cur *mdSection
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### ") {
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &mdSection{heading: strings.TrimPrefix(line, "### ")}
		} else if cur == nil {
			preamble = append(preamble, line)
		} else {
			cur.lines = append(cur.lines, line)
		}
	}
	if cur != nil {
		sections = append(sections, *cur)
	}
	return
}

// isDelegated reports whether an ability file delegates to another spec.
func isDelegated(content string) bool {
	return hasLabel(strings.Split(content, "\n"), "**Spec:**")
}

// delegatedExempt lists the section-presence rules that do not apply to a
// delegated ability, which has no sections by definition.
var delegatedExempt = map[string]bool{
	"ability.missing-inputs":        true,
	"ability.missing-outputs":       true,
	"ability.missing-invariants":    true,
	"ability.missing-failure-modes": true,
	"ability.table-columns":         true,
}

var (
	specSectionOrder         = []string{"Purpose", "Non-Goals", "Success Criteria", "Invariants", "Failure Modes"}
	abilitySectionOrder      = []string{"Inputs", "Outputs", "Invariants", "Failure Modes", "Idempotency", "Composition"}
	decisionSectionOrder     = []string{"Context", "Requirement", "Options", "Decision"}
	scenarioSectionOrder     = []string{"Example"}
	stateMachineSectionOrder = []string{"Orchestrator", "States", "Transitions", "Transition Rules", "Exceptional Flows"}

	nameTypeDescription = []string{"Name", "Type", "Description"}
	failureColumns      = []string{"Failure", "Condition", "Effect"}
	valueMeaning        = []string{"Value", "Meaning"}
)

// sectionOrderMessage returns "" when the recognized sections among names
// appear once each, in the expected order, with every custom section after
// them, and a description of the first problem otherwise.
func sectionOrderMessage(names, expected []string) string {
	pos := make(map[string]int, len(expected))
	for i, e := range expected {
		pos[e] = i
	}
	seen := map[string]bool{}
	last := -1
	lastName := ""
	customSeen := ""
	for _, n := range names {
		idx, recognized := pos[n]
		if !recognized {
			customSeen = n
			continue
		}
		if seen[n] {
			return fmt.Sprintf("section %s appears twice", n)
		}
		seen[n] = true
		if customSeen != "" {
			return fmt.Sprintf("section %s appears after custom section %s", n, customSeen)
		}
		if idx < last {
			return fmt.Sprintf("section %s appears after %s", n, lastName)
		}
		last = idx
		lastName = n
	}
	return ""
}

// placeholderName returns None, Pending, or Open when the section body is a
// placeholder marker, and "" when it is content.
func placeholderName(lines []string) string {
	var nonEmpty []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}
	if len(nonEmpty) != 1 {
		return ""
	}
	return markerPlaceholder(nonEmpty[0])
}

// markerPlaceholder maps a marker line to its placeholder name, or "".
func markerPlaceholder(line string) string {
	switch strings.TrimSpace(line) {
	case "_None._":
		return types.PlaceholderNone
	case "_Pending._":
		return types.PlaceholderPending
	case "_Open._":
		return types.PlaceholderOpen
	}
	return ""
}

// tableHeaders returns the header cells of the first table in lines, or nil.
func tableHeaders(lines []string) []string {
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "|") {
			return format.SplitCells(l)
		}
	}
	return nil
}

func hasTableLine(lines []string) bool {
	return tableHeaders(lines) != nil
}

func hasBullet(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, "- ") {
			return true
		}
	}
	return false
}

func hasLabel(lines []string, label string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, label) {
			return true
		}
	}
	return false
}

func hasFence(lines []string) bool {
	for _, l := range lines {
		if format.IsFence(l) {
			return true
		}
	}
	return false
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// traceLineRe matches a scenario trace: backticked nodes joined by →, where a
// divergence condition is always followed by the node that receives it.
var traceLineRe = regexp.MustCompile("^> `[^`]+`(( — [A-Za-z][A-Za-z0-9]*)? → `[^`]+`)*$")

var domainHeadingRe = regexp.MustCompile(`^## .+ domain$`)

// columnsViolation checks the first table under a section against the expected
// header cells. A section that is absent or a placeholder passes.
func columnsViolation(rule types.Rule, path string, s *mdSection, expected []string) []types.Violation {
	if s == nil || placeholderName(s.lines) != "" {
		return nil
	}
	headers := tableHeaders(s.lines)
	if headers == nil {
		return []types.Violation{newViolation(rule, path, s.heading+" has no table")}
	}
	if !stringsEqual(headers, expected) {
		return []types.Violation{newViolation(rule, path,
			fmt.Sprintf("%s columns are %s; expected %s", s.heading, strings.Join(headers, " | "), strings.Join(expected, " | ")))}
	}
	return nil
}

// evaluateFileRuleV2 checks the file-level rules AASDD v2 adds.
func evaluateFileRuleV2(rule types.Rule, path, content string, ctx specContext) []types.Violation {
	preamble, sections := splitSections(content)
	names := make([]string, len(sections))
	for i, s := range sections {
		names[i] = s.heading
	}
	find := func(h string) *mdSection {
		for i := range sections {
			if sections[i].heading == h {
				return &sections[i]
			}
		}
		return nil
	}
	missing := func(h string) []types.Violation {
		if find(h) == nil {
			return []types.Violation{newViolation(rule, path, "required section missing: ### "+h)}
		}
		return nil
	}
	order := func(expected []string) []types.Violation {
		if msg := sectionOrderMessage(names, expected); msg != "" {
			return []types.Violation{newViolation(rule, path, msg)}
		}
		return nil
	}
	columns := func(heading string, expected []string) []types.Violation {
		return columnsViolation(rule, path, find(heading), expected)
	}

	switch rule.ID {
	// spec.md
	case "spec.missing-purpose":
		return missing("Purpose")
	case "spec.missing-non-goals":
		return missing("Non-Goals")
	case "spec.missing-success-criteria":
		return missing("Success Criteria")
	case "spec.missing-invariants":
		return missing("Invariants")
	case "spec.section-order":
		return order(specSectionOrder)
	case "spec.criteria-columns":
		return columns("Success Criteria", []string{"Criterion", "Abilities", "Scenarios"})
	case "spec.table-columns":
		return columns("Failure Modes", failureColumns)

	// ability.md
	case "ability.missing-invariants":
		return missing("Invariants")
	case "ability.missing-failure-modes":
		return missing("Failure Modes")
	case "ability.section-order":
		if isDelegated(content) {
			return nil
		}
		return order(abilitySectionOrder)
	case "ability.composition-columns":
		return columns("Composition", []string{"Step", "Ability", "Consumes", "Produces"})
	case "ability.table-columns":
		var vs []types.Violation
		vs = append(vs, columns("Inputs", nameTypeDescription)...)
		vs = append(vs, columns("Outputs", nameTypeDescription)...)
		vs = append(vs, columns("Failure Modes", failureColumns)...)
		return vs
	case "ability.delegated-extra-sections":
		if isDelegated(content) && len(sections) > 0 {
			return []types.Violation{newViolation(rule, path, "a delegated ability has no sections; found ### "+sections[0].heading)}
		}
	case "ability.delegated-missing-version":
		if isDelegated(content) && !hasLabel(preamble, "**Version:**") {
			return []types.Violation{newViolation(rule, path, "delegated ability does not declare **Version:**")}
		}

	// decision.md
	case "decision.section-order":
		return order(decisionSectionOrder)
	case "decision.open-without-options":
		if s := find("Decision"); s != nil && placeholderName(s.lines) == types.PlaceholderOpen {
			if o := find("Options"); o == nil || !hasBullet(o.lines) {
				return []types.Violation{newViolation(rule, path, "open decision has no Options list")}
			}
		}

	// scenario.md
	case "scenario.section-order":
		return order(scenarioSectionOrder)
	case "scenario.trace-format":
		for _, line := range preamble {
			if strings.HasPrefix(line, "> ") {
				if !traceLineRe.MatchString(strings.TrimRight(line, " ")) {
					return []types.Violation{newViolation(rule, path, "trace is not a sequence of backticked nodes with a node after every condition: "+line)}
				}
				return nil
			}
		}

	// concept.md
	case "concept.heading-domain":
		if len(preamble) == 0 || !domainHeadingRe.MatchString(preamble[0]) {
			return []types.Violation{newViolation(rule, path, "concept.md must open with ## {DomainName} domain")}
		}
	case "concept.type-table":
		// Sections without a table that follow the last table-bearing section
		// are custom sections; every other section is a type and needs a table.
		lastType := -1
		for i, s := range sections {
			if hasTableLine(s.lines) {
				lastType = i
			}
		}
		if lastType < 0 {
			lastType = len(sections) - 1
		}
		var vs []types.Violation
		for i, s := range sections {
			if i <= lastType && !hasTableLine(s.lines) {
				vs = append(vs, newViolation(rule, path, "type "+s.heading+" has no table"))
			}
		}
		return vs
	case "concept.table-columns":
		var vs []types.Violation
		for i := range sections {
			headers := tableHeaders(sections[i].lines)
			if headers == nil || stringsEqual(headers, nameTypeDescription) || stringsEqual(headers, valueMeaning) {
				continue
			}
			vs = append(vs, newViolation(rule, path,
				fmt.Sprintf("type %s has columns %s; expected Name | Type | Description or Value | Meaning", sections[i].heading, strings.Join(headers, " | "))))
		}
		return vs

	// state-machine.md
	case "state-machine.missing-diagram":
		if !hasFence(preamble) {
			return []types.Violation{newViolation(rule, path, "no fenced diagram before the sections")}
		}
	case "state-machine.missing-orchestrator":
		return missing("Orchestrator")
	case "state-machine.missing-states":
		return missing("States")
	case "state-machine.missing-transitions":
		return missing("Transitions")
	case "state-machine.section-order":
		return order(stateMachineSectionOrder)
	case "state-machine.table-columns":
		var vs []types.Violation
		vs = append(vs, columns("States", []string{"State", "Ability", "Description"})...)
		vs = append(vs, columns("Transitions", []string{"From", "To", "Trigger", "Data Passed Forward"})...)
		if o := find("Orchestrator"); o != nil && hasTableLine(o.lines) {
			if headers := tableHeaders(o.lines); !stringsEqual(headers, nameTypeDescription) {
				vs = append(vs, newViolation(rule, path,
					fmt.Sprintf("Orchestrator-Managed State columns are %s; expected Name | Type | Description", strings.Join(headers, " | "))))
			}
		}
		return vs
	}
	return nil
}
