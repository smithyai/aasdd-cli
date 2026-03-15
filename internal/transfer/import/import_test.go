package import__test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	import_ "github.com/smithyai/aasdd-cli/internal/transfer/import"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// makeSnapshot creates a minimal SpecExport JSON file in a temp dir and returns its path.
func makeSnapshot(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{
			Heading:      "Test Spec",
			AASDDVersion: "v1",
			Version:      "0.1.0",
			Summary:      "A test spec.",
		},
		Abilities: []types.ParsedAbility{
			{
				Path:    "abilities/greet/ability.md",
				Heading: "Greet",
				Purpose: "Says hello.",
				Inputs: &types.Table{
					Headers: []string{"Name", "Type", "Description"},
					Rows: []map[string]string{
						{"Name": "`name`", "Type": "text", "Description": "Recipient name."},
					},
				},
			},
		},
		Scenarios: []types.ParsedScenario{
			{
				Path:        "scenarios/hello/scenario.md",
				Heading:     "Hello",
				Description: "A greeting is produced.",
				Trace:       "Greet",
				Assertions:  []string{"result is non-empty"},
			},
		},
		Concepts: []types.ParsedConcept{
			{
				Path:    "concepts/greeting/concept.md",
				Heading: "Greeting domain",
			},
		},
		Decisions: []types.ParsedDecision{
			{
				Path:        "decisions/lang/decision.md",
				Heading:     "Lang",
				Context:     "Need a language.",
				Requirement: "A language.",
				Decision:    "Go.",
			},
		},
	}
	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	tmp := t.TempDir()
	snapshotPath := filepath.Join(tmp, "snap.json")
	if err := os.WriteFile(snapshotPath, data, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	return snapshotPath
}

// --- Failure modes ---

func TestImport_SourceNotFound(t *testing.T) {
	_, err := import_.Import("/nonexistent-import-xyz.json", t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.SourceNotFound
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFound, got %T: %v", err, err)
	}
}

