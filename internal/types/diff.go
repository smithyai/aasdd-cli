package types

// DiffKind classifies how a construct differs between two specs.
type DiffKind string

const (
	DiffAdded   DiffKind = "Added"
	DiffRemoved DiffKind = "Removed"
	DiffChanged DiffKind = "Changed"
)

// DiffEntry is a single structural difference between two specs.
type DiffEntry struct {
	Path      string   `json:"path"`
	Kind      DiffKind `json:"kind"`
	Construct string   `json:"construct"`
	Detail    string   `json:"detail"`
}

// DiffResult is the outcome of comparing two spec directories.
type DiffResult struct {
	Left    SpecTarget  `json:"left"`
	Right   SpecTarget  `json:"right"`
	Entries []DiffEntry `json:"entries"`
	Changed bool        `json:"changed"`
}
