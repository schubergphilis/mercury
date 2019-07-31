package core

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/models"
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
	ClusterConfig *ClusterConfig     `toml:"cluster" json:"cluster"`           // see cluster.go
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
	Level            string `toml:"level" json:"level"`
	Output           string `toml:"output" json:"output"`
	HealthcheckLevel string `toml:"healthcheck_level" json:"healthcheck_level"`
	ClusterLevel     string `toml:"cluster_level" json:"cluster_level"`
}

func (h *Handler) loadConfig() error {

	// read file
	h.log.Infof("reading config", "type", "core", "file", h.configFile)
	data, err := ioutil.ReadFile(h.configFile)
	if err != nil {
		return err
	}

	// parse file
	config := new(Config)
	f := strings.Split(h.configFile, ".")
	switch f[len(f)-1] {
	case "toml":
		_, err = toml.Decode(string(data), config)
		if err != nil {
			return err
		}
	case "yaml":
		err = yaml.Unmarshal([]byte(data), config)
		if err != nil {
			return err
		}
	}

	// verify details
	h.log.Infof("verifying config", "type", "core", "file", h.configFile)
	if err = config.verify(); err != nil {
		return err
	}

	/*
		log.Debug("Activating new config")
		configLock.Lock()
		config = temp

		log.Info("Config loaded succesfully")
		configLock.Unlock()

		return nil
	}*/
	h.config = config
	return nil
}

func (c *Config) verify() error {
	if err := c.defaultsLogging(); err != nil {
		return err
	}
	if err := c.defaultsHealthCheck(); err != nil {
		return err
	}
	return nil
}

func (c *Config) defaultsLogging() error {
	if c.LoggingConfig.ClusterLevel == "" {
		c.LoggingConfig.ClusterLevel = c.LoggingConfig.Level
	}
	if c.LoggingConfig.HealthcheckLevel == "" {
		c.LoggingConfig.HealthcheckLevel = c.LoggingConfig.Level
	}

	if _, err := logging.ToLevel(c.LoggingConfig.Level); err != nil {
		return fmt.Errorf("invalid main log level: %s", c.LoggingConfig.Level)
	}
	if _, err := logging.ToLevel(c.LoggingConfig.ClusterLevel); err != nil {
		return fmt.Errorf("invalid cluster log level: %s", c.LoggingConfig.ClusterLevel)
	}
	if _, err := logging.ToLevel(c.LoggingConfig.HealthcheckLevel); err != nil {
		return fmt.Errorf("invalid healthcheck log level: %s", c.LoggingConfig.HealthcheckLevel)
	}
	return nil
}

func (c *Config) defaultsHealthCheck() error {
	defaultCheckInterval := 10
	defaultCheckTimeout := 11

	defaultCheck := models.Healthcheck{
		Type:     "tcpconnect",
		Interval: defaultCheckInterval,
		Timeout:  defaultCheckTimeout,
	}
	// if no health check, set default to tcpconnect
	for poolID, pool := range c.Loadbalancer.Pools {
		for backendID, backend := range pool.Backends {
			if len(backend.Healthchecks) == 0 {
				checks := c.Loadbalancer.Pools[poolID].Backends[backendID]
				checks.Healthchecks = append(c.Loadbalancer.Pools[poolID].Backends[backendID].Healthchecks, defaultCheck)
				c.Loadbalancer.Pools[poolID].Backends[backendID] = checks
			}
		}
	}

	for poolID, pool := range c.Loadbalancer.Pools {
		// if there is a pool healthcheck, it applies to all backends/nodes
		poolchecks := []models.Healthcheck{}
		for _, healthcheck := range pool.Healthchecks {
			if healthcheck.Interval == 0 {
				healthcheck.Interval = defaultCheckInterval
			}

			if healthcheck.Timeout == 0 {
				healthcheck.Timeout = defaultCheckTimeout
			}
			poolchecks = append(poolchecks, healthcheck)
		}

		// source ip is the listener of the pool, unless otherwise specified
		sourceIP := pool.Listener.IP

		for backendID, backend := range pool.Backends {
			// collect all nodes we should use the healthcheck(s) for
			targetBackends := make(map[string]int)
			for _, node := range backend.Nodes {
				targetIP := ""
				if node.IP != "" {
					targetIP = node.IP // prefer ip over hostname
				} else {
					targetIP = node.Hostname
				}
				targetBackends[targetIP] = node.Port
			}

			// expand healthchecks based on nodes
			finalchecks := []models.Healthcheck{}
			// add pool checks (if any)
			finalchecks = append(finalchecks, poolchecks...)
			for _, healthcheck := range backend.Healthchecks {

				if healthcheck.SourceIP == "" {
					healthcheck.SourceIP = sourceIP
				}

				if healthcheck.Interval == 0 {
					healthcheck.Interval = defaultCheckInterval
				}

				if healthcheck.Timeout == 0 {
					healthcheck.Timeout = defaultCheckTimeout
				}

				if healthcheck.TargetIP == "" {
					for addr, port := range targetBackends {
						healthcheck.TargetIP = addr
						healthcheck.TargetPort = port
						finalchecks = append(finalchecks, healthcheck)
					}
				} else {
					finalchecks = append(finalchecks, healthcheck)
				}
			}

			// add default check if we have none
			if len(backend.Healthchecks) == 0 {
				finalchecks = append(finalchecks, defaultCheck)
			}

			// apply final checks to backend configuration
			checks := c.Loadbalancer.Pools[poolID].Backends[backendID]
			checks.Healthchecks = finalchecks
			c.Loadbalancer.Pools[poolID].Backends[backendID] = checks

		}
	}

	return nil
}
