package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// Backend is a backend where the proxy can connect to
type Backend struct {
	sync            *sync.RWMutex
	UUID            string
	BalanceMode     string
	ConnectMode     string
	InboundACL      ACLS
	OutboundACL     ACLS
	InboundRule     []string
	OutboundRule    []string
	Statistics      *balancer.Statistics
	Nodes           []*BackendNode
	Hostname        []string
	Fallback        string
	Uptime          time.Time
	ErrorPage       ErrorPage
	MaintenancePage ErrorPage
}

// NewBackend creates a new backend
func NewBackend(uuid string, balancemode string, connectmode string, hostname []string, maxconnections int, errorPage ErrorPage, maintenancePage ErrorPage) *Backend {
	b := &Backend{
		sync:            &sync.RWMutex{},
		UUID:            uuid,
		BalanceMode:     balancemode,
		ConnectMode:     connectmode,
		Hostname:        hostname,
		Statistics:      balancer.NewStatistics(uuid, maxconnections),
		Uptime:          time.Now(),
		ErrorPage:       errorPage,
		MaintenancePage: maintenancePage,
	}
	return b
}

// AddBackendNode adds a backend to the listener
func (b *Backend) AddBackendNode(n *BackendNode) {
	b.sync.Lock()
	defer b.sync.Unlock()
	// Clear statistics for other nodes in order to reset Balancing
	for _, node := range b.Nodes {
		node.Statistics.Reset()
	}
	// Add the new node
	b.Nodes = append(b.Nodes, n)
}

func remove(slice []*BackendNode, s int) []*BackendNode {
	return append(slice[:s], slice[s+1:]...)
}

// UpdateBackendNode update a backend node with a new status
func (b *Backend) UpdateBackendNode(nodeid int, status healthcheck.Status) {
	b.sync.Lock()
	defer b.sync.Unlock()
	if err := b.Nodes[nodeid]; err != nil {
		b.Nodes[nodeid].Status = status
	}
}

// RemoveBackendNode remove a backend node from the listener
func (b *Backend) RemoveBackendNode(nodeid int) {
	b.sync.Lock()
	defer b.sync.Unlock()
	nodes := remove(b.Nodes, nodeid)
	b.Nodes = nodes
	// clear statistics on removal
	for _, node := range b.Nodes {
		node.Statistics.Reset()
	}
}

// RemoveNodeByID remove backend node by ID
func (b *Backend) RemoveNodeByID(uuid string) error {
	for id, node := range b.Nodes {
		if node.UUID == uuid {
			b.RemoveBackendNode(id)
			return nil
		}
	}

	return fmt.Errorf("Unable to find node with uuid:%s", uuid)
}

// GetBackendsUUID Return backend node by ID
func (b *Backend) GetBackendsUUID() (n []string, err error) {
	b.sync.RLock()
	defer b.sync.RUnlock()
	for _, node := range b.Nodes {
		n = append(n, node.UUID)
	}

	return n, fmt.Errorf("Unable to find any nodes for backend: %s", b.UUID)
}

// GetBackendNodeByID Return backend node by ID
func (b *Backend) GetBackendNodeByID(uuid string) (*BackendNode, error) {
	b.sync.RLock()
	defer b.sync.RUnlock()
	for _, node := range b.Nodes {
		if node.UUID == uuid {
			return node, nil
		}
	}

	return nil, fmt.Errorf("Unable to find node with uuid:%s", uuid)
}

// GetBackend Return the first backend
func (b *Backend) GetBackend() (*BackendNode, error) {
	b.sync.RLock()
	defer b.sync.RUnlock()
	for _, node := range b.Nodes {
		return node, nil
	}

	return nil, fmt.Errorf("Unable to find a backend node")
}

// GetBackendNodeBalanced returns a single backend node, based on balancer proto
func (b *Backend) GetBackendNodeBalanced(backendpool, ip, sticky, balancemode string) (*BackendNode, healthcheck.Status, error) {
	b.sync.RLock()
	defer b.sync.RUnlock()
	log := logging.For("Proxy/GetBackendNodeBalanced").WithField("pool", backendpool).WithField("clientip", ip).WithField("sticky", sticky).WithField("mode", balancemode)
	log.Debug("Getting node from proxy backend")

	var onlineNodes []*BackendNode

	for _, n := range b.Nodes {
		if n.Status == healthcheck.Online {
			onlineNodes = append(onlineNodes, n)
		}
	}

	switch len(onlineNodes) {
	case 0: // return error of no nodes
		if len(b.Nodes) > 0 { // 0 online, but there are nodes. so all nodes are in maintenance
			return &BackendNode{}, healthcheck.Maintenance, fmt.Errorf("All backend nodes are in Maintenance in backend %s", backendpool)
		}

		return &BackendNode{}, healthcheck.Offline, fmt.Errorf("Unable to find a node in backend %s", backendpool)

	case 1: // return node if there is only 1 present
		return onlineNodes[0], healthcheck.Online, nil

	default: // balance across N Nodes
		stats := BackendNodeStats(onlineNodes)
		nodes, err := balancer.MultiSort(stats, ip, sticky, balancemode)
		if err != nil {
			return &BackendNode{}, healthcheck.Offline, fmt.Errorf("Unable to parse balance mode %s for backend %s, err: %s", balancemode, backendpool, err)
		}

		for order, node := range nodes {
			log.WithField("order", order).WithField("uuid", node.UUID).WithField("preference", node.Preference).Debug("Online node found")
		}

		node, err := b.GetBackendNodeByID(nodes[0].UUID)
		log.WithField("ip", node.IP).WithField("port", node.Port).WithField("uuid", node.UUID).Debug("Returning node for client")
		if err != nil {
			return &BackendNode{}, healthcheck.Offline, err
		}

		return node, healthcheck.Online, nil
	}

}

// BackendNodeStats gets statistics for backend nodes
func BackendNodeStats(n []*BackendNode) []balancer.Statistics {
	var s []balancer.Statistics
	for _, node := range n {
		s = append(s, *node.Statistics)
	}

	return s
}

// SetACL adds ACLs to the backend
func (b *Backend) SetACL(direction string, acl []ACL) {
	b.sync.Lock()
	defer b.sync.Unlock()
	switch direction {
	case "in":
		b.InboundACL = acl

	case "out":
		b.OutboundACL = acl
	}
}

// SetRules adds Rules to the backend
func (b *Backend) SetRules(direction string, rule []string) {
	b.sync.Lock()
	defer b.sync.Unlock()
	switch direction {
	case "in":
		b.InboundRule = rule

	case "out":
		b.OutboundRule = rule
	}
}

// ClearStats clears the statistics of all nodes of a backend
func (b *Backend) ClearStats() {
	log := logging.For("Proxy/GetBackendNodeBalanced")
	b.sync.Lock()
	defer b.sync.Unlock()
	for _, node := range b.Nodes {
		node.Statistics.Reset()
	}

	log.Debug("Cleared proxy stats")

}

// LoadErrorPage preloads the error page
func (b *Backend) LoadErrorPage(e ErrorPage) error {
	b.ErrorPage = e
	return b.ErrorPage.load()
}
