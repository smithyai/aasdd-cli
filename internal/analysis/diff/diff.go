// Package diff implements the Diff ability.
package diff

import (
	"fmt"
	"os"
	"sort"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// LeftNotFound is returned when the left path does not exist.
type LeftNotFound struct{ Path string }

func (e *LeftNotFound) Error() string { return fmt.Sprintf("left not found: %q", e.Path) }

// LeftNotDirectory is returned when the left path is not a directory.
type LeftNotDirectory struct{ Path string }

func (e *LeftNotDirectory) Error() string {
	return fmt.Sprintf("left is not a directory: %q", e.Path)
}

// RightNotFound is returned when the right path does not exist.
type RightNotFound struct{ Path string }

func (e *RightNotFound) Error() string { return fmt.Sprintf("right not found: %q", e.Path) }

// RightNotDirectory is returned when the right path is not a directory.
type RightNotDirectory struct{ Path string }

func (e *RightNotDirectory) Error() string {
	return fmt.Sprintf("right is not a directory: %q", e.Path)
}

// ReadError is returned when a file under either directory cannot be read.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string { return fmt.Sprintf("read error at %q: %v", e.Path, e.Err) }
func (e *ReadError) Unwrap() error { return e.Err }

// Diff compares two spec directories and returns all structural differences.
func Diff(left, right types.SpecTarget) (types.DiffResult, error) {
	empty := types.DiffResult{}

	lInfo, err := os.Stat(left.Path)
	if err != nil {
		return empty, &LeftNotFound{Path: left.Path}
	}
	if !lInfo.IsDir() {
		return empty, &LeftNotDirectory{Path: left.Path}
	}

	rInfo, err := os.Stat(right.Path)
	if err != nil {
		return empty, &RightNotFound{Path: right.Path}
	}
	if !rInfo.IsDir() {
		return empty, &RightNotDirectory{Path: right.Path}
	}

	lSpec, _, err := export.LoadSpec(left.Path)
	if err != nil {
		return empty, &ReadError{Path: left.Path, Err: err}
	}

	rSpec, _, err := export.LoadSpec(right.Path)
	if err != nil {
		return empty, &ReadError{Path: right.Path, Err: err}
	}

	var entries []types.DiffEntry

	// Compare spec.md
	entries = append(entries, diffSpec(lSpec.ParsedSpecFile, rSpec.ParsedSpecFile)...)

	// Compare abilities by path
	entries = append(entries, diffByPath("ability",
		abilityMap(lSpec.Abilities), abilityMap(rSpec.Abilities),
		func(l, r types.ParsedAbility) []types.DiffEntry { return diffAbility(l, r) },
	)...)

	// Compare concepts by path
	entries = append(entries, diffByPath("concept",
		conceptMap(lSpec.Concepts), conceptMap(rSpec.Concepts),
		func(l, r types.ParsedConcept) []types.DiffEntry { return diffConcept(l, r) },
	)...)

	// Compare decisions by path
	entries = append(entries, diffByPath("decision",
		decisionMap(lSpec.Decisions), decisionMap(rSpec.Decisions),
		func(l, r types.ParsedDecision) []types.DiffEntry { return diffDecision(l, r) },
	)...)

	// Compare scenarios by path
	entries = append(entries, diffByPath("scenario",
		scenarioMap(lSpec.Scenarios), scenarioMap(rSpec.Scenarios),
		func(l, r types.ParsedScenario) []types.DiffEntry { return diffScenario(l, r) },
	)...)

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })

	return types.DiffResult{
		Left:    left,
		Right:   right,
		Entries: entries,
		Changed: len(entries) > 0,
	}, nil
}

// diffByPath compares two maps of keyed constructs, producing Added/Removed/Changed entries.
func diffByPath[T any](construct string, left, right map[string]T, compare func(T, T) []types.DiffEntry) []types.DiffEntry {
	var entries []types.DiffEntry
	for path, l := range left {
		if r, ok := right[path]; ok {
			entries = append(entries, compare(l, r)...)
		} else {
			entries = append(entries, types.DiffEntry{
				Path: path, Kind: types.DiffRemoved, Construct: construct,
				Detail: fmt.Sprintf("%s removed", construct),
			})
		}
	}
	for path := range right {
		if _, ok := left[path]; !ok {
			entries = append(entries, types.DiffEntry{
				Path: path, Kind: types.DiffAdded, Construct: construct,
				Detail: fmt.Sprintf("%s added", construct),
			})
		}
	}
	return entries
}

// --- Spec comparison ---

