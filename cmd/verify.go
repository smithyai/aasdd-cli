package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smithyai/aasdd-cli/internal/analysis/verify"
	"github.com/smithyai/aasdd-cli/internal/analysis/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/format"
	"github.com/smithyai/aasdd-cli/internal/types"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <path>",
	Short: "Check a spec directory for AASDD conformance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}

		result, err := verify.Verify(target, Progress)
		if err != nil {
			var notFound *collect_violations.TargetNotFound
			var isFile *collect_violations.TargetIsFile
			var readErr *collect_violations.ReadError
			switch {
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			case errors.As(err, &isFile):
				fmt.Fprintf(os.Stderr, "error: %s\n", isFile)
			case errors.As(err, &readErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", readErr)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		if Verbose {
			fmt.Fprintf(os.Stderr, "AASDD %s (%d rules)\n", result.AASDDVersion, result.RuleCount)
		}

		if result.Passed && len(result.Violations) == 0 {
			fmt.Println("ok — spec is conformant")
			return nil
		}

		errCount, warnCount := format.WriteViolations(os.Stdout, result, Verbose)

		if result.Passed {
			// Only warnings — still conformant.
			fmt.Printf("\nok — spec is conformant (%d warning(s))\n", warnCount)
			return nil
		}

		fmt.Printf("\n%d error(s), %d warning(s)\n", errCount, warnCount)
		os.Exit(1)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
