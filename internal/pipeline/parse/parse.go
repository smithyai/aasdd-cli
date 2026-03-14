// Package parse implements the Parse ability.
package parse

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/types"
)

// SourceNotFound is returned when the source path does not exist.
type SourceNotFound struct {
	Path string
}

func (e *SourceNotFound) Error() string {
	return fmt.Sprintf("source not found: %q", e.Path)
}

// SourceUnrecognized is returned when the source is a file without a .json extension.
type SourceUnrecognized struct {
	Path string
}

func (e *SourceUnrecognized) Error() string {
	return fmt.Sprintf("source is not a directory or .json file: %q", e.Path)
}

// OutputNotEmpty is returned when the JSON-to-dir output location is occupied.
type OutputNotEmpty struct {
	Path string
}

func (e *OutputNotEmpty) Error() string {
	return fmt.Sprintf("output location is not empty: %q", e.Path)
}

// ParseError is returned when the source JSON cannot be decoded.
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string { return fmt.Sprintf("invalid JSON snapshot: %v", e.Err) }
func (e *ParseError) Unwrap() error { return e.Err }

// WriteError is returned when an output file or directory cannot be written.
type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string { return fmt.Sprintf("write error at %q: %v", e.Path, e.Err) }
func (e *WriteError) Unwrap() error { return e.Err }

// Parse converts a spec directory to JSON, or reconstructs a spec directory from JSON.
// Direction is inferred from source: directory → dir-to-JSON, .json file → JSON-to-dir.
// For dir-to-JSON, output is written to w when outputPath is empty; otherwise to outputPath.
// For JSON-to-dir, outputPath must be provided.
func Parse(source string, outputPath string, flat bool, w io.Writer) (types.ParseResult, error) {
	info, err := os.Stat(source)
	if err != nil {
		return types.ParseResult{}, &SourceNotFound{Path: source}
	}
	if info.IsDir() {
		return dirToJSON(source, outputPath, flat, w)
	}
	if strings.EqualFold(filepath.Ext(source), ".json") {
		return jsonToDir(source, outputPath)
	}
	return types.ParseResult{}, &SourceUnrecognized{Path: source}
}

func dirToJSON(source, outputPath string, flat bool, w io.Writer) (types.ParseResult, error) {
	snapshot, err := readDir(source)
	if err != nil {
		return types.ParseResult{}, err
	}

	var data []byte
	if flat {
		data, err = marshalFlat(snapshot)
	} else {
		data, err = marshalNested(snapshot)
	}
	if err != nil {
		return types.ParseResult{}, &WriteError{Path: outputPath, Err: err}
	}
	data = append(data, '\n')

	if outputPath == "" {
		if _, writeErr := w.Write(data); writeErr != nil {
			return types.ParseResult{}, &WriteError{Path: "-", Err: writeErr}
		}
		return types.ParseResult{
			Direction: types.ParseDirToJSON,
			FileCount: len(snapshot.Files),
		}, nil
	}

	if writeErr := os.WriteFile(outputPath, data, 0o644); writeErr != nil {
		return types.ParseResult{}, &WriteError{Path: outputPath, Err: writeErr}
	}
	return types.ParseResult{
		Direction:  types.ParseDirToJSON,
		FileCount:  len(snapshot.Files),
		OutputPath: outputPath,
	}, nil
}

func jsonToDir(source, outputPath string) (types.ParseResult, error) {
	// Check output location.
	if info, err := os.Stat(outputPath); err == nil {
		if !info.IsDir() {
			return types.ParseResult{}, &OutputNotEmpty{Path: outputPath}
		}
		entries, readErr := os.ReadDir(outputPath)
		if readErr != nil {
			return types.ParseResult{}, &WriteError{Path: outputPath, Err: readErr}
		}
		if len(entries) > 0 {
			return types.ParseResult{}, &OutputNotEmpty{Path: outputPath}
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return types.ParseResult{}, &SourceNotFound{Path: source}
	}

	snapshot, err := unmarshal(data)
	if err != nil {
		return types.ParseResult{}, &ParseError{Err: err}
	}

	for relPath, content := range snapshot.Files {
		fullPath := filepath.Join(outputPath, filepath.FromSlash(relPath))
		if mkErr := os.MkdirAll(filepath.Dir(fullPath), 0o755); mkErr != nil {
			return types.ParseResult{}, &WriteError{Path: fullPath, Err: mkErr}
		}
		if writeErr := os.WriteFile(fullPath, []byte(content), 0o644); writeErr != nil {
			return types.ParseResult{}, &WriteError{Path: fullPath, Err: writeErr}
		}
	}

	return types.ParseResult{
		Direction:  types.ParseJSONToDir,
		FileCount:  len(snapshot.Files),
		OutputPath: outputPath,
	}, nil
}

// specFileNames is the set of file basenames recognised as AASDD spec files.
var specFileNames = map[string]bool{
	"spec.md":     true,
	"ability.md":  true,
	"concept.md":  true,
	"scenario.md": true,
	"decision.md": true,
}

// readDir walks root and returns a SpecSnapshot containing only AASDD spec
// files (spec.md, ability.md, concept.md, scenario.md, decision.md).
// Hidden entries (names starting with '.') are skipped entirely.
func readDir(root string) (types.SpecSnapshot, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !specFileNames[d.Name()] {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		return types.SpecSnapshot{}, err
	}
	return types.SpecSnapshot{Files: files}, nil
}

// marshalFlat serializes a snapshot as a flat path→content JSON object.
func marshalFlat(s types.SpecSnapshot) ([]byte, error) {
	ordered := make(map[string]string, len(s.Files))
	for k, v := range s.Files {
		ordered[k] = v
	}
	return json.MarshalIndent(ordered, "", "  ")
}

// marshalNested serializes a snapshot as a nested directory-tree JSON object.
func marshalNested(s types.SpecSnapshot) ([]byte, error) {
	root := make(map[string]any)
	paths := make([]string, 0, len(s.Files))
	for p := range s.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		node := root
		for i, part := range parts {
			if i == len(parts)-1 {
				node[part] = s.Files[p]
			} else {
				if _, ok := node[part]; !ok {
					node[part] = make(map[string]any)
				}
				node = node[part].(map[string]any)
			}
		}
	}
	return json.MarshalIndent(root, "", "  ")
}

// unmarshal decodes a JSON snapshot in either flat or nested form.
// Both forms are handled uniformly: string values are file contents, object
// values are directory nodes (recursed). Keys may contain "/" in flat form.
func unmarshal(data []byte) (types.SpecSnapshot, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return types.SpecSnapshot{}, err
	}
	files := make(map[string]string)
	if err := flattenNode(raw, "", files); err != nil {
		return types.SpecSnapshot{}, err
	}
	return types.SpecSnapshot{Files: files}, nil
}

func flattenNode(node map[string]any, prefix string, out map[string]string) error {
	for key, val := range node {
		path := key
		if prefix != "" {
			path = prefix + "/" + key
		}
		switch v := val.(type) {
		case string:
			out[path] = v
		case map[string]any:
			if err := flattenNode(v, path, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected value type at %q: %T", path, val)
		}
	}
	return nil
}
