package cmd

import (
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version and date are set via ldflags at build time by GoReleaser.
var (
	version = ""
	date    = ""
)

// Progress controls whether commands stream ok <path> lines for each validated item.
var Progress bool

// Verbose controls whether commands show enriched output (rule descriptions on violations).
var Verbose bool

const repoURL = "https://github.com/smithyai/aasdd-cli"

func buildVersionString() string {
	v := version
	d := date

	if v == "" {
		v = "(devel)"
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" {
				v = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.time" && len(s.Value) >= 10 {
					d = s.Value[:10]
				}
			}
		}
	}

	line1 := "aasdd version " + v
	if d != "" {
		line1 += " (" + d + ")"
	}

	url := repoURL
	if v != "(devel)" {
		url += "/releases/tag/" + v
	}

	return line1 + "\n" + url + "\n"
}

var rootCmd = &cobra.Command{
	Use:     "aasdd",
	Short:   "AASDD CLI — verify, scaffold, export, import, diff, and graph AASDD specs",
	Version: buildVersionString(),
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}")
	rootCmd.PersistentFlags().BoolVarP(&Progress, "progress", "p", false, "print ok <path> for each validated path")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "show enriched output (rule descriptions on violations)")
}
