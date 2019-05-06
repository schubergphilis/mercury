package core

import (
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rdoorn/cluster"
	"github.com/rdoorn/old/glbv2/pkg/tlsconfig"
	"github.com/schubergphilis/mercury.v2/internal/web"
	"github.com/schubergphilis/mercury/pkg/dns"
	yaml "gopkg.in/yaml.v2"
)

// Config holds your main config
type Config struct {
	Logging  LoggingConfig `toml:"logging" json:"logging"`
	Cluster  Cluster       `toml:"cluster" json:"cluster"`
	DNS      dns.Config    `toml:"dns" json:"dns"`
	Settings Settings      `toml:"settings" json:"settings"`
	//Loadbalancer Loadbalancer  `toml:"loadbalancer" json:"loadbalancer"`
	Web web.Config `toml:"web" json:"web"`
}

// Cluster contains the cluster settings
type Cluster struct {
	Binding   ClusterNode         `toml:"binding" json:"binding"`
	Nodes     []ClusterNode       `toml:"nodes" json:"nodes"`
	Settings  cluster.Settings    `toml:"settings" json:"settings"`
	TLSConfig tlsconfig.TLSConfig `toml:"tls" json:"tls"`
}

// ClusterNode contains the connection details of the cluster node
type ClusterNode struct {
	Name    string `toml:"name" json:"name"`
	Addr    string `toml:"addr" json:"addr"`
	AuthKey string `toml:"authkey" json:"authkey"`
}

// Settings contains a list of global application settings
type Settings struct {
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
