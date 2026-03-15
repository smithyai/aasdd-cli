// Package scaffold implements the Scaffold ability.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// TargetNotEmpty is returned when the target directory already contains files.
type TargetNotEmpty struct {
	Path string
}

func (e *TargetNotEmpty) Error() string {
	return fmt.Sprintf("target is not empty: %q", e.Path)
}

// TargetIsFile is returned when the target path exists and is a file, not a
// directory.
type TargetIsFile struct {
	Path string
}

func (e *TargetIsFile) Error() string {
	return fmt.Sprintf("target is a file, not a directory: %q", e.Path)
}

// WriteError is returned when a file cannot be written due to a filesystem
// permission or I/O error. Any files written before the error are left in place.
type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("write error at %q: %v", e.Path, e.Err)
}

func (e *WriteError) Unwrap() error { return e.Err }

// UnknownAASDDVersion is returned when the requested AASDD version is not
// recognised by the tool.
type UnknownAASDDVersion struct {
	Version string
}

func (e *UnknownAASDDVersion) Error() string {
	return fmt.Sprintf("unknown AASDD version: %q", e.Version)
}

const latestVersion = "v1"

// scaffoldFile represents a file to be created during scaffolding.
type scaffoldFile struct {
	RelPath string
	Content string
}

// templatesByVersion maps known AASDD spec versions to the files they scaffold.
var templatesByVersion = map[string][]scaffoldFile{
	"v1": {
		{
			RelPath: "spec.md",
			Content: "## <Name>\n\n**AASDD:** v1\n**Version:** 0.1.0\n\n<One sentence describing what this spec covers.>\n",
		},
		{
			RelPath: "README.md",
			Content: "# Spec\n\nThis directory contains the AASDD spec. See `spec.md` for top-level metadata and invariants.\n",
		},
		{
			RelPath: filepath.Join("abilities", "README.md"),
			Content: "# Abilities\n\nEach subdirectory defines one ability — a named, specifiable behaviour of the system. An ability directory contains an `ability.md` file.\n",
		},
		{
			RelPath: filepath.Join("concepts", "README.md"),
			Content: "# Concepts\n\nEach subdirectory defines one concept domain — the types and data structures shared across abilities. A concept directory contains a `concept.md` file.\n",
		},
		{
			RelPath: filepath.Join("decisions", "README.md"),
			Content: "# Decisions\n\nEach subdirectory records one architectural or design decision. A decision directory contains a `decision.md` file.\n",
		},
		{
			RelPath: filepath.Join("scenarios", "README.md"),
			Content: "# Scenarios\n\nEach subdirectory defines one scenario — a concrete example that exercises an ability end-to-end. A scenario directory contains a `scenario.md` file.\n",
		},
	},
}

