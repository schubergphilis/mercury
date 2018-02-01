package core

import (
	"fmt"
	"log"
	"net/http"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/internal/web"
)

// web interface for healtheck
type webHealthCheckHandler struct {
	title     string
	templates []string
}

func (h webHealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authenticated, username, err := authenticateUser(r)

	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	title := fmt.Sprintf("Mercury %s - Backend Details %s", clusternode, backend)
	page := newPage(title, r.RequestURI)

	templateNames := []string{"backenddetails.tmpl", "header.tmpl", "footer.tmpl"}
	backenddetailsTemplate, err := web.LoadTemplates("static", templateNames)
	if err != nil {
		log.WithField("error", err).Warn("Error loading templates")
	}

	data := struct {
		Pool        config.LoadbalancePool
		Backend     config.BackendPool
		PoolName    string
		BackendName string
		Page        web.Page
	}{poolDetails, backendDetails, pool, backend, *page}

	err = backenddetailsTemplate.ExecuteTemplate(w, "backenddetails", data)
	if err != nil {
		log.WithField("error", err).Warn("Error executing template")
	}

}
