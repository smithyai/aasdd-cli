package types

// ScaffoldResult is the outcome of scaffolding a new spec directory.
type ScaffoldResult struct {
	Target       SpecTarget
	FilesCreated []string
}
