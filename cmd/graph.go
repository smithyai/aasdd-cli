package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/smithyai/aasdd-cli/internal/analysis/graph"
	"github.com/smithyai/aasdd-cli/internal/types"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph <spec>",
	Short: "Generate a dependency graph of a spec directory",
	Long: `Generate a dependency graph showing how abilities relate to each other and to
the concepts they consume and produce. Abilities and concepts are always included.
Use --include to opt in supplementary node kinds.

The graph is written to stdout by default. Use --output to write to a file.
Defaults to Mermaid format; pass --format dot for Graphviz DOT.

Examples:
  aasdd graph ./spec
  aasdd graph ./spec --format dot
  aasdd graph ./spec --output graph.mmd
  aasdd graph ./spec --include scenarios
  aasdd graph ./spec --include decisions
  aasdd graph ./spec --include scenarios,decisions
  aasdd graph ./spec --include state-machine
  aasdd graph ./spec --root verify
  aasdd graph ./spec --root verify --depth 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}
		formatStr, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")
		includeStr, _ := cmd.Flags().GetString("include")
		depth, _ := cmd.Flags().GetInt("depth")
		rootStr, _ := cmd.Flags().GetString("root")

		var format types.GraphFormat
		switch formatStr {
		case "dot":
			format = types.GraphFormatDOT
		default:
			format = types.GraphFormatMermaid
		}

		var include []types.NodeKind
		if includeStr != "" {
			for _, part := range strings.Split(includeStr, ",") {
				part = strings.TrimSpace(part)
				switch part {
				case "scenarios":
					include = append(include, types.NodeKindScenario)
				case "decisions":
					include = append(include, types.NodeKindDecision)
				case "state-machine":
					include = append(include, types.NodeKindStateMachine)
				default:
					fmt.Fprintf(os.Stderr, "error: invalid include value %q: must be one or more of scenarios, decisions, state-machine\n", part)
					os.Exit(1)
				}
			}
		}

		result, err := graph.Graph(target, format, types.GraphOptions{Include: include, Depth: depth, Root: rootStr})
		if err != nil {
			var notFound *graph.TargetNotFound
			var isFile *graph.TargetIsFile
			var readErr *graph.ReadError
			var invalidInclude *graph.InvalidInclude
			var rootNotFound *graph.RootNotFound
			var ambiguousRoot *graph.AmbiguousRoot
			switch {
			case errors.As(err, &notFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", notFound)
			case errors.As(err, &isFile):
				fmt.Fprintf(os.Stderr, "error: %s\n", isFile)
			case errors.As(err, &readErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", readErr)
			case errors.As(err, &invalidInclude):
				fmt.Fprintf(os.Stderr, "error: %s\n", invalidInclude)
			case errors.As(err, &rootNotFound):
				fmt.Fprintf(os.Stderr, "error: %s\n", rootNotFound)
			case errors.As(err, &ambiguousRoot):
				fmt.Fprintf(os.Stderr, "error: %s\n", ambiguousRoot)
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
	graphCmd.Flags().String("include", "", "supplementary node kinds to add: scenarios, decisions, state-machine (comma-separated)")
	graphCmd.Flags().Int("depth", 0, "max ability depth to walk; ≤0 means unlimited")
	graphCmd.Flags().String("root", "", "re-root the graph at this ability (case-insensitive last-segment match)")
	rootCmd.AddCommand(graphCmd)
}
