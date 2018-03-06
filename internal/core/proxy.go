package core

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/proxy"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

// proxies contains all proxy listeners
var proxies = struct {
	sync.RWMutex
	pool map[string]*proxy.Listener
}{pool: make(map[string]*proxy.Listener)}

// InitializeProxies sets up new proxies, and modifies existing one for reloads
func (manager *Manager) InitializeProxies() {
	log := logging.For("core/proxy/init").WithField("func", "proxy")
	log.Info("Initializing Proxies")

	// Lock proxies for writing
	proxies.Lock()
	config.Lock()
	defer proxies.Unlock()
	defer config.Unlock()

	//copy loadbalancer pool
	loadbalancer := make(map[string]config.LoadbalancePool)
	for poolname, pool := range config.GetNoLock().Loadbalancer.Pools {
		loadbalancer[poolname] = pool
	}

	// First time setup, start handler in background
	if len(proxies.pool) == 0 {
		go manager.ProxyHandler()
	}

	// Get all existing proxies, and trim them to keep removableProxy list
	removableProxies := make(map[string]*proxy.Listener)
	for poolname, pool := range proxies.pool {
		removableProxies[poolname] = pool
	}

	// Loop through pools -> proxy vip's
	for poolname, pool := range loadbalancer {
		var newProxy *proxy.Listener

		plog := log.WithField("pool", poolname)

		if pool.Listener.IP == "" {
			plog.Debug("No listener IP for pool, skipping proxy")
			continue
		}

		if existingProxy, ok := proxies.pool[poolname]; ok {
			plog.Debug("Existing proxy")
			// Proxy already exists
			// See if we have a reason to stop it

			existingTLS := existingProxy.TLSConfig
			//newTLS := &tls.Config{}
			newTLS, err := tlsconfig.LoadCertificate(pool.Listener.TLSConfig)
			if err != nil {
				plog.Warn("Error loading certificate")
			}

			// We must go through backends in the same order each run
			// maps however are random, so:
			// lets get all backend names
			// order them alfabeticaly
			// loop over that to add Certificates
			var backendSorted []string
			for backendName := range pool.Backends {
				backendSorted = append(backendSorted, backendName)
			}
			sort.Strings(backendSorted)

			for _, backendName := range backendSorted {
				if pool.Backends[backendName].TLSConfig.CertificateFile != "" {
					tlsconfig.AddCertificate(pool.Backends[backendName].TLSConfig, newTLS)
					log.Debugf("ADDCERT: adding cert for %s", backendName)
				}
			}

			// Update listener if the below changed
			listenerChanged := existingProxy.ListenerMode != pool.Listener.Mode ||
				existingProxy.IP != pool.Listener.IP ||
				existingProxy.Port != pool.Listener.Port ||
				existingProxy.MaxConnections != pool.Listener.MaxConnections ||
				existingProxy.ReadTimeout != pool.Listener.ReadTimeout ||
				existingProxy.WriteTimeout != pool.Listener.WriteTimeout ||
				existingProxy.OCSPStapling != pool.Listener.OCSPStapling ||
				!reflect.DeepEqual(existingTLS.Certificates, newTLS.Certificates)

			// Has listener changed?
			if listenerChanged {
				// Interface changes, we need to restart the proxy, lets stop it
				certchange := !reflect.DeepEqual(existingTLS.Certificates, newTLS.Certificates)
				log.WithField("pool", poolname).Debugf("listener changed - mode:%t ip:%t port:%t, maxcon:%t readtimeout:%t writetimeout:%t ocsp:%t cert:%t",
					existingProxy.ListenerMode != pool.Listener.Mode,
					existingProxy.IP != pool.Listener.IP,
					existingProxy.Port != pool.Listener.Port,
					existingProxy.MaxConnections != pool.Listener.MaxConnections,
					existingProxy.ReadTimeout != pool.Listener.ReadTimeout,
					existingProxy.WriteTimeout != pool.Listener.WriteTimeout,
					existingProxy.OCSPStapling != pool.Listener.OCSPStapling,
					certchange)
				log.WithField("pool", poolname).Info("Restarting existing proxy for new listener settings")
				existingProxy.Stop()
				existingProxy.SetListener(pool.Listener.Mode, pool.Listener.IP, pool.Listener.Port, pool.Listener.MaxConnections, newTLS, pool.Listener.ReadTimeout, pool.Listener.WriteTimeout, pool.Listener.HTTPProto, pool.Listener.OCSPStapling)
				go existingProxy.Start()
			}

			// Continue with new proxy
			newProxy = existingProxy
			// Do not remove this proxy
			delete(removableProxies, poolname)

		} else {
			// We have a non existing proxy, setup a new one
			clog := log.WithField("ip", pool.Listener.IP).WithField("port", pool.Listener.Port).WithField("mode", pool.Listener.Mode).WithField("pool", poolname)
			clog.Debug("Creating new proxy")

			h := sha256.New()
			h.Write([]byte(fmt.Sprintf("%s-%s-%s-%d", poolname, pool.Listener.Mode, pool.Listener.IP, pool.Listener.Port)))
			uuid := fmt.Sprintf("%x", h.Sum(nil))
			newProxy = proxy.New(uuid, poolname, pool.Listener.MaxConnections)

			newTLS, err := tlsconfig.LoadCertificate(pool.Listener.TLSConfig)
			if err != nil {
				plog.Warn("Error loading certificate")
			}

			var backendSorted []string
			for backendName := range pool.Backends {
				backendSorted = append(backendSorted, backendName)
			}

			sort.Strings(backendSorted)

			for _, backendName := range backendSorted {
				if pool.Backends[backendName].TLSConfig.CertificateFile != "" {
					tlsconfig.AddCertificate(pool.Backends[backendName].TLSConfig, newTLS)
					log.Debugf("ADDCERT: adding cert for %s", backendName)
				}
			}

			newProxy.SetListener(pool.Listener.Mode, pool.Listener.IP, pool.Listener.Port, pool.Listener.MaxConnections, newTLS, pool.Listener.ReadTimeout, pool.Listener.WriteTimeout, pool.Listener.HTTPProto, pool.Listener.OCSPStapling)
			go newProxy.Start()
			// Register new proxy
			proxies.pool[poolname] = newProxy
			clog.Debug("Proxy listener started")
		}

		// We now have a proxy listener ready and working, lets add its config dynamicly

		// Get all existing backends, we remove the ones that remain and were not configured
		//var removableBackends map[string]*proxy.Backend
		removableBackends := make(map[string]*proxy.Backend)
		for backendname, backendpool := range newProxy.Backends {
			removableBackends[backendname] = backendpool
		}

		if err := newProxy.LoadErrorPage(pool.ErrorPage); err != nil {
			// This is checked when loading the config
			plog.WithField("file", pool.ErrorPage.File).WithError(err).Warn("Unable to load Error page")
		}

		if err := newProxy.LoadMaintenancePage(pool.MaintenancePage); err != nil {
			// This is checked when loading the config
			plog.WithField("file", pool.MaintenancePage.File).WithError(err).Warn("Unable to load Maintenance page")
		}

		//log.Debugf("proxy:%s Proxy has the following backends before init:%+v", poolname, removableBackends)
		for bid := range removableBackends {
			plog.WithField("backend", bid).Debug("Backend before init")
		}

		// Add ACL's from pool to backend
		// Backend already has its own acl's , we just merge them
		for backendname, backendpool := range pool.Backends {

			// unmark the ones we have in our config, to not be removed
			if _, ok := removableBackends[backendname]; ok {
				delete(removableBackends, backendname)
				plog.WithField("backend", backendname).Debug("Marking backend to keep")
			}

			// Add backend  (will merge if exists)
			plog.WithField("backend", backendname).Info("Adding/Updating backend")
			newProxy.UpdateBackend(backendpool.UUID, backendname, backendpool.BalanceMode.Method, backendpool.ConnectMode, backendpool.HostNames, pool.Listener.MaxConnections, backendpool.ErrorPage, backendpool.MaintenancePage)

			// Use backend to attach acl's
			backend := newProxy.Backends[backendname]

			var inboundACLs []proxy.ACL
			var outboundACLs []proxy.ACL

			// Add pool ACL's
			for _, acl := range pool.InboundACL {
				inboundACLs = append(inboundACLs, acl)
			}
			for _, acl := range pool.OutboundACL {
				outboundACLs = append(outboundACLs, acl)
			}
			// Add backend ACL's
			for _, acl := range backendpool.InboundACL {
				inboundACLs = append(inboundACLs, acl)
			}
			for _, acl := range backendpool.OutboundACL {
				outboundACLs = append(outboundACLs, acl)
			}
			// Replace existing acls with new one
			if !reflect.DeepEqual(backend.InboundACL, inboundACLs) {
				for _, acl := range inboundACLs {
					plog.WithField("backend", backendname).WithField("acl", fmt.Sprintf("%+v", acl)).Debug("Setting inbound ACL")
				}
				backend.SetACL("in", inboundACLs)
			}
			if !reflect.DeepEqual(backend.OutboundACL, outboundACLs) {
				for _, acl := range inboundACLs {
					plog.WithField("backend", backendname).WithField("acl", fmt.Sprintf("%+v", acl)).Debug("Setting outbound ACL")
				}
				backend.SetACL("out", outboundACLs)
			}

			// Check backend Nodes
			// IF node is local check with local config
			// IF node is remote update of removal should be sent at config loading
			if nodes, err := backend.GetBackendsUUID(); err != nil {
				for _, nodeid := range nodes {
					found := false
					for _, node := range backendpool.Nodes {
						if nodeid == node.UUID {
							found = true
						}
					}
					if found == false {
						plog.WithField("backend", backendname).WithField("uuid", nodeid).Debug("Backend node longer exists in config")
						backend.RemoveNodeByID(nodeid)
					}
				}
			}

		} // end of backend loop

		// Remove all backends which remained on the removableBackends
		// these are no longer configured and should be removed
		for backendName := range removableBackends {
			plog.WithField("backend", backendName).Info("Removing unused backend")
			newProxy.RemoveBackend(backendName)
		}

	} // end of pool loop

	// Remove all proxies which remained on the removableProxies listeners
	// these are no longer configured and should be removeHeader
	for proxyName, proxy := range removableProxies {
		log.WithField("pool", proxyName).Info("Stopping unused proxy")
		proxy.Stop()
		delete(proxies.pool, proxyName)
	}

}