func TestImport_SourceNotFile(t *testing.T) {
	_, err := import_.Import(t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.SourceNotFile
	if !errors.As(err, &e) {
		t.Fatalf("expected SourceNotFile, got %T: %v", err, err)
	}
}

func TestImport_OutputNotEmpty_NonEmptyDir(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "existing.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(makeSnapshot(t), out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestImport_OutputNotEmpty_ExistingFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "output")
	if err := os.WriteFile(outPath, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(makeSnapshot(t), outPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.OutputNotEmpty
	if !errors.As(err, &e) {
		t.Fatalf("expected OutputNotEmpty, got %T: %v", err, err)
	}
}

func TestImport_ParseError_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(badFile, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := import_.Import(badFile, t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var e *import_.ParseError
	if !errors.As(err, &e) {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
}

// --- Success ---

func TestImport_FilesExistOnDisk(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPaths := []string{
		"spec.md",
		"abilities/greet/ability.md",
		"scenarios/hello/scenario.md",
		"concepts/greeting/concept.md",
		"decisions/lang/decision.md",
	}
	for _, rel := range expectedPaths {
		full := filepath.Join(out, filepath.FromSlash(rel))
		if _, statErr := os.Stat(full); os.IsNotExist(statErr) {
			t.Errorf("expected file missing: %s", rel)
		}
	}
}

func TestImport_AbilityContentPreserved(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(filepath.Join(out, "abilities", "greet", "ability.md"))
	if readErr != nil {
		t.Fatalf("read ability.md: %v", readErr)
	}
	content := string(data)
	if !contains(content, "Greet") {
		t.Error("ability.md should contain heading \"Greet\"")
	}
	if !contains(content, "Says hello.") {
		t.Error("ability.md should contain purpose text")
	}
}

func TestImport_FileCountInResult(t *testing.T) {
	out := t.TempDir()
	result, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// spec.md + ability.md + scenario.md + concept.md + decision.md = 5
	if result.FileCount != 5 {
		t.Errorf("FileCount=%d, want 5", result.FileCount)
	}
}

func TestImport_OutputPathInResult(t *testing.T) {
	out := t.TempDir()
	result, err := import_.Import(makeSnapshot(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputPath != out {
		t.Errorf("OutputPath=%q, want %q", result.OutputPath, out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- State Machine tests ---

func makeSnapshotWithStateMachine(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{
			Heading:      "Test Spec",
			AASDDVersion: "v1",
			Version:      "0.1.0",
			Summary:      "A test spec.",
		},
		Abilities: []types.ParsedAbility{
			{
				Path:    "abilities/greet/ability.md",
				Heading: "Greet",
				Purpose: "Says hello.",
			},
		},
		StateMachine: &types.ParsedStateMachine{
			Summary:      "Coordinates the lifecycle for a single request.",
			Diagram:      "```mermaid\nstateDiagram-v2\n    [*] --> Processing\n```",
			Orchestrator: "`HandleRequest` owns all transitions.",
			OrchestratorState: &types.Table{
				Headers: []string{"Name", "Type", "Description"},
				Rows: []map[string]string{
					{"Name": "`request`", "Type": "text", "Description": "The incoming request ID."},
				},
			},
			States: &types.Table{
				Headers: []string{"State", "Ability", "Description"},
				Rows: []map[string]string{
					{"State": "`Processing`", "Ability": "`DoWork`", "Description": "Processes the request."},
					{"State": "`Complete`", "Ability": "—", "Description": "Terminal success state."},
				},
			},
			Transitions: &types.Table{
				Headers: []string{"From", "To", "Trigger", "Data Passed Forward"},
				Rows: []map[string]string{
					{"From": "`Processing`", "To": "`Complete`", "Trigger": "done", "Data Passed Forward": "result"},
				},
			},
			TransitionRules: []string{"`Processing` must complete within the configured timeout."},
			ExceptionalFlows: []types.ExceptionalFlow{
				{Heading: "Timeout", Content: "If processing exceeds the deadline, the orchestrator forces a transition to `Failed`."},
			},
		},
	}
	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	tmp := t.TempDir()
	snapshotPath := filepath.Join(tmp, "snap.json")
	if err := os.WriteFile(snapshotPath, data, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	return snapshotPath
}

func TestImport_StateMachineFileCreated(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshotWithStateMachine(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	smPath := filepath.Join(out, "state-machine.md")
	if _, statErr := os.Stat(smPath); os.IsNotExist(statErr) {
		t.Fatal("state-machine.md was not created")
	}
}

func TestImport_StateMachineContainsHeading(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "## State Machine") {
		t.Error("state-machine.md should contain ## State Machine heading")
	}
}

func TestImport_StateMachineContainsSummary(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "Coordinates the lifecycle") {
		t.Error("state-machine.md should contain summary text")
	}
}

func TestImport_StateMachineContainsDiagram(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "```mermaid") {
		t.Error("state-machine.md should contain mermaid diagram")
	}
}

func TestImport_StateMachineContainsOrchestrator(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "### Orchestrator") {
		t.Error("state-machine.md should contain ### Orchestrator")
	}
	if !contains(string(data), "HandleRequest") {
		t.Error("state-machine.md should contain orchestrator text")
	}
}

func TestImport_StateMachineContainsOrchestratorState(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "#### Orchestrator-Managed State") {
		t.Error("state-machine.md should contain #### Orchestrator-Managed State")
	}
}

func TestImport_StateMachineContainsStates(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "### States") {
		t.Error("state-machine.md should contain ### States")
	}
}

func TestImport_StateMachineContainsTransitions(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "### Transitions") {
		t.Error("state-machine.md should contain ### Transitions")
	}
}

func TestImport_StateMachineContainsTransitionRules(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "### Transition Rules") {
		t.Error("state-machine.md should contain ### Transition Rules")
	}
}

func TestImport_StateMachineContainsExceptionalFlows(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithStateMachine(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "state-machine.md"))
	if !contains(string(data), "### Exceptional Flows") {
		t.Error("state-machine.md should contain ### Exceptional Flows")
	}
	if !contains(string(data), "#### Timeout") {
		t.Error("state-machine.md should contain #### Timeout sub-section")
	}
}

func TestImport_StateMachineFileCountIncluded(t *testing.T) {
	baseResult, _ := import_.Import(makeSnapshot(t), t.TempDir())
	smResult, _ := import_.Import(makeSnapshotWithStateMachine(t), t.TempDir())
	// state-machine.md adds 1 file, but SM snapshot has no scenarios/concepts/decisions
	// so compare: base has spec + ability + scenario + concept + decision = 5
	// SM snapshot has spec + ability + state-machine = 3
	if smResult.FileCount != 3 {
		t.Errorf("FileCount with state machine=%d, want 3", smResult.FileCount)
	}
	_ = baseResult
}

func TestImport_NoStateMachineWhenAbsent(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshot(t), out)
	smPath := filepath.Join(out, "state-machine.md")
	if _, statErr := os.Stat(smPath); !os.IsNotExist(statErr) {
		t.Error("state-machine.md should not exist when not in export")
	}
}

// --- renderConceptFile variants ---

func makeSnapshotWithConceptVariants(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{
			Heading:      "Test",
			AASDDVersion: "v1",
			Version:      "0.1.0",
			Summary:      "A test spec.",
		},
		Abilities: []types.ParsedAbility{{Heading: "Greet", Purpose: "Says hi."}},
		Concepts: []types.ParsedConcept{
			{
				Heading: "Greeting domain",
				Intro:   "Types for greetings.",
				Types: []types.ConceptType{
					{
						Name:              "GreetingResult",
						Description:       "The outcome.",
						PropertiesHeading: true,
						Properties: &types.Table{
							Headers: []string{"Name", "Type", "Description"},
							Rows:    []map[string]string{{"Name": "`message`", "Type": "text", "Description": "The greeting."}},
						},
					},
					{
						Name: "Status", // no description, no PropertiesHeading — bare table
						Properties: &types.Table{
							Headers: []string{"Name", "Type", "Description"},
							Rows:    []map[string]string{{"Name": "`ok`", "Type": "bool", "Description": "Success flag."}},
						},
					},
					{
						Name:        "Locale",
						Description: "Locale information.",
						Note:        "Only BCP-47 tags are supported.",
						Properties: &types.Table{
							Headers: []string{"Name", "Type", "Description"},
							Rows:    []map[string]string{{"Name": "`tag`", "Type": "text", "Description": "The locale tag."}},
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "snap.json")
	os.WriteFile(path, data, 0o644)
	return path
}

func TestImport_ConceptWithIntro(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithConceptVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "concepts", "greeting", "concept.md"))
	if !strings.Contains(string(data), "Types for greetings.") {
		t.Error("concept.md should contain intro text")
	}
}

func TestImport_ConceptType_WithPropertiesHeading(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithConceptVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "concepts", "greeting", "concept.md"))
	if !strings.Contains(string(data), "#### Properties") {
		t.Error("concept.md should contain #### Properties when PropertiesHeading=true")
	}
}

func TestImport_ConceptType_BareTable(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithConceptVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "concepts", "greeting", "concept.md"))
	if !strings.Contains(string(data), "### Status") {
		t.Error("concept.md should contain ### Status heading for bare-table type")
	}
}

