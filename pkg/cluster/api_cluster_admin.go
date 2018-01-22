package cluster

import (
	"net/http"
	"strings"
)

type apiClusterAdminHandler struct {
	manager *Manager
}

/*
	Cluster Admin options:
	  request in format: /api/cluster/[manager]/admin/[node]/action
			post data -> additional data in json format

		internal options:
	 	reconnect - reconnect to a node (disconnect, as it reconnects on timeout)
	 	admindown - disconnect a node, and do not reconnect for duration
	 	reload - reload config - passed on to client application
*/

func (h apiClusterAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(strings.TrimLeft(r.URL.Path, "/api/v1/cluster/"), "/")
	if len(path) != 4 {
		apiWriteData(w, 501, apiMessage{Success: false, Data: "Unknown request parameters"})
		return
	}
	node, action := path[2], path[3]
	h.manager.internalMessage <- internalMessage{Type: "api" + action, Node: node}
	apiWriteData(w, 200, apiMessage{Success: true, Data: action + " OK"})
}
