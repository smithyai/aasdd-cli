package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestVerifyOutput_WarningLabelAppears(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spec.md"), "## My Spec\n\n**Status:** Draft\n**Summary:** A test spec.\n")
	result, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := func() string {
		var sb strings.Builder
		format.WriteViolations(&sb, result)
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
	result, err := verify.Verify(types.SpecTarget{Path: dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := func() string {
		var sb strings.Builder
		format.WriteViolations(&sb, result)
		return sb.String()
	}()
	if !strings.Contains(out, "error") {
		t.Errorf("expected output to contain word error, got:\n%s", out)
	}
}
