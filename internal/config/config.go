package config

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/schubergphilis/mercury/internal/web"
	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/cluster"
	"github.com/schubergphilis/mercury/pkg/dns"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/param"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"

	"github.com/BurntSushi/toml"
)

var (
	config     *Config
	configLock sync.RWMutex
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
	// LogLevel holds the level
	LogLevel string
	// LogTarget defines where to write the messages
	LogTarget string
)

const (
	// YES string
	YES            = "yes"
	logConfigLocks = false
)

// Config holds your main config
type Config struct {
	Logging      LoggingConfig `toml:"logging" json:"logging"`
	Cluster      Cluster       `toml:"cluster" json:"cluster"`
	DNS          dns.Config    `toml:"dns" json:"dns"`
	Settings     Settings      `toml:"settings" json:"settings"`
	Loadbalancer Loadbalancer  `toml:"loadbalancer" json:"loadbalancer"`
	Web          web.Config    `toml:"web" json:"web"`
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

// ReloadConfig reloads the config (typically after a hup)
func ReloadConfig() {
	log := logging.For("config/reload")
	log.Info("Reload config file")
	err := LoadConfig(*param.Get().ConfigFile)
	if err != nil {
		FailedReloadTime = time.Now()
		FailedReloadError = err.Error()
		log.Warnf("Error reloading config:%s", err)
	} else {
		ReloadTime = time.Now()
		log.Infof("Reloaded")
	}
}

// LoadConfig a config file
func LoadConfig(file string) error {
	log := logging.For("config/load")
	log.Info("Loading config")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	temp := new(Config)
	f := strings.Split(file, ".")
	switch f[len(f)-1] {
	case "toml":
		log.Debug("Decode toml config")
		_, err = toml.Decode(string(data), temp)
		if err != nil {
			return err
		}
	case "yaml":
		log.Debug("Decode yaml config")
		err = yaml.Unmarshal([]byte(data), temp)
		if err != nil {
			return err
		}
	}

	log.Debug("Check config")

	// Check SSL Certificates
	err = temp.ValidateCertificates()
	if err != nil {
		return err
	}

	// Loadbalance defaults
	if temp.Loadbalancer.Settings.DefaultLoadBalanceMethod == "" {
		temp.Loadbalancer.Settings.DefaultLoadBalanceMethod = "roundrobin"
	}
	// Ensure a default in all backends
	for poolName, pool := range temp.Loadbalancer.Pools {
		if pool.ErrorPage.File != "" {
			if _, err := os.Stat(pool.ErrorPage.File); err != nil {
				return fmt.Errorf("Cannot access error page for pool:%s file:%s error:%s", poolName, pool.ErrorPage.File, err)
			}
		}

		p := temp.Loadbalancer.Pools[poolName]
		if p.ErrorPage.TriggerThreshold == 0 {
			p.ErrorPage.TriggerThreshold = 500
		}

		if p.Listener.Mode == "" {
			p.Listener.Mode = "tcp"
		}

		if p.Listener.OCSPStapling == "" {
			p.Listener.OCSPStapling = YES
		}

		if p.Listener.MaxConnections == 0 {
			p.Listener.MaxConnections = 2048
		}

		if p.Listener.HTTPProto == 0 {
			p.Listener.HTTPProto = 2
		}

		// Default writetimeout for listener is 0 = unlimited time
		// Default readtimeout for listener is 10 seconds
		if p.Listener.ReadTimeout == 0 {
			p.Listener.ReadTimeout = 10
		}

		temp.Loadbalancer.Pools[poolName] = p

		for hid, healthcheck := range temp.Loadbalancer.Pools[poolName].HealthChecks {
			p := temp.Loadbalancer.Pools[poolName]
			if healthcheck.Interval < 1 {
				p.HealthChecks[hid].Interval = 11
			}

			if healthcheck.Timeout < 1 {
				p.HealthChecks[hid].Timeout = 10
			}

			if healthcheck.PINGpackets == 0 {
				p.HealthChecks[hid].PINGpackets = 4
			}

			if healthcheck.PINGtimeout == 0 {
				p.HealthChecks[hid].PINGtimeout = 1
			}

			if healthcheck.Type == "" {
				p.HealthChecks[hid].Type = "tcpconnect"
			}

			temp.Loadbalancer.Pools[poolName] = p
		}

		//log.Debugf("Pool: %s", poolName)
		for backendName, backend := range temp.Loadbalancer.Pools[poolName].Backends {
			h := backend

			if backend.UUID == "" {
				// generate hash uniq to cluster - pool - backend
				hash := sha256.New()
				hash.Write([]byte(fmt.Sprintf("%s-%s-%s", temp.Cluster.Binding.Addr, poolName, backendName)))
				h.UUID = fmt.Sprintf("%x", hash.Sum(nil))
			}

			if backend.ConnectMode == "" {
				h.ConnectMode = temp.Loadbalancer.Pools[poolName].Listener.Mode
			}

			if backend.DNSEntry.IP == "" && temp.Loadbalancer.Pools[poolName].Listener.IP == "" {
				return fmt.Errorf("No IP defined in either the pool's listener IP or the DNSentry IP for backend:%s", backendName)
			}
			// If not DNS Entry IP is set, set the ip to the listener
			if backend.DNSEntry.IP == "" {
				h.DNSEntry.IP = temp.Loadbalancer.Pools[poolName].Listener.IP
			}

			if backend.ErrorPage.File != "" {
				if _, err := os.Stat(backend.ErrorPage.File); err != nil {
					return fmt.Errorf("Cannot access error page for pool:%s backend:%s file:%s error:%s", poolName, backendName, backend.ErrorPage.File, err)
				}
			}

			for hid, healthcheck := range temp.Loadbalancer.Pools[poolName].Backends[backendName].HealthChecks {

				if healthcheck.Interval < 1 {
					h.HealthChecks[hid].Interval = 11
				}

				if healthcheck.Timeout < 1 {
					h.HealthChecks[hid].Timeout = 10
				}

				if healthcheck.PINGpackets == 0 {
					h.HealthChecks[hid].PINGpackets = 4
				}

				if healthcheck.PINGtimeout == 0 {
					h.HealthChecks[hid].PINGtimeout = 1
				}

				if backend.BalanceMode.ActivePassive == YES {
					h.HealthChecks[hid].ActivePassiveID = backend.UUID
				} else {
					h.BalanceMode.ActivePassive = "no"
					h.HealthChecks[hid].ActivePassiveID = ""
				}

				if healthcheck.Type == "" {
					h.HealthChecks[hid].Type = "tcpconnect"
				}
			}

			// Always have atleast 1 check: tcpconnect
			if len(temp.Loadbalancer.Pools[poolName].Backends[backendName].HealthChecks) == 0 {
				tcpconnect := healthcheck.HealthCheck{
					Type:     "tcpconnect",
					Interval: 11,
					Timeout:  10,
				}
				h.HealthChecks = append(h.HealthChecks, tcpconnect)
			}

			if backend.HealthCheckMode == "" {
				h.HealthCheckMode = "all"
			}

			if backend.BalanceMode.ClusterNodes == 0 {
				h.BalanceMode.ClusterNodes = len(temp.Cluster.Nodes)
			}

			if backend.BalanceMode.LocalTopology != "" {
				if val, ok := temp.Loadbalancer.Networks[backend.BalanceMode.LocalTopology]; ok {
					for _, network := range val.CIDRs {
						h.BalanceMode.LocalNetwork = append(h.BalanceMode.LocalNetwork, network)
					}
				} else {
					return fmt.Errorf("Could not find topology name:%s in the defined loadbalancer networks in the config for backend:%s", backend.BalanceMode.LocalTopology, backendName)
				}
			}
			// Default node settings
			for nodeID, node := range temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
				if node.UUID == "" {
					// generate hash uniq to pool - backend - node + port (cluster pool removed for stickyness across clusters)
					hash := sha256.New()
					hash.Write([]byte(fmt.Sprintf("%s-%s-%s-%s", poolName, backendName, node.SafeName(), node.Hostname)))

					//u, _ := uuid.NewV4() // replaced by sha256
					n := node
					//n.UUID = u.String()
					n.UUID = fmt.Sprintf("%x", hash.Sum(nil))
					n.ClusterName = temp.Cluster.Binding.Name
					if n.MaxConnections == 0 {
						n.MaxConnections = pool.Listener.MaxConnections
					}
					h.Nodes[nodeID] = n
					log.Infof("Node:%s UUID:%s", h.Nodes[nodeID].Name(), h.Nodes[nodeID].UUID)
				}
			}

			// Save Backend changes
			temp.Loadbalancer.Pools[poolName].Backends[backendName] = h

			for nodeID, node := range temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
				// load localnetworks based on topology
				if node.LocalTopology != "" {
					if val, ok := temp.Loadbalancer.Networks[node.LocalTopology]; ok {
						for _, network := range val.CIDRs {
							temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nodeID].LocalNetwork = append(temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nodeID].LocalNetwork, network)
						}
					} else {
						return fmt.Errorf("Could not find topology name:%s in the defined loadbalancer networks in the config for backend:%s node:%s", backend.BalanceMode.LocalTopology, backendName, node.Name())
					}
				}
			}

			// Copy node Status if exists
			if Get() != nil {
				log.Debug("Config is not empty, copying node status if it still exists")
				if _, ok := Get().Loadbalancer.Pools[poolName]; ok {
					log.WithField("poolname", poolName).Debug("Existing pool")
					if _, ok := Get().Loadbalancer.Pools[poolName].Backends[backendName]; ok {
						log.WithField("poolname", poolName).WithField("backendname", backendName).Debug("Existing backend")
						for _, oldnode := range Get().Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
							for nid, newnode := range temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
								if oldnode.UUID == newnode.UUID {
									temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nid].Online = oldnode.Online
									temp.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nid].Errors = oldnode.Errors
									log.Debugf("Old node:%s uuid:%s copied to New node:%s uuid:%s", oldnode.Name(), oldnode.UUID, newnode.Name(), newnode.UUID)
								}
							}
						}
					}
				}
			}

		}
	}

	if temp.Web.Binding == "" {
		temp.Web.Binding = "localhost"
	}

	if temp.Web.Port == 0 {
		temp.Web.Port = 9001
	}

	if temp.Settings.ManageNetworkInterfaces == "" {
		temp.Settings.ManageNetworkInterfaces = YES
	}

	if temp.Settings.EnableProxy == "" {
		temp.Settings.EnableProxy = YES
	}

	// Ensure a default in all cluster settings
	saveconfig := false
	s := temp.Cluster.Settings
	if s.ConnectInterval < 1*time.Second {
		s.ConnectInterval = 10
		saveconfig = true
	}

	if s.ConnectTimeout < 1*time.Second {
		s.ConnectTimeout = 10
		saveconfig = true
	}

	if s.PingInterval < 1*time.Second {
		s.PingInterval = 5
		saveconfig = true
	}

	if s.ReadTimeout < 1*time.Second {
		s.ReadTimeout = 11
		saveconfig = true
	}

	if s.JoinDelay < 1 {
		s.JoinDelay = 500 * time.Microsecond
		saveconfig = true
	}

	if saveconfig == true {
		//log.Debugf("Set defaults for cluster settings: (config:%+v new:%+v)", temp.Cluster.Settings, s)
		temp.Cluster.Settings = s
	}

	// Ensure a default in all dns settings
	save := false
	d := temp.DNS
	if d.Binding == "" {
		d.Binding = "localhost"
		save = true
	}

	if d.Port < 1 {
		d.Port = 53
		save = true
	}

	if len(d.AllowedRequests) == 0 {
		// Allow the most common DNS request types
		d.AllowedRequests = []string{"A", "AAAA", "NS", "MX", "SOA", "TXT", "CAA", "ANY", "CNAME", "MB", "MG", "MR", "WKS", "PTR", "HINFO", "MINFO", "SPF"}
		save = true
	}

	for domainName, localDomain := range d.Domains {
		for rid, record := range localDomain.Records {
			if record.Statistics == nil {
				//uid, _ := uuid.NewV4() // replaced by sha256
				//d.Domains[did].Records[rid].UUID = uid.String()
				//d.Domains[did].Records[rid].Statistics = balancer.NewStatistics(uid.String(), 0)

				hash := sha256.New()
				hash.Write([]byte(fmt.Sprintf("%s-%s-%x-%s", domainName, record.Name, record.Type, record.Target)))
				uuid := fmt.Sprintf("%x", hash.Sum(nil))
				d.Domains[domainName].Records[rid].UUID = uuid
				d.Domains[domainName].Records[rid].Statistics = balancer.NewStatistics(uuid, 0)

			}
		}
	}

	if save == true {
		temp.DNS = d
	}

	// ensure this is valid even if not used
	if temp.Web.Auth.Password == nil {
		temp.Web.Auth.Password = &web.AuthPassword{
			Users: make(map[string]string),
		}
	}

	if temp.Web.Auth.LDAP != nil {
		if temp.Web.Auth.LDAP.Method == "" {
			temp.Web.Auth.LDAP.Method = "TLS"
		}
		if temp.Web.Auth.LDAP.Port == 0 {
			temp.Web.Auth.LDAP.Port = 389
		}
	}
	log.Debug("Activating new config")
	configLock.Lock()
	config = temp

	log.Info("Config loaded succesfully")
	configLock.Unlock()

	return nil
}