func diffSpec(l, r types.ParsedSpecFile) []types.DiffEntry {
	var diffs []string
	if l.Heading != r.Heading {
		diffs = append(diffs, fmt.Sprintf("heading: %q → %q", l.Heading, r.Heading))
	}
	if l.AASDDVersion != r.AASDDVersion {
		diffs = append(diffs, fmt.Sprintf("aasdd: %q → %q", l.AASDDVersion, r.AASDDVersion))
	}
	if l.Version != r.Version {
		diffs = append(diffs, fmt.Sprintf("version: %q → %q", l.Version, r.Version))
	}
	if l.Summary != r.Summary {
		diffs = append(diffs, fmt.Sprintf("summary changed"))
	}
	if !slicesEqual(l.Invariants, r.Invariants) {
		diffs = append(diffs, fmt.Sprintf("invariants changed"))
	}
	if len(diffs) == 0 {
		return nil
	}
	detail := diffs[0]
	if len(diffs) > 1 {
		detail = fmt.Sprintf("%d fields changed", len(diffs))
	}
	return []types.DiffEntry{{
		Path: "spec.md", Kind: types.DiffChanged, Construct: "spec", Detail: detail,
	}}
}

// --- Ability comparison ---

func diffAbility(l, r types.ParsedAbility) []types.DiffEntry {
	changed := l.Heading != r.Heading ||
		l.Purpose != r.Purpose ||
		l.OutputsNote != r.OutputsNote ||
		!slicesEqual(l.Invariants, r.Invariants) ||
		!tablesEqual(l.Inputs, r.Inputs) ||
		!tablesEqual(l.Outputs, r.Outputs) ||
		!tablesEqual(l.FailureModes, r.FailureModes) ||
		!customSectionsEqual(l.CustomSections, r.CustomSections)
	if !changed {
		return nil
	}
	return []types.DiffEntry{{
		Path: l.Path, Kind: types.DiffChanged, Construct: "ability", Detail: "ability changed",
	}}
}

// --- Concept comparison ---

func diffConcept(l, r types.ParsedConcept) []types.DiffEntry {
	if l.Heading != r.Heading || l.Intro != r.Intro {
		return []types.DiffEntry{{
			Path: l.Path, Kind: types.DiffChanged, Construct: "concept", Detail: "concept changed",
		}}
	}
	if len(l.Types) != len(r.Types) {
		return []types.DiffEntry{{
			Path: l.Path, Kind: types.DiffChanged, Construct: "concept", Detail: "concept types changed",
		}}
	}
	for i := range l.Types {
		if !conceptTypesEqual(l.Types[i], r.Types[i]) {
			return []types.DiffEntry{{
				Path: l.Path, Kind: types.DiffChanged, Construct: "concept", Detail: "concept types changed",
			}}
		}
	}
	return nil
}

func conceptTypesEqual(a, b types.ConceptType) bool {
	return a.Name == b.Name &&
		a.Description == b.Description &&
		a.PropertiesHeading == b.PropertiesHeading &&
		a.Note == b.Note &&
		tablesEqual(a.Properties, b.Properties)
}

// --- Decision comparison ---

func diffDecision(l, r types.ParsedDecision) []types.DiffEntry {
	if l.Heading != r.Heading || l.Context != r.Context || l.Requirement != r.Requirement || l.Decision != r.Decision {
		return []types.DiffEntry{{
			Path: l.Path, Kind: types.DiffChanged, Construct: "decision", Detail: "decision changed",
		}}
	}
	return nil
}

// --- Scenario comparison ---

func diffScenario(l, r types.ParsedScenario) []types.DiffEntry {
	if l.Heading != r.Heading || l.Description != r.Description || l.Trace != r.Trace || !slicesEqual(l.Assertions, r.Assertions) {
		return []types.DiffEntry{{
			Path: l.Path, Kind: types.DiffChanged, Construct: "scenario", Detail: "scenario changed",
		}}
	}
	return nil
}

// --- Helpers ---

func abilityMap(abilities []types.ParsedAbility) map[string]types.ParsedAbility {
	m := make(map[string]types.ParsedAbility, len(abilities))
	for _, a := range abilities {
		m[a.Path] = a
	}
	return m
}

func conceptMap(concepts []types.ParsedConcept) map[string]types.ParsedConcept {
	m := make(map[string]types.ParsedConcept, len(concepts))
	for _, c := range concepts {
		m[c.Path] = c
	}
	return m
}

func decisionMap(decisions []types.ParsedDecision) map[string]types.ParsedDecision {
	m := make(map[string]types.ParsedDecision, len(decisions))
	for _, d := range decisions {
		m[d.Path] = d
	}
	return m
}

func scenarioMap(scenarios []types.ParsedScenario) map[string]types.ParsedScenario {
	m := make(map[string]types.ParsedScenario, len(scenarios))
	for _, s := range scenarios {
		m[s.Path] = s
	}
	return m
}

func slicesEqual(a, b []string) bool {
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

func tablesEqual(a, b *types.Table) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !slicesEqual(a.Headers, b.Headers) {
		return false
	}
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	for i := range a.Rows {
		if len(a.Rows[i]) != len(b.Rows[i]) {
			return false
		}
		for k, v := range a.Rows[i] {
			if b.Rows[i][k] != v {
				return false
			}
		}
	}
	return true
}

func customSectionsEqual(a, b []types.CustomSection) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Heading != b[i].Heading || a[i].Content != b[i].Content {
			return false
		}
	}
	return true
}
