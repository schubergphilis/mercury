package cluster

import (
	"net/http"
	"time"
)

type apiClusterPublicHandler struct {
	manager *Manager
}

// APIClusterNode contains details of a node we might connect to used for the API
type APIClusterNode struct {
	Name     string        `json:"name"`
	Addr     string        `json:"addr"`
	Status   string        `json:"status"`
	Error    string        `json:"error"`
	JoinTime time.Time     `json:"jointime"`
	Lag      time.Duration `json:"lag"`
	Packets  int64         `json:"packets"`
}

// APIClusterNodeList contains a list of configured/connected nodes used for the API
type APIClusterNodeList struct {
	Nodes map[string]APIClusterNode `json:"nodes"`
}

func (h apiClusterPublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.manager.RLock()
	defer h.manager.RLock()
	var message = &APIClusterNodeList{
		Nodes: make(map[string]APIClusterNode),
	}

	for _, configured := range h.manager.configuredNodes {

		n := APIClusterNode{
			Name:   configured.name,
			Addr:   configured.addr,
			Status: configured.statusStr,
		}

		if active, ok := h.manager.connectedNodes.nodes[configured.name]; ok {
			n.JoinTime = active.joinTime
			n.Lag = active.lag
			n.Packets = active.packets
			n.Status = active.statusStr
			n.Error = active.errorStr
		}
		message.Nodes[configured.name] = n
	}
	apiWriteData(w, http.StatusOK, apiMessage{Success: true, Data: message})
}

func findActiveNode(n []*Node, name string) *Node {
	for _, node := range n {
		if node.name == name {
			return node
		}
	}
	return nil

}
