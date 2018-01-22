package core

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"

	rice "github.com/GeertJohan/go.rice"

	"github.com/schubergphilis/mercury/src/config"
	"github.com/schubergphilis/mercury/src/dns"
	"github.com/schubergphilis/mercury/src/logging"
	"github.com/schubergphilis/mercury/src/proxy"
	"github.com/schubergphilis/mercury/src/tlsconfig"
	"github.com/schubergphilis/mercury/src/web"
)

// FormattedDate date to readable time
func FormattedDate(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC822)
}

const applicationJSONHeader = "application/json"

// WebBackendDetails Provides a detail page for backends
func WebBackendDetails(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/backenddetails").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	r.URL.Query().Get("backend")
	//pool := "INTERNAL_VIP_LB"
	//backend := "ghostbox_lc"
	pool := r.URL.Query().Get("pool")
	backend := r.URL.Query().Get("backend")

	poolDetails := config.Get().Loadbalancer.Pools[pool]
	backendDetails := config.Get().Loadbalancer.Pools[pool].Backends[backend]
	/*
		switch r.Header.Get("Content-type") {
		case applicationJSONHeader:
			data, err := json.Marshal(clusternodes)
			if err != nil {
				fmt.Fprintf(w, "{ error:'%s' }", err)
			}
			fmt.Fprint(w, string(data))

		default:
	*/
	clusternode := config.Get().Cluster.Binding.Name
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
	/*
		}
	*/
}

// WebClusterStatus Provides a status page for Cluster status
func WebClusterStatus(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/clusterstatus").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	clusternode := config.Get().Cluster.Binding.Name
	title := fmt.Sprintf("Mercury %s - Cluster Status", clusternode)
	page := newPage(title, r.RequestURI)

	templateNames := []string{"cluster.tmpl", "header.tmpl", "footer.tmpl"}
	clusterTemplate, err := web.LoadTemplates("static", templateNames)
	if err != nil {
		log.Warnf("Error loading templates: %s", err)
	}

	data := struct {
		ClusterAPIPath string
		Page           web.Page
	}{"/api/v1/cluster/" + clusternode, *page}

	err = clusterTemplate.ExecuteTemplate(w, "cluster", data)
	if err != nil {
		log.WithField("error", err).Warn("Error executing template")
	}

}

// WebProxyStatus Provides a status page for Proxy service
func WebProxyStatus(w http.ResponseWriter, r *http.Request) {
	if pusher, ok := w.(http.Pusher); ok {
		// Push is supported.
		if err := pusher.Push("/static/logo32.png", nil); err != nil {
			log.Printf("Failed to push: %v", err)
		}
		/* type definetion now working with golang yet - so lets not push this one
		if err := pusher.Push("/static/mercury.css", nil); err != nil {
			log.Printf("Failed to push: %v", err)
		}*/
		if err := pusher.Push("/static/list.min.js", nil); err != nil {
			log.Printf("Failed to push: %v", err)
		}
		if err := pusher.Push("/static/jquery.min.js", nil); err != nil {
			log.Printf("Failed to push: %v", err)
		}
	}

	log := logging.For("core/proxystatus").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	switch r.Header.Get("Content-type") {
	case applicationJSONHeader:
		data, err := json.Marshal(proxies.pool)
		if err != nil {
			fmt.Fprintf(w, "{ error:'%s' }", err)
		}
		fmt.Fprint(w, string(data))

	default:
		clusternode := config.Get().Cluster.Binding.Name
		title := fmt.Sprintf("Mercury %s - Proxy Status", clusternode)
		page := newPage(title, r.RequestURI)

		templateNames := []string{"proxy.tmpl", "header.tmpl", "footer.tmpl"}
		proxyTemplate, err := web.LoadTemplates("static", templateNames)
		if err != nil {
			log.Warnf("Error loading templates: %s", err)
		}

		data := struct {
			Proxies map[string]*proxy.Listener
			Page    web.Page
		}{proxies.pool, *page}

		err = proxyTemplate.ExecuteTemplate(w, "proxy", data)
		if err != nil {
			log.WithField("error", err).Warn("Error executing template")
		}

	}
}

