package cluster

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
)

var (
	// ChannelBufferSize the size of the channel buffer
	ChannelBufferSize = 100
)

// InternalMessage is used for internal communication within the cluster
type internalMessage struct {
	Type  string `json:"type"`
	Node  string `json:"node"`
	Error string `json:"error"`
}

// NodeMessage is used for sending private messages between cluster nodes
type NodeMessage struct {
	Node    string      // node to send message to
	Message interface{} // message to send to node
}

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name             string               // name of our cluster node
	authKey          string               // authentication key
	settings         Settings             // adjustable settings
	listener         net.Listener         // our listener
	connectedNodes   *connectionPool      // the list of connected nodes and their sockets
	configuredNodes  map[string]Node      // details of the remote cluster nodes
	newSocket        chan net.Conn        // new clients connecting
	internalMessage  chan internalMessage // internally sent messages within the cluster
	apiRequest       chan APIRequest      // API sent messages to the cluster from the API
	incommingPackets chan Packet          // packets sent to packet manager
	quit             chan bool            // signals exit of listener
	FromCluster      chan Packet          // data received from cluster
	FromClusterAPI   chan APIRequest      // data received from cluster via API interface
	ToCluster        chan interface{}     // data send to cluster
	ToNode           chan NodeMessage     // data send to specific node
	Log              chan string          // logging messages go here
	NodeJoin         chan string          // returns string of the node joining
	NodeLeave        chan string          // returns string of the node leaving
	QuorumState      chan bool            // returns the current quorum state
	useTLS           bool                 // wether or not to use tls
	addr             string               // binding addr
	tls              *tls.Config          // tls config for binding addr
}

var managers = struct {
	sync.RWMutex
	manager       []string
	clusterAPISet bool
}{}

// NewManager creates a new cluster manager
func NewManager(name, authKey string) *Manager {
	m := &Manager{
		name:             name,
		authKey:          authKey,
		settings:         defaultSetting(),
		configuredNodes:  make(map[string]Node),
		connectedNodes:   newConnectionPool(),
		newSocket:        make(chan net.Conn),
		internalMessage:  make(chan internalMessage, 100),
		apiRequest:       make(chan APIRequest, 100),
		incommingPackets: make(chan Packet, 100),
		quit:             make(chan bool),
		FromCluster:      make(chan Packet, ChannelBufferSize),
		FromClusterAPI:   make(chan APIRequest, ChannelBufferSize),
		ToCluster:        make(chan interface{}, ChannelBufferSize),
		ToNode:           make(chan NodeMessage, 100),
		Log:              make(chan string, ChannelBufferSize),
		NodeJoin:         make(chan string, 10),
		NodeLeave:        make(chan string, 10),
		QuorumState:      make(chan bool, 10),
	}
	addManager(m.name)
	if APIEnabled {
		m.addClusterAPI()
	}
	return m
}

func addManager(name string) {
	managers.Lock()
	defer managers.Unlock()
	managers.manager = append(managers.manager, name)
}

func removeManager(name string) {
	managers.Lock()
	defer managers.Unlock()
	var new []string
	for _, mgr := range managers.manager {
		if mgr != name {
			new = append(new, mgr)
		}
	}
	managers.manager = new
}

func (m *Manager) addClusterAPI() {
	managers.Lock()
	defer managers.Unlock()

	http.Handle("/api/v1/cluster/"+m.name+"/admin/", authenticate(apiClusterAdminHandler{manager: m}, m.authKey))
	http.Handle("/api/v1/cluster/"+m.name, apiClusterPublicHandler{manager: m})
	if managers.clusterAPISet == false {
		http.Handle("/api/v1/cluster", apiClusterHandler{})
		managers.clusterAPISet = true
	}
}

// ListenAndServeTLS starts the TLS listener and serves connections to clients
func (m *Manager) ListenAndServeTLS(addr string, tlsConfig *tls.Config) (err error) {
	m.log("%s Starting TLS listener on %s", m.name, addr)
	s := newServer(addr, tlsConfig)
	m.listener, err = s.Listen()
	if err == nil {
		m.start(s, tlsConfig)
	}
	return
}

// ListenAndServe starts the listener and serves connections to clients
func (m *Manager) ListenAndServe(addr string) (err error) {
	m.log("%s Starting listener on %s", m.name, addr)
	s := newServer(addr, &tls.Config{})
	m.listener, err = s.Listen()
	if err == nil {
		m.start(s, &tls.Config{})
	}
	return
}

func (m *Manager) start(s *server, tlsConfig *tls.Config) {
	go m.handleIncommingConnections()         // handles incommin socket connections
	go m.handleOutgoingConnections(tlsConfig) // creates connections to remote nodes
	go m.handlePackets()                      // handles all incomming packets
	go s.Serve(m.newSocket, m.quit)           // accepts new connections and passes them on to the manager
	m.log("%s Cluster quorum state: %t", m.name, m.quorum())
	select {
	case m.QuorumState <- m.quorum(): // quorum update to client application
	default:
	}
	return
}

// Shutdown stops the cluster node
func (m *Manager) Shutdown() {
	m.log("%s Stopping listener on %s", m.name, m.listener.Addr())
	// write exit message to remote cluster
	packet, _ := m.newPacket(&packetNodeShutdown{})
	m.connectedNodes.writeAll(packet)
	// close all connected nodes
	m.connectedNodes.closeAll()
	close(m.quit)
	m.listener.Close()
	removeManager(m.name)
}

