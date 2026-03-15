package export_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// --- Helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// makeSpecDir creates a minimal but valid spec directory in a temp dir.
func makeSpecDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"),
		"## My Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** A test spec.\n")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "ability.md"),
		"## Greet\n\n**Purpose:** Says hello.\n\n### Inputs\n\n| Name | Type | Description |\n| ---- | ---- | ----------- |\n| `name` | text | Recipient name. |\n")
	writeFile(t, filepath.Join(dir, "concepts", "greeting", "concept.md"),
		"## Greeting domain\n\nGreeting types.\n\n### Greeting\n\nA greeting message.\n\n#### Properties\n\n| Name | Type | Description |\n| ---- | ---- | ----------- |\n| `text` | text | The message. |\n")
	writeFile(t, filepath.Join(dir, "decisions", "lang", "decision.md"),
		"## Lang\n\n### Context\n\nNeed a language.\n\n### Requirement\n\nA language.\n\n### Decision\n\nGo.\n")
	return dir
}

// --- Failure modes ---

func TestExport_SourceNotFound(t *testing.T) {
	_, err := export.Export("/nonexistent-export-xyz", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *export.SourceNotFound
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFound, got %T: %v", err, err)
	}
}

func TestExport_SourceNotDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "spec.md")
	writeFile(t, f, "## Spec\n")
	_, err := export.Export(f, "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *export.SourceNotDirectory
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotDirectory, got %T: %v", err, err)
	}
}

// --- Output ---

func TestExport_WritesValidJSON(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var v any
	if jsonErr := json.Unmarshal([]byte(buf.String()), &v); jsonErr != nil {
		t.Errorf("output is not valid JSON: %v", jsonErr)
	}
}

func TestExport_StructuredOutput(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal to SpecExport: %v", jsonErr)
	}
	if exp.Heading != "My Spec" {
		t.Errorf("Heading=%q, want \"My Spec\"", exp.Heading)
	}
	if len(exp.Abilities) == 0 {
		t.Error("Abilities must not be empty")
	} else if exp.Abilities[0].Heading != "Greet" {
		t.Errorf("Abilities[0].Heading=%q, want \"Greet\"", exp.Abilities[0].Heading)
	}
	if len(exp.Concepts) == 0 {
		t.Error("Concepts must not be empty")
	}
	if len(exp.Decisions) == 0 {
		t.Error("Decisions must not be empty")
	}
}

func TestExport_AbilityContainsParsedInputs(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if len(exp.Abilities) == 0 {
		t.Fatal("no abilities")
	}
	a := exp.Abilities[0]
	if a.Inputs == nil {
		t.Fatal("Inputs is nil")
	}
	if len(a.Inputs.Headers) == 0 {
		t.Error("Inputs.Headers is empty")
	}
	if len(a.Inputs.Rows) == 0 {
		t.Error("Inputs.Rows is empty")
	}
}

func TestExport_FileCountNonZero(t *testing.T) {
	var buf strings.Builder
	result, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FileCount == 0 {
		t.Error("FileCount must be > 0")
	}
}

func TestExport_OutputPathEmptyWhenStdout(t *testing.T) {
	var buf strings.Builder
	result, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != "" {
		t.Errorf("OutputPath=%q, want empty", result.OutputPath)
	}
}

func TestExport_ToFile(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "spec.json")
	result, err := export.Export(makeSpecDir(t), outFile, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != outFile {
		t.Errorf("OutputPath=%q, want %q", result.OutputPath, outFile)
	}
	if _, statErr := os.Stat(outFile); statErr != nil {
		t.Errorf("output file does not exist: %v", statErr)
	}
}

func TestExport_NonSpecFilesExcluded(t *testing.T) {
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "README.md"), "# Readme")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "notes.txt"), "scratch")
	writeFile(t, filepath.Join(dir, ".DS_Store"), "binary junk")

	var buf strings.Builder
	result, err := export.Export(dir, "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File count reflects only spec Markdown files (spec.md + ability.md + concept.md + decision.md = 4)
	if result.FileCount == 0 {
		t.Error("FileCount must be > 0")
	}

	// The JSON must be a SpecExport (not a flat map)
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("output is not a valid SpecExport: %v", jsonErr)
	}
}

