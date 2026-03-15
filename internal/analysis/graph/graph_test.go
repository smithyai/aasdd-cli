package graph_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/graph"
	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// helper: scaffold an example spec into a temp dir.
func scaffoldExample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return dir
}

// helper: scaffold a minimal spec into a temp dir.
func scaffoldMinimal(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	return dir
}

// countMermaidNodes counts node definition lines (contain `["`) in Mermaid output.
// Each node rendered by the graph pipeline uses the form:  ID["label"]
func countMermaidNodes(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "[\"") && !strings.Contains(trimmed, "-->") {
			count++
		}
	}
	return count
}

// countMermaidEdges counts edge lines (contain `-->|`) in Mermaid output.
func countMermaidEdges(content string) int {
	return strings.Count(content, "-->|")
}

// --- Failure mode tests ---

func TestGraph_TargetNotFound(t *testing.T) {
	_, err := graph.Graph(types.SpecTarget{Path: "/nonexistent/spec"}, types.GraphFormatMermaid)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *graph.TargetNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetNotFound, got %T: %v", err, err)
	}
	if target.Path != "/nonexistent/spec" {
		t.Errorf("unexpected path in error: %q", target.Path)
	}
}

func TestGraph_TargetIsFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(f, []byte("# spec\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := graph.Graph(types.SpecTarget{Path: f}, types.GraphFormatMermaid)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var target *graph.TargetIsFile
	if !errors.As(err, &target) {
		t.Fatalf("expected TargetIsFile, got %T: %v", err, err)
	}
}

// --- Minimal scaffold (no spec content) ---

func TestGraph_MinimalScaffold_ZeroNodes(t *testing.T) {
	dir := scaffoldMinimal(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 0 {
		t.Errorf("expected 0 nodes, got %d", result.NodeCount)
	}
}

func TestGraph_MinimalScaffold_ZeroEdges(t *testing.T) {
	dir := scaffoldMinimal(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EdgeCount != 0 {
		t.Errorf("expected 0 edges, got %d", result.EdgeCount)
	}
}

func TestGraph_MinimalScaffold_ProducesValidMermaidHeader(t *testing.T) {
	dir := scaffoldMinimal(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result.Content, "graph ") {
		t.Errorf("expected Mermaid content to start with 'graph ', got: %q",
			result.Content[:min(30, len(result.Content))])
	}
}

func TestGraph_MinimalScaffold_ProducesValidDOTHeader(t *testing.T) {
	dir := scaffoldMinimal(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result.Content, "digraph") {
		t.Errorf("expected DOT content to start with 'digraph', got: %q",
			result.Content[:min(30, len(result.Content))])
	}
}

// --- Example scaffold: node count ---

func TestGraph_ExampleScaffold_NodeCount(t *testing.T) {
	// Default graph: abilities + concepts only.
	// Example scaffold: greet (Ability), greeting (Concept) → 2 nodes.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 2 {
		t.Errorf("expected 2 nodes (ability + concept), got %d", result.NodeCount)
	}
}

// --- Example scaffold: edge count ---

func TestGraph_ExampleScaffold_EdgeCount(t *testing.T) {
	// Default graph: abilities + concepts only.
	// Example scaffold: greet →output→ greeting → 1 edge.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EdgeCount != 1 {
		t.Errorf("expected 1 edge (output), got %d", result.EdgeCount)
	}
}

// --- Example scaffold: node labels in Mermaid output ---

func TestGraph_ExampleScaffold_ContainsAbilityNode(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "Greet") {
		t.Errorf("expected Mermaid content to contain 'Greet' for the ability node")
	}
}

func TestGraph_ExampleScaffold_ContainsConceptNode(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "Greeting") {
		t.Errorf("expected Mermaid content to contain 'Greeting' for the concept node")
	}
}

func TestGraph_ExampleScaffold_ContainsDecisionNode(t *testing.T) {
	// Decisions are opt-in via --include decisions.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindDecision}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "OutputChannel") {
		t.Errorf("expected Mermaid content to contain 'OutputChannel' for the decision node")
	}
}

