package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify"
	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

func TestScaffold_DefaultVersionIsV2(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "**AASDD:** v2") {
		t.Errorf("default scaffold should record v2, got:\n%s", data)
	}
}

func verifyNoErrors(t *testing.T, dir string) {
	t.Helper()
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.AASDDVersion != "v2" {
		t.Errorf("verified against %s, want v2", result.AASDDVersion)
	}
	for _, v := range result.Violations {
		if v.Severity == types.SeverityError {
			t.Errorf("error: %s %s: %s", v.Rule, v.Path, v.Message)
		}
	}
	if !result.Passed {
		t.Error("expected the scaffolded spec to pass")
	}
}

func TestScaffold_V2Minimal_VerifiesAsDraft(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v2", false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	verifyNoErrors(t, dir)
}

func TestScaffold_V2Example_VerifiesWithoutWarnings(t *testing.T) {
	dir := t.TempDir()
	if _, err := scaffold.Scaffold(types.SpecTarget{Path: dir}, "v2", true); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	result, err := verify.Verify(types.SpecTarget{Path: dir}, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(result.Violations) != 0 {
		for _, v := range result.Violations {
			t.Errorf("%s %s: %s", v.Rule, v.Path, v.Message)
		}
	}
}