// WebBackendStatus Provides a status page for Backend Status
func WebBackendStatus(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/backendstatus").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	loadbalancer := config.Get().Loadbalancer
	switch r.Header.Get("Content-type") {
	case applicationJSONHeader:
		data, err := json.Marshal(loadbalancer)
		if err != nil {
			fmt.Fprintf(w, "{ error:'%s' }", err)
		}
		fmt.Fprint(w, string(data))

	default:
		clusternode := config.Get().Cluster.Binding.Name
		title := fmt.Sprintf("Mercury %s - Backend Status", clusternode)
		page := newPage(title, r.RequestURI)

		templateNames := []string{"backend.tmpl", "header.tmpl", "footer.tmpl"}
		backendTemplate, err := web.LoadTemplates("static", templateNames)
		if err != nil {
			log.Warnf("Error loading templates: %s", err)
		}

		data := struct {
			Loadbalancer config.Loadbalancer
			Page         web.Page
			ClusterNode  string
		}{loadbalancer, *page, config.Get().Cluster.Binding.Name}

		err = backendTemplate.ExecuteTemplate(w, "backend", data)
		if err != nil {
			log.WithField("error", err).Warn("Error executing template")
		}

	}
}

// WebGLBStatus Provides a status page for GLB
func WebGLBStatus(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/glbstatus").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	dnscache := dns.GetCache()
	switch r.Header.Get("Content-type") {
	case applicationJSONHeader:
		data, err := json.Marshal(dnscache)
		if err != nil {
			fmt.Fprintf(w, "{ error:'%s' }", err)
		}
		fmt.Fprint(w, string(data))

	default:
		clusternode := config.Get().Cluster.Binding.Name
		title := fmt.Sprintf("Mercury %s - GLB Status", clusternode)
		page := newPage(title, r.RequestURI)

		templateNames := []string{"glb.tmpl", "header.tmpl", "footer.tmpl"}
		backendTemplate, err := web.LoadTemplates("static", templateNames)
		if err != nil {
			log.Warnf("Error loading templates: %s", err)
		}

		data := struct {
			DNS  map[string]dns.Domains
			Page web.Page
		}{dnscache, *page}

		err = backendTemplate.ExecuteTemplate(w, "glb", data)
		if err != nil {
			log.WithField("error", err).Warn("Error executing template")
		}

	}
}

// WebLocalDNSStatus Provides a status page for GLB
func WebLocalDNSStatus(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/localdnsstatus").WithField("func", "web")
	w.Header().Add("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")
	dnscache := dns.GetCache()
	switch r.Header.Get("Content-type") {
	case applicationJSONHeader:
		data, err := json.Marshal(dnscache)
		if err != nil {
			fmt.Fprintf(w, "{ error:'%s' }", err)
		}
		fmt.Fprint(w, string(data))

	default:
		clusternode := config.Get().Cluster.Binding.Name
		title := fmt.Sprintf("Mercury %s - LocalDNS Status", clusternode)
		page := newPage(title, r.RequestURI)

		templateNames := []string{"localdns.tmpl", "header.tmpl", "footer.tmpl"}
		backendTemplate, err := web.LoadTemplates("static", templateNames)
		if err != nil {
			log.Warnf("Error loading templates: %s", err)
		}

		data := struct {
			DNS  map[string]dns.Domains
			Page web.Page
		}{dnscache, *page}

		err = backendTemplate.ExecuteTemplate(w, "glb", data)
		if err != nil {
			log.WithField("error", err).Warn("Error executing template")
		}

	}
}

type processInfo struct {
	Version           string
	VersionBuild      string
	VersionSha        string
	StartTime         string
	Uptime            string
	ReloadTime        string
	FailedReloadTime  string
	FailedReloadError string
}

