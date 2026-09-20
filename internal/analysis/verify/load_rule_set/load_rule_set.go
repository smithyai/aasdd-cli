// Package load_rule_set implements the LoadRuleSet ability.
package load_rule_set

import (
	"fmt"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// UnknownAASDDVersion is returned when the requested AASDD version is not
// recognised by the tool.
type UnknownAASDDVersion struct {
	Version string
}

func (e *UnknownAASDDVersion) Error() string {
	return fmt.Sprintf("unknown AASDD version: %q", e.Version)
}

// LatestVersion is the newest AASDD methodology version the tool knows.
const LatestVersion = "v2"

// VersionEntry describes a known AASDD methodology version.
type VersionEntry struct {
	Version string
	Summary string
}

// KnownVersions returns all recognised AASDD versions in order, oldest first.
func KnownVersions() []VersionEntry {
	return []VersionEntry{
		{Version: "v1", Summary: "Initial release — structural rules for spec.md, ability.md, concept.md, scenario.md, and decision.md."},
		{Version: "v2", Summary: "Vision sections, placeholders, composition, delegation, scenario coverage, readiness conditions, and canonical formatting."},
	}
}

func rule(id, description, appliesTo string, severity types.Severity) types.Rule {
	return types.Rule{ID: id, Description: description, AppliesTo: appliesTo, Severity: severity}
}

// v1Rules are the structural rules of AASDD v1.
var v1Rules = []types.Rule{
	rule("directory.missing-spec", "The root directory must contain a spec.md file.", "directory", types.SeverityError),
	rule("directory.missing-abilities", "The root directory must contain an abilities/ subdirectory.", "directory", types.SeverityError),
	rule("directory.missing-concepts", "The root directory must contain a concepts/ subdirectory.", "directory", types.SeverityError),
	rule("abilities.empty", "abilities/ contains no ability subdirectories.", "directory", types.SeverityWarning),
	rule("ability.missing-ability-md", "Each ability directory must contain an ability.md file.", "directory", types.SeverityWarning),
	rule("concepts.empty", "concepts/ contains no concept domain subdirectories.", "directory", types.SeverityWarning),
	rule("concept.missing-concept-md", "Each concept domain directory must contain a concept.md file.", "directory", types.SeverityWarning),
	rule("spec.missing-version", "spec.md must declare **Version:**.", "spec.md", types.SeverityWarning),
	rule("spec.missing-summary", "spec.md must have a summary paragraph after the version metadata.", "spec.md", types.SeverityWarning),
	rule("spec.missing-aasdd-version", "spec.md must declare **AASDD:** to record the methodology version the spec conforms to.", "spec.md", types.SeverityWarning),
	rule("spec.invalid-version-format", "spec.md **Version:** value must be a valid semver string (MAJOR.MINOR.PATCH) with no 'v' prefix.", "spec.md", types.SeverityWarning),
	rule("spec.invalid-aasdd-version", "spec.md **AASDD:** value must be a recognised AASDD version (e.g. v1).", "spec.md", types.SeverityWarning),
	rule("ability.missing-purpose", "ability.md must have a purpose paragraph after the heading.", "ability.md", types.SeverityError),
	rule("ability.missing-inputs", "ability.md must contain a ### Inputs section.", "ability.md", types.SeverityError),
	rule("ability.missing-outputs", "ability.md must contain a ### Outputs section.", "ability.md", types.SeverityError),
	rule("concept.missing-type", "concept.md must define at least one type (### heading).", "concept.md", types.SeverityError),
	rule("scenario.missing-description", "scenario.md must have a description paragraph after the heading.", "scenario.md", types.SeverityError),
	rule("scenario.missing-trace", "scenario.md must contain a blockquote (>) execution trace.", "scenario.md", types.SeverityError),
	rule("decision.missing-context", "decision.md must contain a ## Context section.", "decision.md", types.SeverityError),
	rule("decision.missing-requirement", "decision.md must contain a ## Requirement section.", "decision.md", types.SeverityError),
	rule("decision.missing-decision", "decision.md must contain a ## Decision section.", "decision.md", types.SeverityError),
}

// v2Additions are the rules AASDD v2 adds on top of v1: vision sections,
// placeholders, composition, delegation, cross-references, scenario coverage,
// readiness conditions, and canonical formatting.
var v2Additions = []types.Rule{
	// spec.md
	rule("spec.missing-purpose", "spec.md must contain a ### Purpose section.", "spec.md", types.SeverityError),
	rule("spec.missing-non-goals", "spec.md must contain a ### Non-Goals section.", "spec.md", types.SeverityError),
	rule("spec.missing-success-criteria", "spec.md must contain a ### Success Criteria section.", "spec.md", types.SeverityError),
	rule("spec.missing-invariants", "spec.md must contain a ### Invariants section.", "spec.md", types.SeverityError),
	rule("spec.section-order", "spec.md sections appear in the order Purpose, Non-Goals, Success Criteria, Invariants, Failure Modes, then custom sections.", "spec.md", types.SeverityError),
	rule("spec.criteria-columns", "The Success Criteria table has the columns Criterion, Abilities, and Scenarios.", "spec.md", types.SeverityError),
	rule("spec.table-columns", "A spec-level Failure Modes table has the columns Failure, Condition, and Effect.", "spec.md", types.SeverityError),
	// ability.md
	rule("ability.missing-invariants", "ability.md must contain a ### Invariants section.", "ability.md", types.SeverityError),
	rule("ability.missing-failure-modes", "ability.md must contain a ### Failure Modes section.", "ability.md", types.SeverityError),
	rule("ability.section-order", "ability.md sections appear in the order Inputs, Outputs, Invariants, Failure Modes, Idempotency, Composition, then custom sections.", "ability.md", types.SeverityError),
	rule("ability.composition-columns", "The Composition table has the columns Step, Ability, Consumes, and Produces.", "ability.md", types.SeverityError),
	rule("ability.table-columns", "Inputs and Outputs tables have the columns Name, Type, and Description; the Failure Modes table has Failure, Condition, and Effect.", "ability.md", types.SeverityError),
	rule("ability.delegated-extra-sections", "A delegated ability (one with a **Spec:** label) has no sections.", "ability.md", types.SeverityError),
	rule("ability.delegated-missing-version", "A delegated ability declares the delegated spec's version with **Version:**.", "ability.md", types.SeverityError),
	// decision.md
	rule("decision.section-order", "decision.md sections appear in the order Context, Requirement, Options, Decision, then custom sections.", "decision.md", types.SeverityError),
	rule("decision.open-without-options", "An open decision (_Open._) lists its Options.", "decision.md", types.SeverityError),
	// scenario.md
	rule("scenario.section-order", "scenario.md places its Example section, when present, before any custom section.", "scenario.md", types.SeverityError),
	rule("scenario.trace-format", "The trace is a sequence of backticked nodes joined by →, with divergences written as — Condition →.", "scenario.md", types.SeverityError),
	// concept.md
	rule("concept.heading-domain", "concept.md opens with a ## {DomainName} domain heading.", "concept.md", types.SeverityError),
	rule("concept.type-table", "Every type in concept.md has a Properties table or a Value/Meaning table; sections without a table after the last type are custom sections.", "concept.md", types.SeverityError),
	rule("concept.table-columns", "Every type table in concept.md has the columns Name, Type, and Description, or Value and Meaning.", "concept.md", types.SeverityError),
	// state-machine.md
	rule("state-machine.missing-diagram", "state-machine.md must contain a fenced diagram before its sections.", "state-machine.md", types.SeverityError),
	rule("state-machine.missing-orchestrator", "state-machine.md must contain a ### Orchestrator section.", "state-machine.md", types.SeverityError),
	rule("state-machine.missing-states", "state-machine.md must contain a ### States section.", "state-machine.md", types.SeverityError),
	rule("state-machine.missing-transitions", "state-machine.md must contain a ### Transitions section.", "state-machine.md", types.SeverityError),
	rule("state-machine.section-order", "state-machine.md sections appear in the order Orchestrator, States, Transitions, Transition Rules, Exceptional Flows.", "state-machine.md", types.SeverityError),
	rule("state-machine.table-columns", "The States table has the columns State, Ability, and Description; Transitions has From, To, Trigger, and Data Passed Forward; Orchestrator-Managed State has Name, Type, and Description.", "state-machine.md", types.SeverityError),
	// spec-wide: placeholders and readiness
	rule("placeholder.misplaced", "_None._ is permitted only for Inputs, Failure Modes, and Non-Goals; _Pending._ only for required sections; _Open._ only for a Decision.", "spec", types.SeverityError),
	rule("pending.not-allowed", "At 1.0.0 and above, no section is _Pending._.", "spec", types.SeverityError),
	rule("decision.open-not-allowed", "At 1.0.0 and above, no decision is _Open._.", "spec", types.SeverityError),
	// spec-wide: naming and references
	rule("folder.name-mismatch", "Every ability, scenario, decision, and concept folder is named the kebab-case form of its heading.", "spec", types.SeverityError),
	rule("link.broken", "Every relative link resolves to an existing file, and every anchor to a heading in that file.", "spec", types.SeverityError),
	rule("type.undefined", "Every type in Inputs, Outputs, and Properties is a scalar, a link to a concept, or noted as external by a description that names the spec defining it.", "spec", types.SeverityError),
	// spec-wide: success criteria
	rule("spec.criteria-unknown-ability", "Every name in the Abilities column of Success Criteria is a root ability of this spec.", "spec", types.SeverityError),
	rule("spec.criteria-unserved-root", "Every root ability appears in the Abilities column of at least one success criterion.", "spec", types.SeverityError),
	rule("spec.criteria-unknown-scenario", "Every name in the Scenarios column of Success Criteria is a scenario of this spec.", "spec", types.SeverityError),
	rule("spec.criteria-missing-scenario", "At 1.0.0 and above, every success criterion names at least one scenario.", "spec", types.SeverityError),
	// spec-wide: composition
	rule("ability.missing-composition", "At 1.0.0 and above, every non-leaf ability has a Composition section, unless it is the root ability the state machine orchestrates.", "spec", types.SeverityError),
	rule("ability.composition-on-leaf", "Composition appears only on abilities that have sub-abilities.", "spec", types.SeverityError),
	rule("ability.composition-unknown-child", "Every Ability named in Composition is a direct sub-ability.", "spec", types.SeverityError),
	rule("ability.composition-duplicate-child", "Each sub-ability appears in exactly one Composition row.", "spec", types.SeverityError),
	rule("ability.composition-omits-child", "Every direct sub-ability appears in Composition.", "spec", types.SeverityError),
	rule("ability.composition-step-order", "Composition steps are numbered 1..n, and every source is the parent or an earlier step.", "spec", types.SeverityError),
	rule("ability.composition-unknown-source", "Every value consumed from the parent is a parent input, and every value consumed from a step is produced by that step.", "spec", types.SeverityError),
	rule("ability.composition-unknown-produce", "Every value a sub-ability step produces is an output of that sub-ability, and every value a parent step produces is a parent output or consumed by a later step.", "spec", types.SeverityError),
	// spec-wide: delegation
	rule("ability.delegated-spec-not-found", "A delegated ability's **Spec:** path resolves to a directory containing spec.md (URLs are not checked).", "spec", types.SeverityError),
	rule("ability.delegated-version-mismatch", "A delegated ability's **Version:** equals the delegated spec's version.", "spec", types.SeverityError),
	rule("ability.delegated-multiple-roots", "A delegated spec has exactly one root ability.", "spec", types.SeverityError),
	// spec-wide: scenarios, decisions, state machine
	rule("scenario.unknown-node", "Every trace node is a state of the state machine, or an ability of the spec when there is no state machine.", "spec", types.SeverityError),
	rule("scenario.unknown-condition", "Every divergence condition is a transition trigger or a failure mode name.", "spec", types.SeverityError),
	rule("decision.context-names-no-ability", "A decision's Context names at least one ability of this spec in backticks.", "spec", types.SeverityError),
	rule("state-machine.unknown-ability", "Every ability named in the States table exists in the spec.", "spec", types.SeverityError),
	rule("state-machine.unknown-state", "Every From and To in Transitions is a state in the States table.", "spec", types.SeverityError),
	rule("state-machine.missing-trigger", "Every transition row names its trigger in backticks.", "spec", types.SeverityError),
	// spec-wide: coverage
	rule("coverage.root-failure-mode", "At 1.0.0 and above, every failure mode of every root ability appears as a divergence condition in at least one scenario.", "spec", types.SeverityError),
	rule("coverage.transition", "At 1.0.0 and above, every state machine transition appears in at least one scenario trace.", "spec", types.SeverityError),
	// spec-wide: formatting
	rule("format.crlf", "Files use LF line endings.", "spec", types.SeverityError),
	rule("format.final-newline", "Every file ends with exactly one newline.", "spec", types.SeverityError),
	rule("format.consecutive-blank-lines", "Files contain no consecutive blank lines.", "spec", types.SeverityError),
	rule("format.trailing-whitespace", "Lines have no trailing whitespace.", "spec", types.SeverityWarning),
	rule("format.h2-start", "Every spec file starts with an H2 heading.", "spec", types.SeverityError),
	rule("format.pipe-in-cell", "Table cells contain no pipe characters; escaped pipes are not permitted.", "spec", types.SeverityError),
	rule("format.table-padding", "Tables are in the canonical padded form.", "spec", types.SeverityWarning),
}

// rulesByVersion maps known AASDD versions to their structural rule sets.
var rulesByVersion = map[string][]types.Rule{
	"v1": v1Rules,
	"v2": append(append([]types.Rule{}, v1Rules...), v2Additions...),
}

// LoadRuleSet derives the complete rule set for the given AASDD version.
func LoadRuleSet(aasddVersion string) (types.RuleSet, error) {
	rules, ok := rulesByVersion[aasddVersion]
	if !ok {
		return types.RuleSet{}, &UnknownAASDDVersion{Version: aasddVersion}
	}

	ruleSet := types.RuleSet{
		AASDDVersion: aasddVersion,
		Rules:        rules,
	}

	// Invariant: rule_set.rules is non-empty.
	if len(ruleSet.Rules) == 0 {
		panic("LoadRuleSet: produced an empty rule set — this is a bug")
	}

	// Invariant: all IDs are unique.
	seen := make(map[string]struct{}, len(ruleSet.Rules))
	for _, r := range ruleSet.Rules {
		if _, dup := seen[r.ID]; dup {
			panic(fmt.Sprintf("LoadRuleSet: duplicate rule ID %q — this is a bug", r.ID))
		}
		seen[r.ID] = struct{}{}
	}

	return ruleSet, nil
}
