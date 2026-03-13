package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/smithyai/aasdd-cli/internal/pipeline/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
)

var scaffoldSpecVersion string

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <path>",
	Short: "Create a new AASDD spec directory with stub files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}

		var sv *types.SpecVersion
		if scaffoldSpecVersion != "" {
			sv = &types.SpecVersion{Value: scaffoldSpecVersion}
		}

		result, err := scaffold.Scaffold(target, sv)
		if err != nil {
			var notEmpty *scaffold.TargetNotEmpty
			var unknownVer *scaffold.UnknownSpecVersion
			var writeErr *scaffold.WriteError
			switch {
			case errors.As(err, &notEmpty):
				fmt.Fprintf(os.Stderr, "error: %s\n", notEmpty)
			case errors.As(err, &unknownVer):
				fmt.Fprintf(os.Stderr, "error: %s\n", unknownVer)
			case errors.As(err, &writeErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", writeErr)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("scaffolded %d file(s) in %s\n", len(result.FilesCreated), result.Target.Path)
		for _, f := range result.FilesCreated {
			fmt.Printf("  %s\n", f)
		}
		return nil
	},
}

func init() {
	scaffoldCmd.Flags().StringVar(&scaffoldSpecVersion, "spec-version", "", "AASDD version to scaffold against (default: latest)")
	rootCmd.AddCommand(scaffoldCmd)
}
