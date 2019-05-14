package core

import (
	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury.v2/internal/models"
	"github.com/schubergphilis/mercury/pkg/proxy"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)
import "github.com/schubergphilis/mercury.v2/internal/logging"


// Loadbalancer tree
type LoadbalancerConfig struct {
	Settings LoadbalancerConfigSettings           `toml:"settings" json:"settings"`
	Pools    map[string]LoadbalancerConfigPool    `toml:"pools" json:"pools"`
	Networks map[string]LoadbalancerConfigNetwork `toml:"networks" json:"networks"`
}

// LoadbalancerSettings contains a list of global application settings
type LoadbalancerConfigSettings struct {
	DefaultLoadBalanceMethod string `toml:"default_balance_method"` // "roundrobin, topology, preference"
}

// LoadbalancePool contains a pool to loadbalance
type LoadbalancerConfigPool struct {
	Name            string                                   `json:"name" toml:"name"`                       // pool name
	Listener        LoadbalancerConfigListener               `json:"listener" toml:"listener"`               // listener settings
	Healthchecks    []models.Healthcheck                `json:"healthcheck" toml:"healthcheck"`         // healthcheck to perform for VIP (e.g. internet connectivity)
	HealthcheckMode string                                   `json:"healthcheckmode" toml:"healthcheckmode"` // healthcheck mode (all / any)
	Backends        map[string]LoadbalancerConfigBackendPool `json:"backends" toml:"backends"`               // backend pools
	Online          bool                                     `json:"online" toml:"online"`                   // is pool online?
	InboundACL      []proxy.ACL                              `json:"inboundacls" toml:"inboundacls"`         // acls applied on incomming connections to backend
	OutboundACL     []proxy.ACL                              `json:"outboundacls" toml:"outboundacls"`       // acl's applied on outgoing connections to client
	ErrorPage       proxy.ErrorPage                          `json:"errorpage" toml:"errorpage"`             // alternative error page to show
	MaintenancePage proxy.ErrorPage                          `json:"maintenancepage" toml:"maintenancepage"` // alternative maintenance page to show
}

// LoadbalancerListener is a listener for the loadbalancer
type LoadbalancerConfigListener struct {
	IP             string               `json:"ip" toml:"ip"`                                               // ip of listener
	Port           int                  `json:"port" toml:"port"`                                           // port of listener
	Hostname       string               `json:"hostname" toml:"hostname"`                                   // hostname of listener
	Interface      string               `json:"interface" toml:"interface"`                                 // interface of listener
	Online         bool                 `json:"online" toml:"online"`                                       // is listener online?
	Mode           string               `json:"mode" toml:"mode"`                                           // listener protocol
	Stats          *balancer.Statistics `json:"stats" toml:"stats" yaml:"-"`                                // stats
	TLSConfig      tlsconfig.TLSConfig  `json:"tls" toml:"tls" yaml:"tls"`                                  // TLS config
	MaxConnections int                  `json:"maxconnections" toml:"maxconnections" yaml:"maxconnections"` // maximum connections allowed
	WriteTimeout   int                  `json:"writetimeout" toml:"writetimeout" yaml:"writetimeout"`       // write timeout on server reply to client
	ReadTimeout    int                  `json:"readtimeout" toml:"readtimeout" yaml:"readtimeout"`          // read timeout on client reply to server
	HTTPProto      int                  `json:"httpproto" toml:"httpproto" yaml:"httpproto"`                // force HTP protocol (1 = http/1.x 2 = http/2)
	OCSPStapling   string               `json:"ocspstapling" toml:"ocspstapling" yaml:"ocspstapling"`       // Enable/Disable OCSP Stapling
	//Error          string              `json:"error" toml:"error"` // error??? - not used
}

// BackendNode an cluster node to talk to
type LoadbalancerConfigBackendNode struct {
	*proxy.BackendNode
	Status      models.Status `json:"status" toml:"status" yaml:"-"`
	Errors      []string           `json:"error" toml:"error" yaml:"-"`
	ClusterName string             `json:"clustername" toml:"clustername" yaml:"-"`
}

// DNSEntry for GLB
type DNSEntry struct {
	HostName string `json:"hostname" toml:"hostname"`
	Domain   string `json:"domain" toml:"domain"`
	IP       string `json:"ip" toml:"ip"`
	IP6      string `json:"ip6" toml:"ip6"`
}

