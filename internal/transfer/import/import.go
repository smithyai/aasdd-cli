// Package import_ implements the Import ability.
package import_

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/smithyai/aasdd-cli/internal/format"
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
		relPath := "scenarios/" + format.NameToKebab(scenario.Heading) + "/scenario.md"
		if _, writeErr := writeMD(outputPath, relPath, renderScenarioFile(scenario)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	for _, concept := range exp.Concepts {
		domainName := strings.TrimSuffix(concept.Heading, " domain")
		relPath := "concepts/" + format.NameToKebab(domainName) + "/concept.md"
		if _, writeErr := writeMD(outputPath, relPath, renderConceptFile(concept)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	for _, decision := range exp.Decisions {
		relPath := "decisions/" + format.NameToKebab(decision.Heading) + "/decision.md"
		if _, writeErr := writeMD(outputPath, relPath, renderDecisionFile(decision)); writeErr != nil {
			return types.TransferResult{}, writeErr
		}
		fileCount++
	}

	return types.TransferResult{FileCount: fileCount, OutputPath: outputPath}, nil
}

// writeAbility writes ability.md and recursively writes its sub-abilities.
func writeAbility(root, parentDir string, a types.ParsedAbility) (int, error) {
	relDir := parentDir + "/" + format.NameToKebab(a.Heading)
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

func marker(placeholder string) string {
	switch placeholder {
	case types.PlaceholderPending:
		return "_Pending._"
	case types.PlaceholderOpen:
		return "_Open._"
	default:
		return "_None._"
	}
}

// writeSection writes a ### section. body is the rendered content (already
// ending in a newline) or empty. When the section carries a placeholder it is
// written as that marker; when the body is empty and the section is required,
// _None._ is written; otherwise an empty, optional section is omitted.
func writeSection(b *strings.Builder, heading, body string, placeholders map[string]string, required bool) {
	if ph, ok := placeholders[heading]; ok {
		b.WriteString("\n### " + heading + "\n\n" + marker(ph) + "\n")
		return
	}
	if body != "" {
		b.WriteString("\n### " + heading + "\n\n" + body)
		return
	}
	if required {
		b.WriteString("\n### " + heading + "\n\n_None._\n")
	}
}

func proseBody(text string) string {
	if text == "" {
		return ""
	}
	return text + "\n"
}

func bulletBody(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range items {
		b.WriteString("- " + item + "\n")
	}
	return b.String()
}

func tableBody(t *types.Table) string {
	if t == nil {
		return ""
	}
	return renderTable(t)
}

func writeCustomSections(b *strings.Builder, sections []types.CustomSection) {
	for _, cs := range sections {
		b.WriteString("\n### " + cs.Heading + "\n\n" + cs.Content + "\n")
	}
}

func renderSpecFile(s types.ParsedSpecFile) string {
	var b strings.Builder
	b.WriteString("## " + s.Heading + "\n")
	b.WriteString("\n")
	b.WriteString("**AASDD:** " + s.AASDDVersion + "\n")
	b.WriteString("**Version:** " + s.Version + "\n")
	if s.Summary != "" {
		b.WriteString("\n" + s.Summary + "\n")
	}
	writeSection(&b, "Purpose", proseBody(s.Purpose), s.Placeholders, false)
	writeSection(&b, "Non-Goals", bulletBody(s.NonGoals), s.Placeholders, false)
	writeSection(&b, "Success Criteria", tableBody(s.SuccessCriteria), s.Placeholders, false)
	writeSection(&b, "Invariants", bulletBody(s.Invariants), s.Placeholders, false)
	writeSection(&b, "Failure Modes", tableBody(s.FailureModes), s.Placeholders, false)
	writeCustomSections(&b, s.CustomSections)
	return b.String()
}

func renderAbilityFile(a types.ParsedAbility) string {
	var b strings.Builder
	b.WriteString("## " + a.Heading + "\n")
	if a.Purpose != "" {
		b.WriteString("\n" + a.Purpose + "\n")
	}
	if a.Delegated() {
		b.WriteString("\n**Spec:** " + a.Spec + "\n")
		b.WriteString("**Version:** " + a.SpecVersion + "\n")
		return b.String()
	}
	writeSection(&b, "Inputs", tableBody(a.Inputs), a.Placeholders, true)
	outputs := tableBody(a.Outputs)
	if outputs != "" && a.OutputsNote != "" {
		outputs += "\n" + a.OutputsNote + "\n"
	}
	writeSection(&b, "Outputs", outputs, a.Placeholders, true)
	writeSection(&b, "Invariants", bulletBody(a.Invariants), a.Placeholders, true)
	writeSection(&b, "Failure Modes", tableBody(a.FailureModes), a.Placeholders, true)
	writeSection(&b, "Idempotency", proseBody(a.Idempotency), a.Placeholders, false)
	writeSection(&b, "Composition", tableBody(a.Composition), a.Placeholders, false)
	writeCustomSections(&b, a.CustomSections)
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
	writeCustomSections(&b, c.CustomSections)
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
	writeSection(&b, "Example", proseBody(s.Example), nil, false)
	writeCustomSections(&b, s.CustomSections)
	return b.String()
}

func renderDecisionFile(d types.ParsedDecision) string {
	var b strings.Builder
	b.WriteString("## " + d.Heading + "\n")
	writeSection(&b, "Context", proseBody(d.Context), d.Placeholders, false)
	writeSection(&b, "Requirement", proseBody(d.Requirement), d.Placeholders, false)
	writeSection(&b, "Options", bulletBody(d.Options), d.Placeholders, false)
	writeSection(&b, "Decision", proseBody(d.Decision), d.Placeholders, false)
	writeCustomSections(&b, d.CustomSections)
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
