package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/schubergphilis/mercury/internal/core.v2"
	"github.com/schubergphilis/mercury/internal/logging"
	// Only enabled for profiling
)

// version is set during makefile
var version string
var versionBuild string
var versionSha string

// Initialize package
func init() {
	core.Version = version
	core.VersionBuild = versionBuild
	core.VersionSha = versionSha
	core.StartTime = time.Now()
}

// main start
func main() {
	logger, err := logging.NewZap("stdout", "mercury.log")
	if err != nil {
		panic(err)
	}

	// parse parameters
	configFile := flag.String("config-file", "../../test/mercury.toml", "path to your mercury toml confg file")
	pidFile := flag.String("pid-file", "/run/mercury.pid", "path to your pid file")
	logLevel := flag.String("loglevel", "info", "log level [debug|info|warn|fatal]")
	showVersion := flag.Bool("version", false, "display version")
	flag.Parse()

	// show version only, if requested
	if *showVersion {
		logger.Infof("current version", "version", "1.2")
		os.Exit(0)
	}

	// setup handler
	handler := core.New(
		core.WithLogger(logger),          // set logger
		core.WithLogLevel(*logLevel),     // set log level (param/default)
		core.WithPidFile(*pidFile),       // set pid file (param/default)
		core.WithConfigFile(*configFile), // load config file (param/default)
	)

	// start the core handler
	go handler.Start()

	// wait for sigint or sigterm for cleanup - note that sigterm cannot be caught
	sigterm := make(chan os.Signal, 10)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	for {
		select {
		case <-sigterm:
			handler.LogProvider.Warnf("Program killed by signal!")
			handler.Quit <- struct{}{}
			return

		case <-sighup:
			handler.LogProvider.Warnf("Program received HUP signal!")
			handler.Reload <- struct{}{}
		}
	}

}
