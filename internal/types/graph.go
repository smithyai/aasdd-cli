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
	NodeKindAbility  NodeKind = "Ability"
	NodeKindConcept  NodeKind = "Concept"
	NodeKindDecision NodeKind = "Decision"
	NodeKindScenario NodeKind = "Scenario"
)

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
