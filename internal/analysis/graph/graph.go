// Package graph implements the Graph ability.
package graph

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	"github.com/smithyai/aasdd-cli/internal/types"
)

// TargetNotFound is returned when target.path does not exist.
type TargetNotFound struct{ Path string }

func (e *TargetNotFound) Error() string {
	return fmt.Sprintf("target not found: %q", e.Path)
}

// TargetIsFile is returned when target.path is a file, not a directory.
type TargetIsFile struct{ Path string }

func (e *TargetIsFile) Error() string {
	return fmt.Sprintf("target is a file, not a directory: %q", e.Path)
}

// ReadError is returned when a file under target cannot be read.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("read error at %q: %v", e.Path, e.Err)
}

func (e *ReadError) Unwrap() error { return e.Err }

// Graph generates a dependency graph for the spec rooted at target.
func Graph(target types.SpecTarget, format types.GraphFormat) (types.GraphResult, error) {
	empty := types.GraphResult{}

	info, err := os.Stat(target.Path)
	if err != nil {
		return empty, &TargetNotFound{Path: target.Path}
	}
	if !info.IsDir() {
		return empty, &TargetIsFile{Path: target.Path}
	}

	spec, _, err := export.LoadSpec(target.Path)
	if err != nil {
		return empty, &ReadError{Path: target.Path, Err: err}
	}

	nodes, edges := buildGraph(spec)

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		return edges[i].To < edges[j].To
	})

	var content string
	switch format {
	case types.GraphFormatDOT:
		content = renderDOT(nodes, edges)
	default:
		content = renderMermaid(nodes, edges)
	}

	return types.GraphResult{
		Target:    target,
		Format:    format,
		Content:   content,
		NodeCount: len(nodes),
		EdgeCount: len(edges),
	}, nil
}

// buildGraph constructs the full set of nodes and edges from a loaded spec.
func buildGraph(spec types.SpecExport) ([]types.GraphNode, []types.GraphEdge) {
	var nodes []types.GraphNode
	var edges []types.GraphEdge

	// headingToID maps lowercased ability heading → ability node ID.
	headingToID := make(map[string]string)

	var walkAbility func(a types.ParsedAbility)
	walkAbility = func(a types.ParsedAbility) {
		id := abilityNodeID(a.Path)
		nodes = append(nodes, types.GraphNode{ID: id, Kind: types.NodeKindAbility, Label: a.Heading})
		headingToID[strings.ToLower(a.Heading)] = id

		for _, sub := range a.SubAbilities {
			walkAbility(sub)
			edges = append(edges, types.GraphEdge{
				From:     id,
				To:       abilityNodeID(sub.Path),
				Relation: "sub-ability",
			})
		}

		if a.Inputs != nil {
			for _, row := range a.Inputs.Rows {
				for _, cid := range conceptRefsFromCell(row["Type"]) {
					edges = append(edges, types.GraphEdge{From: id, To: cid, Relation: "input"})
				}
			}
		}
		if a.Outputs != nil {
			for _, row := range a.Outputs.Rows {
				for _, cid := range conceptRefsFromCell(row["Type"]) {
					edges = append(edges, types.GraphEdge{From: id, To: cid, Relation: "output"})
				}
			}
		}
	}

	for _, a := range spec.Abilities {
		walkAbility(a)
	}

	for _, c := range spec.Concepts {
		nodes = append(nodes, types.GraphNode{
			ID:    stripPathParts(c.Path, "concepts/", "/concept.md"),
			Kind:  types.NodeKindConcept,
			Label: c.Heading,
		})
	}

	for _, d := range spec.Decisions {
		nodes = append(nodes, types.GraphNode{
			ID:    stripPathParts(d.Path, "decisions/", "/decision.md"),
			Kind:  types.NodeKindDecision,
			Label: d.Heading,
		})
	}

	for _, s := range spec.Scenarios {
		sid := stripPathParts(s.Path, "scenarios/", "/scenario.md")
		nodes = append(nodes, types.GraphNode{ID: sid, Kind: types.NodeKindScenario, Label: s.Heading})

		for _, ref := range traceRefs(s.Trace) {
			if aid, ok := headingToID[strings.ToLower(ref)]; ok {
				edges = append(edges, types.GraphEdge{From: sid, To: aid, Relation: "traces"})
			}
		}
	}

	edges = deduplicateEdges(edges)
	return nodes, edges
}