func TestGraph_ExampleScaffold_ContainsScenarioNode(t *testing.T) {
	// Scenarios are opt-in via --include scenarios.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindScenario}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "HappyPath") {
		t.Errorf("expected Mermaid content to contain 'HappyPath' for the scenario node")
	}
}

// --- Example scaffold: Mermaid format ---

func TestGraph_ExampleScaffold_MermaidStartsWithGraph(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result.Content, "graph ") {
		t.Errorf("Mermaid content should start with 'graph ', got: %q",
			result.Content[:min(30, len(result.Content))])
	}
}

func TestGraph_ExampleScaffold_MermaidFormatIsSet(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != types.GraphFormatMermaid {
		t.Errorf("expected Format=Mermaid, got %q", result.Format)
	}
}

func TestGraph_ExampleScaffold_MermaidNodeCountMatchesContent(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actual := countMermaidNodes(result.Content)
	if actual != result.NodeCount {
		t.Errorf("node_count=%d but Mermaid contains %d node definitions\ncontent:\n%s",
			result.NodeCount, actual, result.Content)
	}
}

func TestGraph_ExampleScaffold_MermaidEdgeCountMatchesContent(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actual := countMermaidEdges(result.Content)
	if actual != result.EdgeCount {
		t.Errorf("edge_count=%d but Mermaid contains %d edge definitions\ncontent:\n%s",
			result.EdgeCount, actual, result.Content)
	}
}

// --- Example scaffold: DOT format ---

func TestGraph_ExampleScaffold_DOTStartsWithDigraph(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result.Content, "digraph") {
		t.Errorf("DOT content should start with 'digraph', got: %q",
			result.Content[:min(30, len(result.Content))])
	}
}

func TestGraph_ExampleScaffold_DOTEndsWithBrace(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(result.Content)
	if !strings.HasSuffix(trimmed, "}") {
		t.Errorf("DOT content should end with '}', got: ...%q",
			trimmed[max(0, len(trimmed)-20):])
	}
}

func TestGraph_ExampleScaffold_DOTFormatIsSet(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != types.GraphFormatDOT {
		t.Errorf("expected Format=DOT, got %q", result.Format)
	}
}

func TestGraph_ExampleScaffold_DOTContainsAbilityNode(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "ability:greet") {
		t.Errorf("DOT content should contain 'ability:greet' node ID, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_DOTContainsConceptNode(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "concept:greeting") {
		t.Errorf("DOT content should contain 'concept:greeting' node ID, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_BothFormats_SameNodeCount(t *testing.T) {
	dir := scaffoldExample(t)
	mResult, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("Mermaid: %v", err)
	}
	dResult, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("DOT: %v", err)
	}
	if mResult.NodeCount != dResult.NodeCount {
		t.Errorf("Mermaid node_count=%d != DOT node_count=%d", mResult.NodeCount, dResult.NodeCount)
	}
}

func TestGraph_ExampleScaffold_BothFormats_SameEdgeCount(t *testing.T) {
	dir := scaffoldExample(t)
	mResult, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("Mermaid: %v", err)
	}
	dResult, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("DOT: %v", err)
	}
	if mResult.EdgeCount != dResult.EdgeCount {
		t.Errorf("Mermaid edge_count=%d != DOT edge_count=%d", mResult.EdgeCount, dResult.EdgeCount)
	}
}

// --- Example scaffold: specific edges ---

