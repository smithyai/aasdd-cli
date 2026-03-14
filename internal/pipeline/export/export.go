// Package export implements the Export ability.
package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// LoadSpec walks the source directory and returns a SpecExport plus a total file count.
func LoadSpec(source string) (types.SpecExport, int, error) {
	exp := types.SpecExport{}
	count := 0

	if data, readErr := os.ReadFile(filepath.Join(source, "spec.md")); readErr == nil {
		exp.ParsedSpecFile = parseSpecFile(string(data))
		count++
	}

	abilitiesDir := filepath.Join(source, "abilities")
	if entries, readErr := os.ReadDir(abilitiesDir); readErr == nil {
		sortEntries(entries)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ability, n, err := loadAbilityDir(source, "abilities/"+entry.Name())
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
			if sdata, readErr := os.ReadFile(scenFile); readErr == nil {
				scenario := parseScenarioFile(string(sdata))
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
			if data, readErr := os.ReadFile(conceptFile); readErr == nil {
				concept := parseConceptFile(string(data))
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
			if data, readErr := os.ReadFile(decisionFile); readErr == nil {
				decision := parseDecisionFile(string(data))
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
func loadAbilityDir(source, relDir string) (types.ParsedAbility, int, error) {
	absDir := filepath.Join(source, filepath.FromSlash(relDir))
	data, err := os.ReadFile(filepath.Join(absDir, "ability.md"))
	if err != nil {
		return types.ParsedAbility{}, 0, err
	}
	ability := parseAbilityFile(string(data))
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
				sub, n, subErr := loadAbilityDir(source, subRelDir)
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

func parseSpecFile(content string) types.ParsedSpecFile {
	var s types.ParsedSpecFile
	section := ""
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			s.Heading = strings.TrimPrefix(line, "## ")
		case strings.HasPrefix(line, "**AASDD:**"):
			s.AASDDVersion = strings.TrimSpace(strings.TrimPrefix(line, "**AASDD:**"))
		case strings.HasPrefix(line, "**Version:**"):
			s.Version = strings.TrimSpace(strings.TrimPrefix(line, "**Version:**"))
		case strings.HasPrefix(line, "**Summary:**"):
			s.Summary = strings.TrimSpace(strings.TrimPrefix(line, "**Summary:**"))
		case line == "### Invariants":
			section = "invariants"
		case strings.HasPrefix(line, "### "):
			section = ""
		case section == "invariants" && strings.HasPrefix(line, "- "):
			s.Invariants = append(s.Invariants, strings.TrimPrefix(line, "- "))
		}
	}
	return s
}

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

func parseAbilityFile(content string) types.ParsedAbility {
	var a types.ParsedAbility
	preamble, sections := splitH3(content)

	purposeIdx := -1
	for i, line := range preamble {
		if strings.HasPrefix(line, "## ") {
			a.Heading = strings.TrimPrefix(line, "## ")
		} else if strings.HasPrefix(line, "**Purpose:**") {
			purposeIdx = i
		}
	}
	if purposeIdx >= 0 {
		first := strings.TrimSpace(strings.TrimPrefix(preamble[purposeIdx], "**Purpose:**"))
		var parts []string
		if first != "" {
			parts = append(parts, first)
		}
		parts = append(parts, preamble[purposeIdx+1:]...)
		a.Purpose = strings.TrimSpace(strings.Join(parts, "\n"))
	}

	for _, sec := range sections {
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
		case "Notes":
			a.Notes = strings.TrimSpace(strings.Join(sec.lines, "\n"))
		case "Visualization":
			a.Visualization = extractMermaid(sec.lines)
		}
	}
	return a
}

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

	for _, sec := range sections {
		c.Types = append(c.Types, parseConceptType(sec.heading, sec.lines))
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

func parseScenarioFile(content string) types.ParsedScenario {
	var s types.ParsedScenario
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## "):
			s.Heading = strings.TrimPrefix(line, "## ")
		case strings.HasPrefix(line, "**Description:**"):
			s.Description = strings.TrimSpace(strings.TrimPrefix(line, "**Description:**"))
		case strings.HasPrefix(trimmed, "> `") && strings.HasSuffix(trimmed, "`"):
			s.Trace = strings.Trim(strings.TrimPrefix(trimmed, ">"), " `")
		case strings.HasPrefix(line, "- "):
			s.Assertions = append(s.Assertions, strings.TrimPrefix(line, "- "))
		}
	}
	return s
}

func parseDecisionFile(content string) types.ParsedDecision {
	var d types.ParsedDecision
	preamble, sections := splitH3(content)
	for _, line := range preamble {
		if strings.HasPrefix(line, "## ") {
			d.Heading = strings.TrimPrefix(line, "## ")
		}
	}
	for _, sec := range sections {
		text := strings.TrimSpace(strings.Join(sec.lines, "\n"))
		switch sec.heading {
		case "Context":
			d.Context = text
		case "Requirement":
			d.Requirement = text
		case "Decision":
			d.Decision = text
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

func extractMermaid(lines []string) string {
	inBlock := false
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "```mermaid" {
			inBlock = true
			continue
		}
		if strings.TrimSpace(line) == "```" && inBlock {
			inBlock = false
			continue
		}
		if inBlock {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
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
