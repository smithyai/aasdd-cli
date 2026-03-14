package cmd

import (
	"fmt"

	"github.com/smithyai/aasdd-cli/internal/pipeline/verify/load_rule_set"
	"github.com/spf13/cobra"
)

var listVersionsCmd = &cobra.Command{
	Use:   "list-versions",
	Short: "List all AASDD methodology versions known to the tool",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		versions := load_rule_set.KnownVersions()
		latest := versions[len(versions)-1].Version
		for _, v := range versions {
			marker := "  "
			if v.Version == latest {
				marker = "* "
			}
			fmt.Printf("%s%-6s %s\n", marker, v.Version, v.Summary)
		}
	},
}

func init() {
	rootCmd.AddCommand(listVersionsCmd)
}
