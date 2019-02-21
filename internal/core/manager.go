package core

import (
	"fmt"
	"runtime"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/internal/web"
	"github.com/schubergphilis/mercury/pkg/cluster"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
)

const (
	// YES when yes smimply isn't good enough
	YES = "yes"
)

// Manager main
type Manager struct {
	cluster                         *cluster.Manager
	healthchecks                    chan healthcheck.CheckResult
	dnsrefresh                      chan bool
	dnsdiscard                      chan string
	dnsoffline                      chan string
	dnsupdates                      chan *config.ClusterPacketGlobalDNSUpdate
	dnsremove                       chan *config.ClusterPacketGlobalDNSRemove
	clearStatsProxyBackend          chan *config.ClusterPacketClearProxyStatistics
	clusterGlbalDNSStatisticsUpdate chan *config.ClusterPacketGlbalDNSStatisticsUpdate
	addProxyBackend                 chan *config.ProxyBackendNodeUpdate
	removeProxyBackend              chan *config.ProxyBackendNodeUpdate
	proxyBackendStatisticsUpdate    chan *config.ProxyBackendStatisticsUpdate
	healthManager                   *healthcheck.Manager
	webAuthenticator                web.Auth
}

// NewManager creates a new manager
func NewManager() *Manager {
	manager := &Manager{
		healthchecks:                    make(chan healthcheck.CheckResult),
		dnsrefresh:                      make(chan bool),
		dnsdiscard:                      make(chan string),
		dnsoffline:                      make(chan string),
		dnsupdates:                      make(chan *config.ClusterPacketGlobalDNSUpdate),
		dnsremove:                       make(chan *config.ClusterPacketGlobalDNSRemove),
		addProxyBackend:                 make(chan *config.ProxyBackendNodeUpdate),
		removeProxyBackend:              make(chan *config.ProxyBackendNodeUpdate),
		proxyBackendStatisticsUpdate:    make(chan *config.ProxyBackendStatisticsUpdate),
		clusterGlbalDNSStatisticsUpdate: make(chan *config.ClusterPacketGlbalDNSStatisticsUpdate),
		clearStatsProxyBackend:          make(chan *config.ClusterPacketClearProxyStatistics),
	}
	return manager
}

// Initialize the service
func Initialize(reload <-chan bool) {
	log := logging.For("core/manager/init")
	log.Debug("Initializing Manager")

	manager := NewManager()

	// Create IP's
	CreateListeners()

	// Cluster communication
	go manager.InitializeCluster()

	// HealthCheck's
	manager.healthManager = healthcheck.NewManager()
	go manager.HealthHandler(manager.healthManager)
	go manager.InitializeHealthChecks(manager.healthManager)

	//log.Fatalf("web auth: %+v", config.Get().Web.Auth)
	if config.Get().Web.Auth.LDAP != nil {
		manager.webAuthenticator = config.Get().Web.Auth.LDAP
	} else {
		manager.webAuthenticator = config.Get().Web.Auth.Password
	}
	manager.setupAPI()

	// Create Listeners for Loadbalancer
	if config.Get().Settings.EnableProxy == YES {
		go manager.InitializeProxies()
		go manager.GetAllProxyStatsHandler()
	}

	// DNS updates
	go manager.InitializeDNSUpdates()
	go manager.StartDNSServer()

	// Webserver
	go manager.InitializeWebserver()

	for {
		select {
		case <-reload:
			log.Info("Reloading Manager")
			stats := new(runtime.MemStats)
			runtime.ReadMemStats(stats)
			log.WithField("memory", fmt.Sprintf("%5.2fk", float64(stats.Alloc)/1024)).Infof("Memory usage before reload")
			// Reload log level
			go logging.Configure(config.Get().Logging.Output, config.Get().Logging.Level)
			// Create new listeners if any
			CreateListeners()
			// Start new DNS Listeners (if changed)
			go manager.StartDNSServer()
			go UpdateDNSConfig()
			manager.dnsrefresh <- true

			// Start new healthchecks, and send exits to no longer used ones
			go manager.InitializeHealthChecks(manager.healthManager)
			// Re-read proxies, and update where needed
			// This needs to be after the healthchecks have been evacuated
			go manager.InitializeProxies()
			if config.Get().Web.Auth.LDAP != nil {
				manager.webAuthenticator = config.Get().Web.Auth.LDAP
			} else {
				manager.webAuthenticator = config.Get().Web.Auth.Password
			}
			// force cargbage collection due to golang map[] memory leakage
			// https://github.com/golang/go/issues/20135
			runtime.GC()
			log.WithField("memory", fmt.Sprintf("%5.2fk", float64(stats.Alloc)/1024)).Infof("Memory usage after reload")
		}
	}
}

// Cleanup the service
func Cleanup() {
	log := logging.For("core/manager")
	log.Debug("Cleaning up...")
	RemoveListeners()
}

// DumpNodes dumps the current state of all backend nodes
func DumpNodes() {
	for pn, pool := range config.Get().Loadbalancer.Pools {
		for bn, backend := range pool.Backends {
			for nn, node := range backend.Nodes {
				fmt.Printf("MEM DUMP OF CONFIG: pool:%s backend:%s node:%d %+v\n", pn, bn, nn, node)
			}
		}
	}
}
