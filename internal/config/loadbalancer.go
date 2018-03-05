package config

import (
	"fmt"
	"strings"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/proxy"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

// Loadbalancer tree
type Loadbalancer struct {
	Settings LoadbalancerSettings       `toml:"settings" json:"settings"`
	Pools    map[string]LoadbalancePool `toml:"pools" json:"pools"`
	Networks map[string]Network         `toml:"networks" json:"networks"`
}

// LoadbalancerSettings contains a list of global application settings
type LoadbalancerSettings struct {
	DefaultLoadBalanceMethod string `toml:"default_balance_method"` // "roundrobin, topology, preference"
}

// LoadbalancePool contains a pool to loadbalance
type LoadbalancePool struct {
	Name            string                    `json:"name" toml:"name"`                       // pool name
	Listener        LoadbalancerListener      `json:"listener" toml:"listener"`               // listener settings
	HealthChecks    []healthcheck.HealthCheck `json:"healthcheck" toml:"healthcheck"`         // healthcheck to perform for VIP (e.g. internet connectivity)
	HealthCheckMode string                    `json:"healthcheckmode" toml:"healthcheckmode"` // healthcheck mode (all / any)
	Backends        map[string]BackendPool    `json:"backends" toml:"backends"`               // backend pools
	Online          bool                      `json:"online" toml:"online"`                   // is pool online?
	Stats           *balancer.Statistics      `json:"stats" toml:"stats" yaml:"-"`            // statis
	InboundACL      []proxy.ACL               `json:"inboundacls" toml:"inboundacls"`         // acls applied on incomming connections to backend
	OutboundACL     []proxy.ACL               `json:"outboundacls" toml:"outboundacls"`       // acl's applied on outgoing connections to client
	ErrorPage       proxy.ErrorPage           `json:"errorpage" toml:"errorpage"`             // alternative error page to show
}

// LoadbalancerListener is a listener for the loadbalancer
type LoadbalancerListener struct {
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
type BackendNode struct {
	*proxy.BackendNode
	Status      healthcheck.Status `json:"status" toml:"status" yaml:"-"`
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
type BackendPool struct {
	Nodes           []*BackendNode            `json:"nodes" toml:"nodes"`                     // backend nodes
	HealthChecks    []healthcheck.HealthCheck `json:"healthcheck" toml:"healthcheck"`         // healthcheck to perform on each backend node
	HealthCheckMode string                    `json:"healthcheckmode" toml:"healthcheckmode"` // healthcheck mode (all / any)
	DNSEntry        DNSEntry                  `json:"dnsentry" toml:"dnsentry"`               // glb dns entry for this backend
	Online          bool                      `json:"online" toml:"online"`                   // is backend pool online
	BalanceMode     BalanceMode               `json:"balance" toml:"balance"`                 // loadbalance method
	Stats           *balancer.Statistics      `json:"stats" toml:"stats" yaml:"-"`            // statistics
	ConnectMode     string                    `json:"connectmode" toml:"connectmode"`         // protocol to use when connecting to backend
	InboundACL      []proxy.ACL               `json:"inboundacls" toml:"inboundacls"`         // acl's to apply on requests sent to server
	OutboundACL     []proxy.ACL               `json:"outboundacls" toml:"outboundacls"`       // acl's to apply on replies to client
	HostNames       []string                  `json:"hostnames" toml:"hostnames"`             // hostnames requests we reply to on http
	UUID            string                    `json:"uuid" toml:"uuid"`                       // uuid of backend pool
	TLSConfig       tlsconfig.TLSConfig       `json:"tls" toml:"tls" yaml:"tls"`              // tls configuratuin
	Crossconnects   bool                      `json:"crossconnects" toml:"crossconnects"`     // allow cluster cross-connects (e.g. each server can connect to all backends)
	ErrorPage       proxy.ErrorPage           `json:"errorpage" toml:"errorpage"`             // alternative error page to show
}

// BalanceMode Which type of loadbalancing to use
type BalanceMode struct {
	Method        string   `json:"method" toml:"method"`                 // balance method for the backend
	LocalTopology string   `json:"local_topology" toml:"local_topology"` // overrides localnetwork
	ActivePassive string   `json:"active_passive" toml:"active_passive"` // active_passive only affects monitoring: when "yes" only alert if there are no nodes up
	Preference    int      `json:"preference" toml:"preference"`         // used for preference based loadbalancing
	LocalNetwork  []string `json:"local_network" toml:"local_network"`   // used for topology based loadbalancing
	ClusterNodes  int      `json:"clusternodes" toml:"clusternodes"`     // affects monitoring only: how many cluster nodes serve this backend
}

// Network Contains network information
type Network struct {
	CIDRs []string
}

// Name Return a identifyable name that includes the port
func (n BackendNode) Name() string {
	return fmt.Sprintf("%s_%d", n.SafeName(), n.Port)
}

// ServerName returns the node name, either hostname or ip
func (n BackendNode) ServerName() string {
	if n.Hostname != "" {
		return n.Hostname
	}
	return n.IP
}

// SafeName returns the node name without special characters
func (n BackendNode) SafeName() string {
	r := strings.NewReplacer(".", "_",
		"-", "_")
	return r.Replace(n.ServerName())
}

// UpdateStatus updates node status
func UpdateNodeStatus(poolName string, backendName string, nodeUUID string, status healthcheck.Status, err []string) {
	configLock.Lock()
	defer configLock.Unlock()
	if _, ok := config.Loadbalancer.Pools[poolName]; ok {
		if _, ok := config.Loadbalancer.Pools[poolName].Backends[backendName]; ok {
			for nid, node := range config.Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
				if node.UUID == nodeUUID {
					config.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nid].Status = status
					config.Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nid].Errors = err
				}
			}
		}
	}
}