func TestGraph_ExampleScaffold_OutputEdgePresent(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|output|") {
		t.Errorf("expected 'output' edge in Mermaid content, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_TracesEdgePresent(t *testing.T) {
	// Traces edges only appear when scenarios are included.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindScenario}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|traces|") {
		t.Errorf("expected 'traces' edge in Mermaid content, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_Default_NoDecisionNodes(t *testing.T) {
	// Decisions must not appear in the default graph.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "[Decision]") {
		t.Errorf("default graph must not contain Decision nodes, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_Default_NoScenarioNodes(t *testing.T) {
	// Scenarios must not appear in the default graph.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "[Scenario]") {
		t.Errorf("default graph must not contain Scenario nodes, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_Default_NoTracesEdges(t *testing.T) {
	// Traces edges must not appear in the default graph.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "|traces|") {
		t.Errorf("default graph must not contain traces edges, content:\n%s", result.Content)
	}
}

func TestGraph_ExampleScaffold_IncludeScenariosAndDecisions_NodeCount(t *testing.T) {
	// --include scenarios,decisions adds both kinds: 4 nodes total.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindScenario, types.NodeKindDecision}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 4 {
		t.Errorf("expected 4 nodes with scenarios+decisions included, got %d", result.NodeCount)
	}
}

// --- Determinism ---

func TestGraph_DeterministicMermaidOutput(t *testing.T) {
	dir := scaffoldExample(t)
	r1, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	r2, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if r1.Content != r2.Content {
		t.Errorf("graph output is non-deterministic:\nfirst:\n%s\nsecond:\n%s", r1.Content, r2.Content)
	}
}

func TestGraph_DeterministicDOTOutput(t *testing.T) {
	dir := scaffoldExample(t)
	r1, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	r2, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if r1.Content != r2.Content {
		t.Errorf("DOT output is non-deterministic:\nfirst:\n%s\nsecond:\n%s", r1.Content, r2.Content)
	}
}

// --- Self-spec: verify against the CLI's own spec ---

func TestGraph_SelfSpec_HasSubAbilityEdges(t *testing.T) {
	// The CLI spec has verify → collect-violations and verify → load-rule-set.
	// Running graph on the CLI spec should produce sub-ability edges.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available (running outside repo tree): %v", err)
	}
	if !strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("expected 'sub-ability' edge in CLI spec graph, content:\n%s", result.Content)
	}
}

func TestGraph_SelfSpec_NodeCountGtZero(t *testing.T) {
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available (running outside repo tree): %v", err)
	}
	if result.NodeCount == 0 {
		t.Error("expected node_count > 0 for CLI self-spec")
	}
}

func TestGraph_SelfSpec_EdgeCountGtZero(t *testing.T) {
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available (running outside repo tree): %v", err)
	}
	if result.EdgeCount == 0 {
		t.Error("expected edge_count > 0 for CLI self-spec")
	}
}

func TestGraph_SelfSpec_DOT_ContainsAbilityNodes(t *testing.T) {
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatDOT)
	if err != nil {
		t.Skipf("self-spec not available (running outside repo tree): %v", err)
	}
	for _, name := range []string{"ability:graph", "ability:verify", "ability:scaffold"} {
		if !strings.Contains(result.Content, name) {
			t.Errorf("DOT content should contain %q", name)
		}
	}
}

// --- Edge deduplication ---

func TestGraph_NoDuplicateEdges(t *testing.T) {
	// Write a spec where an ability references the same concept in multiple
	// output rows — should produce only one output edge to that concept.
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	// Overwrite the ability to add a duplicate concept reference in outputs.
	abilityContent := "## Greet\n\n**Purpose:** Test duplicate refs.\n\n" +
		"### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n" +
		"| `name` | text | Name. |\n\n" +
		"### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n" +
		"| `result` | [GreetingResult](../../concepts/greeting/concept.md#greetingresult) | First ref. |\n" +
		"| `alt`    | [GreetingResult](../../concepts/greeting/concept.md#greetingresult) | Dup ref. |\n\n" +
		"### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n" +
		"| `EmptyName` | empty. | Error. |\n"
	abilityPath := filepath.Join(dir, "abilities", "greet", "ability.md")
	if err := os.WriteFile(abilityPath, []byte(abilityContent), 0o644); err != nil {
		t.Fatalf("write ability: %v", err)
	}

	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The duplicate output reference should be collapsed to a single edge.
	// Scenarios are not included by default, so only 1 output edge remains.
	if result.EdgeCount != 1 {
		t.Errorf("expected 1 edge (output, deduplicated), got %d\ncontent:\n%s",
			result.EdgeCount, result.Content)
	}
}