// BackendPool nodes and details
type LoadbalancerConfigBackendPool struct {
	Nodes           []*LoadbalancerConfigBackendNode `json:"nodes" toml:"nodes"`                     // backend nodes
	Healthchecks    []models.Healthcheck        `json:"healthchecks" toml:"healthchecks"`       // healthchecks to perform on each backend node
	HealthcheckMode string                           `json:"healthcheckmode" toml:"healthcheckmode"` // healthcheck mode (all / any)
	DNSEntry        DNSEntry                         `json:"dnsentry" toml:"dnsentry"`               // glb dns entry for this backend
	Online          bool                             `json:"online" toml:"online"`                   // is backend pool online
	BalanceMode     LoadbalancerConfigBalanceMode    `json:"balance" toml:"balance"`                 // loadbalance method
	Stats           *balancer.Statistics             `json:"stats" toml:"stats" yaml:"-"`            // statistics
	ConnectMode     string                           `json:"connectmode" toml:"connectmode"`         // protocol to use when connecting to backend
	InboundACL      []proxy.ACL                      `json:"inboundacls" toml:"inboundacls"`         // acl's to apply on requests sent to server
	OutboundACL     []proxy.ACL                      `json:"outboundacls" toml:"outboundacls"`       // acl's to apply on replies to client
	HostNames       []string                         `json:"hostnames" toml:"hostnames"`             // hostnames requests we reply to on http
	UUID            string                           `json:"uuid" toml:"uuid"`                       // uuid of backend pool
	TLSConfig       tlsconfig.TLSConfig              `json:"tls" toml:"tls" yaml:"tls"`              // tls configuratuin
	Crossconnects   bool                             `json:"crossconnects" toml:"crossconnects"`     // allow cluster cross-connects (e.g. each server can connect to all backends)
	ErrorPage       proxy.ErrorPage                  `json:"errorpage" toml:"errorpage"`             // alternative error page to show
	MaintenancePage proxy.ErrorPage                  `json:"maintenancepage" toml:"maintenancepage"` // alternative maintenance page to show
}

// BalanceMode Which type of loadbalancing to use
type LoadbalancerConfigBalanceMode struct {
	Method              string   `json:"method" toml:"method"`                               // balance method for the backend
	LocalTopology       string   `json:"local_topology" toml:"local_topology"`               // overrides localnetwork
	ActivePassive       string   `json:"active_passive" toml:"active_passive"`               // active_passive only affects monitoring: when "yes" only alert if there are no nodes up
	Preference          int      `json:"preference" toml:"preference"`                       // used for preference based loadbalancing
	LocalNetwork        []string `json:"local_network" toml:"local_network"`                 // used for topology based loadbalancing
	ClusterNodes        int      `json:"clusternodes" toml:"clusternodes"`                   // Depricated: affects monitoring only: how many cluster nodes serve this backend
	ServingClusterNodes int      `json:"serving_cluster_nodes" toml:"serving_cluster_nodes"` // affects monitoring only: how many cluster nodes serve this backend
	ServingBackendNodes int      `json:"serving_backend_nodes" toml:"serving_backend_nodes"` // affects monitoring only: how many backend nodes serve this backend
}

// Network Contains network information
type LoadbalancerConfigNetwork struct {
	CIDRs []string
}

type Loadbalancer struct{
  log logging.SimpleLogger
  config *LoadbalancerConfig
  quit   chan struct{}

  updateProxyStatistics chan struct{}
  updateHealthcheck chan struct{}
  refreshDNS chan struct{} // ???
}

func (l *Loadbalancer) LocalHandler() {
  for {
    select {
		case proxyStatistics := <-l.updateProxyStatistics:
      l.log.Debugf("update of proxy statistics", "packet", proxyStatistics)

      // received backend statistic update, send this to the cluster
			/*cgstats := &config.ClusterPacketGlbalDNSStatisticsUpdate{
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
			go clusterProxyStatsBroadcast(cl, cgstats)*/

		case healthcheck := <-l.updateHealthcheck:
      l.log.Debugf("update of healthcheck", "packet", healthcheck)
			// Local Healthcheck update received from local healthcheck workers
			// We only reach this update if a Healthcheck changed state (either up or down)
			// If Changed we need to:
			// - update our local config, so we are aware that a node is online/offline

			// Update node status in memory
			//config.UpdateNodeStatus(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID, healthcheck.ReportedStatus, healthcheck.ErrorMsg)

			// Get UUID of node
			// node, err := config.GetNodeByUUID(healthcheck.PoolName, healthcheck.BackendName, healthcheck.NodeUUID)
			// If we have proxy enabled send node update to proxy

			/*pool := config.Get().Loadbalancer.Pools[healthcheck.PoolName]
			if config.Get().Settings.EnableProxy == YES && pool.Listener.IP != "" {
				go clusterClearProxyStatistics(cl, healthcheck.PoolName, healthcheck.BackendName)
				go manager.updateProxyBackendNode(healthcheck.PoolName, healthcheck.BackendName, node)
			}*/

			// Send update to DNS
			//go manager.sendDNSUpdate(cl, healthcheck.PoolName, healthcheck.BackendName)

		case _ = <-l.refreshDNS:
			// On a reload refresh existing and remove unused dns entries
			//log.WithField("func", "core").Debug("dnsreload")
			//go manager.dnsRefresh(cl)
		}
  }
}
