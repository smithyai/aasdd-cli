// Package import_ implements the Import ability.
package import_

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
func Import(source, outputPath string) (types.ImportResult, error) {
	info, err := os.Stat(source)
	if err != nil {
		return types.ImportResult{}, &SourceNotFound{Path: source}
	}
	if info.IsDir() {
		return types.ImportResult{}, &SourceNotFile{Path: source}
	}

	if info, statErr := os.Stat(outputPath); statErr == nil {
		if !info.IsDir() {
			return types.ImportResult{}, &OutputNotEmpty{Path: outputPath}
		}
		entries, readErr := os.ReadDir(outputPath)
		if readErr != nil {
			return types.ImportResult{}, &WriteError{Path: outputPath, Err: readErr}
		}
		if len(entries) > 0 {
			return types.ImportResult{}, &OutputNotEmpty{Path: outputPath}
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return types.ImportResult{}, &SourceNotFound{Path: source}
	}

	var exp types.SpecExport
	if err := json.Unmarshal(data, &exp); err != nil {
		return types.ImportResult{}, &ParseError{Err: err}
	}

	fileCount := 0

	if _, writeErr := writeMD(outputPath, "spec.md", renderSpecFile(exp.ParsedSpecFile)); writeErr != nil {
		return types.ImportResult{}, writeErr
	}
	fileCount++

	for _, ability := range exp.Abilities {
		n, writeErr := writeAbility(outputPath, ability)
		if writeErr != nil {
			return types.ImportResult{}, writeErr
		}
		fileCount += n
	}

	for _, scenario := range exp.Scenarios {
		if _, writeErr := writeMD(outputPath, scenario.Path, renderScenarioFile(scenario)); writeErr != nil {
			return types.ImportResult{}, writeErr
		}
		fileCount++
	}

	for _, concept := range exp.Concepts {
		if _, writeErr := writeMD(outputPath, concept.Path, renderConceptFile(concept)); writeErr != nil {
			return types.ImportResult{}, writeErr
		}
		fileCount++
	}

	for _, decision := range exp.Decisions {
		if _, writeErr := writeMD(outputPath, decision.Path, renderDecisionFile(decision)); writeErr != nil {
			return types.ImportResult{}, writeErr
		}
		fileCount++
	}

	return types.ImportResult{FileCount: fileCount, OutputPath: outputPath}, nil
}

// writeAbility writes ability.md and recursively writes its sub-abilities.
func writeAbility(root string, a types.ParsedAbility) (int, error) {
	if _, writeErr := writeMD(root, a.Path, renderAbilityFile(a)); writeErr != nil {
		return 0, writeErr
	}
	count := 1

	for _, sub := range a.SubAbilities {
		n, writeErr := writeAbility(root, sub)
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
	b.WriteString("**Status:** " + s.Status + "\n")
	b.WriteString("**Summary:** " + s.Summary + "\n")
	if len(s.Invariants) > 0 {
		b.WriteString("\n### Invariants\n\n")
		for _, inv := range s.Invariants {
			b.WriteString("- " + inv + "\n")
		}
	}
	return b.String()
}

func renderAbilityFile(a types.ParsedAbility) string {
	var b strings.Builder
	b.WriteString("## " + a.Heading + "\n")
	if a.Purpose != "" {
		b.WriteString("\n**Purpose:** " + a.Purpose + "\n")
	}
	if a.Inputs != nil {
		b.WriteString("\n### Inputs\n\n")
		b.WriteString(renderTable(a.Inputs))
	}
	if a.Outputs != nil {
		b.WriteString("\n### Outputs\n\n")
		b.WriteString(renderTable(a.Outputs))
		if a.OutputsNote != "" {
			b.WriteString("\n" + a.OutputsNote + "\n")
		}
	}
	if len(a.Invariants) > 0 {
		b.WriteString("\n### Invariants\n\n")
		for _, inv := range a.Invariants {
			b.WriteString("- " + inv + "\n")
		}
	}
	if a.FailureModes != nil {
		b.WriteString("\n### Failure Modes\n\n")
		b.WriteString(renderTable(a.FailureModes))
	}
	if a.Notes != "" {
		b.WriteString("\n### Notes\n\n" + a.Notes + "\n")
	}
	if a.Visualization != "" {
		b.WriteString("\n### Visualization\n\n```mermaid\n" + a.Visualization + "\n```\n")
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
		b.WriteString("\n**Description:** " + s.Description + "\n")
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
	return b.String()
}

func renderTable(t *types.Table) string {
	if t == nil || len(t.Headers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| ")
	b.WriteString(strings.Join(t.Headers, " | "))
	b.WriteString(" |\n| ")
	seps := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		w := max(3, len(h))
		seps[i] = strings.Repeat("-", w)
	}
	b.WriteString(strings.Join(seps, " | "))
	b.WriteString(" |\n")
	for _, row := range t.Rows {
		b.WriteString("| ")
		cells := make([]string, len(t.Headers))
		for i, h := range t.Headers {
			cells[i] = row[h]
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}