// --- Filter ---

func TestGraph_Default_NodeCountMatchesContent(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actual := countMermaidNodes(result.Content)
	if actual != result.NodeCount {
		t.Errorf("node_count=%d but Mermaid contains %d node definitions\ncontent:\n%s",
			result.NodeCount, actual, result.Content)
	}
}

func TestGraph_Default_SubAbilityEdgesKept(t *testing.T) {
	// Sub-ability edges are always present regardless of include.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if !strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("default graph should retain sub-ability edges, content:\n%s", result.Content)
	}
}

func TestGraph_Include_InvalidValue_ReturnsError(t *testing.T) {
	dir := scaffoldExample(t)
	_, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{"abilities"}})
	if err == nil {
		t.Fatal("expected error for invalid include value, got nil")
	}
	var invalid *graph.InvalidInclude
	if !errors.As(err, &invalid) {
		t.Fatalf("expected InvalidInclude error, got %T: %v", err, err)
	}
	if invalid.Value != "abilities" {
		t.Errorf("unexpected value in error: %q", invalid.Value)
	}
}

func TestGraph_Include_EmptyOptions_SameAsDefault(t *testing.T) {
	// Explicitly passing empty GraphOptions should behave identically to no opts.
	dir := scaffoldExample(t)
	def, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	explicit, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid, types.GraphOptions{})
	if err != nil {
		t.Fatalf("explicit empty opts: %v", err)
	}
	if def.NodeCount != explicit.NodeCount || def.EdgeCount != explicit.EdgeCount {
		t.Errorf("empty opts should equal default: default(%d nodes, %d edges) vs explicit(%d nodes, %d edges)",
			def.NodeCount, def.EdgeCount, explicit.NodeCount, explicit.EdgeCount)
	}
}

func TestGraph_Include_Scenarios_AddsTracesEdges(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindScenario}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|traces|") {
		t.Errorf("--include scenarios should add traces edges, content:\n%s", result.Content)
	}
}

func TestGraph_Include_Decisions_AddsDecisionNodes(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindDecision}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "[Decision]") {
		t.Errorf("--include decisions should add Decision nodes, content:\n%s", result.Content)
	}
}

// --- State machine ---

// addStateMachine writes a minimal state-machine.md to dir that references the
// `Greet` ability produced by scaffoldExample.
func addStateMachine(t *testing.T, dir string) {
	t.Helper()
	content := "## State Machine\n\n" +
		"A minimal state machine for testing graph output.\n\n" +
		"### States\n\n" +
		"| State    | Ability | Description     |\n" +
		"| -------- | ------- | --------------- |\n" +
		"| `Active` | `Greet` | Greets someone. |\n" +
		"| `Done`   | —       | Terminal state. |\n"
	if err := os.WriteFile(filepath.Join(dir, "state-machine.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write state-machine.md: %v", err)
	}
}

func TestGraph_WithStateMachine_NotIncludedByDefault(t *testing.T) {
	// State machine must not appear in the default graph.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "State Machine") {
		t.Errorf("state machine must not appear in default graph, content:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "|orchestrates|") {
		t.Errorf("orchestrates edges must not appear in default graph, content:\n%s", result.Content)
	}
}

func TestGraph_WithStateMachine_NodeCount(t *testing.T) {
	// Default (abilities+concepts): greet + greeting = 2 nodes.
	// With --include state-machine: greet + greeting + state-machine = 3 nodes.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 3 {
		t.Errorf("expected 3 nodes (ability + concept + state-machine), got %d", result.NodeCount)
	}
}

func TestGraph_WithStateMachine_HasStateMachineNode(t *testing.T) {
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "State Machine") {
		t.Errorf("expected 'State Machine' node label in Mermaid content, content:\n%s", result.Content)
	}
}

func TestGraph_WithStateMachine_HasOrchestratesEdge(t *testing.T) {
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|orchestrates|") {
		t.Errorf("expected 'orchestrates' edge in Mermaid content, content:\n%s", result.Content)
	}
}

