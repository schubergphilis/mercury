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
			apiWriteData(w, 405, apiMessage{Success: false, Error: "invalid request"})
		}

		data, err := h.manager.healthManager.JSONAuthorized(path[5])
		if err != nil {
			apiWriteData(w, 501, apiMessage{Success: false, Error: err.Error()})
			return
		}
		apiWriteJSONData(w, http.StatusOK, apiMessage{Success: true, Data: string(data)})
	case "POST":
		//                             1   2  3            4     5    6      7
		// expect a url in the format: api v1 healthchecks admin UUID ACTION STATUS
		// where action is the type to change, and status to what we set it
		path := strings.Split(r.RequestURI, "/")

		if len(path) < 7 {
			apiWriteData(w, 405, apiMessage{Success: false, Error: "invalid request"})
			return
		}

		switch path[6] {
		case "status":
			err := h.manager.healthManager.SetStatus(path[5], path[7])
			if err != nil {
				apiWriteData(w, 501, apiMessage{Success: false, Error: err.Error()})
				return
			}
			apiWriteData(w, 200, apiMessage{Success: true})
		}
		apiWriteData(w, 405, apiMessage{Success: false, Error: fmt.Sprintf("unknown action: %s", path[6])})

	}

}
