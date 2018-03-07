package core

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/cluster"
	"github.com/schubergphilis/mercury/pkg/dns"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/proxy"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
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

			go clusterDNSUpdateSingleBroadcastAll(cl, node)

		case node := <-cl.NodeLeave:
			log.WithField("func", "core").Debug("Leave")
			// client left the cluster
			log.WithField("func", "cluster").WithField("client", node).WithField("request", "clusterLeave").Info("Received cluster leave update")
			go manager.BackendNodeDiscard(node)

			// If a node goes offline, mark all entries as offline
			manager.dnsoffline <- node

		case packet := <-cl.FromCluster:
			log.WithField("func", "core").Debug("FromCluster")

			switch packet.DataType {
			case "config.ClusterPacketConfigRequest":

				log.WithField("func", "core").Debug("RequestConfig")
				// Ignore config requests from self
				log.WithField("client", packet.Name).WithField("request", packet.DataType).Info("Sending config")
				go clusterDNSUpdateSingleBroadcastAll(cl, packet.Name)

			case "config.ClusterPacketGlobalDNSUpdate":
				log.WithField("func", "core").Debug("globalDNSUpdate")
				dnsupdate := &config.ClusterPacketGlobalDNSUpdate{}
				err := packet.Message(dnsupdate)
				if err != nil {
					log.Warn("Unable to parse ClusterGlobalDNSUpdate request: %s", err.Error())
					continue
				}

				log.WithField("func", "dns").WithField("client", packet.Name).WithField("request", packet.DataType).WithField("pool", dnsupdate.PoolName).WithField("backend", dnsupdate.BackendName).WithField("uuid", dnsupdate.BackendUUID).WithField("hostname", dnsupdate.DNSEntry.HostName).WithField("domain", dnsupdate.DNSEntry.Domain).WithField("ip", dnsupdate.DNSEntry.IP).WithField("ip6", dnsupdate.DNSEntry.IP6).WithField("status", dnsupdate.Status.String()).Info("Received cluster dns update")
				manager.dnsupdates <- dnsupdate

			case "config.ClusterPacketGlobalDNSRemove":
				log.WithField("func", "core").Debug("globalDNSRemove")
				dnsremove := &config.ClusterPacketGlobalDNSRemove{}
				err := packet.Message(dnsremove)
				if err != nil {
					log.Warn("Unable to parse ClusterGlobalDNSRemove request: %s", err.Error())
					continue
				}

				log.WithField("func", "dns").WithField("client", packet.Name).WithField("request", packet.DataType).WithField("cluster", dnsremove.ClusterNode).WithField("domain", dnsremove.Domain).WithField("hostname", dnsremove.Hostname).Info("Received cluster dns removal")
				manager.dnsremove <- dnsremove

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

		case healthcheck := <-manager.healthchecks:
			log.WithField("func", "core").Debug("healthcheck")
			clog := log.WithField("pool", healthcheck.PoolName).WithField("backend", healthcheck.BackendName).WithField("nodeuuid", healthcheck.NodeUUID).WithField("node", healthcheck.NodeName).WithField("status", healthcheck.ReportedStatus.String()).WithField("func", "healthcheck")
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

			// Update node status in memory
			config.UpdateNodeStatus(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID, healthcheck.ReportedStatus, healthcheck.ErrorMsg)

			// Get UUID of node
			clog.WithField("searchnode", healthcheck.NodeUUID).Debug("Search node by uuid")
			node, err := config.GetNodeByUUID(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID)
			if err != nil {
				clog.WithField("error", err).Warn("ignoring healthcheck update for unknown node")
				continue
			}
			clog.WithField("foundnode", node.Name()).Debug("Updated status health")

			// If we have proxy enabled send node update to proxy
			pool := config.Get().Loadbalancer.Pools[healthcheck.PoolName]
			if config.Get().Settings.EnableProxy == YES && pool.Listener.IP != "" {
				go clusterClearProxyStatistics(cl, healthcheck.PoolName, healthcheck.BackendName)
				go manager.updateProxyBackendNode(healthcheck.PoolName, healthcheck.BackendName, node)
			}

			// Send update to DNS
			go manager.sendDNSUpdate(cl, healthcheck.PoolName, healthcheck.BackendName)

		case _ = <-manager.dnsrefresh:
			// On a reload refresh existing and remove unused dns entries
			log.WithField("func", "core").Debug("dnsreload")
			go manager.dnsRefresh(cl)
		}
	}
}

func (manager *Manager) dnsRefresh(cl *cluster.Manager) {
	log := logging.For("core/cluster/dnsrefresh")
	// Get existing entries
	existingEntries := make(map[string][]string)
	cache := dns.GetCache()
	if _, ok := cache[config.Get().Cluster.Binding.Name]; ok {
		for domain := range cache[config.Get().Cluster.Binding.Name].Domains {
			for _, record := range cache[config.Get().Cluster.Binding.Name].Domains[domain].Records {
				if record.Local == false {
					existingEntries[domain] = append(existingEntries[domain], record.Name)
				}
			}
		}
	}

	for poolName := range config.Get().Loadbalancer.Pools {
		for backendName, backend := range config.Get().Loadbalancer.Pools[poolName].Backends {
			for i := len(existingEntries[backend.DNSEntry.Domain]) - 1; i >= 0; i-- {
				if existingEntries[backend.DNSEntry.Domain][i] == backend.DNSEntry.HostName {
					existingEntries[backend.DNSEntry.Domain] = append(existingEntries[backend.DNSEntry.Domain][:i], existingEntries[backend.DNSEntry.Domain][i+1:]...)
				}
			}
			manager.sendDNSUpdate(cl, poolName, backendName)
		}
	}

	for domain, existingRecords := range existingEntries {
		for _, existingRecord := range existingRecords {
			log.Infof("Deleting unused DNS entry: %s %s\n", domain, existingRecord)
			dnsremove := &config.ClusterPacketGlobalDNSRemove{
				ClusterNode: config.Get().Cluster.Binding.Name,
				Domain:      domain,
				Hostname:    existingRecord,
			}

			// Send removal request to local dns manager
			manager.dnsremove <- dnsremove

			// Send removal to cluster for other nodes to remove this entry
			cl.ToCluster <- dnsremove
		}
	}
}

