package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smithyai/aasdd-cli/internal/authoring/scaffold"
	"github.com/smithyai/aasdd-cli/internal/types"
	"github.com/spf13/cobra"
)

var scaffoldExample bool
var scaffoldAASDDVersion string

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <path>",
	Short: "Create a new AASDD spec directory with stub files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := types.SpecTarget{Path: args[0]}

		version := scaffoldAASDDVersion

		if Verbose {
			fmt.Fprintf(os.Stderr, "scaffolding %s\n", target.Path)
		}

		result, err := scaffold.Scaffold(target, version, scaffoldExample)
		if err != nil {
			var notEmpty *scaffold.TargetNotEmpty
			var isFile *scaffold.TargetIsFile
			var writeErr *scaffold.WriteError
			var unknownVer *scaffold.UnknownAASDDVersion
			switch {
			case errors.As(err, &isFile):
				fmt.Fprintf(os.Stderr, "error: %s\n", isFile)
			case errors.As(err, &notEmpty):
				fmt.Fprintf(os.Stderr, "error: %s\n", notEmpty)
			case errors.As(err, &writeErr):
				fmt.Fprintf(os.Stderr, "error: %s\n", writeErr)
			case errors.As(err, &unknownVer):
				fmt.Fprintf(os.Stderr, "error: %s\n", unknownVer)
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
	rootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.Flags().BoolVarP(&scaffoldExample, "example", "e", false, "populate with a worked example instead of empty stubs")
	scaffoldCmd.Flags().StringVar(&scaffoldAASDDVersion, "aasdd-version", "", "AASDD version to scaffold for (default: latest; e.g. --aasdd-version v1)")
}
