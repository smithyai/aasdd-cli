package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smithyai/aasdd-cli/internal/analysis/graph"
	"github.com/smithyai/aasdd-cli/internal/types"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph <spec>",
	Short: "Generate a dependency graph of a spec directory",
	Long: `Generate a dependency graph showing how abilities, concepts, decisions, and
scenarios relate to each other.

The graph is written to stdout by default. Use --output to write to a file.
Defaults to Mermaid format; pass --format dot for Graphviz DOT.

Examples:
  aasdd graph ./spec
  aasdd graph ./spec --format dot
  aasdd graph ./spec --output graph.mmd`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}
		formatStr, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")

		var format types.GraphFormat
		switch formatStr {
		case "dot":
			format = types.GraphFormatDOT
		default:
			format = types.GraphFormatMermaid
		}

		result, err := graph.Graph(target, format)
		if err != nil {
			var notFound *graph.TargetNotFound
			var isFile *graph.TargetIsFile
			var readErr *graph.ReadError
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

		if outputPath != "" {
			if writeErr := os.WriteFile(outputPath, []byte(result.Content), 0o644); writeErr != nil {
				fmt.Fprintf(os.Stderr, "error writing output: %v\n", writeErr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "wrote %d nodes, %d edges → %s\n",
				result.NodeCount, result.EdgeCount, outputPath)
			return nil
		}

		fmt.Print(result.Content)
		return nil
	},
}

func init() {
	graphCmd.Flags().StringP("format", "f", "mermaid", "output format: mermaid or dot")
	graphCmd.Flags().StringP("output", "o", "", "write graph to file instead of stdout")
	rootCmd.AddCommand(graphCmd)
}
