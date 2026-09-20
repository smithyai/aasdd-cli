// Package export implements the Export ability.
package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// SourceNotFound is returned when the source path does not exist.
type SourceNotFound struct{ Path string }

func (e *SourceNotFound) Error() string {
	return fmt.Sprintf("source not found: %q", e.Path)
}

// SourceNotDirectory is returned when the source path is not a directory.
type SourceNotDirectory struct{ Path string }

func (e *SourceNotDirectory) Error() string {
	return fmt.Sprintf("source is not a directory: %q", e.Path)
}

// WriteError is returned when an output file cannot be written.
type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string { return fmt.Sprintf("write error at %q: %v", e.Path, e.Err) }
func (e *WriteError) Unwrap() error { return e.Err }

// Export serializes a spec directory into a structured SpecExport written as JSON.
// Output is written to w when outputPath is empty; otherwise to the file at outputPath.
func Export(source, outputPath string, w io.Writer) (types.TransferResult, error) {
	info, err := os.Stat(source)
	if err != nil {
		return types.TransferResult{}, &SourceNotFound{Path: source}
	}
	if !info.IsDir() {
		return types.TransferResult{}, &SourceNotDirectory{Path: source}
	}

	exp, fileCount, err := LoadSpec(source)
	if err != nil {
		return types.TransferResult{}, err
	}

	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		return types.TransferResult{}, &WriteError{Path: outputPath, Err: err}
	}
	data = append(data, '\n')

	if outputPath == "" {
		if _, writeErr := w.Write(data); writeErr != nil {
			return types.TransferResult{}, &WriteError{Path: "-", Err: writeErr}
		}
		return types.TransferResult{FileCount: fileCount}, nil
	}

	if writeErr := os.WriteFile(outputPath, data, 0o644); writeErr != nil {
		return types.TransferResult{}, &WriteError{Path: outputPath, Err: writeErr}
	}
	return types.TransferResult{FileCount: fileCount, OutputPath: outputPath}, nil
}

// NormalizeNewlines converts CRLF and bare CR line endings to LF.
func NormalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// readSpecFile reads a spec file and normalizes its line endings.
func readSpecFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return NormalizeNewlines(string(data)), true
}

var aasddLabelRe = regexp.MustCompile(`(?m)^\*\*AASDD:\*\*[ \t]*(\S*)`)

// LoadSpec walks the source directory and returns a SpecExport plus a total file count.
//
// A spec written against AASDD v1 is parsed with v1 semantics: the sections v2
// added (Purpose, Non-Goals, Success Criteria, Composition, Options) are custom
// sections in a v1 spec and are preserved as such.
func LoadSpec(source string) (types.SpecExport, int, error) {
	exp := types.SpecExport{}
	count := 0
	legacy := false

	if content, ok := readSpecFile(filepath.Join(source, "spec.md")); ok {
		if m := aasddLabelRe.FindStringSubmatch(content); m != nil && m[1] == "v1" {
			legacy = true
		}
		exp.ParsedSpecFile = parseSpecFile(content, legacy)
		count++
	}

	if content, ok := readSpecFile(filepath.Join(source, "state-machine.md")); ok {
		sm := parseStateMachineFile(content)
		exp.StateMachine = &sm
		count++
	}

	abilitiesDir := filepath.Join(source, "abilities")
	if entries, readErr := os.ReadDir(abilitiesDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ability, n, err := loadAbilityDir(source, "abilities/"+entry.Name(), legacy)
			if err == nil {
				exp.Abilities = append(exp.Abilities, ability)
				count += n
			}
		}
	}

	scenariosDir := filepath.Join(source, "scenarios")
	if entries, readErr := os.ReadDir(scenariosDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			scenFile := filepath.Join(scenariosDir, entry.Name(), "scenario.md")
			if content, ok := readSpecFile(scenFile); ok {
				scenario := parseScenarioFile(content)
				scenario.Path = "scenarios/" + entry.Name() + "/scenario.md"
				exp.Scenarios = append(exp.Scenarios, scenario)
				count++
			}
		}
	}

	conceptsDir := filepath.Join(source, "concepts")
	if entries, readErr := os.ReadDir(conceptsDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			conceptFile := filepath.Join(conceptsDir, entry.Name(), "concept.md")
			if content, ok := readSpecFile(conceptFile); ok {
				concept := parseConceptFile(content)
				concept.Path = "concepts/" + entry.Name() + "/concept.md"
				exp.Concepts = append(exp.Concepts, concept)
				count++
			}
		}
	}

	decisionsDir := filepath.Join(source, "decisions")
	if entries, readErr := os.ReadDir(decisionsDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			decisionFile := filepath.Join(decisionsDir, entry.Name(), "decision.md")
			if content, ok := readSpecFile(decisionFile); ok {
				decision := parseDecisionFile(content, legacy)
				decision.Path = "decisions/" + entry.Name() + "/decision.md"
				exp.Decisions = append(exp.Decisions, decision)
				count++
			}
		}
	}

	return exp, count, nil
}

