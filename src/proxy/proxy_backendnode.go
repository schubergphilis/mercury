package proxy

import (
	"time"

	"github.com/schubergphilis/mercury/src/balancer"
)

// BackendNode is a backendnode where the proxy can connect to
type BackendNode struct {
	UUID           string
	IP             string
	Hostname       string
	Port           int
	Statistics     *balancer.Statistics
	Uptime         time.Time
	MaxConnections int
	Preference     int
	LocalTopology  string   `json:"local_topology" toml:"local_topology"` // overrides localnetwork
	LocalNetwork   []string `json:"local_network" toml:"local_network"`   // used for topology based loadbalancing

}

// NewBackendNode creates a new node for a proxy backend
func NewBackendNode(UUID string, IP string, hostname string, port int, maxconnections int, topology []string, preference int) *BackendNode {
	b := &BackendNode{
		UUID:       UUID,
		IP:         IP,
		Hostname:   hostname,
		Port:       port,
		Uptime:     time.Now(),
		Statistics: balancer.NewStatistics(UUID, maxconnections),
	}
	b.Statistics.Topology = topology
	b.Statistics.Preference = preference
	b.LocalNetwork = topology
	b.Preference = preference
	return b
}

// Name returns the node name, either hostname or ip
func (a *BackendNode) Name() string {
	if a.Hostname != "" {
		return a.Hostname
	}
	return a.IP
}
