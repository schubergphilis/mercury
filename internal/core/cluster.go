package core

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/cluster"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/proxy"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

const (
/*globalDNSUpdate      = "globalDNSUpdate"
globalDNSStatistics  = "globalDNSStatistics"
clearProxyStatistics = "clearProxyStatistics"
backendNodeUpdate    = "backendNodeUpdate"*/
)

// ClusterClient is the interface between the program and the cluster
func (manager *Manager) ClusterClient(cl *cluster.Manager) {
	log := logging.For("core/cluster/client")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)

	for {
		select {
		case _ = <-signalChan:
			cl.StateDump()
		case node := <-cl.NodeJoin:
			log.WithField("func", "core").Debug("Join")
			log.WithField("func", "cluster").WithField("client", node).WithField("request", "clusterJoin").Info("Received cluster join update")
			cl.ToNode <- cluster.NodeMessage{Node: node, Message: config.ClusterPacketConfigRequest{}}

			log.WithField("func", "core").Debug("Join - sending channel")
			// If a node (re)-joins remove its old entries, since uuid's might have changed or uuids might be gone
			manager.dnsdiscard <- node
			log.WithField("func", "core").Debug("Join - sending channel OK")

			go clusterDNSUpdateSingleBroadcastAll(cl, node)
			log.WithField("func", "core").Debug("Join OK")

		case node := <-cl.NodeLeave:
			log.WithField("func", "core").Debug("Leave")
			// client left the cluster
			log.WithField("func", "cluster").WithField("client", node).WithField("request", "clusterLeave").Info("Received cluster leave update")
			go manager.BackendNodeDiscard(node)

			// If a node goes offline, mark all entries as offline
			manager.dnsoffline <- node
			log.WithField("func", "core").Debug("Leave OK")

		case packet := <-cl.FromCluster:
			log.WithField("func", "core").Debug("FromCluster")

			switch packet.DataType {
			case "config.ClusterPacketConfigRequest":

				log.WithField("func", "core").Debug("RequestConfig")
				// Ignore config requests from self
				log.WithField("client", packet.Name).WithField("request", packet.DataType).Info("Sending config")
				go clusterDNSUpdateSingleBroadcastAll(cl, packet.Name)
				log.WithField("func", "core").Debug("RequestConfig OK")

			case "config.ClusterPacketGlobalDNSUpdate":
				log.WithField("func", "core").Debug("globalDNSUpdate")
				dnsupdate := &config.ClusterPacketGlobalDNSUpdate{}
				err := packet.Message(dnsupdate)
				if err != nil {
					log.Warn("Unable to parse ClusterGlobalDNSUpdate request: %s", err.Error())
					continue
				}
				log.WithField("func", "dns").WithField("client", packet.Name).WithField("request", packet.DataType).WithField("pool", dnsupdate.PoolName).WithField("backend", dnsupdate.BackendName).WithField("uuid", dnsupdate.BackendUUID).WithField("hostname", dnsupdate.DNSEntry.HostName).WithField("domain", dnsupdate.DNSEntry.Domain).WithField("ip", dnsupdate.DNSEntry.IP).WithField("ip6", dnsupdate.DNSEntry.IP6).WithField("online", dnsupdate.Online).Info("Received cluster dns update")
				manager.dnsupdates <- dnsupdate
				log.WithField("func", "core").Debug("globalDNSUpdate OK")

			case "config.ClusterPacketGlbalDNSStatisticsUpdate":
				log.WithField("func", "core").Debug("globalDNSStatistics")
				su := &config.ClusterPacketGlbalDNSStatisticsUpdate{}
				err := packet.Message(su)
				if err != nil {
					log.Warn("Unable to parse ClusterGlbalDNSStatisticsUpdate request: %s", err.Error())
					continue
				}
				su.ClusterNode = packet.Name

				dnsEntry, err := getDNSentry(su.PoolName, su.BackendName)
				if err != nil {
					log.WithField("pool", su.PoolName).WithField("backend", su.BackendName).WithError(err).Debug("DNS Get Entry error")
					continue
				}
				su.DNSEntry = dnsEntry
				manager.clusterGlbalDNSStatisticsUpdate <- su
				log.WithField("func", "core").Debug("globalDNSStatistics OK")

			case "config.ClusterPacketClearProxyStatistics":
				log.WithField("func", "core").Debug("clearProxyStatistics")
				su := &config.ClusterPacketClearProxyStatistics{}
				log.WithField("data", packet.DataMessage).Debug("Clear proxy stats")

				err := packet.Message(su)
				if err != nil {
					log.Warn("Unable to parse ClusterClearProxyStatistics request: %s", err.Error())
					continue
				}
				log.Debug("Clear proxy status update to clearStatsProxyBackend")
				manager.clearStatsProxyBackend <- su
				log.Debug("Clear proxy stats done")
				log.WithField("func", "core").Debug("clearProxyStatistics OK")

			default:
				log.WithField("client", packet.Name).WithField("request", packet.DataType).WithField("data", packet.DataMessage).Warn("Recieved unknown cluster request")
			}

		case proxyBackendStatistics := <-manager.proxyBackendStatisticsUpdate:
			log.WithField("func", "core").Debug("proxyBackendStatisticsUpdate")
			clog := log.WithField("pool", proxyBackendStatistics.PoolName).WithField("backend", proxyBackendStatistics.BackendName)
			clog.Debug("Received proxy statistics")
			cgstats := &config.ClusterPacketGlbalDNSStatisticsUpdate{
				BackendName:       proxyBackendStatistics.BackendName,
				PoolName:          proxyBackendStatistics.PoolName,
				ClusterNode:       config.Get().Cluster.Binding.Name,
				UUID:              proxyBackendStatistics.Statistics.UUID,
				ClientsConnected:  proxyBackendStatistics.Statistics.ClientsConnected,
				ClientsConnects:   proxyBackendStatistics.Statistics.ClientsConnects,
				RX:                proxyBackendStatistics.Statistics.RX,
				TX:                proxyBackendStatistics.Statistics.TX,
				ResponseTimeValue: proxyBackendStatistics.Statistics.ResponseTimeValue,
			}
			go clusterProxyStatsBroadcast(cl, cgstats)
			log.WithField("func", "core").Debug("proxyBackendStatisticsUpdate OK")

		case healthcheck := <-manager.healthchecks:
			log.WithField("func", "core").Debug("healthcheck")
			clog := log.WithField("pool", healthcheck.PoolName).WithField("backend", healthcheck.BackendName).WithField("nodeuuid", healthcheck.NodeUUID).WithField("node", healthcheck.NodeName).WithField("online", healthcheck.Online).WithField("func", "healthcheck")
			clog.Info("Received healthcheck update")
			// Local HealthCheck update received from local healthcheck workers
			// We only reach this update if a Healthcheck changed state (either up or down)
			// If Changed we need to:
			// - update our local config, so we are aware that a node is online/offline
			if _, ok := config.Get().Loadbalancer.Pools[healthcheck.PoolName]; !ok {
				clog.Warn("Pool no longer exists, discarding healthcheck update")
				continue
			}
			if _, ok := config.Get().Loadbalancer.Pools[healthcheck.PoolName].Backends[healthcheck.BackendName]; !ok {
				clog.Warn("Backend of pool no longer exists, discarding healthcheck update")
				continue
			}
			config.UpdateNodeStatus(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID, healthcheck.Online, healthcheck.ErrorMsg)

			pool := config.Get().Loadbalancer.Pools[healthcheck.PoolName]
			backend := config.Get().Loadbalancer.Pools[healthcheck.PoolName].Backends[healthcheck.BackendName]
			clog.WithField("searchnode", healthcheck.NodeUUID).Debug("Search node by uuid")
			node, err := config.GetNodeByUUID(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID)
			//node, err := backend.GetNodeByUUID(healthcheck.NodeUUID)
			if err != nil {
				clog.WithField("error", err).Warn("ignoring healthcheck update for unknown node")
				continue
			}
			//clog.WithField("foundnode", node.Name()).Debug("Updating status health")
			//node.UpdateStatus(healthcheck.Online, healthcheck.ErrorMsg)
			clog.WithField("foundnode", node.Name()).Debug("Updated status health")

			if config.Get().Settings.EnableProxy == YES && pool.Listener.IP != "" {
				go clusterClearProxyStatistics(cl, healthcheck.PoolName, healthcheck.BackendName)
				// - send the update to our proxy server if enabled
				go manager.updateProxyBackendNode(healthcheck.PoolName, healthcheck.BackendName, node)
			}

			// - send DNS updates to local dns servers
			// TODO newDNSUpdate that acts on both cluster and non cluster version
			dnsupdate := &config.ClusterPacketGlobalDNSUpdate{
				ClusterNode: config.Get().Cluster.Binding.Name,
				PoolName:    healthcheck.PoolName,
				BackendName: healthcheck.BackendName,
				DNSEntry:    backend.DNSEntry,
				BalanceMode: backend.BalanceMode,
				BackendUUID: backend.UUID,
				Online:      getLocalNodeStatus(healthcheck.PoolName, healthcheck.BackendName),
			}

			manager.dnsupdates <- dnsupdate
			// - send DNS update to cluster nodes
			go clusterDNSUpdateBroadcast(cl, config.Get().Cluster.Binding.Name, healthcheck.PoolName, healthcheck.BackendName, backend.DNSEntry, backend.BalanceMode, backend.UUID)
			clog.WithField("func", "core").WithField("foundnode", node.Name()).Debug("healthcheck OK")
		}
	}
}

