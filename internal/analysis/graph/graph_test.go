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
	// Example scaffold creates: greet (Ability), greeting (Concept),
	// output-channel (Decision), happy-path (Scenario) → 4 nodes total.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NodeCount != 4 {
		t.Errorf("expected 4 nodes, got %d", result.NodeCount)
	}
}

// --- Example scaffold: edge count ---

func TestGraph_ExampleScaffold_EdgeCount(t *testing.T) {
	// greet →output→ greeting, happy-path →traces→ greet → 2 edges total.
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EdgeCount != 2 {
		t.Errorf("expected 2 edges, got %d", result.EdgeCount)
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
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "OutputChannel") {
		t.Errorf("expected Mermaid content to contain 'OutputChannel' for the decision node")
	}
}

func TestGraph_ExampleScaffold_ContainsScenarioNode(t *testing.T) {
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
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
	dir := scaffoldExample(t)
	result, err := graph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "|traces|") {
		t.Errorf("expected 'traces' edge in Mermaid content, content:\n%s", result.Content)
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
	if result.EdgeCount != 2 {
		t.Errorf("expected 2 edges (1 output + 1 traces, deduplicated), got %d\ncontent:\n%s",
			result.EdgeCount, result.Content)
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
