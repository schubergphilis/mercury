package main

import (
	core "github.com/schubergphilis/mercury.v2/internal/core.v2"
	"github.com/schubergphilis/mercury.v2/internal/logging"
)

// Only enabled for profiling

const (
	defaultLogLevel  = "warn"
	defaultLogTarget = "stdout"
)

// main start
func main() {
	logger, err := logging.NewZap()
	if err != nil {
		panic(err)
	}

	handler := core.New(
		core.WithLogger(logger),
	)

	handler.Log.Debugf("reading profile (credentials + api endpoint)")

	/*
		// Default logging before reading the config
		logging.Configure(defaultLogTarget, defaultLogLevel)
		log := logging.For("main")

		switch {
		case *param.Get().Debug == true:
			config.LogLevel = "debug"
		}
		logging.Configure(config.LogTarget, config.LogLevel)

		if *param.Get().Version == true {
			log.WithField("version", config.Version).WithField("build", config.VersionBuild).WithField("gitsha", config.VersionSha).Info("Mercury version")
			return
		}
		log.WithField("file", *param.Get().ConfigFile).Debug("Reading config file")

		err := config.LoadConfig(*param.Get().ConfigFile)
		if err != nil {
			log.WithField("file", *param.Get().ConfigFile).WithField("error", err).Fatal("Error reading config file")
		}

		// If we are checking the config, we can exit safely here
		if *param.Get().CheckConfig == true {
			return
		}

		switch {
		case *param.Get().CheckGLB == true:
			os.Exit(check.GLB())
		case *param.Get().CheckBackend == true:
			os.Exit(check.Backend())
		}

		logging.Configure(config.Get().Logging.Output, config.Get().Logging.Level)

		lock, err := lockfile.New(*param.Get().PidFile)
		if err != nil {
			proc, err := lock.GetOwner()
			if err == nil {
				log = log.WithField("pid", proc.Pid)
			}
			log.WithField("file", *param.Get().PidFile).WithField("error", err).Fatalf("Create pid failed")
		}
		err = lock.TryLock()
		if err != nil {
			proc, err := lock.GetOwner()
			if err == nil {
				log = log.WithField("pid", proc.Pid)
			}
			log.WithField("file", *param.Get().PidFile).WithField("error", err).Fatalf("Create pid failed")
		}

		defer lock.Unlock()

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
				return

			case <-sighup:
				log.Warn("Program received HUP signal!")
				config.ReloadConfig()
				logging.Configure(config.Get().Logging.Output, config.Get().Logging.Level)
				reload <- true
			}
		}
	*/
}