func getDNSentry(poolname, backendname string) (config.DNSEntry, error) {
	config.Lock()
	defer config.Unlock()
	if _, ok := config.GetNoLock().Loadbalancer.Pools[poolname]; ok {
		if _, ok := config.GetNoLock().Loadbalancer.Pools[poolname].Backends[backendname]; ok {
			return config.GetNoLock().Loadbalancer.Pools[poolname].Backends[backendname].DNSEntry, nil
		}
	}
	return config.DNSEntry{}, fmt.Errorf("unkown pool/backend: %s/%s", poolname, backendname)
}

func (manager *Manager) updateProxyBackendNode(poolName string, backendName string, node config.BackendNode) {
	log := logging.For("core/cluster/proxybackendupdate").WithField("func", "proxy")
	proxyupdate := &config.ProxyBackendNodeUpdate{
		PoolName:        poolName,
		BackendName:     backendName,
		BackendNode:     proxy.BackendNode{IP: node.IP, Port: node.Port, Hostname: node.Hostname, MaxConnections: node.MaxConnections, LocalNetwork: node.LocalNetwork, Preference: node.Preference},
		BackendNodeUUID: node.UUID,
	}
	if config.Get().Settings.EnableProxy == YES {
		clog := log.WithField("pool", proxyupdate.PoolName).WithField("backend", proxyupdate.BackendName).WithField("uuid", proxyupdate.BackendNodeUUID).WithField("online", node.Online).WithField("ip", node.IP).WithField("port", node.Port)

		if node.Online == true {
			clog.Warnf("Adding backend to proxy")
			manager.addProxyBackend <- proxyupdate
			clog.Warnf("Adding backend to proxy OK")

		} else {
			clog.Warnf("Removing backend from proxy")
			manager.removeProxyBackend <- proxyupdate
			clog.Warnf("Removing backend from proxy OK")
		}
	}
}