// ValidateCertificates checks if all provided SSL certificates are correct
func (c *Config) ValidateCertificates() error {
	// Test Pool/Backend Certificate
	for poolName, pool := range c.Loadbalancer.Pools {
		if strings.EqualFold(pool.Listener.Mode, "https") {

			// Check if we have a main certificate FILE
			certcount := 0
			if pool.Listener.TLSConfig.CertificateProvided() {
				if err := pool.Listener.TLSConfig.Valid(); err != nil {
					return fmt.Errorf("Certificate issue for pool:%s %s", poolName, err.Error())
				}

				certcount++
			}

			// Check if we have certificates on a backend
			for backendName, backend := range pool.Backends {
				if backend.TLSConfig.CertificateProvided() {
					if err := pool.Listener.TLSConfig.Valid(); err != nil {
						return fmt.Errorf("Certificate issue for pool:%s backend:%s %s", poolName, backendName, err.Error())
					}

					certcount++
				}
			}

			if certcount == 0 {
				return fmt.Errorf("No certificate file specified for HTTPS mode on pool %s", poolName)
			}
		}
	}

	// Test Web Service Certificate
	if c.Web.TLSConfig.CertificateProvided() {
		if err := c.Web.TLSConfig.Valid(); err != nil {
			return fmt.Errorf("Could not load TLS configuration for Mercury Web Service: %s", err)
		}
	}

	// Test Cluster Service Certificate
	if c.Cluster.TLSConfig.CertificateProvided() {
		if err := c.Cluster.TLSConfig.Valid(); err != nil {
			return fmt.Errorf("Could not load TLS configuration for Mercury Cluster Service: %s", err)
		}
	}
	return nil
}

