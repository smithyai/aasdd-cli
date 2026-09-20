package collect_violations

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	"github.com/smithyai/aasdd-cli/internal/types"
)

var tickRe = regexp.MustCompile("`([^`]+)`")

// ticks returns every backticked identifier in s, in order.
func ticks(s string) []string {
	ms := tickRe.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

func firstTick(s string) string {
	if t := ticks(s); len(t) > 0 {
		return t[0]
	}
	return ""
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// abilityNode is an ability with its place in the tree.
type abilityNode struct {
	a        *types.ParsedAbility
	parent   *abilityNode
	children []*abilityNode
	root     bool
}

func flattenAbilities(list []types.ParsedAbility, parent *abilityNode, out *[]*abilityNode) {
	for i := range list {
		n := &abilityNode{a: &list[i], parent: parent, root: parent == nil}
		*out = append(*out, n)
		if parent != nil {
			parent.children = append(parent.children, n)
		}
		flattenAbilities(list[i].SubAbilities, n, out)
	}
}

type transition struct{ from, to, trigger string }

type trace struct {
	path  string
	nodes []string
	conds []string
}

// specModel is the parsed spec plus the indexes the spec-wide rules need.
type specModel struct {
	dir             string
	exp             types.SpecExport
	nodes           []*abilityNode
	abilityNames    map[string]bool
	rootNames       []string
	rootSet         map[string]bool
	scenarioFolders map[string]bool
	failureNames    map[string]bool
	states          map[string]bool
	triggers        map[string]bool
	transitions     []transition
	traces          []trace
}

func buildModel(dir string, exp types.SpecExport) *specModel {
	m := &specModel{
		dir:             dir,
		exp:             exp,
		abilityNames:    map[string]bool{},
		rootSet:         map[string]bool{},
		scenarioFolders: map[string]bool{},
		failureNames:    map[string]bool{},
		states:          map[string]bool{},
		triggers:        map[string]bool{},
	}
	flattenAbilities(m.exp.Abilities, nil, &m.nodes)
	for _, n := range m.nodes {
		m.abilityNames[n.a.Heading] = true
		if n.root {
			m.rootNames = append(m.rootNames, n.a.Heading)
			m.rootSet[n.a.Heading] = true
		}
		if n.a.FailureModes != nil {
			for _, row := range n.a.FailureModes.Rows {
				if f := firstTick(row["Failure"]); f != "" {
					m.failureNames[f] = true
				}
			}
		}
	}
	for _, s := range m.exp.Scenarios {
		m.scenarioFolders[path.Base(path.Dir(s.Path))] = true
		nodes, conds := parseTrace(s.Trace)
		m.traces = append(m.traces, trace{path: s.Path, nodes: nodes, conds: conds})
	}
	if sm := m.exp.StateMachine; sm != nil {
		if sm.States != nil {
			for _, row := range sm.States.Rows {
				if s := firstTick(row["State"]); s != "" {
					m.states[s] = true
				}
			}
		}
		if sm.Transitions != nil {
			for _, row := range sm.Transitions.Rows {
				t := transition{from: firstTick(row["From"]), to: firstTick(row["To"]), trigger: firstTick(row["Trigger"])}
				m.transitions = append(m.transitions, t)
				if t.trigger != "" {
					m.triggers[t.trigger] = true
				}
			}
		}
	}
	return m
}

// parseTrace splits a stored trace ("A` → `B` — Cond → `C") into nodes and the
// divergence condition on the edge leaving each node.
func parseTrace(trace string) (nodes, conds []string) {
	if strings.TrimSpace(trace) == "" {
		return nil, nil
	}
	for _, seg := range strings.Split(trace, " → ") {
		seg = strings.TrimSpace(seg)
		cond := ""
		if i := strings.Index(seg, " — "); i >= 0 {
			cond = strings.TrimSpace(seg[i+len(" — "):])
			seg = seg[:i]
		}
		nodes = append(nodes, strings.Trim(seg, "`"))
		conds = append(conds, strings.Trim(cond, "`"))
	}
	return nodes, conds
}

func (m *specModel) abs(rel string) string {
	return filepath.Join(m.dir, filepath.FromSlash(rel))
}

var specFileNames = map[string]bool{
	"spec.md": true, "ability.md": true, "concept.md": true,
	"scenario.md": true, "decision.md": true, "state-machine.md": true,
}

// specFiles lists every spec file under the directory, sorted by slash path so
// the order is the same on every platform.
func (m *specModel) specFiles() []string {
	var files []string
	_ = filepath.WalkDir(m.dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if specFileNames[d.Name()] {
			files = append(files, p)
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return filepath.ToSlash(files[i]) < filepath.ToSlash(files[j]) })
	return files
}

// evaluateSpecRules evaluates every spec-wide rule against the parsed spec.
func evaluateSpecRules(rules []types.Rule, dir string, exp types.SpecExport, ctx specContext) []types.Violation {
	m := buildModel(dir, exp)
	var out []types.Violation
	for _, r := range rules {
		out = append(out, m.evaluate(r, ctx)...)
	}
	return out
}

func (m *specModel) evaluate(rule types.Rule, ctx specContext) []types.Violation {
	switch rule.ID {
	case "placeholder.misplaced":
		return m.misplacedPlaceholders(rule)
	case "pending.not-allowed":
		if !ctx.ready {
			return nil
		}
		return m.pendingNotAllowed(rule)
	case "decision.open-not-allowed":
		if !ctx.ready {
			return nil
		}
		return m.openNotAllowed(rule)
	case "folder.name-mismatch":
		return m.folderNames(rule)
	case "link.broken":
		return m.brokenLinks(rule)
	case "type.undefined":
		return m.undefinedTypes(rule)
	case "spec.criteria-unknown-ability", "spec.criteria-unserved-root", "spec.criteria-unknown-scenario", "spec.criteria-missing-scenario":
		return m.criteria(rule, ctx)
	case "ability.missing-composition", "ability.composition-on-leaf", "ability.composition-unknown-child",
		"ability.composition-duplicate-child", "ability.composition-omits-child", "ability.composition-step-order",
		"ability.composition-unknown-source", "ability.composition-unknown-produce":
		return m.composition(rule, ctx)
	case "ability.delegated-spec-not-found", "ability.delegated-version-mismatch", "ability.delegated-multiple-roots":
		return m.delegation(rule)
	case "scenario.unknown-node", "scenario.unknown-condition":
		return m.scenarioRefs(rule)
	case "decision.context-names-no-ability":
		return m.decisionContexts(rule)
	case "state-machine.unknown-ability", "state-machine.unknown-state", "state-machine.missing-trigger":
		return m.stateMachine(rule)
	case "coverage.root-failure-mode", "coverage.transition":
		if !ctx.ready {
			return nil
		}
		return m.coverage(rule)
	case "format.crlf", "format.final-newline", "format.consecutive-blank-lines", "format.trailing-whitespace",
		"format.h2-start", "format.pipe-in-cell", "format.table-padding":
		return m.formatting(rule)
	}
	return nil
}

// --- placeholders ---

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Where each placeholder is permitted. A placeholder outside these sets is
// misplaced; the sets match the rule description in load_rule_set.
var (
	specNoneAllowed            = map[string]bool{"Non-Goals": true, "Failure Modes": true}
	specPendingAllowed         = map[string]bool{"Purpose": true, "Non-Goals": true, "Success Criteria": true, "Invariants": true}
	abilityNoneAllowed         = map[string]bool{"Inputs": true, "Failure Modes": true}
	abilityPendingAllowed      = map[string]bool{"Inputs": true, "Outputs": true, "Invariants": true, "Failure Modes": true}
	decisionPendingAllowed     = map[string]bool{"Context": true, "Requirement": true, "Decision": true}
	decisionOpenAllowed        = map[string]bool{"Decision": true}
	stateMachinePendingAllowed = map[string]bool{"Orchestrator": true, "States": true, "Transitions": true}
)

func placementOK(placeholder, section string, none, pending, open map[string]bool) bool {
	switch placeholder {
	case types.PlaceholderNone:
		return none[section]
	case types.PlaceholderPending:
		return pending[section]
	case types.PlaceholderOpen:
		return open[section]
	}
	return true
}

func (m *specModel) misplacedPlaceholders(rule types.Rule) []types.Violation {
	var out []types.Violation
	check := func(p string, placeholders map[string]string, none, pending, open map[string]bool) {
		for _, section := range sortedKeys(placeholders) {
			ph := placeholders[section]
			if !placementOK(ph, section, none, pending, open) {
				out = append(out, newViolation(rule, p, fmt.Sprintf("_%s._ is not permitted for section %s", ph, section)))
			}
		}
	}
	custom := func(p string, sections []types.CustomSection) {
		for _, cs := range sections {
			if ph := markerPlaceholder(cs.Content); ph != "" {
				out = append(out, newViolation(rule, p, fmt.Sprintf("_%s._ is not permitted in custom section %s", ph, cs.Heading)))
			}
		}
	}
	check(m.abs("spec.md"), m.exp.Placeholders, specNoneAllowed, specPendingAllowed, nil)
	custom(m.abs("spec.md"), m.exp.CustomSections)
	for _, n := range m.nodes {
		check(m.abs(n.a.Path), n.a.Placeholders, abilityNoneAllowed, abilityPendingAllowed, nil)
		custom(m.abs(n.a.Path), n.a.CustomSections)
		if ph := markerPlaceholder(n.a.Idempotency); ph != "" {
			out = append(out, newViolation(rule, m.abs(n.a.Path), fmt.Sprintf("_%s._ is not permitted for section Idempotency", ph)))
		}
	}
	for _, c := range m.exp.Concepts {
		custom(m.abs(c.Path), c.CustomSections)
	}
	for _, d := range m.exp.Decisions {
		check(m.abs(d.Path), d.Placeholders, nil, decisionPendingAllowed, decisionOpenAllowed)
		custom(m.abs(d.Path), d.CustomSections)
	}
	for _, s := range m.exp.Scenarios {
		if ph := markerPlaceholder(s.Example); ph != "" {
			out = append(out, newViolation(rule, m.abs(s.Path), fmt.Sprintf("_%s._ is not permitted for section Example", ph)))
		}
		custom(m.abs(s.Path), s.CustomSections)
	}
	if sm := m.exp.StateMachine; sm != nil {
		check(m.abs("state-machine.md"), sm.Placeholders, nil, stateMachinePendingAllowed, nil)
		custom(m.abs("state-machine.md"), sm.CustomSections)
	}
	return out
}

func (m *specModel) pendingNotAllowed(rule types.Rule) []types.Violation {
	var out []types.Violation
	check := func(p string, placeholders map[string]string) {
		for _, section := range sortedKeys(placeholders) {
			if placeholders[section] == types.PlaceholderPending {
				out = append(out, newViolation(rule, p, fmt.Sprintf("section %s is _Pending._ in a spec at %s", section, m.exp.Version)))
			}
		}
	}
	check(m.abs("spec.md"), m.exp.Placeholders)
	for _, n := range m.nodes {
		check(m.abs(n.a.Path), n.a.Placeholders)
	}
	for _, d := range m.exp.Decisions {
		check(m.abs(d.Path), d.Placeholders)
	}
	if sm := m.exp.StateMachine; sm != nil {
		check(m.abs("state-machine.md"), sm.Placeholders)
	}
	return out
}

func (m *specModel) openNotAllowed(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, d := range m.exp.Decisions {
		if d.Open() {
			out = append(out, newViolation(rule, m.abs(d.Path), fmt.Sprintf("decision %s is _Open._ in a spec at %s", d.Heading, m.exp.Version)))
		}
	}
	return out
}

// --- naming ---

func (m *specModel) folderNames(rule types.Rule) []types.Violation {
	var out []types.Violation
	check := func(rel, heading, want string) {
		got := path.Base(path.Dir(rel))
		if got != want {
			out = append(out, newViolation(rule, m.abs(rel), fmt.Sprintf("folder %q should be %q for heading %q", got, want, heading)))
		}
	}
	for _, n := range m.nodes {
		check(n.a.Path, n.a.Heading, format.NameToKebab(n.a.Heading))
	}
	for _, s := range m.exp.Scenarios {
		check(s.Path, s.Heading, format.NameToKebab(s.Heading))
	}
	for _, d := range m.exp.Decisions {
		check(d.Path, d.Heading, format.NameToKebab(d.Heading))
	}
	for _, c := range m.exp.Concepts {
		check(c.Path, c.Heading, format.NameToKebab(strings.TrimSuffix(c.Heading, " domain")))
	}
	return out
}

// --- links and types ---

var linkRe = regexp.MustCompile(`\]\(([^)]+)\)`)
var headingRe = regexp.MustCompile(`(?m)^#{1,6} (.*)$`)

func anchorize(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

func headingAnchors(file string) map[string]bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	anchors := map[string]bool{}
	for _, mm := range headingRe.FindAllStringSubmatch(export.NormalizeNewlines(string(data)), -1) {
		anchors[anchorize(mm[1])] = true
	}
	return anchors
}

// existsExact reports whether p exists with exactly this spelling. Below root
// every path element is compared case-sensitively against its directory
// listing, so a link that only works on a case-insensitive filesystem is
// reported on every platform.
func existsExact(p, root string) bool {
	if _, err := os.Stat(p); err != nil {
		return false
	}
	cur := filepath.Clean(p)
	root = filepath.Clean(root)
	for strings.HasPrefix(cur, root+string(filepath.Separator)) {
		parent := filepath.Dir(cur)
		entries, err := os.ReadDir(parent)
		if err != nil {
			return true
		}
		found := false
		for _, e := range entries {
			if e.Name() == filepath.Base(cur) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		cur = parent
	}
	return true
}

func (m *specModel) brokenLinks(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, f := range m.specFiles() {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := export.NormalizeNewlines(string(data))
		for _, mm := range linkRe.FindAllStringSubmatch(content, -1) {
			target := mm[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			parts := strings.SplitN(target, "#", 2)
			absPath := f
			if parts[0] != "" {
				absPath = filepath.Join(filepath.Dir(f), filepath.FromSlash(parts[0]))
			}
			if !existsExact(absPath, m.dir) {
				out = append(out, newViolation(rule, f, "broken link "+target))
				continue
			}
			if len(parts) == 2 && parts[1] != "" && !headingAnchors(absPath)[parts[1]] {
				out = append(out, newViolation(rule, f, "missing anchor "+target))
			}
		}
	}
	return out
}

var scalarTypes = map[string]bool{"text": true, "number": true, "boolean": true, "timestamp": true}

// typeParts strips structural modifiers from a Type cell and returns the
// underlying type expressions.
func typeParts(cell string) []string {
	s := strings.TrimSpace(cell)
	for {
		switch {
		case strings.HasPrefix(s, "optional "):
			s = strings.TrimPrefix(s, "optional ")
		case strings.HasPrefix(s, "list of "):
			s = strings.TrimPrefix(s, "list of ")
		default:
			if strings.HasPrefix(s, "map of ") {
				rest := strings.TrimPrefix(s, "map of ")
				if i := strings.Index(rest, " to "); i >= 0 {
					return append(typeParts(rest[:i]), typeParts(rest[i+len(" to "):])...)
				}
			}
			return []string{s}
		}
	}
}

// externalNoted reports whether a description notes that a type is defined
// elsewhere: it says so directly, or it names the spec that defines it.
func externalNoted(description string) bool {
	d := strings.ToLower(description)
	return strings.Contains(d, "external") || strings.Contains(d, "another spec") ||
		strings.Contains(d, "defined in") || strings.Contains(d, "spec")
}

func (m *specModel) undefinedTypes(rule types.Rule) []types.Violation {
	var out []types.Violation
	check := func(p string, t *types.Table) {
		if t == nil {
			return
		}
		for _, row := range t.Rows {
			cell := strings.TrimSpace(row["Type"])
			if cell == "" || cell == "—" {
				continue
			}
			for _, part := range typeParts(cell) {
				if strings.HasPrefix(part, "[") || scalarTypes[part] || externalNoted(row["Description"]) {
					continue
				}
				out = append(out, newViolation(rule, p, fmt.Sprintf("type %q is neither a scalar, a link to a concept, nor noted as external", part)))
			}
		}
	}
	for _, n := range m.nodes {
		if n.a.Delegated() {
			continue
		}
		check(m.abs(n.a.Path), n.a.Inputs)
		check(m.abs(n.a.Path), n.a.Outputs)
	}
	for _, c := range m.exp.Concepts {
		for _, ct := range c.Types {
			if ct.Properties != nil && contains(ct.Properties.Headers, "Type") {
				check(m.abs(c.Path), ct.Properties)
			}
		}
	}
	return out
}

// --- success criteria ---

func (m *specModel) criteria(rule types.Rule, ctx specContext) []types.Violation {
	sc := m.exp.SuccessCriteria
	if sc == nil {
		return nil
	}
	p := m.abs("spec.md")
	var out []types.Violation
	named := map[string]bool{}
	for _, row := range sc.Rows {
		for _, a := range ticks(row["Abilities"]) {
			named[a] = true
			if rule.ID == "spec.criteria-unknown-ability" && !m.rootSet[a] {
				out = append(out, newViolation(rule, p, fmt.Sprintf("criterion names %s, which is not a root ability", a)))
			}
		}
		scenarios := ticks(row["Scenarios"])
		if rule.ID == "spec.criteria-missing-scenario" && ctx.ready && len(scenarios) == 0 {
			out = append(out, newViolation(rule, p, "criterion has no scenario: "+row["Criterion"]))
		}
		if rule.ID == "spec.criteria-unknown-scenario" {
			for _, s := range scenarios {
				if !m.scenarioFolders[s] {
					out = append(out, newViolation(rule, p, fmt.Sprintf("criterion names scenario %s, which does not exist", s)))
				}
			}
		}
	}
	if rule.ID == "spec.criteria-unserved-root" {
		for _, r := range m.rootNames {
			if !named[r] {
				out = append(out, newViolation(rule, p, fmt.Sprintf("root ability %s serves no success criterion", r)))
			}
		}
	}
	return out
}

// --- composition ---

var sourceRe = regexp.MustCompile(`from (parent|step (\d+))`)

func tableColumnTicks(t *types.Table, column string) []string {
	if t == nil {
		return nil
	}
	var out []string
	for _, row := range t.Rows {
		if v := firstTick(row[column]); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (m *specModel) orchestrates(children []*abilityNode) bool {
	sm := m.exp.StateMachine
	if sm == nil || sm.States == nil {
		return false
	}
	for _, row := range sm.States.Rows {
		for _, name := range ticks(row["Ability"]) {
			for _, c := range children {
				if c.a.Heading == name {
					return true
				}
			}
		}
	}
	return false
}

// consumedLater reports whether a value produced by step (1-based) is consumed
// from that step by any later row.
func consumedLater(rows []map[string]string, step int, value string) bool {
	for j := step; j < len(rows); j++ {
		for _, clause := range strings.Split(rows[j]["Consumes"], ", ") {
			src := sourceRe.FindStringSubmatch(clause)
			if src == nil || src[1] == "parent" {
				continue
			}
			k, _ := strconv.Atoi(src[2])
			names := ticks(clause)
			if k == step && len(names) > 0 && names[len(names)-1] == value {
				return true
			}
		}
	}
	return false
}

func (m *specModel) composition(rule types.Rule, ctx specContext) []types.Violation {
	var out []types.Violation
	for _, n := range m.nodes {
		if n.a.Delegated() {
			continue
		}
		p := m.abs(n.a.Path)
		emit := func(id, msg string) {
			if rule.ID == id {
				out = append(out, newViolation(rule, p, msg))
			}
		}
		children := n.children
		comp := n.a.Composition
		if comp == nil {
			if len(children) > 0 && ctx.ready && !(n.root && m.orchestrates(children)) {
				emit("ability.missing-composition", "non-leaf ability has no Composition section")
			}
			continue
		}
		if len(children) == 0 {
			emit("ability.composition-on-leaf", "Composition on an ability with no sub-abilities")
			continue
		}
		childByName := map[string]*abilityNode{}
		for _, c := range children {
			childByName[c.a.Heading] = c
		}
		seen := map[string]bool{}
		var parentInputs, parentOutputs []string
		if n.a.Inputs != nil {
			parentInputs = tableColumnTicks(n.a.Inputs, "Name")
		}
		if n.a.Outputs != nil {
			parentOutputs = tableColumnTicks(n.a.Outputs, "Name")
		}
		rows := comp.Rows
		for i, row := range rows {
			step := i + 1
			if s, err := strconv.Atoi(strings.TrimSpace(row["Step"])); err != nil || s != step {
				emit("ability.composition-step-order", fmt.Sprintf("row %d is numbered %q; expected %d", step, row["Step"], step))
			}
			abilityCell := strings.TrimSpace(row["Ability"])
			var child *abilityNode
			if abilityCell != "—" {
				name := firstTick(abilityCell)
				if name == "" {
					emit("ability.composition-unknown-child", fmt.Sprintf("step %d names no ability", step))
				} else if c, ok := childByName[name]; !ok {
					emit("ability.composition-unknown-child", fmt.Sprintf("step %d names %s, which is not a sub-ability", step, name))
				} else {
					if seen[name] {
						emit("ability.composition-duplicate-child", fmt.Sprintf("step %d names %s a second time", step, name))
					}
					seen[name] = true
					child = c
				}
			}
			for _, produced := range ticks(row["Produces"]) {
				switch {
				case child != nil:
					if child.a.Outputs != nil && !contains(tableColumnTicks(child.a.Outputs, "Name"), produced) {
						emit("ability.composition-unknown-produce", fmt.Sprintf("step %d produces %s, which %s does not output", step, produced, child.a.Heading))
					}
				case abilityCell == "—":
					if contains(parentOutputs, produced) || consumedLater(rows, step, produced) {
						continue
					}
					emit("ability.composition-unknown-produce", fmt.Sprintf("step %d produces %s, which is neither a parent output nor consumed by a later step", step, produced))
				}
			}
			for _, clause := range strings.Split(row["Consumes"], ", ") {
				clause = strings.TrimSpace(clause)
				if clause == "" || clause == "—" {
					continue
				}
				names := ticks(clause)
				src := sourceRe.FindStringSubmatch(clause)
				if src == nil || len(names) == 0 {
					emit("ability.composition-unknown-source", fmt.Sprintf("step %d consumes %q without a source", step, clause))
					continue
				}
				value := names[len(names)-1]
				if src[1] == "parent" {
					if n.a.Inputs != nil && !contains(parentInputs, value) {
						emit("ability.composition-unknown-source", fmt.Sprintf("step %d consumes %s from parent, which is not a parent input", step, value))
					}
					continue
				}
				k, _ := strconv.Atoi(src[2])
				if k < 1 || k >= step {
					emit("ability.composition-step-order", fmt.Sprintf("step %d consumes from step %d", step, k))
					continue
				}
				if !contains(ticks(rows[k-1]["Produces"]), value) {
					emit("ability.composition-unknown-source", fmt.Sprintf("step %d consumes %s, which step %d does not produce", step, value, k))
				}
			}
		}
		for _, c := range children {
			if !seen[c.a.Heading] {
				emit("ability.composition-omits-child", "Composition omits sub-ability "+c.a.Heading)
			}
		}
	}
	return out
}

// --- delegation ---

func (m *specModel) delegation(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, n := range m.nodes {
		if !n.a.Delegated() || strings.Contains(n.a.Spec, "://") {
			continue
		}
		p := m.abs(n.a.Path)
		target := filepath.Join(filepath.Dir(p), filepath.FromSlash(n.a.Spec))
		data, err := os.ReadFile(filepath.Join(target, "spec.md"))
		if err != nil {
			if rule.ID == "ability.delegated-spec-not-found" {
				out = append(out, newViolation(rule, p, fmt.Sprintf("delegated spec %s has no spec.md", n.a.Spec)))
			}
			continue
		}
		if rule.ID == "ability.delegated-version-mismatch" {
			version := ""
			if mm := versionLineRe.FindStringSubmatch(export.NormalizeNewlines(string(data))); mm != nil {
				version = mm[1]
			}
			if version != n.a.SpecVersion {
				out = append(out, newViolation(rule, p, fmt.Sprintf("delegated version is %s but %s is at %s", n.a.SpecVersion, n.a.Spec, version)))
			}
		}
		if rule.ID == "ability.delegated-multiple-roots" {
			roots := 0
			if entries, readErr := os.ReadDir(filepath.Join(target, "abilities")); readErr == nil {
				for _, e := range entries {
					if e.IsDir() {
						if _, statErr := os.Stat(filepath.Join(target, "abilities", e.Name(), "ability.md")); statErr == nil {
							roots++
						}
					}
				}
			}
			if roots != 1 {
				out = append(out, newViolation(rule, p, fmt.Sprintf("delegated spec %s has %d root abilities; expected exactly one", n.a.Spec, roots)))
			}
		}
	}
	return out
}

// --- scenarios, decisions, state machine ---

func (m *specModel) scenarioRefs(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, t := range m.traces {
		p := m.abs(t.path)
		for i, node := range t.nodes {
			if rule.ID == "scenario.unknown-node" {
				known := m.abilityNames[node]
				if m.exp.StateMachine != nil {
					known = m.states[node]
				}
				if !known {
					out = append(out, newViolation(rule, p, "unknown trace node "+node))
				}
			}
			if rule.ID == "scenario.unknown-condition" && t.conds[i] != "" && !m.triggers[t.conds[i]] && !m.failureNames[t.conds[i]] {
				out = append(out, newViolation(rule, p, "unknown divergence condition "+t.conds[i]))
			}
		}
	}
	return out
}

func (m *specModel) decisionContexts(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, d := range m.exp.Decisions {
		if d.Placeholders["Context"] != "" {
			continue // a pending Context is reported by the readiness rules
		}
		names := false
		for _, t := range ticks(d.Context) {
			if m.abilityNames[t] {
				names = true
				break
			}
		}
		if !names {
			out = append(out, newViolation(rule, m.abs(d.Path), "Context names no ability of this spec"))
		}
	}
	return out
}

func (m *specModel) stateMachine(rule types.Rule) []types.Violation {
	sm := m.exp.StateMachine
	if sm == nil {
		return nil
	}
	p := m.abs("state-machine.md")
	var out []types.Violation
	switch rule.ID {
	case "state-machine.unknown-ability":
		if sm.States != nil {
			for _, row := range sm.States.Rows {
				for _, name := range ticks(row["Ability"]) {
					if !m.abilityNames[name] {
						out = append(out, newViolation(rule, p, fmt.Sprintf("state %s maps to unknown ability %s", firstTick(row["State"]), name)))
					}
				}
			}
		}
	case "state-machine.unknown-state":
		for _, t := range m.transitions {
			if !m.states[t.from] {
				out = append(out, newViolation(rule, p, "transition from unknown state "+t.from))
			}
			if !m.states[t.to] {
				out = append(out, newViolation(rule, p, "transition to unknown state "+t.to))
			}
		}
	case "state-machine.missing-trigger":
		for _, t := range m.transitions {
			if t.trigger == "" {
				out = append(out, newViolation(rule, p, fmt.Sprintf("transition %s → %s names no trigger", t.from, t.to)))
			}
		}
	}
	return out
}

// --- coverage ---

func (m *specModel) coverage(rule types.Rule) []types.Violation {
	var out []types.Violation
	switch rule.ID {
	case "coverage.root-failure-mode":
		for _, n := range m.nodes {
			if !n.root || n.a.Delegated() || n.a.FailureModes == nil {
				continue
			}
			for _, name := range tableColumnTicks(n.a.FailureModes, "Failure") {
				found := false
				for _, t := range m.traces {
					if contains(t.conds, name) {
						found = true
						break
					}
				}
				if !found {
					out = append(out, newViolation(rule, m.abs(n.a.Path), fmt.Sprintf("failure mode %s appears in no scenario", name)))
				}
			}
		}
	case "coverage.transition":
		for _, tr := range m.transitions {
			dup := 0
			for _, other := range m.transitions {
				if other.from == tr.from && other.to == tr.to {
					dup++
				}
			}
			found := false
			for _, t := range m.traces {
				for i := 0; i+1 < len(t.nodes); i++ {
					if t.nodes[i] == tr.from && t.nodes[i+1] == tr.to && (dup < 2 || t.conds[i] == tr.trigger) {
						found = true
					}
				}
			}
			if !found {
				out = append(out, newViolation(rule, m.abs("state-machine.md"), fmt.Sprintf("transition %s → %s (%s) appears in no scenario", tr.from, tr.to, tr.trigger)))
			}
		}
	}
	return out
}

// --- formatting ---

var trailingWhitespaceRe = regexp.MustCompile(`[ \t]\n`)

func (m *specModel) formatting(rule types.Rule) []types.Violation {
	var out []types.Violation
	for _, f := range m.specFiles() {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s := string(raw)
		norm := export.NormalizeNewlines(s)
		msg := ""
		switch rule.ID {
		case "format.crlf":
			if strings.Contains(s, "\r") {
				msg = "file contains carriage returns; use LF line endings"
			}
		case "format.final-newline":
			if !strings.HasSuffix(norm, "\n") || strings.HasSuffix(norm, "\n\n") {
				msg = "file must end with exactly one newline"
			}
		case "format.consecutive-blank-lines":
			if strings.Contains(norm, "\n\n\n") {
				msg = "file contains consecutive blank lines"
			}
		case "format.trailing-whitespace":
			if trailingWhitespaceRe.MatchString(norm) {
				msg = "file contains trailing whitespace"
			}
		case "format.h2-start":
			if !strings.HasPrefix(norm, "## ") {
				msg = "file must start with an H2 heading"
			}
		case "format.pipe-in-cell":
			inFence := false
			for _, line := range strings.Split(norm, "\n") {
				if format.IsFence(line) {
					inFence = !inFence
					continue
				}
				if !inFence && strings.HasPrefix(line, "|") && strings.Contains(line, `\|`) {
					msg = "table cell contains an escaped pipe"
					break
				}
			}
		case "format.table-padding":
			if format.PadDocument(norm) != norm {
				msg = "tables are not in the canonical padded form"
			}
		}
		if msg != "" {
			out = append(out, newViolation(rule, f, msg))
		}
	}
	return out
}
