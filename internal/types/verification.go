package types

// Rule is a single structural check derived from the AASDD conventions for a given spec version.
type Rule struct {
	ID          string
	Description string
	AppliesTo   string
}

// RuleSet is the complete set of structural rules for a given AASDD spec version.
type RuleSet struct {
	SpecVersion SpecVersion
	Rules       []Rule
}

// Violation is a single rule failure found during verification.
type Violation struct {
	Rule    string // matches Rule.ID
	Path    string
	Message string
}

// VerificationResult is the outcome of verifying a spec directory.
type VerificationResult struct {
	Target     SpecTarget
	Violations []Violation
	Passed     bool
}
