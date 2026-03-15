package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smithyai/aasdd-cli/internal/analysis/diff"
	"github.com/smithyai/aasdd-cli/internal/types"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <left> <right>",
	Short: "Compare two spec directories for structural differences",
	Long: `Compare two spec directories and report all structural differences.

Differences are classified as Added (exists only in right), Removed (exists
only in left), or Changed (exists in both but differs structurally).

Comparison is structural, not textual — whitespace and formatting differences
that do not change parsed content are ignored.

Examples:
  aasdd diff ./spec-v1 ./spec-v2
  aasdd diff ./spec ./imported-spec`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		left := types.SpecTarget{Path: args[0]}
		right := types.SpecTarget{Path: args[1]}

		result, err := diff.Diff(left, right)
		if err != nil {
			var leftNotFound *diff.LeftNotFound
			var leftNotDir *diff.LeftNotDirectory
			var rightNotFound *diff.RightNotFound
			var rightNotDir *diff.RightNotDirectory
			var readErr *diff.ReadError
			switch {
			case errors.As(err, &leftNotFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", leftNotFound)
			case errors.As(err, &leftNotDir):
				fmt.Fprintf(os.Stderr, "error: %s\n", leftNotDir)
			case errors.As(err, &rightNotFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", rightNotFound)
			case errors.As(err, &rightNotDir):
				fmt.Fprintf(os.Stderr, "error: %s\n", rightNotDir)
			case errors.As(err, &readErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", readErr)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		if !result.Changed {
			fmt.Println("ok — no structural differences")
			return nil
		}

		for _, entry := range result.Entries {
			line := fmt.Sprintf("%s  %s  %s", entry.Kind, entry.Construct, entry.Path)
			if Verbose && entry.Detail != "" {
				line += "  " + entry.Detail
			}
			fmt.Println(line)
		}

		fmt.Printf("\n%d difference(s)\n", len(result.Entries))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
