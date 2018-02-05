package core

import (
	"net/http"
)

// Public API
type apiHealthCheckPublicHandler struct {
	manager *Manager
}

func (h apiHealthCheckPublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h.manager.healthManager.JSON()
	if err != nil {
		apiWriteData(w, 501, apiMessage{Success: false, Data: err.Error()})
		return
	}
	apiWriteJSONData(w, http.StatusOK, apiMessage{Success: true, Data: string(data)})
}

// Authorized personel only
type apiHealthCheckAdminHandler struct {
	manager *Manager
}

func (h apiHealthCheckAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h.manager.healthManager.JSON()
	if err != nil {
		apiWriteData(w, 501, apiMessage{Success: false, Data: err.Error()})
		return
	}
	apiWriteJSONData(w, http.StatusOK, apiMessage{Success: true, Data: string(data)})

}
