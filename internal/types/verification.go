package types

// Severity indicates whether a rule violation blocks conformance or is advisory only.
type Severity int

const (
	SeverityError   Severity = iota // Spec is structurally unusable.
	SeverityWarning                 // Spec is sound but violates a convention or is missing tooling metadata.
)

// Rule is a single structural check derived from the AASDD conventions for a given spec version.
type Rule struct {
	ID          string
	Description string
	AppliesTo   string
	Severity    Severity
}

// RuleSet is the complete set of structural rules for a given AASDD version.
type RuleSet struct {
	AASDDVersion string
	Rules        []Rule
}

// Violation is a single rule failure found during verification.
type Violation struct {
	Rule        string // matches Rule.ID
	Severity    Severity
	Description string // copied from Rule.Description
	Path        string
	Message     string
}

// VerificationResult is the outcome of verifying a spec directory.
type VerificationResult struct {
	Target        SpecTarget
	AASDDVersion  string // AASDD version used during verification
	RuleCount     int    // number of rules evaluated
	Violations    []Violation
	Passed        bool     // true when there are no Error-severity violations
	SpecFileNames []string // basenames of files that at least one rule applies to
}
