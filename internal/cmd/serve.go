package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/schubergphilis/mercury.v3/internal/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func serveCmd() *cobra.Command {
	// Serve
	command := &cobra.Command{
		Use:   "serve",
		Short: "start the mercury loadbalancer",
		Run:   serve(),
	}
	// Serve Flags
	command.PersistentFlags().String("pid-file", "/var/run/mercury.pid", "location of the pid file")
	viper.BindPFlag("pid_file", command.PersistentFlags().Lookup("pid-file"))
	return command
}

func serve() func(command *cobra.Command, args []string) {
	return func(command *cobra.Command, args []string) {
		/*fmt.Printf("config file: %+v\n", viper.GetString("config_file"))
		fmt.Printf("pid file: %+v\n", viper.GetString("pid_file"))
		fmt.Printf("log output: %+v\n", viper.GetStringSlice("log_output"))
		fmt.Printf("log level: %+v\n", viper.GetString("log_level"))*/

		var config core.Config
		if err := viper.Unmarshal(&config); err != nil {
			panic(fmt.Errorf("failed to unmarshal config: %s", err))
		}

		if err := config.Verify(); err != nil {
			panic(fmt.Errorf("failed to verify config: %s", err))
		}

		// Start the application
		core := core.New()
		core.Enable(&config)

		// wait for sigint or sigterm for cleanup - note that sigterm cannot be caught
		sigterm := make(chan os.Signal, 10)
		signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

		sighup := make(chan os.Signal, 1)
		signal.Notify(sighup, syscall.SIGHUP)

		for {
			select {
			case <-sigterm:
				core.Warnf("Program killed by signal!")
				core.Stop()
				return

			case <-sighup:
				core.Warnf("Program received HUP signal!")
				viper.ReadInConfig()
				if err := viper.Unmarshal(&config); err != nil {
					panic(fmt.Errorf("failed to unmarshal config: %s", err))
				}

				if err := config.Verify(); err != nil {
					panic(fmt.Errorf("failed to verify config: %s", err))
				}
				core.Enable(&config)
			}
		}

	}
}
