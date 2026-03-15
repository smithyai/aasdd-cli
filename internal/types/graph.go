package types

// GraphFormat is the output format for a rendered graph.
type GraphFormat string

const (
	GraphFormatMermaid GraphFormat = "Mermaid"
	GraphFormatDOT     GraphFormat = "DOT"
)

// NodeKind classifies the construct a graph node represents.
type NodeKind string

const (
	NodeKindAbility      NodeKind = "Ability"
	NodeKindConcept      NodeKind = "Concept"
	NodeKindDecision     NodeKind = "Decision"
	NodeKindScenario     NodeKind = "Scenario"
	NodeKindStateMachine NodeKind = "StateMachine"
)

// GraphOptions controls optional parameters for the Graph ability.
type GraphOptions struct {
	// Include lists supplementary node kinds to add beyond the default
	// ability-anchored graph (abilities + concepts). Valid values:
	// NodeKindScenario, NodeKindDecision, NodeKindStateMachine.
	Include []NodeKind
	// Depth limits how many levels of sub-abilities are walked. ≤0 means unlimited.
	Depth int
	// Root re-roots the graph at the ability whose last path segment matches
	// this name (case-insensitive). Empty means the spec root.
	Root string
}

// GraphNode is a single node in the spec dependency graph.
type GraphNode struct {
	ID    string   `json:"id"`
	Kind  NodeKind `json:"kind"`
	Label string   `json:"label"`
}

// GraphEdge is a directed relationship between two nodes.
type GraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// GraphResult is the outcome of generating a spec dependency graph.
type GraphResult struct {
	Target    SpecTarget  `json:"target"`
	Format    GraphFormat `json:"format"`
	Content   string      `json:"content"`
	NodeCount int         `json:"node_count"`
	EdgeCount int         `json:"edge_count"`
}