func TestExport_ConceptContainsParsedTypes(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDir(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if len(exp.Concepts) == 0 {
		t.Fatal("no concepts")
	}
	c := exp.Concepts[0]
	if len(c.Types) == 0 {
		t.Error("concept has no types")
	} else if c.Types[0].Properties == nil {
		t.Error("concept type has no properties table")
	}
}

// --- State Machine tests ---

const testStateMachineMD = `## State Machine

Coordinates the lifecycle for a single request.

` + "```mermaid\nstateDiagram-v2\n    [*] --> Processing\n    Processing --> Complete: done\n    Processing --> Failed: error\n    Complete --> [*]\n    Failed --> [*]\n```" + `

### Orchestrator

` + "`HandleRequest`" + ` owns all transitions. It invokes each sub-ability in sequence.

#### Orchestrator-Managed State

| Name      | Type | Description              |
| --------- | ---- | ------------------------ |
| ` + "`request`" + ` | text | The incoming request ID. |

### States

| State        | Ability     | Description                |
| ------------ | ----------- | -------------------------- |
| ` + "`Processing`" + ` | ` + "`DoWork`" + `    | Processes the request.     |
| ` + "`Complete`" + `   | —           | Terminal success state.    |
| ` + "`Failed`" + `     | —           | Terminal failure state.    |

### Transitions

| From         | To         | Trigger          | Data Passed Forward |
| ------------ | ---------- | ---------------- | ------------------- |
| ` + "`Processing`" + ` | ` + "`Complete`" + ` | done             | result              |
| ` + "`Processing`" + ` | ` + "`Failed`" + `   | error            | —                   |

### Transition Rules

- ` + "`Processing`" + ` must complete within the configured timeout.

### Exceptional Flows

#### Timeout

If processing exceeds the deadline, the orchestrator forces a transition to ` + "`Failed`" + `.
`

func makeSpecDirWithStateMachine(t *testing.T) string {
	t.Helper()
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "state-machine.md"), testStateMachineMD)
	return dir
}

func TestExport_StateMachineParsed(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	if jsonErr := json.Unmarshal([]byte(buf.String()), &exp); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if exp.StateMachine == nil {
		t.Fatal("StateMachine is nil")
	}
}

func TestExport_StateMachineSummary(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	want := "Coordinates the lifecycle for a single request."
	if exp.StateMachine.Summary != want {
		t.Errorf("Summary=%q, want %q", exp.StateMachine.Summary, want)
	}
}

func TestExport_StateMachineDiagram(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if !strings.Contains(exp.StateMachine.Diagram, "stateDiagram-v2") {
		t.Errorf("Diagram does not contain mermaid content: %q", exp.StateMachine.Diagram)
	}
	if !strings.HasPrefix(exp.StateMachine.Diagram, "```mermaid") {
		t.Error("Diagram should include code fences")
	}
}

func TestExport_StateMachineOrchestrator(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if !strings.Contains(exp.StateMachine.Orchestrator, "HandleRequest") {
		t.Errorf("Orchestrator=%q, want to contain HandleRequest", exp.StateMachine.Orchestrator)
	}
}

func TestExport_StateMachineOrchestratorState(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if exp.StateMachine.OrchestratorState == nil {
		t.Fatal("OrchestratorState is nil")
	}
	if len(exp.StateMachine.OrchestratorState.Rows) != 1 {
		t.Errorf("OrchestratorState rows=%d, want 1", len(exp.StateMachine.OrchestratorState.Rows))
	}
}

func TestExport_StateMachineStates(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if exp.StateMachine.States == nil {
		t.Fatal("States is nil")
	}
	if len(exp.StateMachine.States.Rows) != 3 {
		t.Errorf("States rows=%d, want 3", len(exp.StateMachine.States.Rows))
	}
}

func TestExport_StateMachineTransitions(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if exp.StateMachine.Transitions == nil {
		t.Fatal("Transitions is nil")
	}
	if len(exp.StateMachine.Transitions.Rows) != 2 {
		t.Errorf("Transitions rows=%d, want 2", len(exp.StateMachine.Transitions.Rows))
	}
}

func TestExport_StateMachineTransitionRules(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.StateMachine.TransitionRules) != 1 {
		t.Errorf("TransitionRules len=%d, want 1", len(exp.StateMachine.TransitionRules))
	}
}

