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

// UnknownSpecVersion is returned when the requested spec version is not
// recognised by the tool.
type UnknownSpecVersion struct {
	Version string
}

func (e *UnknownSpecVersion) Error() string {
	return fmt.Sprintf("unknown spec version: %q", e.Version)
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
			Content: "## <Name>\n\n**AASDD:** v1\n**Version:** 0.1.0\n**Status:** Draft\n**Summary:** <One sentence describing what this spec covers.>\n",
		},
		{
			RelPath: filepath.Join("abilities", ".gitkeep"),
			Content: "",
		},
		{
			RelPath: filepath.Join("concepts", ".gitkeep"),
			Content: "",
		},
		{
			RelPath: filepath.Join("decisions", ".gitkeep"),
			Content: "",
		},
	},
}

// Scaffold creates a spec directory populated with the correct structure and
// stub files for a new AASDD spec.
//
// When specVersion is nil, the latest known version is used.
func Scaffold(target types.SpecTarget, specVersion *types.SpecVersion) (types.ScaffoldResult, error) {
	resolved := latestVersion
	if specVersion != nil {
		resolved = specVersion.Value
	}

	templates, ok := templatesByVersion[resolved]
	if !ok {
		return types.ScaffoldResult{}, &UnknownSpecVersion{Version: resolved}
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
