package cmd

import (
	"errors"
	"fmt"
	"os"

	import_ "github.com/smithyai/aasdd-cli/internal/transfer/import"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <source> <output>",
	Short: "Reconstruct a spec directory from a snapshot",
	Long: `Reconstruct a spec directory on disk from a previously exported snapshot.

The <source> argument must be a JSON snapshot file produced by 'aasdd export'.
The <output> directory must be empty or nonexistent.

Examples:
  aasdd import spec.json ./spec`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := import_.Import(args[0], args[1])
		if err != nil {
			var notFound *import_.SourceNotFound
			var notFile *import_.SourceNotFile
			var notEmpty *import_.OutputNotEmpty
			var parseErr *import_.ParseError
			var writeErr *import_.WriteError
			switch {
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			case errors.As(err, &notFile):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFile)
			case errors.As(err, &notEmpty):
				fmt.Fprintf(os.Stderr, "error: %s\n", notEmpty)
			case errors.As(err, &parseErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", parseErr)
			case errors.As(err, &writeErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", writeErr)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "wrote %d file(s) to %s\n", result.FileCount, result.OutputPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