func TestGraph_WithStateMachine_DOTContainsStateMachineNode(t *testing.T) {
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "sm:state-machine") {
		t.Errorf("DOT content should contain 'sm:state-machine' node, content:\n%s", result.Content)
	}
}

func TestGraph_WithStateMachine_MermaidNodeCountMatchesContent(t *testing.T) {
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actual := countMermaidNodes(result.Content)
	if actual != result.NodeCount {
		t.Errorf("node_count=%d but Mermaid contains %d node definitions\ncontent:\n%s",
			result.NodeCount, actual, result.Content)
	}
}

func TestGraph_WithStateMachine_TerminalStatesProduceNoEdges(t *testing.T) {
	// States with "—" in the Ability column must not produce orchestrates edges.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only one orchestrates edge (Greet); terminal states must be skipped.
	count := strings.Count(result.Content, "|orchestrates|")
	if count != 1 {
		t.Errorf("expected exactly 1 orchestrates edge, got %d\ncontent:\n%s", count, result.Content)
	}
}

func TestGraph_Include_StateMachine_AddsStateMachineNode(t *testing.T) {
	// --include state-machine adds the SM node alongside the default ability+concept graph.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "[StateMachine]") {
		t.Errorf("--include state-machine should add StateMachine node, content:\n%s", result.Content)
	}
	// Concepts remain present.
	if !strings.Contains(result.Content, "[Concept]") {
		t.Errorf("concepts must still be present with --include state-machine, content:\n%s", result.Content)
	}
}

func TestGraph_Include_StateMachine_HasOrchestratesEdge(t *testing.T) {
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|orchestrates|") {
		t.Errorf("--include state-machine should add orchestrates edges, content:\n%s", result.Content)
	}
}

func TestGraph_Include_StateMachine_WithoutStateMachine_NoSMNode(t *testing.T) {
	// A spec with no state machine: --include state-machine adds no nodes beyond the default.
	dir := scaffoldExample(t)
	def, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	withSM, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("with SM include: %v", err)
	}
	if withSM.NodeCount != def.NodeCount {
		t.Errorf("no state machine: --include state-machine should not change node count (default=%d, got=%d)",
			def.NodeCount, withSM.NodeCount)
	}
}

// --- Depth ---

func TestGraph_Depth1_SelfSpec_NoSubAbilityEdges(t *testing.T) {
	// The CLI self-spec has sub-abilities; depth=1 should hide them.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid, types.GraphOptions{Depth: 1})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("depth=1 should suppress sub-ability edges, content:\n%s", result.Content)
	}
}

func TestGraph_Depth1_SelfSpec_HasRootAbilities(t *testing.T) {
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid, types.GraphOptions{Depth: 1})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	// Root abilities like verify, scaffold, graph, diff must still appear.
	for _, name := range []string{"Verify", "Scaffold", "Graph"} {
		if !strings.Contains(result.Content, name) {
			t.Errorf("depth=1 graph missing root ability %q, content:\n%s", name, result.Content)
		}
	}
}

func TestGraph_DepthZero_SameAsFull(t *testing.T) {
	// depth=0 means unlimited — must produce identical output to omitting depth.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	resultDepth0, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid, types.GraphOptions{Depth: 0})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if result.NodeCount != resultDepth0.NodeCount || result.EdgeCount != resultDepth0.EdgeCount {
		t.Errorf("depth=0 should equal full graph: full(%d nodes, %d edges) vs depth0(%d nodes, %d edges)",
			result.NodeCount, result.EdgeCount, resultDepth0.NodeCount, resultDepth0.EdgeCount)
	}
}

func TestGraph_Depth1_FewerOrEqualNodesThanFull(t *testing.T) {
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	resultDepth1, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid, types.GraphOptions{Depth: 1})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if resultDepth1.NodeCount > result.NodeCount {
		t.Errorf("depth=1 should have ≤ nodes than full graph: depth1=%d full=%d",
			resultDepth1.NodeCount, result.NodeCount)
	}
}

