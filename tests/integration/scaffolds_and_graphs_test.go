package integration_test

import (
	"strings"
	"testing"

	gph "github.com/smithyai/aasdd-cli/internal/analysis/graph"
	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScenario_ScaffoldsAndGraphs(t *testing.T) {
	// Scaffold uses example=true so the spec has abilities, concepts,
	// decisions, and scenarios for the graph to traverse.
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	result, err := gph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatMermaid)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}

	if result.NodeCount == 0 {
		t.Error("Graph.result.node_count should be greater than zero")
	}
	if result.EdgeCount == 0 {
		t.Error("Graph.result.edge_count should be greater than zero")
	}
	if result.Content == "" {
		t.Error("Graph.result.content should be non-empty")
	}
	if !strings.HasPrefix(result.Content, "graph ") {
		t.Errorf("Mermaid content should start with 'graph ', got: %q",
			result.Content[:min20(result.Content)])
	}
}

func TestScenario_ScaffoldsAndGraphsInDOTFormat(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v1", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	result, err := gph.Graph(types.SpecTarget{Path: dir}, types.GraphFormatDOT)
	if err != nil {
		t.Fatalf("graph (DOT): %v", err)
	}

	if result.NodeCount == 0 {
		t.Error("Graph.result.node_count should be greater than zero")
	}
	if result.EdgeCount == 0 {
		t.Error("Graph.result.edge_count should be greater than zero")
	}
	if !strings.HasPrefix(result.Content, "digraph") {
		t.Errorf("DOT content should start with 'digraph', got: %q",
			result.Content[:min20(result.Content)])
	}
}

func min20(s string) int {
	if len(s) < 20 {
		return len(s)
	}
	return 20
}
