package cluster

import (
	"net/http"
)

type apiClusterHandler struct {
	manager *Manager
}

func (h apiClusterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	managers.Lock()
	defer managers.Unlock()
	apiWriteData(w, http.StatusOK, apiMessage{Success: true, Data: managers.manager})
}