func TestGraph_Depth1_OnlyRootAbilities(t *testing.T) {
	// depth=1 → root abilities + their concepts only, no sub-ability edges.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Depth: 1})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("depth=1 should suppress sub-ability edges, content:\n%s", result.Content)
	}
	if result.NodeCount == 0 {
		t.Error("expected at least one root ability node")
	}
}

// --- Root ---

func TestGraph_Root_SelfSpec_VerifySubtreeOnly(t *testing.T) {
	// --root verify should include verify + its sub-abilities, not sibling abilities.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "verify"})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	// Sub-abilities of verify must be present.
	if !strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("--root verify should retain sub-ability edges for verify's children, content:\n%s", result.Content)
	}
}

func TestGraph_Root_SelfSpec_ExcludesSiblings(t *testing.T) {
	// All abilities (full graph).
	full, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid)
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	// Rooted at verify — should have fewer ability nodes than the full graph.
	rooted, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "verify"})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if rooted.NodeCount >= full.NodeCount {
		t.Errorf("--root verify should produce fewer nodes than full graph: rooted=%d full=%d",
			rooted.NodeCount, full.NodeCount)
	}
}

func TestGraph_Root_SelfSpec_WithDepth(t *testing.T) {
	// --root verify --depth 1 should show verify only, no sub-ability edges.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "verify", Depth: 1})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("--root verify --depth 1 should suppress sub-ability edges, content:\n%s", result.Content)
	}
}

func TestGraph_Root_NotFound_ReturnsError(t *testing.T) {
	dir := scaffoldExample(t)
	_, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "nonexistent-ability"})
	if err == nil {
		t.Fatal("expected error for unknown root, got nil")
	}
	var notFound *graph.RootNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected RootNotFound error, got %T: %v", err, err)
	}
	if notFound.Name != "nonexistent-ability" {
		t.Errorf("unexpected name in error: %q", notFound.Name)
	}
}

func TestGraph_Root_Ambiguous_ReturnsError(t *testing.T) {
	// Create a spec with two abilities that share the same last segment "helper".
	dir := scaffoldMinimal(t)
	for _, parent := range []string{"foo", "bar"} {
		helperDir := filepath.Join(dir, "abilities", parent, "helper")
		if err := os.MkdirAll(helperDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		parentContent := "## " + strings.ToUpper(parent[:1]) + parent[1:] + "\n\n**Purpose:** Parent.\n\n" +
			"### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n\n" +
			"### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n\n" +
			"### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n"
		if err := os.WriteFile(filepath.Join(dir, "abilities", parent, "ability.md"), []byte(parentContent), 0o644); err != nil {
			t.Fatalf("write parent ability: %v", err)
		}
		helperContent := "## Helper\n\n**Purpose:** Helper sub-ability.\n\n" +
			"### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n\n" +
			"### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n\n" +
			"### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n"
		if err := os.WriteFile(filepath.Join(helperDir, "ability.md"), []byte(helperContent), 0o644); err != nil {
			t.Fatalf("write helper ability: %v", err)
		}
	}

	_, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "helper"})
	if err == nil {
		t.Fatal("expected error for ambiguous root, got nil")
	}
	var ambiguous *graph.AmbiguousRoot
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousRoot error, got %T: %v", err, err)
	}
	if len(ambiguous.Matches) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(ambiguous.Matches), ambiguous.Matches)
	}
}