// loadAbilityDir loads ability.md from relDir (relative to source root),
// and recursively loads sub-abilities.
func loadAbilityDir(source, relDir string, legacy bool) (types.ParsedAbility, int, error) {
	absDir := filepath.Join(source, filepath.FromSlash(relDir))
	content, ok := readSpecFile(filepath.Join(absDir, "ability.md"))
	if !ok {
		return types.ParsedAbility{}, 0, fmt.Errorf("ability.md not readable in %s", relDir)
	}
	ability := parseAbilityFile(content, legacy)
	ability.Path = relDir + "/ability.md"
	count := 1

	if entries, readErr := os.ReadDir(absDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subRelDir := relDir + "/" + entry.Name()
			if _, statErr := os.Stat(filepath.Join(absDir, entry.Name(), "ability.md")); statErr == nil {
				sub, n, subErr := loadAbilityDir(source, subRelDir, legacy)
				if subErr == nil {
					ability.SubAbilities = append(ability.SubAbilities, sub)
					count += n
				}
			}
		}
	}

	return ability, count, nil
}

func sortEntries(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
}

// --- Markdown parsers ---

// h3Section holds body lines under a single ### heading.
type h3Section struct {
	heading string
	lines   []string
}

// splitH3 splits content into preamble lines and ### sections.
func splitH3(content string) (preamble []string, sections []h3Section) {
	var cur *h3Section
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### ") {
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &h3Section{heading: strings.TrimPrefix(line, "### ")}
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

// placeholderOf returns the placeholder a section body carries, or "" when the
// body is content. A body is a placeholder when its only non-empty line is
// exactly _None._, _Pending._, or _Open._.
func placeholderOf(lines []string) string {
	var nonEmpty []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}
	if len(nonEmpty) != 1 {
		return ""
	}
	switch nonEmpty[0] {
	case "_None._":
		return types.PlaceholderNone
	case "_Pending._":
		return types.PlaceholderPending
	case "_Open._":
		return types.PlaceholderOpen
	}
	return ""
}

func setPlaceholder(m map[string]string, section, placeholder string) map[string]string {
	if m == nil {
		m = make(map[string]string)
	}
	m[section] = placeholder
	return m
}

// hasContent reports whether any line is non-empty.
func hasContent(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return true
		}
	}
	return false
}

// hasTable reports whether the lines contain a Markdown table.
func hasTable(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "|") {
			return true
		}
	}
	return false
}

var specSections = map[string]bool{"Purpose": true, "Non-Goals": true, "Success Criteria": true, "Invariants": true, "Failure Modes": true}
var abilitySections = map[string]bool{"Inputs": true, "Outputs": true, "Invariants": true, "Failure Modes": true, "Idempotency": true, "Composition": true}
var decisionSections = map[string]bool{"Context": true, "Requirement": true, "Options": true, "Decision": true}
var stateMachineSections = map[string]bool{"Orchestrator": true, "States": true, "Transitions": true, "Transition Rules": true, "Exceptional Flows": true}

// v2-only section headings that a v1 spec may use as custom sections.
var v2SpecSections = map[string]bool{"Purpose": true, "Non-Goals": true, "Success Criteria": true}

