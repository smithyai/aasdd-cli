package live_test

import (
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/types"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerify_OwnSpec(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	specDir := filepath.Join(repoRoot, "spec")
	result, err := verify.Verify(types.SpecTarget{Path: specDir}, nil)
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	for _, v := range result.Violations {
		label := "warning"
		if v.Severity == types.SeverityError {
			label = "error"
		}
		t.Logf("%s  %s  %s", label, v.Rule, v.Path)
	}
	if result.Passed {
		return
	}
	errCount := 0
	for _, v := range result.Violations {
		if v.Severity == types.SeverityError {
			errCount++
		}
	}
	t.Errorf("own spec is not conformant: %d error(s) found", errCount)
}
