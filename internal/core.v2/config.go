package core

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/schubergphilis/mercury.v2/internal/web"
	yaml "gopkg.in/yaml.v2"
)

var (
	// Version of application
	Version string
	// VersionBuild number
	VersionBuild string
	// VersionSha git commit of build
	VersionSha string
	// StartTime of application
	StartTime time.Time
	// ReloadTime last time a reload was successfull
	ReloadTime time.Time
	// FailedReloadTime last time a reload failed
	FailedReloadTime time.Time
	// FailedReloadError last time a reload failed
	FailedReloadError string
)

// Config holds your main config
type Config struct {
	LoggingConfig LoggingConfig      `toml:"logging" json:"logging"`
	Settings      SettingsConfig     `toml:"settings" json:"settings"`
	DNSConfig     DNSConfig          `toml:"dns" json:"dns"`                   // see dns.go
	ClusterConfig ClusterConfig      `toml:"cluster" json:"cluster"`           // see cluster.go
	Loadbalancer  LoadbalancerConfig `toml:"loadbalancer" json:"loadbalancer"` // see loadbalancer.go
	Web           web.Config         `toml:"web" json:"web"`
}

// Settings contains a list of global application settings
type SettingsConfig struct {
	ManageNetworkInterfaces string `toml:"manage_network_interfaces"` // do network interface config (e.g. bind ip's)
	EnableProxy             string `toml:"enable_proxy"`              // start proxies, or let another app handle this
}

// LoggingConfig log config
type LoggingConfig struct {
	Level  string `toml:"level" json:"level"`
	Output string `toml:"output" json:"output"`
}

func (h *Handler) loadConfig() (*Config, error) {

	// read file
	h.Log.Infof("reading config", "type", "core", "file", h.configFile)
	data, err := ioutil.ReadFile(h.configFile)
	if err != nil {
		return nil, err
	}

	// parse file
	config := new(Config)
	f := strings.Split(h.configFile, ".")
	switch f[len(f)-1] {
	case "toml":
		_, err = toml.Decode(string(data), config)
		if err != nil {
			return nil, err
		}
	case "yaml":
		err = yaml.Unmarshal([]byte(data), config)
		if err != nil {
			return nil, err
		}
	}

	// verify details
	h.Log.Infof("verifying config", "type", "core", "file", h.configFile)
	if err = config.verify(); err != nil {
		return nil, err
	}

	/*
		log.Debug("Activating new config")
		configLock.Lock()
		config = temp

		log.Info("Config loaded succesfully")
		configLock.Unlock()

		return nil
	}*/
	return config, nil
}

func (c Config) verify() error {
	return nil
}