// sendDNSUpdate sends dns update to local node, and to the cluster nodes
func (manager *Manager) sendDNSUpdate(cl *cluster.Manager, poolName string, backendName string) error {

	if _, ok := config.Get().Loadbalancer.Pools[poolName]; !ok {
		return fmt.Errorf("Pool no longer exists, discarding dns update")
	}

	if _, ok := config.Get().Loadbalancer.Pools[poolName].Backends[backendName]; !ok {
		return fmt.Errorf("Backend of pool no longer exists, discarding dns update")
	}

	backend := config.Get().Loadbalancer.Pools[poolName].Backends[backendName]
	// - send DNS updates to local dns servers
	dnsupdate := &config.ClusterPacketGlobalDNSUpdate{
		ClusterNode: config.Get().Cluster.Binding.Name,
		PoolName:    poolName,
		BackendName: backendName,
		DNSEntry:    backend.DNSEntry,
		BalanceMode: backend.BalanceMode,
		BackendUUID: backend.UUID,
		Status:      getLocalNodeStatus(poolName, backendName),
	}

	manager.dnsupdates <- dnsupdate
	// - send DNS update to cluster nodes
	go clusterDNSUpdateBroadcast(cl, config.Get().Cluster.Binding.Name, poolName, backendName, backend.DNSEntry, backend.BalanceMode, backend.UUID)
	return nil
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
		BackendNode:     proxy.BackendNode{IP: node.IP, Port: node.Port, Hostname: node.Hostname, MaxConnections: node.MaxConnections, LocalNetwork: node.LocalNetwork, Preference: node.Preference, Status: node.Status},
		BackendNodeUUID: node.UUID,
	}
	if config.Get().Settings.EnableProxy == YES {
		clog := log.WithField("pool", proxyupdate.PoolName).WithField("backend", proxyupdate.BackendName).WithField("uuid", proxyupdate.BackendNodeUUID).WithField("status", node.Status).WithField("ip", node.IP).WithField("port", node.Port)

		switch node.Status {
		case healthcheck.Online:
			clog.Warnf("Set proxy to Online")
			manager.addProxyBackend <- proxyupdate // add or update
		case healthcheck.Maintenance:
			clog.Warnf("Set proxy to Maintenance")
			manager.addProxyBackend <- proxyupdate // add or update
		default:
			clog.Warnf("Remove proxy due to offline")
			manager.removeProxyBackend <- proxyupdate
		}
	}
}

// InitializeCluster sets up the cluster, starts it, and starts the client
func (manager *Manager) InitializeCluster() {
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

	go writeClusterLog(cl)
	go manager.ClusterClient(cl)
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
			log.WithField("oldstatus", n.Status).WithField("status", node.Status).Debug("Update of existing node")
			// We already know this node, only update its status and error
			config.UpdateBackendNode(pool, backend, nid, node.Status, node.Errors)
			return
		}
	}

	log.WithField("status", node.Status).Debug("Update of new node")
	// It is a new Node, add it
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
						n.Status = healthcheck.Offline
						go manager.updateProxyBackendNode(pid, bid, *n)
						// Remove node from our config
						go config.RemoveBackendNodeUUID(pid, bid, n.UUID)
					}
				}
			}
		}
	}
}

func getLocalNodeStatus(poolName, backendName string) healthcheck.Status {
	var online = 0
	var maintenance = 0
	config.RLock()
	defer config.RUnlock()
	for _, node := range config.GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
		if node.Status == healthcheck.Online && node.ClusterName == config.GetNoLock().Cluster.Binding.Name {
			online++
		}
		if node.Status == healthcheck.Maintenance && node.ClusterName == config.GetNoLock().Cluster.Binding.Name {
			maintenance++
		}
	}
	// if any node is online, we are online
	if online > 0 {
		return healthcheck.Online
	}

	// if we are not online, but we have nodes in maintenance, then status is maintenance
	if maintenance > 0 {
		return healthcheck.Maintenance
	}

	// if no nodes are online or in maintenance, we are offline
	return healthcheck.Offline
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
	status := getLocalNodeStatus(poolName, backendName)
	dnsupdate := config.ClusterPacketGlobalDNSUpdate{
		ClusterNode: clusterNode,
		PoolName:    poolName,
		BackendName: backendName,
		DNSEntry:    dnsEntry,
		BalanceMode: balanceMode,
		BackendUUID: backendUUID,
		Status:      status,
	}
	cl.ToCluster <- dnsupdate
}

func clusterDNSUpdateBroadcast(cl *cluster.Manager, clusterNode string, poolName string, backendName string, dnsEntry config.DNSEntry, balanceMode config.BalanceMode, backendUUID string) {
	status := getLocalNodeStatus(poolName, backendName)
	dnsupdate := config.ClusterPacketGlobalDNSUpdate{
		ClusterNode: clusterNode,
		PoolName:    poolName,
		BackendName: backendName,
		DNSEntry:    dnsEntry,
		BalanceMode: balanceMode,
		BackendUUID: backendUUID,
		Status:      status,
	}
	cl.ToCluster <- dnsupdate
}