// abilityNodeID derives a stable node ID from an ability's file path.
// "abilities/verify/ability.md"                    → "verify"
// "abilities/verify/collect-violations/ability.md" → "verify/collect-violations"
func abilityNodeID(path string) string {
	s := strings.TrimPrefix(path, "abilities/")
	s = strings.TrimSuffix(s, "/ability.md")
	return s
}

// stripPathParts removes a known prefix and suffix from a path to produce a
// stable node ID.
func stripPathParts(path, prefix, suffix string) string {
	s := strings.TrimPrefix(path, prefix)
	s = strings.TrimSuffix(s, suffix)
	return s
}

// conceptLinkRE matches Markdown links whose href contains a concept path.
var conceptLinkRE = regexp.MustCompile(`concepts/([^/]+)/concept\.md`)

// conceptRefsFromCell extracts concept node IDs referenced in a Markdown table
// cell value (e.g. "[GreetingResult](../../concepts/greeting/concept.md#x)").
func conceptRefsFromCell(cell string) []string {
	matches := conceptLinkRE.FindAllStringSubmatch(cell, -1)
	seen := make(map[string]bool)
	var ids []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}
	return ids
}

// traceTokenRE matches sequences of word characters (ability heading tokens).
var traceTokenRE = regexp.MustCompile(`[A-Za-z][A-Za-z0-9-]*`)

// traceRefs extracts ability name references from a parsed scenario trace string.
// The trace has already been through parseScenarioFile, so leading/trailing
// backticks and the ">" prefix have been stripped.
func traceRefs(trace string) []string {
	tokens := traceTokenRE.FindAllString(trace, -1)
	seen := make(map[string]bool)
	var refs []string
	for _, t := range tokens {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			refs = append(refs, t)
		}
	}
	return refs
}

// deduplicateEdges removes exact duplicate (from, relation, to) triples.
func deduplicateEdges(edges []types.GraphEdge) []types.GraphEdge {
	type key struct{ from, relation, to string }
	seen := make(map[key]bool)
	result := make([]types.GraphEdge, 0, len(edges))
	for _, e := range edges {
		k := key{e.From, e.Relation, e.To}
		if !seen[k] {
			seen[k] = true
			result = append(result, e)
		}
	}
	return result
}

// --- Mermaid rendering ---

// mermaidSafeID converts a node ID and kind to a safe Mermaid diagram node
// identifier by replacing non-alphanumeric characters and prefixing with a
// kind abbreviation to prevent collisions between same-named constructs.
func mermaidSafeID(id string, kind types.NodeKind) string {
	prefix := "A"
	switch kind {
	case types.NodeKindConcept:
		prefix = "C"
	case types.NodeKindDecision:
		prefix = "D"
	case types.NodeKindScenario:
		prefix = "S"
	}
	safe := regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(id, "_")
	safe = strings.Trim(safe, "_")
	return prefix + "_" + safe
}

func escapeMermaidLabel(s string) string {
	return strings.ReplaceAll(s, `"`, "#quot;")
}

