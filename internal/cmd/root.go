package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// nolint: gochecknoglobals
	configFile string
)

// order of priority:
// parameter
// environment variable
// config??
// default

// Execute parses the command parameters and starts the requested process
func Execute() {

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Root
	rootCmd := &cobra.Command{
		Use:   filepath.Base(os.Args[0]),
		Short: "Mercury loadbalancer",
		Long:  `a dns based global-loadbalancer see https://github.com/schubergphilis/mercury/ for details`,
	}
	// Root Flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", "/etc/mercury/mercury.toml", "location of the config file")
	viper.BindPFlag("config_file", rootCmd.PersistentFlags().Lookup("config-file"))

	rootCmd.PersistentFlags().String("log-level", "info", "level of logging (debug, info, warn, error)")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	rootCmd.PersistentFlags().StringSlice("log-output", []string{"stdout"}, "where to output the logs seperated by spaces")
	viper.BindPFlag("log_output", rootCmd.PersistentFlags().Lookup("log-output"))

	rootCmd.AddCommand(
		serveCmd(),
		checkCmd(),
		versionCmd(),
	)

	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		viper.AddConfigPath(home)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if configFile != "" {
			log.Println("config specified but unable to read it, using defaults")
		}
	}
}