func uptime(t time.Duration) string {
	if (t.Hours() / 24) > 1 {
		return fmt.Sprintf("%.0fd", t.Hours()/24)
	} else if t.Hours() > 1 {
		return fmt.Sprintf("%.0fh", t.Hours())
	} else if t.Minutes() > 1 {
		return fmt.Sprintf("%.0fm", t.Minutes())
	} else if t.Seconds() > 1 {
		return fmt.Sprintf("%.0fs", t.Seconds())
	}
	return fmt.Sprintf("%.0dns", t.Nanoseconds())
}

// WebRoot serves Webserver's root folder
func WebRoot(w http.ResponseWriter, r *http.Request) {
	log := logging.For("core/webroot").WithField("func", "web")
	page := newPage("Mercury Global Loadbalancer", r.RequestURI)

	templateNames := []string{"root.tmpl", "header.tmpl", "footer.tmpl"}
	template, err := web.LoadTemplates("static", templateNames)
	if err != nil {
		log.Warnf("Error loading templates: %s", err)
	}

	data := struct {
		ProcessInfo processInfo
		Page        web.Page
	}{processInfo{
		strings.TrimSuffix(config.Version, "\""),
		strings.TrimSuffix(config.VersionBuild, "\""),
		strings.TrimSuffix(config.VersionSha, "\""),
		FormattedDate(config.StartTime),
		uptime(time.Since(config.StartTime)),
		FormattedDate(config.ReloadTime),
		FormattedDate(config.FailedReloadTime),
		config.FailedReloadError,
	}, *page}

	err = template.ExecuteTemplate(w, "root", data)
	if err != nil {
		log.WithField("error", err).Warn("Error executing template")
	}
}

func newPage(title, uri string) *web.Page {
	return &web.Page{
		Title:    title,
		URI:      uri,
		Hostname: config.Get().Cluster.Binding.Name,
		Time:     time.Now(),
	}
}

// NewServer sets up a new webserver
func NewServer(ip string, port int) (s *http.Server, l net.Listener, err error) {
	s = &http.Server{
		Addr: fmt.Sprintf("%s:%d", ip, port),
		//Handler:        goweb.DefaultHttpHandler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	box, _ := rice.FindBox("static")
	staticContent := http.StripPrefix("/static/", http.FileServer(box.HTTPBox()))

	http.Handle("/static/", staticContent)

	fs := http.FileServer(http.Dir(config.Get().Web.Path))
	http.Handle("/internal/", http.StripPrefix("/internal/", fs))

	//http.HandleFunc("/glb", dns.WebGLBStatus)
	http.HandleFunc("/glb", WebGLBStatus)
	http.HandleFunc("/localdns", WebLocalDNSStatus)
	http.HandleFunc("/backend", WebBackendStatus)
	http.HandleFunc("/proxy", WebProxyStatus)
	http.HandleFunc("/cluster", WebClusterStatus)
	http.HandleFunc("/backenddetails", WebBackendDetails)
	http.HandleFunc("/", WebRoot)

	l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	//  return s
	return
}

// InitializeWebserver starts the webserver
func InitializeWebserver() {
	log := logging.For("core/webserver").WithField("ip", config.Get().Web.Binding).WithField("port", config.Get().Web.Port).WithField("func", "web")
	log.Info("Starting web server")
	server, listener, err := NewServer(config.Get().Web.Binding, config.Get().Web.Port)

	if config.Get().Web.TLSConfig.CertificateFile != "" {
		log.WithField("file", config.Get().Web.TLSConfig.CertificateFile).Debug("Enabling SSL for web service")
		tlsconf, terr := tlsconfig.LoadCertificate(config.Get().Web.TLSConfig)
		if terr != nil {
			log.WithField("error", terr).Errorf("Failed to load SSL certificate for web service")
		}
		server.TLSConfig = tlsconf
		http2.ConfigureServer(server, &http2.Server{})

		listener = tls.NewListener(listener, server.TLSConfig)
	}
	if err == nil {
		defer listener.Close()
		log.Info("Started web server")
		err = server.Serve(listener)
		if err != nil {
			log.Fatalf("Error Starting Webservice: %s", err)
		}
	}
}