// ProxyHandler is the interface between the cluster and the proxies, passing along updates
func (manager *Manager) ProxyHandler() {
	log := logging.For("core/proxy/handler").WithField("func", "proxy")
	log.Debug("Calling proxy handler")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)

	for {
		select {
		// Wait for proxy backend updates
		case _ = <-signalChan:
			for _, proxy := range proxies.pool {
				proxy.Debug()
			}

		case update := <-manager.addProxyBackend: // add or update
			log.Debug("UpdateProxyBackend")

			plog := log.WithField("pool", update.PoolName).WithField("backend", update.BackendName).WithField("uuid", update.BackendNodeUUID)

			// Check if packet is for a proxy we have
			plog.Debugf("proxyGetBackend")
			backend, err := proxyGetBackend(update.PoolName, update.BackendName)
			if err != nil {
				plog.WithError(err).Debug("Unable to get backend")
				continue
			}

			plog.Debugf("proxyGetNodeByUUID")
			nodeid, err := proxyGetNodeByUUID(backend, update.BackendNodeUUID)
			if err != nil {
				plog.WithError(err).Debug("Unable to get backend node by UUID")
				continue
			}

			// Node exists, update existing
			if nodeid >= 0 {
				plog.WithField("node", update.BackendNode.Name()).WithField("ip", update.BackendNode.IP).WithField("port", update.BackendNode.Port).Debug("Update proxy node")
				backend.UpdateBackendNode(nodeid, update.BackendNode.Status)
				continue
			}

			// New node, add
			backendNode := proxy.NewBackendNode(update.BackendNodeUUID, update.BackendNode.IP, update.BackendNode.Hostname, update.BackendNode.Port, update.BackendNode.MaxConnections, update.BackendNode.LocalNetwork, update.BackendNode.Preference, update.BackendNode.Status) // max connections = 1 -> not used
			plog.WithField("node", backendNode.Name()).WithField("ip", backendNode.IP).WithField("port", backendNode.Port).Debug("Add proxy node")
			backend.AddBackendNode(backendNode)

		case update := <-manager.removeProxyBackend:
			log.Debug("removeProxyBackend")
			plog := log.WithField("pool", update.PoolName).WithField("backend", update.BackendName).WithField("uuid", update.BackendNodeUUID)

			// Check if packet is for a proxy we have
			plog.Debugf("proxyGetBackend")
			backend, err := proxyGetBackend(update.PoolName, update.BackendName)
			if err != nil {
				plog.WithError(err).Debug("Unable to get backend")
				continue
			}

			plog.Debugf("proxyGetNodeByUUID")
			nodeid, err := proxyGetNodeByUUID(backend, update.BackendNodeUUID)
			if err != nil {
				plog.WithError(err).Debug("Unable to get backend node by UUID")
				continue
			}

			// Remove proxy backend
			if nodeid >= 0 {
				plog.WithField("node", update.BackendNode.Name()).WithField("ip", update.BackendNode.IP).WithField("port", update.BackendNode.Port).Debug("Remove proxy node")
				backend.RemoveBackendNode(nodeid)
			}

		case update := <-manager.clearStatsProxyBackend:
			// Check if packet is for a proxy we have
			log.Debug("clearStatsProxyBackend")
			log.WithField("pool", update.PoolName).WithField("backend", update.BackendName).Debug("Clearing proxy stats")
			backend, err := proxyGetBackend(update.PoolName, update.BackendName)
			if err != nil {
				log.WithError(err).Debug("Unable to get backend")
				continue
			}

			backend.ClearStats()
		}
	}
}