func parseSpecFile(content string, legacy bool) types.ParsedSpecFile {
	var s types.ParsedSpecFile
	preamble, sections := splitH3(content)

	var summaryLines []string
	pastMeta := false
	for _, line := range preamble {
		switch {
		case strings.HasPrefix(line, "## "):
			s.Heading = strings.TrimPrefix(line, "## ")
		case strings.HasPrefix(line, "**AASDD:**"):
			s.AASDDVersion = strings.TrimSpace(strings.TrimPrefix(line, "**AASDD:**"))
			pastMeta = true
		case strings.HasPrefix(line, "**Version:**"):
			s.Version = strings.TrimSpace(strings.TrimPrefix(line, "**Version:**"))
			pastMeta = true
		case pastMeta && strings.TrimSpace(line) != "":
			summaryLines = append(summaryLines, line)
		}
	}
	s.Summary = strings.TrimSpace(strings.Join(summaryLines, "\n"))

	for _, sec := range sections {
		if legacy && v2SpecSections[sec.heading] {
			s.CustomSections = append(s.CustomSections, customSection(sec))
			continue
		}
		if ph := placeholderOf(sec.lines); ph != "" && specSections[sec.heading] {
			s.Placeholders = setPlaceholder(s.Placeholders, sec.heading, ph)
			continue
		}
		switch sec.heading {
		case "Purpose":
			s.Purpose = trimmedText(sec.lines)
		case "Non-Goals":
			if bullets := extractBullets(sec.lines); len(bullets) > 0 || !hasContent(sec.lines) {
				s.NonGoals = bullets
			} else {
				s.CustomSections = append(s.CustomSections, customSection(sec))
			}
		case "Success Criteria":
			if t := parseTable(sec.lines); t != nil || !hasContent(sec.lines) {
				s.SuccessCriteria = t
			} else {
				s.CustomSections = append(s.CustomSections, customSection(sec))
			}
		case "Invariants":
			s.Invariants = extractBullets(sec.lines)
		case "Failure Modes":
			if t := parseTable(sec.lines); t != nil || !hasContent(sec.lines) {
				s.FailureModes = t
			} else {
				s.CustomSections = append(s.CustomSections, customSection(sec))
			}
		default:
			s.CustomSections = append(s.CustomSections, customSection(sec))
		}
	}
	return s
}

func parseAbilityFile(content string, legacy bool) types.ParsedAbility {
	var a types.ParsedAbility
	preamble, sections := splitH3(content)

	var purposeLines []string
	pastHeading := false
	for _, line := range preamble {
		switch {
		case strings.HasPrefix(line, "## "):
			a.Heading = strings.TrimPrefix(line, "## ")
			pastHeading = true
		case strings.HasPrefix(line, "**Spec:**"):
			a.Spec = strings.TrimSpace(strings.TrimPrefix(line, "**Spec:**"))
		case strings.HasPrefix(line, "**Version:**"):
			a.SpecVersion = strings.TrimSpace(strings.TrimPrefix(line, "**Version:**"))
		case pastHeading && strings.TrimSpace(line) != "":
			purposeLines = append(purposeLines, line)
		}
	}
	a.Purpose = strings.TrimSpace(strings.Join(purposeLines, "\n"))

	for _, sec := range sections {
		if legacy && sec.heading == "Composition" {
			a.CustomSections = append(a.CustomSections, customSection(sec))
			continue
		}
		if ph := placeholderOf(sec.lines); ph != "" && abilitySections[sec.heading] {
			a.Placeholders = setPlaceholder(a.Placeholders, sec.heading, ph)
			continue
		}
		switch sec.heading {
		case "Inputs":
			a.Inputs = parseTable(sec.lines)
		case "Outputs":
			a.Outputs = parseTable(sec.lines)
			a.OutputsNote = outputsNote(sec.lines)
		case "Invariants":
			a.Invariants = extractBullets(sec.lines)
		case "Failure Modes":
			a.FailureModes = parseTable(sec.lines)
		case "Idempotency":
			a.Idempotency = trimmedText(sec.lines)
		case "Composition":
			if t := parseTable(sec.lines); t != nil || !hasContent(sec.lines) {
				a.Composition = t
			} else {
				a.CustomSections = append(a.CustomSections, customSection(sec))
			}
		default:
			a.CustomSections = append(a.CustomSections, customSection(sec))
		}
	}
	return a
}

// parseConceptFile parses a concept.md file. Every ### section up to and
// including the last one that carries a table is a type; sections without a
// table that follow the last type are custom sections.
func parseConceptFile(content string) types.ParsedConcept {
	var c types.ParsedConcept
	preamble, sections := splitH3(content)

	var introLines []string
	for _, line := range preamble {
		if strings.HasPrefix(line, "## ") {
			c.Heading = strings.TrimPrefix(line, "## ")
		} else {
			introLines = append(introLines, line)
		}
	}
	c.Intro = strings.TrimSpace(strings.Join(introLines, "\n"))

	lastType := -1
	for i, sec := range sections {
		if hasTable(sec.lines) {
			lastType = i
		}
	}
	if lastType < 0 {
		lastType = len(sections) - 1
	}
	for i, sec := range sections {
		if i <= lastType {
			c.Types = append(c.Types, parseConceptType(sec.heading, sec.lines))
		} else {
			c.CustomSections = append(c.CustomSections, customSection(sec))
		}
	}
	return c
}

