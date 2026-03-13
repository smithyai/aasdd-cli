package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/collect_violations"
	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/load_rule_set"
	"github.com/smithyai/aasdd-cli/internal/types"
)

var verifySpecVersion string

var verifyCmd = &cobra.Command{
	Use:   "verify <path>",
	Short: "Check a spec directory for AASDD conformance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}

		var sv *types.SpecVersion
		if verifySpecVersion != "" {
			sv = &types.SpecVersion{Value: verifySpecVersion}
		}

		result, err := verify.Verify(target, sv)
		if err != nil {
			var unknownVer *load_rule_set.UnknownSpecVersion
			var notFound *collect_violations.TargetNotFound
			switch {
			case errors.As(err, &unknownVer):
				fmt.Fprintf(os.Stderr, "error: %s\n", unknownVer)
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		if result.Passed {
			fmt.Println("ok — spec is conformant")
			return nil
		}

		for _, v := range result.Violations {
			fmt.Printf("  %s  %s\n    %s\n", v.Rule, v.Path, v.Message)
		}
		fmt.Printf("\n%d violation(s) found\n", len(result.Violations))
		os.Exit(1)
		return nil
	},
}

func init() {
	verifyCmd.Flags().StringVar(&verifySpecVersion, "spec-version", "", "AASDD version to verify against (default: latest)")
	rootCmd.AddCommand(verifyCmd)
}