func TestImport_ConceptType_WithNote(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithConceptVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "concepts", "greeting", "concept.md"))
	if !strings.Contains(string(data), "Only BCP-47 tags are supported.") {
		t.Error("concept.md should contain note text")
	}
}

func TestImport_ConceptType_WithDescription(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithConceptVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "concepts", "greeting", "concept.md"))
	if !strings.Contains(string(data), "The outcome.") {
		t.Error("concept.md should contain type description")
	}
}

// --- renderSpecFile with invariants ---

func makeSnapshotWithSpecInvariants(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{
			Heading:      "Test",
			AASDDVersion: "v1",
			Version:      "0.1.0",
			Summary:      "A test.",
			Invariants:   []string{"name must be non-empty", "result is deterministic"},
		},
		Abilities: []types.ParsedAbility{{Heading: "Greet", Purpose: "Says hi."}},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "snap.json")
	os.WriteFile(path, data, 0o644)
	return path
}

func TestImport_SpecFileInvariants(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithSpecInvariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "spec.md"))
	content := string(data)
	if !strings.Contains(content, "### Invariants") {
		t.Error("spec.md should contain ### Invariants")
	}
	if !strings.Contains(content, "name must be non-empty") {
		t.Error("spec.md should contain first invariant")
	}
}

// --- renderAbilityFile with outputs note and custom sections ---