func parseConceptType(name string, lines []string) types.ConceptType {
	ct := types.ConceptType{Name: name}
	var descLines []string
	var noteLines []string
	var tableLines []string
	state := "desc"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch state {
		case "desc":
			switch {
			case trimmed == "#### Properties":
				ct.PropertiesHeading = true
				state = "table"
			case strings.HasPrefix(trimmed, "|"):
				state = "table"
				tableLines = append(tableLines, trimmed)
			case strings.HasPrefix(trimmed, "> **Note:**"):
				noteLines = append(noteLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "> **Note:**")))
				state = "note"
			default:
				descLines = append(descLines, line)
			}
		case "table":
			switch {
			case strings.HasPrefix(trimmed, "|"):
				tableLines = append(tableLines, trimmed)
			case trimmed == "" && len(tableLines) > 0:
				state = "posttable"
			}
		case "posttable":
			if strings.HasPrefix(trimmed, "> **Note:**") {
				noteLines = append(noteLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "> **Note:**")))
			}
		}
	}

	ct.Description = strings.TrimSpace(strings.Join(descLines, "\n"))
	if len(tableLines) >= 2 {
		ct.Properties = parseTable(tableLines)
	}
	if len(noteLines) > 0 {
		ct.Note = strings.TrimSpace(strings.Join(noteLines, "\n"))
	}
	return ct
}

// traceRe matches a trace line: backticked nodes joined by →, each optionally
// followed by a divergence condition that is itself followed by a node.
var traceRe = regexp.MustCompile("^> `[^`]+`(( — [^→`]+)? → `[^`]+`)*$")

func parseScenarioFile(content string) types.ParsedScenario {
	var s types.ParsedScenario
	preamble, sections := splitH3(content)

	var descLines []string
	pastHeading := false
	for _, line := range preamble {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## "):
			s.Heading = strings.TrimPrefix(line, "## ")
			pastHeading = true
		case s.Trace == "" && traceRe.MatchString(trimmed):
			s.Trace = strings.Trim(strings.TrimPrefix(trimmed, ">"), " `")
		case strings.HasPrefix(line, "- "):
			s.Assertions = append(s.Assertions, strings.TrimPrefix(line, "- "))
		case pastHeading && s.Trace == "" && trimmed != "":
			descLines = append(descLines, line)
		}
	}
	s.Description = strings.TrimSpace(strings.Join(descLines, "\n"))

	for _, sec := range sections {
		switch sec.heading {
		case "Example":
			s.Example = trimmedText(sec.lines)
		default:
			s.CustomSections = append(s.CustomSections, customSection(sec))
		}
	}
	return s
}

func parseDecisionFile(content string, legacy bool) types.ParsedDecision {
	var d types.ParsedDecision
	preamble, sections := splitH3(content)
	for _, line := range preamble {
		if strings.HasPrefix(line, "## ") {
			d.Heading = strings.TrimPrefix(line, "## ")
		}
	}
	for _, sec := range sections {
		if legacy && sec.heading == "Options" {
			d.CustomSections = append(d.CustomSections, customSection(sec))
			continue
		}
		if ph := placeholderOf(sec.lines); ph != "" && decisionSections[sec.heading] {
			d.Placeholders = setPlaceholder(d.Placeholders, sec.heading, ph)
			continue
		}
		switch sec.heading {
		case "Context":
			d.Context = trimmedText(sec.lines)
		case "Requirement":
			d.Requirement = trimmedText(sec.lines)
		case "Options":
			if bullets := extractBullets(sec.lines); len(bullets) > 0 || !hasContent(sec.lines) {
				d.Options = bullets
			} else {
				d.CustomSections = append(d.CustomSections, customSection(sec))
			}
		case "Decision":
			d.Decision = trimmedText(sec.lines)
		default:
			d.CustomSections = append(d.CustomSections, customSection(sec))
		}
	}
	return d
}

