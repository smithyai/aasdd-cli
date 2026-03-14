package integration_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestVerifyOutput_WarningLabelAppears(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n\n**Summary:** A test spec.\n")
	if err := os.MkdirAll(filepath.Join(dir, "abilities"), 0o755); err != nil {
		t.Fatalf("mkdir abilities: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "concepts"), 0o755); err != nil {
		t.Fatalf("mkdir concepts: %v", err)
	}
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := func() string {
		var sb strings.Builder
		format.WriteViolations(&sb, result, false)
		return sb.String()
	}()
	if !strings.Contains(out, "warning") {
		t.Errorf("expected output to contain word warning, got:\n%s", out)
	}
	if strings.Contains(out, "error") {
		t.Errorf("expected no error label in output, got:\n%s", out)
	}
}

func TestVerifyOutput_ErrorLabelAppears(t *testing.T) {
	dir := t.TempDir()
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := func() string {
		var sb strings.Builder
		format.WriteViolations(&sb, result, false)
		return sb.String()
	}()
	if !strings.Contains(out, "error") {
		t.Errorf("expected output to contain word error, got:\n%s", out)
	}
}

func TestVerifyOutput_ProgressAndVerboseTogether(t *testing.T) {
	dir := t.TempDir()
	// A spec.md that will produce a warning (missing **Version:**) —
	// gives us both a progress line for the file and a verbose description on the violation.
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n\n**AASDD:** v1\n**Summary:** A test spec.\n")

	// Capture stderr to verify progress output.
	oldStderr := os.Stderr
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stderr = w

	result, err := verify.Verify(types.SpecTarget{Path: dir}, true)

	w.Close()
	os.Stderr = oldStderr
	captured, _ := io.ReadAll(r)
	progressOut := string(captured)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// --progress: stderr should have received at least one ok line.
	if !strings.Contains(progressOut, "ok") {
		t.Errorf("expected progress output to contain 'ok', got:\n%s", progressOut)
	}
	// The root directory marker must appear.
	if !strings.Contains(progressOut, "ok  .") {
		t.Errorf("expected progress output to contain 'ok  .', got:\n%s", progressOut)
	}
	// spec.md should appear as a validated file.
	if !strings.Contains(progressOut, "spec.md") {
		t.Errorf("expected progress output to contain 'spec.md', got:\n%s", progressOut)
	}

	// --verbose: WriteViolations with verbose=true should include the rule description.
	var violationBuf strings.Builder
	format.WriteViolations(&violationBuf, result, true)
	violationOut := violationBuf.String()
	if len(result.Violations) > 0 && result.Violations[0].Description == "" {
		t.Error("expected violations to carry a non-empty Description when verbose=true")
	}
	for _, v := range result.Violations {
		if !strings.Contains(violationOut, v.Description) {
			t.Errorf("expected violation output to contain description %q, got:\n%s", v.Description, violationOut)
		}
	}
}
