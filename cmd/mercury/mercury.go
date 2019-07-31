package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/nightlyone/lockfile"
	"github.com/schubergphilis/mercury/internal/check"
	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/internal/core"
	"github.com/schubergphilis/mercury/pkg/dns"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/param"

	// Only enabled for profiling
	"net/http"
	"net/http/pprof"
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

	addr, ok := os.LookupEnv("PROFILER_ADDR")
	if ok {
		log.Infof("Starting profiler at http://%s", addr)
		go EnableProfiler(addr)
	}
	if *param.Get().KeyDir == "" {
		basedir := path.Dir(*param.Get().ConfigFile)
		param.Get().KeyDir = &basedir
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
		log.WithField("version", config.Version).WithField("build", config.VersionBuild).WithField("gitsha", config.VersionSha).Info("Mercury version")
		return
	}
	log.WithField("file", *param.Get().ConfigFile).Debug("Reading config file")

	if *param.Get().CreateKeySigningKey {
		createKeySigningKey()
	}

	if *param.Get().ReadSigningKey {
		readKeySigningKey()
	}

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
}

// EnableProfiler starts the profiler on localhost port 6060
func EnableProfiler(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", pprof.Index)
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	http.ListenAndServe(addr, mux)
}

func createKeySigningKey() {
	log := logging.For("main/keysigning")

	if *param.Get().KeyDir == "" {
		log.Errorf("--key-file must be provided when generating keys")
		os.Exit(255)
	}
	if *param.Get().SigningZone == "" {
		log.Errorf("--signing-zone must be provided when generating keys")
		os.Exit(255)
	}
	keys := dns.NewKeyStore()
	keys.Load(*param.Get().KeyDir)

	var algorithm uint8
	switch *param.Get().SigningAlgorithm {
	case "RSASHA256":
		algorithm = 8
	case "RSASHA512":
		algorithm = 10
	case "ECDSAP256SHA256":
		algorithm = 13
	case "ECDSAP348SHA348":
		algorithm = 14
	default:
		log.Errorf("unknown algorithm: %s", *param.Get().SigningAlgorithm)
		os.Exit(255)
	}
	key, err := dns.NewPrivateKey(dns.KeySigningKey, algorithm)
	if err != nil {
		log.Errorf("error creating key: %s", err)
		os.Exit(255)
	}
	keys.SetRollover(dns.KeySigningKey, *param.Get().SigningZone, time.Duration(*param.Get().SigningKeyTTL)*time.Second, key)
	keys.Save(*param.Get().KeyDir)

	displaySigningKey(keys, *param.Get().SigningZone)
	/*if err := dns.GenerateKey(dns.KeySigningKey, *param.Get().SigningAlgorithm, *param.Get().SigningZone, *param.Get().KeyDir); err != nil {
	  	log.Errorf("error creating key: %s", err)
	  	os.Exit(255)
	  }
	*/
	os.Exit(0)
}

func readKeySigningKey() {
	log := logging.For("main/keysigning")

	if *param.Get().SigningZone == "" {
		log.Errorf("--signing-zone must be provided when generating keys")
		os.Exit(255)
	}

	keys := dns.NewKeyStore()
	keys.Load(*param.Get().KeyDir)
	displaySigningKey(keys, *param.Get().SigningZone)
	os.Exit(0)
}

func displaySigningKey(keys *dns.KeyStore, zone string) {
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}

	fmt.Printf("The following keys are now available:\n")
	for z, keys := range keys.KeySigningKeys.Keys {
		if z == zone {
			for _, key := range keys {
				DNSKEY := key.DNSKEY(dns.KeySigningKey, zone)
				log.Printf("KSK valid till %s, record: %s\n", key.Deactivate, DNSKEY.ToDS(2))
			}
		}
	}
}