func proxyGetBackend(poolname, backendname string) (*proxy.Backend, error) {
	proxies.RLock()
	defer proxies.RUnlock()

	if _, ok := proxies.pool[poolname]; !ok {
		return nil, fmt.Errorf("Received update for unknown pool:%s", poolname)
	}

	if backend, ok := proxies.pool[poolname].Backends[backendname]; ok {
		return backend, nil
	}

	return nil, fmt.Errorf("proxy:%s Received update for unknown backend:%s", poolname, backendname)
}

func proxyGetNodeByUUID(backend *proxy.Backend, uuid string) (int, error) {
	proxies.RLock()
	defer proxies.RUnlock()

	nodeid := -1
	// Check if the update is for a pool we have in our proxy listener

	// Check if we already have a UUID of this node
	for bnid, bn := range backend.Nodes {
		if bn.UUID == uuid {
			nodeid = bnid
		}
	}

	return nodeid, nil
}

// GetAllProxyStatsHandler periodicly gets all stats
func (manager *Manager) GetAllProxyStatsHandler() {
	ticker := time.NewTicker(500 * time.Millisecond)
	var oldstats []*config.ProxyBackendStatisticsUpdate
	for {
		select {
		case <-ticker.C:
			newstats := manager.GetAllProxyStats()
			for _, new := range newstats {
				for _, old := range oldstats {
					if old.Statistics.UUID == new.Statistics.UUID {
						if old.Statistics.ClientsConnects != new.Statistics.ClientsConnects {
							manager.proxyBackendStatisticsUpdate <- new
						}
					} // uuid
				} // old
			} // new
			oldstats = newstats

		}
	}
}

// GetAllProxyStats gets all proxy statistics and sends them to the proxy handler
func (manager *Manager) GetAllProxyStats() []*config.ProxyBackendStatisticsUpdate {
	proxies.RLock()
	defer proxies.RUnlock()
	var stats []*config.ProxyBackendStatisticsUpdate
	for poolname, pool := range proxies.pool {
		for backendname := range pool.Backends {
			s := proxies.pool[poolname].GetBackendStats(backendname)
			ps := &config.ProxyBackendStatisticsUpdate{
				PoolName:    poolname,
				BackendName: backendname,
				Statistics:  *s,
			}
			stats = append(stats, ps)
		}
	}
	return stats
}
