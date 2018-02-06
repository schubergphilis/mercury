package core

import (
	"fmt"
	"net/http"
	"strings"
)

// Public API
type apiHealthCheckPublicHandler struct {
	manager *Manager
}

// Public API returns the filtered json of the workers
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

// Private API returns the unfiltered json, or executes commands
func (h apiHealthCheckAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		path := strings.Split(r.RequestURI, "/")
		if len(path) < 5 {
			apiWriteData(w, 405, apiMessage{Success: false, Data: fmt.Errorf("invalid request")})
		}

		fmt.Printf("URL: %+v", path)
		data, err := h.manager.healthManager.JSONAuthorized(path[5])
		if err != nil {
			apiWriteData(w, 501, apiMessage{Success: false, Data: err.Error()})
			return
		}
		apiWriteJSONData(w, http.StatusOK, apiMessage{Success: true, Data: string(data)})
	case "POST":
	}

}