func makeSnapshotWithAbilityVariants(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{Heading: "Test", AASDDVersion: "v1", Version: "0.1.0"},
		Abilities: []types.ParsedAbility{
			{
				Heading: "Greet",
				Purpose: "Says hello.",
				Outputs: &types.Table{
					Headers: []string{"Name", "Type", "Description"},
					Rows:    []map[string]string{{"Name": "`result`", "Type": "text", "Description": "The greeting."}},
				},
				OutputsNote: "Returns empty string if name was absent.",
				CustomSections: []types.CustomSection{
					{Heading: "Notes", Content: "This ability logs all invocations."},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "snap.json")
	os.WriteFile(path, data, 0o644)
	return path
}

func TestImport_AbilityWithOutputsNote(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithAbilityVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "abilities", "greet", "ability.md"))
	if !strings.Contains(string(data), "Returns empty string if name was absent.") {
		t.Error("ability.md should contain outputs note")
	}
}

func TestImport_AbilityWithCustomSection(t *testing.T) {
	out := t.TempDir()
	import_.Import(makeSnapshotWithAbilityVariants(t), out)
	data, _ := os.ReadFile(filepath.Join(out, "abilities", "greet", "ability.md"))
	content := string(data)
	if !strings.Contains(content, "### Notes") {
		t.Error("ability.md should contain ### Notes custom section")
	}
	if !strings.Contains(content, "logs all invocations") {
		t.Error("ability.md should contain custom section content")
	}
}

// --- Sub-ability written to disk ---

func makeSnapshotWithSubAbility(t *testing.T) string {
	t.Helper()
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{Heading: "Test", AASDDVersion: "v1", Version: "0.1.0"},
		Abilities: []types.ParsedAbility{
			{
				Heading: "Greet",
				Purpose: "Says hello.",
				SubAbilities: []types.ParsedAbility{
					{
						Heading: "Validate",
						Purpose: "Validates the input.",
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "snap.json")
	os.WriteFile(path, data, 0o644)
	return path
}

func TestImport_SubAbilityWritten(t *testing.T) {
	out := t.TempDir()
	_, err := import_.Import(makeSnapshotWithSubAbility(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	subPath := filepath.Join(out, "abilities", "greet", "validate", "ability.md")
	if _, statErr := os.Stat(subPath); os.IsNotExist(statErr) {
		t.Error("sub-ability file was not created")
	}
}

func TestImport_SubAbilityFileCountIncluded(t *testing.T) {
	out := t.TempDir()
	result, err := import_.Import(makeSnapshotWithSubAbility(t), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// spec.md + greet/ability.md + greet/validate/ability.md = 3
	if result.FileCount != 3 {
		t.Errorf("FileCount=%d, want 3", result.FileCount)
	}
}

// --- pascalToKebab via PascalCase and acronym headings ---

func TestImport_PascalCaseDecisionPath(t *testing.T) {
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{Heading: "Test", AASDDVersion: "v1", Version: "0.1.0"},
		Abilities:      []types.ParsedAbility{{Heading: "Greet", Purpose: "Says hi."}},
		Decisions: []types.ParsedDecision{
			{Heading: "OutputChannel", Context: "ctx", Requirement: "req", Decision: "dec"},
		},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	snapPath := filepath.Join(tmp, "snap.json")
	os.WriteFile(snapPath, data, 0o644)
	out := t.TempDir()
	import_.Import(snapPath, out)
	expected := filepath.Join(out, "decisions", "output-channel", "decision.md")
	if _, statErr := os.Stat(expected); os.IsNotExist(statErr) {
		t.Error("decision file not created at expected kebab-case path: decisions/output-channel/decision.md")
	}
}

func TestImport_AcronymDecisionPath(t *testing.T) {
	exp := types.SpecExport{
		ParsedSpecFile: types.ParsedSpecFile{Heading: "Test", AASDDVersion: "v1", Version: "0.1.0"},
		Abilities:      []types.ParsedAbility{{Heading: "Greet", Purpose: "Says hi."}},
		Decisions: []types.ParsedDecision{
			{Heading: "HTTPTransport", Context: "ctx", Requirement: "req", Decision: "dec"},
		},
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	tmp := t.TempDir()
	snapPath := filepath.Join(tmp, "snap.json")
	os.WriteFile(snapPath, data, 0o644)
	out := t.TempDir()
	import_.Import(snapPath, out)
	expected := filepath.Join(out, "decisions", "http-transport", "decision.md")
	if _, statErr := os.Stat(expected); os.IsNotExist(statErr) {
		t.Error("decision file not created at expected kebab-case path for acronym: decisions/http-transport/decision.md")
	}
}
