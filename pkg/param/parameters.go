package param

/*
 * param package handles the cli parameters
 */

import (
	"flag"
	"sync"
)

// Config is the cmd parameter output
type Config struct {
	ConfigFile   *string
	PidFile      *string
	CheckGLB     *bool
	CheckConfig  *bool
	CheckBackend *bool
	Debug        *bool
	Version      *bool
	BackendName  *string
	PoolName     *string
	DNSName      *string
	ClusterOnly  *bool
}

var (
	config     *Config
	configLock sync.RWMutex
)

func Init() {
	c := Config{
		ConfigFile:   flag.String("config-file", "../../test/mercury.toml", "path to your mercury toml confg file"),
		PidFile:      flag.String("pid-file", "/run/mercury.pid", "path to your pid file"),
		CheckGLB:     flag.Bool("check-glb", false, "gives you a GLB report"),
		CheckConfig:  flag.Bool("check-config", false, "does a config check"),
		CheckBackend: flag.Bool("check-backend", false, "gives you a Backend report"),
		Debug:        flag.Bool("debug", false, "force logging to debug mode"),
		Version:      flag.Bool("version", false, "display version"),
		BackendName:  flag.String("backend-name", "", "only check selected backend name"),
		PoolName:     flag.String("pool-name", "", "only check selected pool name"),
		DNSName:      flag.String("dns-name", "", "only check selected dns name"),
		ClusterOnly:  flag.Bool("cluster-only", false, "only check cluster"),
	}
	flag.Parse()
	config = &c
}

// Get Allows you to get a parameter
func Get() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

// SetConfig sets the config file
func SetConfig(file string) {
	configLock.Lock()
	defer configLock.Unlock()
	config.ConfigFile = &file
}
