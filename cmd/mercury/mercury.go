package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/nightlyone/lockfile"
	"github.com/schubergphilis/mercury/internal/check"
	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/internal/core"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/param"

	// Only enabled for profiling
	"net/http"
	"net/http/pprof"

	"github.com/stackimpact/stackimpact-go"
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
	param.Init()

	log := logging.For("main")

	stackimpactAPI, ok := os.LookupEnv("STACKIMPACT_API")
	if ok {
		log.Infof("Starting stackimpact profiler")
		agent := stackimpact.Start(stackimpact.Options{
			AgentKey: stackimpactAPI,
			AppName:  "mercury",
		})
		span := agent.Profile()
		defer span.Stop()
	}

	addr, ok := os.LookupEnv("PROFILER_ADDR")
	if ok {
		log.Infof("Starting profiler at http://%s", addr)
		go EnableProfiler(addr)
	}

	// Default logging before reading the config
	config.LogTarget = "stdout"
	switch {
	case *param.Get().Debug == true:
		config.LogLevel = "debug"
	case *param.Get().CheckGLB == true || *param.Get().CheckBackend == true || *param.Get().CheckConfig == true:
		config.LogLevel = "warn"
	default:
		config.LogLevel = "info"
	}
	logging.Configure(config.LogTarget, config.LogLevel)

	if *param.Get().Version == true {
		log.WithField("version", config.Version).WithField("build", config.VersionBuild).WithField("gitsha", config.VersionSha).WithField("golang", runtime.Version()).Info("Mercury version")
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

	case *param.Get().CheckEndpoints == true:
		os.Exit(check.Endpoints())

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
}

// EnableProfiler starts the profiler on localhost port 6060
func EnableProfiler(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", pprof.Index)
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	http.ListenAndServe(addr, mux)
}