// Get returns the pointer to the latest config loaded
func Get() *Config {
	log := logging.For("config/get")
	_, file, no, ok := runtime.Caller(1)
	if ok && logConfigLocks {
		log.Debugf("getconfig for %s#%d", file, no)
	}

	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

// GetNoLock is Pointer to config without locking
func GetNoLock() *Config {
	return config
}

// Lock the config for Writes
func Lock() {
	log := logging.For("config/lock")
	_, file, no, ok := runtime.Caller(1)
	if ok && logConfigLocks {
		log.Debugf("lockconfig for %s#%d", file, no)
	}
	configLock.Lock()
}

// Unlock the config for Writes
func Unlock() {
	log := logging.For("config/unlock")
	_, file, no, ok := runtime.Caller(1)
	if ok && logConfigLocks {
		log.Debugf("unlockconfig for %s#%d", file, no)
	}
	configLock.Unlock()
}

// RLock the config for reads
func RLock() {
	log := logging.For("config/rlock")
	_, file, no, ok := runtime.Caller(1)
	if ok && logConfigLocks {
		log.Debugf("rlockconfig for %s#%d", file, no)
	}
	configLock.RLock()
}

// RUnlock the config for reads
func RUnlock() {
	log := logging.For("config/runlock")
	_, file, no, ok := runtime.Caller(1)
	if ok && logConfigLocks {
		log.Debugf("runlockconfig for %s#%d", file, no)
	}
	configLock.RUnlock()
}
