package format

import (
	"fmt"
	"github.com/smithyai/aasdd-cli/internal/types"
	"io"
)

func ViolationLabel(severity types.Severity) string {
	if severity == types.SeverityWarning {
		return "warning"
	}
	return "error"
}

func WriteViolations(w io.Writer, result types.VerificationResult) (errCount, warnCount int) {
	for _, v := range result.Violations {
		label := ViolationLabel(v.Severity)
		if v.Severity == types.SeverityWarning {
			warnCount++
		} else {
			errCount++
		}
		fmt.Fprintf(w, "  %s  %s  %s\n    %s\n", label, v.Rule, v.Path, v.Message)
	}
	return
}
