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
}

var (
	config     *Config
	configLock sync.RWMutex
)

func init() {
	c := Config{
		ConfigFile:   flag.String("config-file", "../../test/mercury.toml", "path to your mercury toml confg file"),
		PidFile:      flag.String("pid-file", "/run/mercury.pid", "path to your pid file"),
		CheckGLB:     flag.Bool("check-glb", false, "gives you a GLB report"),
		CheckConfig:  flag.Bool("check-config", false, "does a config check"),
		CheckBackend: flag.Bool("check-backend", false, "gives you a Backend report"),
		Debug:        flag.Bool("debug", false, "force logging to debug mode"),
		Version:      flag.Bool("version", false, "display version"),
	}
	flag.Parse()
	config = &c
	/*
		log := logging.For("cmd/parameters")
		log.WithField("file", *c.ConfigFile).Debug("Parameter for config file")
		log.WithField("file", *c.PidFile).Debug("Parameter for pid file")
		log.WithField("bool", *c.CheckBackend).Debug("Parameter for check backend")
		log.WithField("bool", *c.CheckGLB).Debug("Parameter for check glb")
		log.WithField("bool", *c.CheckConfig).Debug("Parameter for check config")
		log.WithField("bool", *c.Debug).Debug("Parameter for debug")
		log.WithField("bool", *c.Version).Debug("Parameter for version")
	*/
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