// InitializeCluster sets up the cluster, starts it, and starts the client
func (manager *Manager) InitializeCluster() {
	//log := logging.For("core/cluster/init")
	cluster.ChannelBufferSize = 100
	cl := cluster.NewManager(config.Get().Cluster.Binding.Name, config.Get().Cluster.Binding.AuthKey)
	configured := cl.NodesConfigured()
	for _, node := range config.Get().Cluster.Nodes {
		if _, ok := configured[node.Name]; ok {
			delete(configured, node.Name)
		}
		// Add newly configured nodes
		if !cl.NodeConfigured(node.Name) {
			cl.AddNode(node.Name, node.Addr)
		}
	}
	//  remove old cluster nodes
	for name := range configured {
		cl.RemoveNode(name)
	}
	tlsConfig, err := tlsconfig.LoadCertificate(config.Get().Cluster.TLSConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = cl.ListenAndServeTLS(config.Get().Cluster.Binding.Addr, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}
	//go cluster.Start()
	go writeClusterLog(cl)
	go manager.ClusterClient(cl)
	//log.Info("Ready to serve!")

}

func writeClusterLog(cl *cluster.Manager) {
	cluster.LogTraffic = false
	log := logging.For("cluster/log").WithField("func", "cluster").WithField("manager", config.Get().Cluster.Binding.Name)
	for {
		select {
		case logEntry := <-cl.Log:
			log.Info(logEntry)
		}
	}
}

// BackendNodeUpdate updates the backend nodes of a single backendpool
func (manager *Manager) BackendNodeUpdate(pool string, backend string, node *config.BackendNode) {
	log := logging.For("core/cluster/Update").WithField("func", "proxy")
	for nid, n := range config.Get().Loadbalancer.Pools[pool].Backends[backend].Nodes {
		if n.UUID == node.UUID {
			log.WithField("wasonline", n.Online).WithField("online", node.Online).Debug("Update of existing node")
			// We already know this node, only update its status and error
			config.UpdateBackendNode(pool, backend, nid, node.Online, node.Errors)
			return
		}
	}
	log.WithField("online", node.Online).Debug("Update of new node")
	// It is a new Node, add it
	// TODO: we need to think about locking
	config.AddBackendNode(pool, backend, node)
}

// BackendNodeDiscard removes backend nodes of a backendpool based on node name
func (manager *Manager) BackendNodeDiscard(node string) {
	config.RLock()
	defer config.RUnlock()
	log := logging.For("core/cluster/Discard").WithField("func", "proxy")
	for pid := range config.GetNoLock().Loadbalancer.Pools {
		for bid := range config.GetNoLock().Loadbalancer.Pools[pid].Backends {
			for nid := len(config.GetNoLock().Loadbalancer.Pools[pid].Backends[bid].Nodes) - 1; nid >= 0; nid-- {
				if _, ok := config.GetNoLock().Loadbalancer.Pools[pid].Backends[bid]; ok {
					n := config.GetNoLock().Loadbalancer.Pools[pid].Backends[bid].Nodes[nid]
					if n.ClusterName == node {
						log.WithField("pool", pid).WithField("backend", bid).WithField("hostname", n.Hostname).WithField("port", n.Port).WithField("uuid", n.UUID).Warnf("Discarding backend node of leaving cluster")
						n.Online = false
						go manager.updateProxyBackendNode(pid, bid, *n)
						// Remove node from our config
						go config.RemoveBackendNodeUUID(pid, bid, n.UUID)
					}
				}
			}
		}
	}
}

func getLocalNodeStatus(poolName, backendName string) bool {
	var online = 0
	config.RLock()
	defer config.RUnlock()
	for _, node := range config.GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
		if node.Online == true && node.ClusterName == config.GetNoLock().Cluster.Binding.Name {
			online++
		}
	}
	if online > 0 {
		return true
	}
	return false
}

