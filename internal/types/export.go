package types

// Placeholder values recorded when a section carries a marker instead of content.
const (
	PlaceholderNone    = "None"    // the section was _None._
	PlaceholderPending = "Pending" // the section was _Pending._
	PlaceholderOpen    = "Open"    // the section was _Open._
)

// SpecExport is the top-level structured representation of a spec directory.
type SpecExport struct {
	ParsedSpecFile
	Abilities    []ParsedAbility     `json:"abilities,omitempty"`
	Concepts     []ParsedConcept     `json:"concepts,omitempty"`
	Decisions    []ParsedDecision    `json:"decisions,omitempty"`
	Scenarios    []ParsedScenario    `json:"scenarios,omitempty"`
	StateMachine *ParsedStateMachine `json:"state_machine,omitempty"`
}

// CustomSection is an author-defined section with no methodology semantics.
// Content is stored as raw markdown (no parsing).
type CustomSection struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// ParsedSpecFile holds the structured content of a spec.md file.
type ParsedSpecFile struct {
	Heading         string            `json:"heading"`
	AASDDVersion    string            `json:"aasdd"`
	Version         string            `json:"version"`
	Summary         string            `json:"summary"`
	Purpose         string            `json:"purpose,omitempty"`
	NonGoals        []string          `json:"non_goals,omitempty"`
	SuccessCriteria *Table            `json:"success_criteria,omitempty"`
	Invariants      []string          `json:"invariants,omitempty"`
	FailureModes    *Table            `json:"failure_modes,omitempty"`
	Placeholders    map[string]string `json:"placeholders,omitempty"`
	CustomSections  []CustomSection   `json:"custom_sections,omitempty"`
}

// Table is a parsed Markdown table with column order preserved.
type Table struct {
	Headers []string            `json:"headers"`
	Rows    []map[string]string `json:"rows"`
}

// ParsedAbility holds the structured content of an ability.md file.
type ParsedAbility struct {
	Path           string            `json:"-"`
	Heading        string            `json:"heading"`
	Purpose        string            `json:"purpose,omitempty"`
	Spec           string            `json:"spec,omitempty"`
	SpecVersion    string            `json:"spec_version,omitempty"`
	Inputs         *Table            `json:"inputs,omitempty"`
	Outputs        *Table            `json:"outputs,omitempty"`
	OutputsNote    string            `json:"outputs_note,omitempty"`
	Invariants     []string          `json:"invariants,omitempty"`
	FailureModes   *Table            `json:"failure_modes,omitempty"`
	Idempotency    string            `json:"idempotency,omitempty"`
	Composition    *Table            `json:"composition,omitempty"`
	Placeholders   map[string]string `json:"placeholders,omitempty"`
	CustomSections []CustomSection   `json:"custom_sections,omitempty"`
	SubAbilities   []ParsedAbility   `json:"sub_abilities,omitempty"`
}

// Delegated reports whether the ability is defined by another spec.
func (a ParsedAbility) Delegated() bool { return a.Spec != "" }

// ParsedConcept holds the structured content of a concept.md file.
type ParsedConcept struct {
	Path           string          `json:"-"`
	Heading        string          `json:"heading"`
	Intro          string          `json:"intro,omitempty"`
	Types          []ConceptType   `json:"types,omitempty"`
	CustomSections []CustomSection `json:"custom_sections,omitempty"`
}

// ConceptType is a named type within a concept.md file.
type ConceptType struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	PropertiesHeading bool   `json:"properties_heading,omitempty"`
	Note              string `json:"note,omitempty"`
	Properties        *Table `json:"properties,omitempty"`
}

// ParsedScenario holds the structured content of a scenario.md file.
type ParsedScenario struct {
	Path           string          `json:"-"`
	Heading        string          `json:"heading"`
	Description    string          `json:"description,omitempty"`
	Trace          string          `json:"trace"`
	Assertions     []string        `json:"assertions,omitempty"`
	Example        string          `json:"example,omitempty"`
	CustomSections []CustomSection `json:"custom_sections,omitempty"`
}

// ParsedDecision holds the structured content of a decision.md file.
type ParsedDecision struct {
	Path           string            `json:"-"`
	Heading        string            `json:"heading"`
	Context        string            `json:"context,omitempty"`
	Requirement    string            `json:"requirement,omitempty"`
	Options        []string          `json:"options,omitempty"`
	Decision       string            `json:"decision,omitempty"`
	Placeholders   map[string]string `json:"placeholders,omitempty"`
	CustomSections []CustomSection   `json:"custom_sections,omitempty"`
}

// Open reports whether the decision has not been made yet.
func (d ParsedDecision) Open() bool { return d.Placeholders["Decision"] == PlaceholderOpen }

// ParsedStateMachine holds the structured content of a state-machine.md file.
type ParsedStateMachine struct {
	Summary           string            `json:"summary"`
	Diagram           string            `json:"diagram"`
	Orchestrator      string            `json:"orchestrator"`
	OrchestratorState *Table            `json:"orchestrator_managed_state,omitempty"`
	States            *Table            `json:"states"`
	Transitions       *Table            `json:"transitions"`
	TransitionRules   []string          `json:"transition_rules,omitempty"`
	ExceptionalFlows  []ExceptionalFlow `json:"exceptional_flows,omitempty"`
	Placeholders      map[string]string `json:"placeholders,omitempty"`
	CustomSections    []CustomSection   `json:"custom_sections,omitempty"`
}

// ExceptionalFlow is a named exceptional flow within a state machine.
type ExceptionalFlow struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// TransferResult is the outcome of a transfer operation (export or import).
type TransferResult struct {
	FileCount  int    `json:"file_count"`
	OutputPath string `json:"output_path,omitempty"`
}