func renderMermaid(nodes []types.GraphNode, edges []types.GraphEdge) string {
	// Build a mapping from node ID + kind to Mermaid safe ID. Because
	// ability IDs and concept IDs can collide (e.g. "graph"), we key on
	// both the ID and the kind.
	type nodeKey struct {
		id   string
		kind types.NodeKind
	}
	safeMermaidID := make(map[nodeKey]string)
	for _, n := range nodes {
		k := nodeKey{n.ID, n.Kind}
		safeMermaidID[k] = mermaidSafeID(n.ID, n.Kind)
	}

	// Also build a lookup from node ID to node (kind) for edge rendering.
	// If two nodes share the same ID (different kinds), we need to know
	// which kind each edge endpoint refers to. Edges from abilities go to
	// concepts; edges between abilities or from scenarios to abilities can
	// be resolved by elimination.
	nodeByID := make(map[string][]types.GraphNode)
	for _, n := range nodes {
		nodeByID[n.ID] = append(nodeByID[n.ID], n)
	}

	lookupMermaidID := func(id string, preferKind types.NodeKind) string {
		ns := nodeByID[id]
		if len(ns) == 1 {
			return safeMermaidID[nodeKey{ns[0].ID, ns[0].Kind}]
		}
		// Ambiguous: prefer the requested kind.
		for _, n := range ns {
			if n.Kind == preferKind {
				return safeMermaidID[nodeKey{n.ID, n.Kind}]
			}
		}
		// Fallback: first match.
		if len(ns) > 0 {
			return safeMermaidID[nodeKey{ns[0].ID, ns[0].Kind}]
		}
		return mermaidSafeID(id, preferKind)
	}

	var sb strings.Builder
	sb.WriteString("graph LR\n")

	for _, n := range nodes {
		mid := safeMermaidID[nodeKey{n.ID, n.Kind}]
		label := escapeMermaidLabel(fmt.Sprintf("%s [%s]", n.Label, string(n.Kind)))
		fmt.Fprintf(&sb, "    %s[\"%s\"]\n", mid, label)
	}

	for _, e := range edges {
		var fromKind, toKind types.NodeKind
		switch e.Relation {
		case "sub-ability":
			fromKind, toKind = types.NodeKindAbility, types.NodeKindAbility
		case "input", "output":
			fromKind, toKind = types.NodeKindAbility, types.NodeKindConcept
		case "traces":
			fromKind, toKind = types.NodeKindScenario, types.NodeKindAbility
		}
		from := lookupMermaidID(e.From, fromKind)
		to := lookupMermaidID(e.To, toKind)
		fmt.Fprintf(&sb, "    %s -->|%s| %s\n", from, e.Relation, to)
	}

	return sb.String()
}

// --- DOT rendering ---

func dotShape(kind types.NodeKind) string {
	switch kind {
	case types.NodeKindConcept:
		return "ellipse"
	case types.NodeKindDecision:
		return "diamond"
	case types.NodeKindScenario:
		return "parallelogram"
	default:
		return "box"
	}
}

// dotNodeID returns a DOT-safe node identifier. Because DOT allows any string
// as a quoted identifier, we use kind-prefixed IDs to avoid merging same-named
// constructs of different kinds.
func dotNodeID(id string, kind types.NodeKind) string {
	prefix := "ability"
	switch kind {
	case types.NodeKindConcept:
		prefix = "concept"
	case types.NodeKindDecision:
		prefix = "decision"
	case types.NodeKindScenario:
		prefix = "scenario"
	}
	return prefix + ":" + id
}

func renderDOT(nodes []types.GraphNode, edges []types.GraphEdge) string {
	nodeByID := make(map[string][]types.GraphNode)
	for _, n := range nodes {
		nodeByID[n.ID] = append(nodeByID[n.ID], n)
	}

	lookupDotID := func(id string, preferKind types.NodeKind) string {
		ns := nodeByID[id]
		if len(ns) == 1 {
			return dotNodeID(ns[0].ID, ns[0].Kind)
		}
		for _, n := range ns {
			if n.Kind == preferKind {
				return dotNodeID(n.ID, n.Kind)
			}
		}
		if len(ns) > 0 {
			return dotNodeID(ns[0].ID, ns[0].Kind)
		}
		return dotNodeID(id, preferKind)
	}

	var sb strings.Builder
	sb.WriteString("digraph spec {\n")
	sb.WriteString("  rankdir=LR;\n")

	for _, n := range nodes {
		nid := dotNodeID(n.ID, n.Kind)
		label := fmt.Sprintf("%s\\n[%s]", n.Label, string(n.Kind))
		shape := dotShape(n.Kind)
		fmt.Fprintf(&sb, "  %q [label=%q shape=%s];\n", nid, label, shape)
	}

	for _, e := range edges {
		var fromKind, toKind types.NodeKind
		switch e.Relation {
		case "sub-ability":
			fromKind, toKind = types.NodeKindAbility, types.NodeKindAbility
		case "input", "output":
			fromKind, toKind = types.NodeKindAbility, types.NodeKindConcept
		case "traces":
			fromKind, toKind = types.NodeKindScenario, types.NodeKindAbility
		}
		from := lookupDotID(e.From, fromKind)
		to := lookupDotID(e.To, toKind)
		fmt.Fprintf(&sb, "  %q -> %q [label=%q];\n", from, to, e.Relation)
	}

	sb.WriteString("}\n")
	return sb.String()
}