func TestExport_StateMachineExceptionalFlows(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.StateMachine.ExceptionalFlows) != 1 {
		t.Fatalf("ExceptionalFlows len=%d, want 1", len(exp.StateMachine.ExceptionalFlows))
	}
	if exp.StateMachine.ExceptionalFlows[0].Heading != "Timeout" {
		t.Errorf("ExceptionalFlows[0].Heading=%q, want Timeout", exp.StateMachine.ExceptionalFlows[0].Heading)
	}
}

func TestExport_StateMachineCountsInFileCount(t *testing.T) {
	var buf strings.Builder
	baseResult, _ := export.Export(makeSpecDir(t), "", &buf)
	buf.Reset()
	smResult, _ := export.Export(makeSpecDirWithStateMachine(t), "", &buf)
	if smResult.FileCount != baseResult.FileCount+1 {
		t.Errorf("FileCount with state machine=%d, want %d", smResult.FileCount, baseResult.FileCount+1)
	}
}

func TestExport_NoStateMachineWhenAbsent(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDir(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if exp.StateMachine != nil {
		t.Error("StateMachine should be nil when no state-machine.md exists")
	}
}

// --- Table parsing tolerance tests ---

// makeSpecDirWithAbility writes a spec dir whose ability.md uses a custom Inputs table.
func makeSpecDirWithAbility(t *testing.T, abilityMD string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"),
		"## My Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** A test spec.\n")
	writeFile(t, filepath.Join(dir, "abilities", "greet", "ability.md"), abilityMD)
	return dir
}

func TestExport_Table_ExtraCellPaddingTrimmed(t *testing.T) {
	// Cells have excessive surrounding whitespace — values must be trimmed.
	abilityMD := "## Greet\n\nSays hello.\n\n### Inputs\n\n" +
		"|   Name   |   Type   |   Description   |\n" +
		"|   ----   |   ----   |   -----------   |\n" +
		"|   `name`   |   text   |   Recipient name.   |\n"
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithAbility(t, abilityMD), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 || exp.Abilities[0].Inputs == nil {
		t.Fatal("no inputs parsed")
	}
	row := exp.Abilities[0].Inputs.Rows[0]
	if row["Name"] != "`name`" {
		t.Errorf("Name=%q, want \"`name`\"", row["Name"])
	}
	if row["Type"] != "text" {
		t.Errorf("Type=%q, want \"text\"", row["Type"])
	}
	if row["Description"] != "Recipient name." {
		t.Errorf("Description=%q, want \"Recipient name.\"", row["Description"])
	}
}

func TestExport_Table_RaggedSeparatorIgnored(t *testing.T) {
	// Separator row uses varying dash counts and alignment colons — must be skipped.
	abilityMD := "## Greet\n\nSays hello.\n\n### Inputs\n\n" +
		"| Name | Type | Description |\n" +
		"| :--- | :----: | ---: |\n" +
		"| `name` | text | Recipient name. |\n"
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithAbility(t, abilityMD), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 || exp.Abilities[0].Inputs == nil {
		t.Fatal("no inputs parsed")
	}
	if len(exp.Abilities[0].Inputs.Rows) != 1 {
		t.Errorf("rows=%d, want 1", len(exp.Abilities[0].Inputs.Rows))
	}
}

func TestExport_Table_MisalignedColumnsParsedCorrectly(t *testing.T) {
	// Columns are deliberately misaligned — all cells must still map to the right header.
	abilityMD := "## Greet\n\nSays hello.\n\n### Inputs\n\n" +
		"| Name | Type | Description |\n" +
		"| - | - | - |\n" +
		"| `name` | text | Recipient name. |\n" +
		"| `locale` | text | Locale code. |\n"
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithAbility(t, abilityMD), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 || exp.Abilities[0].Inputs == nil {
		t.Fatal("no inputs parsed")
	}
	rows := exp.Abilities[0].Inputs.Rows
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(rows))
	}
	if rows[1]["Name"] != "`locale`" {
		t.Errorf("rows[1][Name]=%q, want \"`locale`\"", rows[1]["Name"])
	}
	if rows[1]["Type"] != "text" {
		t.Errorf("rows[1][Type]=%q, want \"text\"", rows[1]["Type"])
	}
}

// --- Scenario parsing tests ---