// GetNodeByUUID returns the node based on UUID
func GetNodeByUUID(poolName string, backendName string, nodeUUID string) (BackendNode, error) {
	configLock.Lock()
	defer configLock.Unlock()
	if _, ok := config.Loadbalancer.Pools[poolName]; ok {
		if _, ok := config.Loadbalancer.Pools[poolName].Backends[backendName]; ok {
			for _, node := range config.Loadbalancer.Pools[poolName].Backends[backendName].Nodes {
				if node.UUID == nodeUUID {
					return *node, nil
				}
			}
		}
	}
	return BackendNode{}, fmt.Errorf("node UUID not found:%s", nodeUUID)
}

// PoolsCopy returns a copy of pool
func (l *Loadbalancer) PoolsCopy() map[string]LoadbalancePool {
	configLock.RLock()
	defer configLock.RUnlock()
	copy := make(map[string]LoadbalancePool, len(l.Pools))
	for poolname, pool := range l.Pools {
		copy[poolname] = pool
	}
	return copy
}

func removeBackendNodeID(s []*BackendNode, i int) []*BackendNode {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// RemoveBackendNodeUUID removed a backend node to a backend
func RemoveBackendNodeUUID(poolName string, backendName string, nodeUUID string) {
	Lock()
	defer Unlock()
	backend := GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName]
	for nid := len(GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName].Nodes) - 1; nid >= 0; nid-- {
		if backend.Nodes[nid].UUID == nodeUUID {
			nodes := removeBackendNodeID(backend.Nodes, nid)
			backend.Nodes = nodes
		}
	}
	GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName] = backend
}

// AddBackendNode adds a backend node to a backend
func AddBackendNode(poolName string, backendName string, node *BackendNode) {
	Lock()
	defer Unlock()
	if backend, ok := GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName]; ok {
		backend.Nodes = append(backend.Nodes, node)
		GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName] = backend
	}
}

// UpdateBackendNode updates status of online and error of backend node
func UpdateBackendNode(poolName string, backendName string, nodeID int, status healthcheck.Status, err []string) {
	Lock()
	defer Unlock()
	if _, ok := GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName]; ok {
		GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nodeID].Status = status
		GetNoLock().Loadbalancer.Pools[poolName].Backends[backendName].Nodes[nodeID].Errors = err
	}
}
