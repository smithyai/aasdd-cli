package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smithyai/aasdd-cli/internal/transfer/export"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <source> [output]",
	Short: "Export a spec directory to a portable snapshot file",
	Long: `Serialize a spec directory into a portable JSON snapshot file.

Every spec file (spec.md, ability.md, concept.md, scenario.md, decision.md)
under <source> is included. All other files and hidden entries are excluded.

The snapshot can later be reconstructed into a spec directory with 'aasdd import'.

Examples:
  aasdd export ./spec
  aasdd export ./spec spec.json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		outputPath := ""
		if len(args) == 2 {
			outputPath = args[1]
		}

		result, err := export.Export(source, outputPath, os.Stdout)
		if err != nil {
			var notFound *export.SourceNotFound
			var notDir *export.SourceNotDirectory
			var writeErr *export.WriteError
			switch {
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			case errors.As(err, &notDir):
				fmt.Fprintf(os.Stderr, "error: %s\n", notDir)
			case errors.As(err, &writeErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", writeErr)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		if result.OutputPath != "" {
			fmt.Fprintf(os.Stderr, "wrote %d file(s) to %s\n", result.FileCount, result.OutputPath)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