// exampleTemplatesByVersion maps known AASDD spec versions to worked-example templates
// used when --example is passed to the scaffold command.
var exampleTemplatesByVersion = map[string][]scaffoldFile{
	"v1": {
		{
			RelPath: "spec.md",
			Content: "## Greeter\n\n**AASDD:** v1\n**Version:** 0.1.0\n\nA minimal service that produces a personalised greeting for a given name.\n",
		},
		{
			RelPath: "README.md",
			Content: "# Spec\n\nThis directory contains the AASDD spec. See `spec.md` for top-level metadata and invariants.\n",
		},
		{
			RelPath: filepath.Join("abilities", "README.md"),
			Content: "# Abilities\n\nEach subdirectory defines one ability — a named, specifiable behaviour of the system. An ability directory contains an `ability.md` file.\n",
		},
		{
			RelPath: filepath.Join("abilities", "greet", "ability.md"),
			Content: "## Greet\n\nProduces a personalised greeting for the given name.\n\n### Inputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `name` | text | The name to greet. Must be non-empty. |\n\n### Outputs\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `result` | [GreetingResult](../../concepts/greeting/concept.md#greetingresult) | The produced greeting. |\n\n### Invariants\n\n- `result.message` contains the value of `name`.\n\n### State Machine\n\n| From | Event | To | Action |\n| --- | --- | --- | --- |\n| `Idle` | `Greet(name)` | `Done` | Build and store the greeting in `result.message`. |\n\n### Failure Modes\n\n| Failure | Condition | Effect |\n| --- | --- | --- |\n| `EmptyName` | `name` is empty. | Error returned to caller. |\n",
		},
		{
			RelPath: filepath.Join("concepts", "README.md"),
			Content: "# Concepts\n\nEach subdirectory defines one concept domain — the types and data structures shared across abilities. A concept directory contains a `concept.md` file.\n",
		},
		{
			RelPath: filepath.Join("concepts", "greeting", "concept.md"),
			Content: "## Greeting domain\n\nTypes representing the output of a greeting operation.\n\n### GreetingResult\n\nThe outcome of a greet operation.\n\n#### Properties\n\n| Name | Type | Description |\n| --- | --- | --- |\n| `message` | text | The full greeting string (e.g. `Hello, Alice!`). |\n",
		},
		{
			RelPath: filepath.Join("decisions", "README.md"),
			Content: "# Decisions\n\nEach subdirectory records one architectural or design decision. A decision directory contains a `decision.md` file.\n",
		},
		{
			RelPath: filepath.Join("decisions", "output-channel", "decision.md"),
			Content: "## OutputChannel\n\n### Context\n\nThe greeting must be delivered to the caller without coupling the core logic to a specific I/O channel.\n\n### Requirement\n\nAn output mechanism that keeps rendering concerns separate from the ability logic.\n\n### Decision\n\nReturn value — `Greet` returns a `GreetingResult` to its caller. The caller is responsible for rendering the message.\n",
		},
		{
			RelPath: filepath.Join("scenarios", "README.md"),
			Content: "# Scenarios\n\nEach subdirectory defines one scenario — a concrete example that exercises an ability end-to-end. A scenario directory contains a `scenario.md` file.\n",
		},
		{
			RelPath: filepath.Join("scenarios", "happy-path", "scenario.md"),
			Content: "## HappyPath\n\nA non-empty name produces a greeting that contains that name.\n\n> `Greet`\n\n- `name` is `Alice`\n- `result.message` contains `Alice`\n",
		},
	},
}

// Scaffold creates a spec directory populated with the correct structure and
// stub files for a new AASDD spec. Pass an empty aasddVersion to use the
// latest version. When example is true, a worked example is written instead
// of minimal stubs.
func Scaffold(target types.SpecTarget, aasddVersion string, example bool) (types.ScaffoldResult, error) {
	if aasddVersion == "" {
		aasddVersion = latestVersion
	}
	templates, ok := templatesByVersion[aasddVersion]
	if !ok {
		return types.ScaffoldResult{}, &UnknownAASDDVersion{Version: aasddVersion}
	}
	if example {
		exTmpl, ok := exampleTemplatesByVersion[aasddVersion]
		if !ok {
			return types.ScaffoldResult{}, &UnknownAASDDVersion{Version: aasddVersion}
		}
		templates = exTmpl
	}

	// Failure mode: TargetIsFile — target exists and is a file, not a directory.
	if info, err := os.Stat(target.Path); err == nil && !info.IsDir() {
		return types.ScaffoldResult{}, &TargetIsFile{Path: target.Path}
	}

	// Failure mode: TargetNotEmpty — directory exists and already contains files.
	if entries, err := os.ReadDir(target.Path); err == nil && len(entries) > 0 {
		return types.ScaffoldResult{}, &TargetNotEmpty{Path: target.Path}
	}

	var filesCreated []string

	for _, tmpl := range templates {
		fullPath := filepath.Join(target.Path, tmpl.RelPath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return types.ScaffoldResult{}, &WriteError{Path: fullPath, Err: err}
		}

		if err := os.WriteFile(fullPath, []byte(tmpl.Content), 0o644); err != nil {
			return types.ScaffoldResult{}, &WriteError{Path: fullPath, Err: err}
		}

		filesCreated = append(filesCreated, fullPath)
	}

	result := types.ScaffoldResult{
		Target:       target,
		FilesCreated: filesCreated,
	}

	// Invariant: result.files_created is non-empty.
	if len(result.FilesCreated) == 0 {
		panic("Scaffold: produced no files — this is a bug")
	}

	// Invariant: every file in result.files_created exists on disk.
	for _, p := range result.FilesCreated {
		if _, err := os.Stat(p); err != nil {
			panic(fmt.Sprintf("Scaffold: created file %q does not exist after write — this is a bug", p))
		}
	}

	return result, nil
}
