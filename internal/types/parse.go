package types

// ParseDirection indicates whether a Parse operation reads a directory or a JSON snapshot.
type ParseDirection string

const (
	ParseDirToJSON ParseDirection = "dir-to-json"
	ParseJSONToDir ParseDirection = "json-to-dir"
)

// SpecSnapshot is a portable, path-indexed representation of a spec directory's file contents.
// All paths use forward slashes and are relative to the spec root.
type SpecSnapshot struct {
	Files map[string]string
}

// ParseResult is the outcome of a Parse operation.
type ParseResult struct {
	Direction  ParseDirection
	FileCount  int
	OutputPath string // empty when written to stdout
}