// quorum returns quorum state based on configured vs connected nodes
func (m *Manager) quorum() bool {
	m.RLock()
	defer m.RUnlock()
	switch len(m.configuredNodes) {
	case 0:
		return true // single node
	case 1:
		return true // 2 cluster node, we don't send quorum loss, as that would nullify the additional node
	default:
		return float64(len(m.configuredNodes)+1)/2 < float64(m.connectedNodes.count()+1) // +1 to add our selves
	}
}

func (m *Manager) updateQuorum() {
	m.log("%s Cluster quorum state: %t", m.name, m.quorum())
	select {
	case m.QuorumState <- m.quorum(): // quorum update to client application
	default:
	}
}

// AddNode adds a cluster node to the cluster to be connected to
func (m *Manager) AddNode(nodeName, nodeAddr string) {
	m.Lock()
	defer m.Unlock()
	m.configuredNodes[nodeName] = Node{
		name:      nodeName,
		addr:      nodeAddr,
		statusStr: StatusOffline,
	}
	select {
	case m.internalMessage <- internalMessage{Type: "nodeadd", Node: nodeName}:
	default:
	}
}

// NodesConfigured returns all nodes configured to be part of the cluster
func (m *Manager) NodesConfigured() map[string]bool {
	node := make(map[string]bool)
	m.RLock()
	defer m.RUnlock()
	for name := range m.configuredNodes {
		node[name] = true
	}

	return node
}

// NodeConfigured returns true or false if a node is configured in the manager
func (m *Manager) NodeConfigured(nodeName string) bool {
	m.RLock()
	defer m.RUnlock()
	if _, ok := m.configuredNodes[nodeName]; ok {
		return true
	}

	return false
}

// RemoveNode remove a cluster node from the list of servers to connect to, and close its connections
func (m *Manager) RemoveNode(nodeName string) {
	m.Lock()
	defer m.Unlock()
	m.log("%s is removing node %s", m.name, nodeName)
	if _, ok := m.configuredNodes[nodeName]; ok {
		delete(m.configuredNodes, nodeName)
	}

	select {
	case m.internalMessage <- internalMessage{Type: "noderemove", Node: nodeName}:
	default:
	}

	m.connectedNodes.close(nodeName)
}

func (m *Manager) getConfiguredNodes() (nodes []Node) {
	m.RLock()
	defer m.RUnlock()
	for _, node := range m.configuredNodes {
		nodes = append(nodes, node)
	}

	return
}

// StateDump dumps the current state of the cluster to the log
func (m *Manager) StateDump() {
	m.log("cluster state:")
	for _, node := range m.configuredNodes {
		m.log("configured nodes: %+v", node)
	}

	for _, node := range m.connectedNodes.nodes {
		m.log("connected nodes: %+v", node)
	}
}

// Name returns the name of a cluster node
func (m *Manager) Name() string {
	return m.name
}

func (m *Manager) ReceivedLogging() chan string {
	return m.Log
}

func (m *Manager) ReceivedNodeJoin() chan string {
	return m.NodeJoin
}

func (m *Manager) ReceivedNodeLeave() chan string {
	return m.NodeLeave
}

func (m *Manager) ReceivedFromCluster() chan Packet {
	return m.FromCluster
}

func (m *Manager) ReceivedFromClusterAPI() chan APIRequest {
	return m.FromClusterAPI
}

func (m *Manager) SendToCluster() chan interface{} {
	return m.ToCluster
}

func (m *Manager) SendToNode() chan NodeMessage {
	return m.ToNode
}

func New() *Manager {
	m := &Manager{
		settings:         defaultSetting(),
		configuredNodes:  make(map[string]Node),
		connectedNodes:   newConnectionPool(),
		newSocket:        make(chan net.Conn),
		internalMessage:  make(chan internalMessage, 100),
		apiRequest:       make(chan APIRequest, 100),
		incommingPackets: make(chan Packet, 100),
		quit:             make(chan bool),
		FromCluster:      make(chan Packet, ChannelBufferSize),
		FromClusterAPI:   make(chan APIRequest, ChannelBufferSize),
		ToCluster:        make(chan interface{}, ChannelBufferSize),
		ToNode:           make(chan NodeMessage, 100),
		Log:              make(chan string, ChannelBufferSize),
		NodeJoin:         make(chan string, 10),
		NodeLeave:        make(chan string, 10),
		QuorumState:      make(chan bool, 10),
	}
	return m
}

func (m *Manager) WithName(name string) {
	m.name = name
}

func (m *Manager) WithKey(key string) {
	m.authKey = key
}

func (m *Manager) WithAddr(addr string) {
	m.addr = addr
}

func (m *Manager) WithTLS(tls *tls.Config) {
	m.tls = tls
}

func (m *Manager) Start() {
	m.quit = make(chan bool)

	addManager(m.name)
	/*if APIEnabled {
		m.addClusterAPI()
	}*/

	if m.tls == nil {
		err := m.ListenAndServe(m.addr)
		if err != nil {

		}
	} else {
		err := m.ListenAndServeTLS(m.addr, m.tls)
		if err != nil {

		}
	}
}

func (m *Manager) Stop() {
	m.Shutdown()
}
