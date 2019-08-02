package cmd

import (
	"github.com/spf13/cobra"
)

func checkCmd() *cobra.Command {
	// Serve
	command := &cobra.Command{
		Use:   "serve",
		Short: "start the mercury loadbalancer",
		// Run:   serve(),
	}
	// Serve Flags
	/*serveCmd.PersistentFlags().String("pid-file", "/var/run/mercury.pid", "location of the pid file")
	viper.BindPFlag("pid-file", serveCmd.PersistentFlags().Lookup("pid-file"))*/
	return command
}