func clusterClearProxyStatistics(cl *cluster.Manager, poolname string, backendname string) {

	c := &config.ClusterPacketClearProxyStatistics{
		PoolName:    poolname,
		BackendName: backendname,
	}

	cl.ToCluster <- c
}

func clusterProxyStatsBroadcast(cl *cluster.Manager, stats *config.ClusterPacketGlbalDNSStatisticsUpdate) {
	cl.ToCluster <- stats
}

func clusterDNSUpdateSingleBroadcastAll(cl *cluster.Manager, client string) {
	log := logging.For("core/cluster/dnsbroadcast").WithField("func", "dns")
	config.RLock()
	defer config.RUnlock()
	for poolName, pool := range config.GetNoLock().Loadbalancer.Pools {
		for backendName, backend := range pool.Backends {
			log.WithField("pool", poolName).WithField("backend", backendName).WithField("client", client).Debug("Broadcasting DNS update to cluster client")
			go clusterDNSUpdateSingle(cl, client, config.GetNoLock().Cluster.Binding.Name, poolName, backendName, backend.DNSEntry, backend.BalanceMode, backend.UUID)
		}
	}
}

func clusterDNSUpdateBroadcastAll(cl *cluster.Manager) {
	log := logging.For("core/cluster/dnsbroadcast").WithField("func", "dns")
	config.RLock()
	defer config.RUnlock()
	for poolName, pool := range config.GetNoLock().Loadbalancer.Pools {
		for backendName, backend := range pool.Backends {
			log.WithField("pool", poolName).WithField("backend", backendName).Debug("Broadcasting DNS update to all clusters")
			go clusterDNSUpdateBroadcast(cl, config.GetNoLock().Cluster.Binding.Name, poolName, backendName, backend.DNSEntry, backend.BalanceMode, backend.UUID)
		}
	}
}

func clusterDNSUpdateSingle(cl *cluster.Manager, client string, clusterNode string, poolName string, backendName string, dnsEntry config.DNSEntry, balanceMode config.BalanceMode, backendUUID string) {
	online := getLocalNodeStatus(poolName, backendName)

	dnsupdate := config.ClusterPacketGlobalDNSUpdate{
		ClusterNode: clusterNode,
		PoolName:    poolName,
		BackendName: backendName,
		DNSEntry:    dnsEntry,
		BalanceMode: balanceMode,
		BackendUUID: backendUUID,
		Online:      online,
	}
	cl.ToCluster <- dnsupdate
}

func clusterDNSUpdateBroadcast(cl *cluster.Manager, clusterNode string, poolName string, backendName string, dnsEntry config.DNSEntry, balanceMode config.BalanceMode, backendUUID string) {
	online := getLocalNodeStatus(poolName, backendName)

	dnsupdate := config.ClusterPacketGlobalDNSUpdate{
		ClusterNode: clusterNode,
		PoolName:    poolName,
		BackendName: backendName,
		DNSEntry:    dnsEntry,
		BalanceMode: balanceMode,
		BackendUUID: backendUUID,
		Online:      online,
	}
	cl.ToCluster <- dnsupdate
}
