package core

import (
	"fmt"
	"net/http"

	"github.com/schubergphilis/mercury/internal/web"
)

// web interface for healtheck
type webHealthCheckHandler struct {
	title         string
	templateFiles []string
	template      string
}

func (h webHealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authenticated, username, err := authenticateUser(r)

	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")

	page := newPage(h.title, r.RequestURI)
	backenddetailsTemplate, err := web.LoadTemplates("static", h.templateFiles)
	if err != nil {
		webWriteError(w, 500, fmt.Sprintf("unable to load template: %s", err.Error()))
		return
	}

	data := struct {
		Page          web.Page
		Authenticated bool
		Username      string
	}{*page, authenticated, username}

	err = backenddetailsTemplate.ExecuteTemplate(w, h.template, data)
	if err != nil {
		webWriteError(w, 500, fmt.Sprintf("unable to execute template: %s", err.Error()))
	}

}
