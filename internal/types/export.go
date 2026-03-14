package types

// SpecExport is the top-level structured representation of a spec directory.
type SpecExport struct {
	ParsedSpecFile
	Abilities []ParsedAbility  `json:"abilities,omitempty"`
	Concepts  []ParsedConcept  `json:"concepts,omitempty"`
	Decisions []ParsedDecision `json:"decisions,omitempty"`
	Scenarios []ParsedScenario `json:"scenarios,omitempty"`
}

// ParsedSpecFile holds the structured content of a spec.md file.
type ParsedSpecFile struct {
	Heading      string   `json:"heading"`
	AASDDVersion string   `json:"aasdd"`
	Version      string   `json:"version"`
	Summary      string   `json:"summary"`
	Invariants   []string `json:"invariants,omitempty"`
}

// Table is a parsed Markdown table with column order preserved.
type Table struct {
	Headers []string            `json:"headers"`
	Rows    []map[string]string `json:"rows"`
}

// ParsedAbility holds the structured content of an ability.md file.
type ParsedAbility struct {
	Path          string          `json:"path"`
	Heading       string          `json:"heading"`
	Purpose       string          `json:"purpose,omitempty"`
	Inputs        *Table          `json:"inputs,omitempty"`
	Outputs       *Table          `json:"outputs,omitempty"`
	OutputsNote   string          `json:"outputs_note,omitempty"`
	Invariants    []string        `json:"invariants,omitempty"`
	FailureModes  *Table          `json:"failure_modes,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	Visualization string          `json:"visualization,omitempty"`
	SubAbilities  []ParsedAbility `json:"sub_abilities,omitempty"`
}

// ParsedConcept holds the structured content of a concept.md file.
type ParsedConcept struct {
	Path    string        `json:"path"`
	Heading string        `json:"heading"`
	Intro   string        `json:"intro,omitempty"`
	Types   []ConceptType `json:"types,omitempty"`
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
	Path        string   `json:"path"`
	Heading     string   `json:"heading"`
	Description string   `json:"description,omitempty"`
	Trace       string   `json:"trace"`
	Assertions  []string `json:"assertions,omitempty"`
}

// ParsedDecision holds the structured content of a decision.md file.
type ParsedDecision struct {
	Path        string `json:"path"`
	Heading     string `json:"heading"`
	Context     string `json:"context,omitempty"`
	Requirement string `json:"requirement,omitempty"`
	Decision    string `json:"decision,omitempty"`
}

// TransferResult is the outcome of a transfer operation (export or import).
type TransferResult struct {
	FileCount  int    `json:"file_count"`
	OutputPath string `json:"output_path,omitempty"`
}
