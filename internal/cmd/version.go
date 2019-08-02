package cmd

import (
	"fmt"

	"github.com/schubergphilis/mercury.v3/internal/core"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "shows the version of Mercury",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version: %s (build: %s)\nSha: %s\n", core.Name, core.Version, core.VersionBuild, core.VersionSha)
		},
	}
	return cmd
}
