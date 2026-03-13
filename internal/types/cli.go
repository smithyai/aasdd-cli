package types

// SpecTarget is the spec directory path supplied by the caller.
type SpecTarget struct {
	Path string
}

// SpecVersion is a pinned AASDD spec version to verify or scaffold against.
type SpecVersion struct {
	Value string
}