// parseTable parses lines that form a Markdown table and returns a typed Table.
func parseTable(lines []string) *types.Table {
	var tlines []string
	for _, line := range lines {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "|") {
			tlines = append(tlines, t)
		}
	}
	if len(tlines) < 2 {
		return nil
	}
	rawHeaders := strings.Split(strings.Trim(tlines[0], "|"), "|")
	headers := make([]string, len(rawHeaders))
	for i, h := range rawHeaders {
		headers[i] = strings.TrimSpace(h)
	}
	var rows []map[string]string
	for _, line := range tlines[2:] {
		rawCells := strings.Split(strings.Trim(line, "|"), "|")
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(rawCells) {
				row[h] = strings.TrimSpace(rawCells[i])
			}
		}
		rows = append(rows, row)
	}
	return &types.Table{Headers: headers, Rows: rows}
}

func extractBullets(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			out = append(out, strings.TrimPrefix(line, "- "))
		}
	}
	return out
}

func trimmedText(lines []string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func customSection(sec h3Section) types.CustomSection {
	return types.CustomSection{
		Heading: sec.heading,
		Content: trimmedText(sec.lines),
	}
}

func outputsNote(lines []string) string {
	lastTable := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			lastTable = i
		}
	}
	if lastTable < 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[lastTable+1:], "\n"))
}

// parseStateMachineFile parses the content of a state-machine.md file.
func parseStateMachineFile(content string) types.ParsedStateMachine {
	var sm types.ParsedStateMachine
	preamble, sections := splitH3(content)

	// Extract summary and diagram from preamble (everything before ### sections).
	var summaryLines []string
	var diagramLines []string
	inDiagram := false
	pastHeading := false
	for _, line := range preamble {
		if strings.HasPrefix(line, "## ") {
			pastHeading = true
			continue
		}
		if !pastHeading {
			continue
		}
		if strings.HasPrefix(line, "```") && !inDiagram {
			inDiagram = true
			diagramLines = append(diagramLines, line)
			continue
		}
		if inDiagram {
			diagramLines = append(diagramLines, line)
			if strings.TrimSpace(line) == "```" {
				inDiagram = false
			}
			continue
		}
		summaryLines = append(summaryLines, line)
	}
	sm.Summary = strings.TrimSpace(strings.Join(summaryLines, "\n"))
	sm.Diagram = strings.TrimSpace(strings.Join(diagramLines, "\n"))

	for _, sec := range sections {
		if ph := placeholderOf(sec.lines); ph != "" && stateMachineSections[sec.heading] {
			sm.Placeholders = setPlaceholder(sm.Placeholders, sec.heading, ph)
			continue
		}
		switch sec.heading {
		case "Orchestrator":
			sm.Orchestrator, sm.OrchestratorState = parseOrchestratorSection(sec.lines)
		case "States":
			sm.States = parseTable(sec.lines)
		case "Transitions":
			sm.Transitions = parseTable(sec.lines)
		case "Transition Rules":
			sm.TransitionRules = extractBullets(sec.lines)
		case "Exceptional Flows":
			sm.ExceptionalFlows = parseExceptionalFlows(sec.lines)
		default:
			sm.CustomSections = append(sm.CustomSections, customSection(sec))
		}
	}
	return sm
}

// parseOrchestratorSection extracts the orchestrator text and optional
// Orchestrator-Managed State table from an Orchestrator H3 section.
func parseOrchestratorSection(lines []string) (string, *types.Table) {
	var textLines []string
	var tableLines []string
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "#### Orchestrator-Managed State" {
			inTable = true
			continue
		}
		if inTable {
			if strings.HasPrefix(trimmed, "|") {
				tableLines = append(tableLines, trimmed)
			}
		} else {
			textLines = append(textLines, line)
		}
	}
	text := strings.TrimSpace(strings.Join(textLines, "\n"))
	var table *types.Table
	if len(tableLines) >= 2 {
		table = parseTable(tableLines)
	}
	return text, table
}

// parseExceptionalFlows splits H4 sub-sections into ExceptionalFlow entries.
func parseExceptionalFlows(lines []string) []types.ExceptionalFlow {
	var flows []types.ExceptionalFlow
	var current *types.ExceptionalFlow
	for _, line := range lines {
		if strings.HasPrefix(line, "#### ") {
			if current != nil {
				current.Content = strings.TrimSpace(current.Content)
				flows = append(flows, *current)
			}
			heading := strings.TrimPrefix(line, "#### ")
			current = &types.ExceptionalFlow{Heading: heading}
		} else if current != nil {
			current.Content += line + "\n"
		}
	}
	if current != nil {
		current.Content = strings.TrimSpace(current.Content)
		flows = append(flows, *current)
	}
	return flows
}
