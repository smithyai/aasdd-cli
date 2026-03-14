package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/pipeline/parse"
	"github.com/spf13/cobra"
)

var parseFlat bool

var parseCmd = &cobra.Command{
	Use:   "parse <source> [output]",
	Short: "Convert a spec directory to JSON, or reconstruct a directory from JSON",
	Long: `Convert a spec directory to a JSON snapshot, or reconstruct a spec directory from one.

Direction is inferred from the source path:
  directory   → dir-to-JSON (output defaults to stdout)
  .json file  → JSON-to-dir (output path required)

Examples:
  aasdd parse ./spec
  aasdd parse ./spec --flat
  aasdd parse ./spec spec.json
  aasdd parse ./spec spec.json --flat
  aasdd parse spec.json ./spec`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		outputPath := ""
		if len(args) == 2 {
			outputPath = args[1]
		}

		// For JSON-to-dir, an output directory is required.
		if len(args) == 1 && strings.EqualFold(filepath.Ext(source), ".json") {
			fmt.Fprintln(os.Stderr, "error: output directory required when source is a JSON file")
			os.Exit(1)
		}

		result, err := parse.Parse(source, outputPath, parseFlat, os.Stdout)
		if err != nil {
			var notFound *parse.SourceNotFound
			var unrecognized *parse.SourceUnrecognized
			var notEmpty *parse.OutputNotEmpty
			var parseErr *parse.ParseError
			var writeErr *parse.WriteError
			switch {
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			case errors.As(err, &unrecognized):
				fmt.Fprintf(os.Stderr, "error: %s\n", unrecognized)
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

		if result.OutputPath != "" {
			fmt.Fprintf(os.Stderr, "wrote %d file(s) to %s\n", result.FileCount, result.OutputPath)
		}

		return nil
	},
}

func init() {
	parseCmd.Flags().BoolVarP(&parseFlat, "flat", "f", false, "emit flat path→content JSON (dir-to-JSON only)")
	rootCmd.AddCommand(parseCmd)
}
