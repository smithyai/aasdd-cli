package format

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/types"
)

func ViolationLabel(severity types.Severity) string {
	if severity == types.SeverityWarning {
		return "warning"
	}
	return "error"
}

// Status prefix labels — all padded to 4 characters so columns align.
const (
	treePrefixOK   = "ok  "
	treePrefixWarn = "warn"
	treePrefixErr  = "err "
	treePrefixInfo = "info"
	treePrefixDir  = "    " // directories have no status
)

// WriteFileTree prints an indented directory tree of result.Target.Path to w,
// annotating every file with its verification status:
//
//	ok   – spec file, no violations
//	warn – spec file, warning-severity violations only
//	err  – spec file, at least one error-severity violation
//	info – file not matched by any rule
//
// Directory entries are printed without a status prefix to serve as structural
// markers. Unreadable entries are silently skipped.
func WriteFileTree(w io.Writer, result types.VerificationResult) {
	specFileNames := make(map[string]struct{}, len(result.SpecFileNames))
	for _, name := range result.SpecFileNames {
		specFileNames[name] = struct{}{}
	}

	// Build absolute-path → worst Severity map for violated files.
	// SeverityError = 0, SeverityWarning = 1 — lower value is worse.
	worstSeverity := make(map[string]types.Severity)
	hasViolation := make(map[string]bool)
	for _, v := range result.Violations {
		hasViolation[v.Path] = true
		if existing, seen := worstSeverity[v.Path]; !seen || v.Severity < existing {
			worstSeverity[v.Path] = v.Severity
		}
	}

	_ = filepath.WalkDir(result.Target.Path, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}
		rel, _ := filepath.Rel(result.Target.Path, path)
		if rel == "." {
			return nil // skip the root itself
		}
		depth := strings.Count(rel, string(filepath.Separator))
		indent := strings.Repeat("  ", depth)
		name := d.Name()

		if d.IsDir() {
			fmt.Fprintf(w, "%s  %s%s/\n", treePrefixDir, indent, name)
			return nil
		}

		prefix := treeFilePrefix(path, name, specFileNames, worstSeverity, hasViolation)
		fmt.Fprintf(w, "%s  %s%s\n", prefix, indent, name)
		return nil
	})
}

func treeFilePrefix(
	absPath, basename string,
	specFileNames map[string]struct{},
	worstSeverity map[string]types.Severity,
	hasViolation map[string]bool,
) string {
	if hasViolation[absPath] {
		if worstSeverity[absPath] == types.SeverityError {
			return treePrefixErr
		}
		return treePrefixWarn
	}
	if _, ok := specFileNames[basename]; ok {
		return treePrefixOK
	}
	return treePrefixInfo
}

func WriteViolations(w io.Writer, result types.VerificationResult, verbose bool) (errCount, warnCount int) {
	for _, v := range result.Violations {
		label := ViolationLabel(v.Severity)
		if v.Severity == types.SeverityWarning {
			warnCount++
		} else {
			errCount++
		}
		fmt.Fprintf(w, "  %s  %s  %s\n    %s\n", label, v.Rule, v.Path, v.Message)
		if verbose && v.Description != "" {
			fmt.Fprintf(w, "    %s\n", v.Description)
		}
	}
	return
}
