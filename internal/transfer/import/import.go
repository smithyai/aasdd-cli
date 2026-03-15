// Package import_ implements the Import ability.
package import_

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// SourceNotFound is returned when the source path does not exist.
type SourceNotFound struct{ Path string }

func (e *SourceNotFound) Error() string {
	return fmt.Sprintf("source not found: %q", e.Path)
}

// SourceNotFile is returned when the source path is a directory, not a file.
type SourceNotFile struct{ Path string }

func (e *SourceNotFile) Error() string {
	return fmt.Sprintf("source is not a file: %q", e.Path)
}

// OutputNotEmpty is returned when the output location is occupied.
type OutputNotEmpty struct{ Path string }

func (e *OutputNotEmpty) Error() string {
	return fmt.Sprintf("output location is not empty: %q", e.Path)
}

// ParseError is returned when the source file cannot be decoded.
type ParseError struct{ Err error }

func (e *ParseError) Error() string { return fmt.Sprintf("invalid export file: %v", e.Err) }
func (e *ParseError) Unwrap() error { return e.Err }

// WriteError is returned when an output file cannot be written.
type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string { return fmt.Sprintf("write error at %q: %v", e.Path, e.Err) }
func (e *WriteError) Unwrap() error { return e.Err }

// Import reconstructs a spec directory on disk from a previously exported structured file.
func Import(source, outputPath string) (types.TransferResult, error) {
	info, err := os.Stat(source)
	if err != nil {
		return types.TransferResult{}, &SourceNotFound{Path: source}
	}
	if info.IsDir() {
		return types.TransferResult{}, &SourceNotFile{Path: source}
	}

	if info, statErr := os.Stat(outputPath); statErr == nil {
		if !info.IsDir() {
			return types.TransferResult{}, &OutputNotEmpty{Path: outputPath}
		}
		entries, readErr := os.ReadDir(outputPath)
		if readErr != nil {
			return types.TransferResult{}, &WriteError{Path: outputPath, Err: readErr}
		}
		if len(entries) > 0 {
			return types.TransferResult{}, &OutputNotEmpty{Path: outputPath}
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return types.TransferResult{}, &SourceNotFound{Path: source}
	}

	var exp types.SpecExport
	if err := json.Unmarshal(data, &exp); err != nil {
		return types.TransferResult{}, &ParseError{Err: err}
	}

	fileCount := 0

	if _, writeErr := writeMD(outputPath, "spec.md", renderSpecFile(exp.ParsedSpecFile)); writeErr != nil {
		return types.TransferResult{}, writeErr
	}
	fileCount++

	if exp.StateMachine != nil {
		if _, writeErr := writeMD(outputPath, "state-machine.md", renderStateMachineFile(*exp.StateMachine)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	for _, ability := range exp.Abilities {
		n, writeErr := writeAbility(outputPath, "abilities", ability)
		if writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount += n
	}

	for _, scenario := range exp.Scenarios {
		relPath := "scenarios/" + pascalToKebab(scenario.Heading) + "/scenario.md"
		if _, writeErr := writeMD(outputPath, relPath, renderScenarioFile(scenario)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	for _, concept := range exp.Concepts {
		domainName := strings.TrimSuffix(concept.Heading, " domain")
		relPath := "concepts/" + pascalToKebab(domainName) + "/concept.md"
		if _, writeErr := writeMD(outputPath, relPath, renderConceptFile(concept)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	for _, decision := range exp.Decisions {
		relPath := "decisions/" + pascalToKebab(decision.Heading) + "/decision.md"
		if _, writeErr := writeMD(outputPath, relPath, renderDecisionFile(decision)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	return types.TransferResult{FileCount: fileCount, OutputPath: outputPath}, nil
}

// writeAbility writes ability.md and recursively writes its sub-abilities.
func writeAbility(root, parentDir string, a types.ParsedAbility) (int, error) {
	relDir := parentDir + "/" + pascalToKebab(a.Heading)
	relPath := relDir + "/ability.md"
	if _, writeErr := writeMD(root, relPath, renderAbilityFile(a)); writeErr != nil {
		return 0, writeErr
	}
	count := 1

	for _, sub := range a.SubAbilities {
		n, writeErr := writeAbility(root, relDir, sub)
		if writeErr != nil {
			return count, writeErr
		}
		count += n
	}

	return count, nil
}

// writeMD writes content as a Markdown file at root/relPath, creating parent dirs as needed.
func writeMD(root, relPath, content string) (string, error) {
	fullPath := filepath.Join(root, filepath.FromSlash(relPath))
	if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
		return "", &WriteError{Path: fullPath, Err: mkErr}
	}
	if writeErr := os.WriteFile(fullPath, []byte(content), 0o644); writeErr != nil {
		return "", &WriteError{Path: fullPath, Err: writeErr}
	}
	return fullPath, nil
}

// --- Markdown renderers ---

func renderSpecFile(s types.ParsedSpecFile) string {
	var b strings.Builder
	b.WriteString("## " + s.Heading + "\n")
	b.WriteString("\n")
	b.WriteString("**AASDD:** " + s.AASDDVersion + "\n")
	b.WriteString("**Version:** " + s.Version + "\n")
	if s.Summary != "" {
		b.WriteString("\n" + s.Summary + "\n")
	}
	if len(s.Invariants) > 0 {
		b.WriteString("\n### Invariants\n\n")
		for _, inv := range s.Invariants {
			b.WriteString("- " + inv + "\n")
		}
	}
	for _, cs := range s.CustomSections {
		b.WriteString("\n### " + cs.Heading + "\n\n" + cs.Content + "\n")
	}
	return b.String()
}

func renderAbilityFile(a types.ParsedAbility) string {
	var b strings.Builder
	b.WriteString("## " + a.Heading + "\n")
	if a.Purpose != "" {
		b.WriteString("\n" + a.Purpose + "\n")
	}
	b.WriteString("\n### Inputs\n\n")
	if a.Inputs != nil {
		b.WriteString(renderTable(a.Inputs))
	} else {
		b.WriteString("_None._\n")
	}
	b.WriteString("\n### Outputs\n\n")
	if a.Outputs != nil {
		b.WriteString(renderTable(a.Outputs))
		if a.OutputsNote != "" {
			b.WriteString("\n" + a.OutputsNote + "\n")
		}
	} else {
		b.WriteString("_None._\n")
	}
	b.WriteString("\n### Invariants\n\n")
	if len(a.Invariants) > 0 {
		for _, inv := range a.Invariants {
			b.WriteString("- " + inv + "\n")
		}
	} else {
		b.WriteString("_None._\n")
	}
	b.WriteString("\n### Failure Modes\n\n")
	if a.FailureModes != nil {
		b.WriteString(renderTable(a.FailureModes))
	} else {
		b.WriteString("_None._\n")
	}
	for _, cs := range a.CustomSections {
		b.WriteString("\n### " + cs.Heading + "\n\n" + cs.Content + "\n")
	}
	return b.String()
}

func renderConceptFile(c types.ParsedConcept) string {
	var b strings.Builder
	b.WriteString("## " + c.Heading + "\n")
	if c.Intro != "" {
		b.WriteString("\n" + c.Intro + "\n")
	}
	for _, t := range c.Types {
		b.WriteString("\n### " + t.Name + "\n")
		if t.Description != "" {
			b.WriteString("\n" + t.Description + "\n")
		}
		if t.Properties != nil {
			if t.PropertiesHeading {
				b.WriteString("\n#### Properties\n\n")
			} else {
				b.WriteString("\n")
			}
			b.WriteString(renderTable(t.Properties))
		}
		if t.Note != "" {
			b.WriteString("\n> **Note:** " + t.Note + "\n")
		}
	}
	return b.String()
}

func renderScenarioFile(s types.ParsedScenario) string {
	var b strings.Builder
	b.WriteString("## " + s.Heading + "\n")
	if s.Description != "" {
		b.WriteString("\n" + s.Description + "\n")
	}
	b.WriteString("\n> `" + s.Trace + "`\n")
	if len(s.Assertions) > 0 {
		b.WriteString("\n")
		for _, assertion := range s.Assertions {
			b.WriteString("- " + assertion + "\n")
		}
	}
	return b.String()
}

func renderDecisionFile(d types.ParsedDecision) string {
	var b strings.Builder
	b.WriteString("## " + d.Heading + "\n")
	if d.Context != "" {
		b.WriteString("\n### Context\n\n" + d.Context + "\n")
	}
	if d.Requirement != "" {
		b.WriteString("\n### Requirement\n\n" + d.Requirement + "\n")
	}
	if d.Decision != "" {
		b.WriteString("\n### Decision\n\n" + d.Decision + "\n")
	}
	for _, cs := range d.CustomSections {
		b.WriteString("\n### " + cs.Heading + "\n\n" + cs.Content + "\n")
	}
	return b.String()
}

func renderStateMachineFile(sm types.ParsedStateMachine) string {
	var b strings.Builder
	b.WriteString("## State Machine\n")
	if sm.Summary != "" {
		b.WriteString("\n" + sm.Summary + "\n")
	}
	if sm.Diagram != "" {
		b.WriteString("\n" + sm.Diagram + "\n")
	}
	b.WriteString("\n### Orchestrator\n")
	if sm.Orchestrator != "" {
		b.WriteString("\n" + sm.Orchestrator + "\n")
	}
	if sm.OrchestratorState != nil {
		b.WriteString("\n#### Orchestrator-Managed State\n\n")
		b.WriteString(renderTable(sm.OrchestratorState))
	}
	b.WriteString("\n### States\n\n")
	if sm.States != nil {
		b.WriteString(renderTable(sm.States))
	}
	b.WriteString("\n### Transitions\n\n")
	if sm.Transitions != nil {
		b.WriteString(renderTable(sm.Transitions))
	}
	if len(sm.TransitionRules) > 0 {
		b.WriteString("\n### Transition Rules\n\n")
		for _, rule := range sm.TransitionRules {
			b.WriteString("- " + rule + "\n")
		}
	}
	if len(sm.ExceptionalFlows) > 0 {
		b.WriteString("\n### Exceptional Flows\n")
		for _, flow := range sm.ExceptionalFlows {
			b.WriteString("\n#### " + flow.Heading + "\n")
			if flow.Content != "" {
				b.WriteString("\n" + flow.Content + "\n")
			}
		}
	}
	return b.String()
}

func renderTable(t *types.Table) string {
	if t == nil || len(t.Headers) == 0 {
		return ""
	}

	// Calculate max width for each column.
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range t.Rows {
		for i, h := range t.Headers {
			if v := utf8.RuneCountInString(row[h]); v > widths[i] {
				widths[i] = v
			}
		}
	}
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
	}

	var b strings.Builder
	// Header row
	b.WriteString("| ")
	for i, h := range t.Headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(h)
		b.WriteString(strings.Repeat(" ", widths[i]-utf8.RuneCountInString(h)))
	}
	b.WriteString(" |\n")
	// Separator row
	b.WriteString("| ")
	for i := range t.Headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(strings.Repeat("-", widths[i]))
	}
	b.WriteString(" |\n")
	// Data rows
	for _, row := range t.Rows {
		b.WriteString("| ")
		for i, h := range t.Headers {
			if i > 0 {
				b.WriteString(" | ")
			}
			v := row[h]
			b.WriteString(v)
			b.WriteString(strings.Repeat(" ", widths[i]-utf8.RuneCountInString(v)))
		}
		b.WriteString(" |\n")
	}
	return b.String()
}

// pascalToKebab converts a PascalCase string to kebab-case.
// Consecutive uppercase letters are treated as an acronym
// (e.g., "CLI" → "cli", "HTTPServer" → "http-server").
func pascalToKebab(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Insert hyphen before this uppercase letter when:
			// - not the first character, AND
			// - either the previous char is lowercase, OR
			//   the next char is lowercase (end of an acronym like "HTTP" in "HTTPServer")
			if i > 0 {
				prevLower := unicode.IsLower(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if prevLower || nextLower {
					b.WriteByte('-')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