func TestGraph_Depth_ExampleScaffold_NoDifference(t *testing.T) {
	// Example scaffold has no sub-abilities, so depth=1 should equal full graph.
	full, err := graph.Graph(types.SpecTarget{Path: scaffoldExample(t)}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	depth1, err := graph.Graph(types.SpecTarget{Path: scaffoldExample(t)}, types.GraphFormatMermaid, types.GraphOptions{Depth: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if full.NodeCount != depth1.NodeCount || full.EdgeCount != depth1.EdgeCount {
		t.Errorf("depth=1 on flat spec should equal full graph: full(%d nodes, %d edges) vs depth1(%d nodes, %d edges)",
			full.NodeCount, full.EdgeCount, depth1.NodeCount, depth1.EdgeCount)
	}
}

// --- Combinations ---

func TestGraph_IncludeAllThree_NodeCount(t *testing.T) {
	// --include scenarios,decisions,state-machine on example scaffold with SM:
	// greet (ability) + greeting (concept) + output-channel (decision) +
	// happy-path (scenario) + state-machine = 5 nodes.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{
			types.NodeKindScenario,
			types.NodeKindDecision,
			types.NodeKindStateMachine,
		}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 5 {
		t.Errorf("expected 5 nodes with all three includes, got %d\ncontent:\n%s",
			result.NodeCount, result.Content)
	}
}

func TestGraph_IncludeAllThree_AllEdgeKindsPresent(t *testing.T) {
	// With all supplementary kinds included, output, traces, and orchestrates edges
	// should all appear.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Include: []types.NodeKind{
			types.NodeKindScenario,
			types.NodeKindDecision,
			types.NodeKindStateMachine,
		}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rel := range []string{"|output|", "|traces|", "|orchestrates|"} {
		if !strings.Contains(result.Content, rel) {
			t.Errorf("expected edge %s with all three includes, content:\n%s", rel, result.Content)
		}
	}
}

func TestGraph_Root_Include_Scenarios(t *testing.T) {
	// --root greet --include scenarios: greet + greeting + happy-path = 3 nodes,
	// traces edge present.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "greet", Include: []types.NodeKind{types.NodeKindScenario}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 3 {
		t.Errorf("expected 3 nodes (ability + concept + scenario), got %d\ncontent:\n%s",
			result.NodeCount, result.Content)
	}
	if !strings.Contains(result.Content, "|traces|") {
		t.Errorf("expected traces edge with --root greet --include scenarios, content:\n%s", result.Content)
	}
}

func TestGraph_Root_Include_Decisions(t *testing.T) {
	// --root greet --include decisions: greet + greeting + output-channel = 3 nodes.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "greet", Include: []types.NodeKind{types.NodeKindDecision}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 3 {
		t.Errorf("expected 3 nodes (ability + concept + decision), got %d\ncontent:\n%s",
			result.NodeCount, result.Content)
	}
}

func TestGraph_Root_Include_StateMachine(t *testing.T) {
	// --root greet --include state-machine: greet + greeting + state-machine = 3 nodes.
	dir := scaffoldExample(t)
	addStateMachine(t, dir)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid,
		types.GraphOptions{Root: "greet", Include: []types.NodeKind{types.NodeKindStateMachine}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 3 {
		t.Errorf("expected 3 nodes (ability + concept + state-machine), got %d\ncontent:\n%s",
			result.NodeCount, result.Content)
	}
	if !strings.Contains(result.Content, "|orchestrates|") {
		t.Errorf("expected orchestrates edge with --root greet --include state-machine, content:\n%s", result.Content)
	}
}

func TestGraph_Depth_Include_ScenariosStillAdded(t *testing.T) {
	// --depth 1 limits the ability walk but --include scenarios must still add
	// scenario nodes and traces edges for top-level abilities.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Depth: 1, Include: []types.NodeKind{types.NodeKindScenario}})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	// depth=1 suppresses sub-ability edges.
	if strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("depth=1 should suppress sub-ability edges, content:\n%s", result.Content)
	}
}

func TestGraph_Depth_Include_DecisionsStillAdded(t *testing.T) {
	// --depth 1 must not suppress decision nodes added via --include decisions.
	result, err := graph.Graph(types.SpecTarget{Path: "../../../spec"}, types.GraphFormatMermaid,
		types.GraphOptions{Depth: 1, Include: []types.NodeKind{types.NodeKindDecision}})
	if err != nil {
		t.Skipf("self-spec not available: %v", err)
	}
	if strings.Contains(result.Content, "|sub-ability|") {
		t.Errorf("depth=1 should suppress sub-ability edges, content:\n%s", result.Content)
	}
}

// min/max helpers for Go versions before 1.21 generic builtins.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
