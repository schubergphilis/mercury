package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/schubergphilis/mercury/src/check"
	"github.com/schubergphilis/mercury/src/config"
	"github.com/schubergphilis/mercury/src/core"
	"github.com/schubergphilis/mercury/src/logging"
	"github.com/schubergphilis/mercury/src/param"

	"github.com/Wang/pid"
	// Only enabled for profiling
	// _ "net/http/pprof"
)

// version is set during makefile
var version string
var versionBuild string
var versionSha string

// Initialize package
func init() {
	config.Version = version
	config.VersionBuild = versionBuild
	config.VersionSha = versionSha
	config.StartTime = time.Now()
}

// main start
func main() {
	logging.Configure("stdout", "info")

	log := logging.For("main")
	//log.WithFields(logging.Fields{"request_id": request_id, "user_ip": user_ip})

	// pprof profiling disabled by default
	// go http.ListenAndServe("localhost:6060", nil)
	// runtime.SetBlockProfileRate(1)
	// runtime.SetMutexProfileFraction(1)

	// Default logging before reading the config
	config.LogTarget = "stdout"
	if *param.Get().Debug == true {
		config.LogLevel = "debug"
	} else if *param.Get().CheckGLB == true || *param.Get().CheckBackend == true || *param.Get().CheckConfig == true {
		config.LogLevel = "warn"
	} else {
		config.LogLevel = "info"
	}
	logging.Configure(config.LogTarget, config.LogLevel)

	if *param.Get().Version == true {
		log.WithField("version", config.Version).WithField("build", config.VersionBuild).WithField("gitsha", config.VersionSha).Info("Mercury version")
		return
	}
	log.WithField("file", *param.Get().ConfigFile).Debug("Reading config file")

	err := config.LoadConfig(*param.Get().ConfigFile)
	if err != nil {
		//log.Fatalf("Error loading config file:%s", err)
		log.WithField("file", *param.Get().ConfigFile).WithField("error", err).Fatal("Error reading config file")
	}

	// IF we are checking the config, we can exit safely here
	if *param.Get().CheckConfig == true {
		return
	}

	if *param.Get().CheckGLB == true {
		os.Exit(check.GLB())
	} else if *param.Get().CheckBackend == true {
		os.Exit(check.Backend())
	}

	logging.Configure(config.Get().Logging.Output, config.Get().Logging.Level)
	//log.Debug("Starting")
	//log.WithField("key", "value").Error("key error")

	pidValue, err := pid.Create(*param.Get().PidFile)
	if err != nil {
		log.WithField("file", *param.Get().PidFile).WithField("error", err).Fatalf("Create pid failed")
	}
	log.WithField("pid", pidValue).Info("New pid")

	reload := make(chan bool, 1)
	go core.Initialize(reload)

	// wait for sigint or sigterm for cleanup - note that sigterm cannot be caught
	sigterm := make(chan os.Signal, 10)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	for {
		select {
		case <-sigterm:
			log.Warn("Program killed by signal!")
			core.Cleanup()
			//p.Stop()
			return
		case <-sighup:
			log.Warn("Program received HUP signal!")
			config.ReloadConfig()
			logging.Configure(config.Get().Logging.Output, config.Get().Logging.Level)
			reload <- true
		}
	}
}