func makeSpecDirWithScenario(t *testing.T) string {
	t.Helper()
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "scenarios", "greeting", "scenario.md"),
		"## Greeting\n\nA greeting is produced.\n\n> `Greet`\n\n- result is non-empty\n- result contains the name\n")
	return dir
}

func TestExport_ScenarioParsed(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithScenario(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Scenarios) == 0 {
		t.Fatal("no scenarios parsed")
	}
}

func TestExport_ScenarioFields(t *testing.T) {
	var buf strings.Builder
	export.Export(makeSpecDirWithScenario(t), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	s := exp.Scenarios[0]
	if s.Heading != "Greeting" {
		t.Errorf("Heading=%q, want \"Greeting\"", s.Heading)
	}
	if s.Description != "A greeting is produced." {
		t.Errorf("Description=%q, want \"A greeting is produced.\"", s.Description)
	}
	if s.Trace != "Greet" {
		t.Errorf("Trace=%q, want \"Greet\"", s.Trace)
	}
	if len(s.Assertions) != 2 {
		t.Errorf("Assertions len=%d, want 2", len(s.Assertions))
	}
}

// --- Custom section and outputs note ---

func TestExport_AbilityCustomSection(t *testing.T) {
	abilityMD := "## Greet\n\nSays hello.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n_None._\n\n### Invariants\n\n_None._\n\n### Failure Modes\n\n_None._\n\n### Notes\n\nThis ability logs all invocations.\n"
	var buf strings.Builder
	export.Export(makeSpecDirWithAbility(t, abilityMD), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 {
		t.Fatal("no abilities")
	}
	a := exp.Abilities[0]
	if len(a.CustomSections) == 0 {
		t.Fatal("no custom sections")
	}
	if a.CustomSections[0].Heading != "Notes" {
		t.Errorf("CustomSections[0].Heading=%q, want \"Notes\"", a.CustomSections[0].Heading)
	}
	if !strings.Contains(a.CustomSections[0].Content, "logs all invocations") {
		t.Errorf("CustomSections[0].Content=%q missing expected text", a.CustomSections[0].Content)
	}
}

func TestExport_AbilityOutputsNote(t *testing.T) {
	abilityMD := "## Greet\n\nSays hello.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n| Name | Type | Description |\n| ---- | ---- | ----------- |\n| `result` | text | The greeting. |\n\nReturns empty string if name was absent.\n\n### Invariants\n\n_None._\n\n### Failure Modes\n\n_None._\n"
	var buf strings.Builder
	export.Export(makeSpecDirWithAbility(t, abilityMD), "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 {
		t.Fatal("no abilities")
	}
	if !strings.Contains(exp.Abilities[0].OutputsNote, "Returns empty string") {
		t.Errorf("OutputsNote=%q, want to contain \"Returns empty string\"", exp.Abilities[0].OutputsNote)
	}
}

// --- Spec invariants ---

func TestExport_SpecInvariants(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"),
		"## My Spec\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Summary:** A test spec.\n\n### Invariants\n\n- `name` must be non-empty\n- result is deterministic\n")
	var buf strings.Builder
	export.Export(dir, "", &buf)
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Invariants) != 2 {
		t.Errorf("Invariants len=%d, want 2", len(exp.Invariants))
	}
}

// --- Sub-ability parsing ---

func makeSpecDirWithSubAbility(t *testing.T) string {
	t.Helper()
	dir := makeSpecDir(t)
	writeFile(t, filepath.Join(dir, "abilities", "greet", "validate", "ability.md"),
		"## Validate\n\nValidates the input.\n\n### Inputs\n\n_None._\n\n### Outputs\n\n_None._\n\n### Invariants\n\n_None._\n\n### Failure Modes\n\n_None._\n")
	return dir
}

func TestExport_SubAbilityParsed(t *testing.T) {
	var buf strings.Builder
	_, err := export.Export(makeSpecDirWithSubAbility(t), "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var exp types.SpecExport
	json.Unmarshal([]byte(buf.String()), &exp)
	if len(exp.Abilities) == 0 {
		t.Fatal("no abilities")
	}
	if len(exp.Abilities[0].SubAbilities) == 0 {
		t.Fatal("no sub-abilities parsed")
	}
	if exp.Abilities[0].SubAbilities[0].Heading != "Validate" {
		t.Errorf("SubAbility Heading=%q, want \"Validate\"", exp.Abilities[0].SubAbilities[0].Heading)
	}
}
