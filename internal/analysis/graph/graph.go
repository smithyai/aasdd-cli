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

// InvalidInclude is returned when an include value is not a valid supplementary NodeKind.
type InvalidInclude struct{ Value string }

func (e *InvalidInclude) Error() string {
	return fmt.Sprintf("invalid include value %q: must be one or more of scenarios, decisions, state-machine", e.Value)
}

// RootNotFound is returned when the specified root ability name does not match any ability.
type RootNotFound struct{ Name string }

func (e *RootNotFound) Error() string {
	return fmt.Sprintf("root ability %q not found", e.Name)
}

// AmbiguousRoot is returned when the root name matches more than one ability.
type AmbiguousRoot struct {
	Name    string
	Matches []string
}

func (e *AmbiguousRoot) Error() string {
	return fmt.Sprintf("root ability %q is ambiguous; matches: %s", e.Name, strings.Join(e.Matches, ", "))
}

// Graph generates a dependency graph for the spec rooted at target.
// opts is optional; pass a GraphOptions to restrict by filter or depth.
func Graph(target types.SpecTarget, format types.GraphFormat, opts ...types.GraphOptions) (types.GraphResult, error) {
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

	var opt types.GraphOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Validate and build the include set.
	includeSet := make(map[types.NodeKind]bool)
	for _, k := range opt.Include {
		switch k {
		case types.NodeKindScenario, types.NodeKindDecision, types.NodeKindStateMachine:
			includeSet[k] = true
		default:
			return empty, &InvalidInclude{Value: string(k)}
		}
	}

	// Re-root at the specified ability if requested.
	if opt.Root != "" {
		match, err := findAbility(spec.Abilities, opt.Root)
		if err != nil {
			return empty, err
		}
		spec.Abilities = []types.ParsedAbility{match}
	}

	nodes, edges := buildGraph(spec, opt.Depth)
	nodes, edges = applyIncludes(nodes, edges, includeSet)

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
// maxDepth controls how many levels of sub-abilities are walked; ≤0 means unlimited.
func buildGraph(spec types.SpecExport, maxDepth int) ([]types.GraphNode, []types.GraphEdge) {
	var nodes []types.GraphNode
	var edges []types.GraphEdge

	// headingToID maps lowercased ability heading → ability node ID.
	headingToID := make(map[string]string)
	// squishToNodeID maps squished (lowercase, hyphen-stripped) last-segment of an
	// ability node ID → full node ID. Used to match PascalCase ability names in
	// the state machine States table (e.g. "AnalyzeContent" → "moderate-content/analyze-content").
	squishToNodeID := make(map[string]string)

	var walkAbility func(a types.ParsedAbility, depth int)
	walkAbility = func(a types.ParsedAbility, depth int) {
		id := abilityNodeID(a.Path)
		nodes = append(nodes, types.GraphNode{ID: id, Kind: types.NodeKindAbility, Label: a.Heading})
		headingToID[strings.ToLower(a.Heading)] = id
		lastSeg := id[strings.LastIndex(id, "/")+1:]
		squishToNodeID[strings.ToLower(strings.ReplaceAll(lastSeg, "-", ""))] = id

		if maxDepth <= 0 || depth < maxDepth {
			for _, sub := range a.SubAbilities {
				walkAbility(sub, depth+1)
				edges = append(edges, types.GraphEdge{
					From:     id,
					To:       abilityNodeID(sub.Path),
					Relation: "sub-ability",
				})
			}
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
		walkAbility(a, 1)
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

	if spec.StateMachine != nil {
		nodes = append(nodes, types.GraphNode{
			ID:    "state-machine",
			Kind:  types.NodeKindStateMachine,
			Label: "State Machine",
		})
		if spec.StateMachine.States != nil {
			for _, row := range spec.StateMachine.States.Rows {
				abilityName := strings.Trim(row["Ability"], "` ")
				if abilityName == "" || abilityName == "\u2014" {
					continue
				}
				squished := strings.ToLower(strings.ReplaceAll(abilityName, "-", ""))
				if aid, ok := squishToNodeID[squished]; ok {
					edges = append(edges, types.GraphEdge{
						From:     "state-machine",
						To:       aid,
						Relation: "orchestrates",
					})
				}
			}
		}
	}

	edges = deduplicateEdges(edges)
	return nodes, edges
}

// applyIncludes post-processes the full node/edge set to produce the
// ability-anchored graph. Abilities and concepts are always retained.
// Scenarios, decisions, and state machine are only retained when their
// NodeKind appears in include.
func applyIncludes(nodes []types.GraphNode, edges []types.GraphEdge, include map[types.NodeKind]bool) ([]types.GraphNode, []types.GraphEdge) {
	var filteredNodes []types.GraphNode
	for _, n := range nodes {
		switch n.Kind {
		case types.NodeKindAbility, types.NodeKindConcept:
			filteredNodes = append(filteredNodes, n)
		default:
			if include[n.Kind] {
				filteredNodes = append(filteredNodes, n)
			}
		}
	}

	var filteredEdges []types.GraphEdge
	for _, e := range edges {
		switch e.Relation {
		case "sub-ability", "input", "output":
			filteredEdges = append(filteredEdges, e)
		case "traces":
			if include[types.NodeKindScenario] {
				filteredEdges = append(filteredEdges, e)
			}
		case "orchestrates":
			if include[types.NodeKindStateMachine] {
				filteredEdges = append(filteredEdges, e)
			}
		}
	}

	return filteredNodes, filteredEdges
}

// findAbility searches the ability tree for an ability whose last path segment
// or full ID matches name (case-insensitive). Returns AmbiguousRoot if more
// than one ability matches, or RootNotFound if none match.
func findAbility(abilities []types.ParsedAbility, name string) (types.ParsedAbility, error) {
	lower := strings.ToLower(name)
	var matches []types.ParsedAbility
	var matchPaths []string

	var search func([]types.ParsedAbility)
	search = func(abs []types.ParsedAbility) {
		for _, a := range abs {
			id := abilityNodeID(a.Path)
			lastSeg := id
			if idx := strings.LastIndex(id, "/"); idx >= 0 {
				lastSeg = id[idx+1:]
			}
			if strings.ToLower(lastSeg) == lower || strings.ToLower(id) == lower {
				matches = append(matches, a)
				matchPaths = append(matchPaths, id)
			}
			search(a.SubAbilities)
		}
	}
	search(abilities)

	switch len(matches) {
	case 0:
		return types.ParsedAbility{}, &RootNotFound{Name: name}
	case 1:
		return matches[0], nil
	default:
		return types.ParsedAbility{}, &AmbiguousRoot{Name: name, Matches: matchPaths}
	}
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
	case types.NodeKindStateMachine:
		prefix = "SM"
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
		case "orchestrates":
			fromKind, toKind = types.NodeKindStateMachine, types.NodeKindAbility
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
	case types.NodeKindStateMachine:
		return "hexagon"
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
	case types.NodeKindStateMachine:
		prefix = "sm"
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
		case "orchestrates":
			fromKind, toKind = types.NodeKindStateMachine, types.NodeKindAbility
		}
		from := lookupDotID(e.From, fromKind)
		to := lookupDotID(e.To, toKind)
		fmt.Fprintf(&sb, "  %q -> %q [label=%q];\n", from, to, e.Relation)
	}

	sb.WriteString("}\n")
	return sb.String()
}
