package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/http2"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

const (
	// YES when yes simply isn't enough
	YES = "yes"
	// HTTP string
	HTTP = "http"
	// HTTPS string
	HTTPS = "https"
)

// Listener contains the config for the proxy listener
type Listener struct {
	UUID            string
	Name            string
	SourceIP        string
	IP              string
	Port            int
	ListenerMode    string // Protocol the listener expects
	HTTPProto       int    // HTTP Version Protocol the listener expects
	Backends        map[string]*Backend
	TLSConfig       *tls.Config // TLS Config
	MaxConnections  int
	socket          *limitListener
	Statistics      *balancer.Statistics
	stop            chan bool
	ErrorPage       ErrorPage
	MaintenancePage ErrorPage
	ReadTimeout     int // Timeout in seconds to wait for the client sending the request - https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	WriteTimeout    int // Timeout in seconds to wait for server reply to client
	Uptime          time.Time
	OCSPStapling    string // use OCSP Stapling
}

// New creates a new proxy for using a listener
func New(uuid string, name string, maxconnections int) *Listener {
	return &Listener{
		UUID:       uuid,
		Name:       name,
		Backends:   make(map[string]*Backend),
		stop:       make(chan bool, 1),
		Statistics: balancer.NewStatistics(uuid, maxconnections),
		Uptime:     time.Now(),
	}
}

// GetBackend Return the first backend
func (l *Listener) GetBackend() (*Backend, error) {
	for _, backend := range l.Backends {
		return backend, nil
	}
	return nil, fmt.Errorf("Unable to find a backend")
}

// AddBackend adds a backend to an existing proxy
func (l *Listener) AddBackend(uuid string, name string, balancemode string, connectmode string, hostname []string, maxconnections int, errorPage ErrorPage, maintenancePage ErrorPage) {
	b := NewBackend(uuid, balancemode, connectmode, hostname, maxconnections, errorPage, maintenancePage)
	l.Backends[name] = b
}

// Start the listener
func (l *Listener) Start() {
	log := logging.For("proxy/listener/start").WithField("pool", l.Name).WithField("localip", l.IP).WithField("localport", l.Port).WithField("mode", l.ListenerMode)
	if l.SourceIP != "" {
		log = log.WithField("sourceip", l.SourceIP)
	}

	log.Debug("Starting listener")

	var httpsrv *http.Server
	var tcplistener net.Listener
	var listener net.Listener
	var err error
	ocspQuit := make(chan bool)
	switch l.ListenerMode {
	case "tcp":
		// Start listener, and do actions based on that, do other functions
		tcplistener, err = l.NewTCPProxy()
		if err != nil {
			log.WithField("error", err).Error("Error starting TCP proxy listener")
			return
		}
		go l.TCPProxy(tcplistener)

	case HTTP:
		proxy := l.NewHTTPProxy()
		httpsrv = &http.Server{
			ReadTimeout:  time.Duration(l.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(l.WriteTimeout) * time.Second,
			Addr:         fmt.Sprintf("%s:%d", l.IP, l.Port),
			Handler:      proxy,
			ErrorLog:     logging.StandardLog("listener/http"),
		}

		listener, err = net.Listen("tcp", httpsrv.Addr)
		if err != nil {
			log.WithField("error", err).Error("Error starting HTTP proxy listener")
			return
		}

		l.socket = limitListenerConnections(listener.(*net.TCPListener), l.MaxConnections)
		go httpsrv.Serve(l.socket)

	case HTTPS:
		proxy := l.NewHTTPProxy()

		l.TLSConfig.GetClientCertificate = func(t *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			log.Debugf("Client requestinfo: %+v", t)
			return nil, nil
		}

		l.TLSConfig.GetCertificate = func(t *tls.ClientHelloInfo) (*tls.Certificate, error) {
			log.Debugf("Client Hello: %+v", t)
			return nil, nil
		}

		l.TLSConfig.GetConfigForClient = func(t *tls.ClientHelloInfo) (*tls.Config, error) {
			log.WithField("client_tls_support", fmt.Sprintf("%+v", t)).WithField("handshake", "getConfigForClient").Debug("SSL Handhake")
			return nil, nil
		}

		httpsrv = &http.Server{
			ReadTimeout:  time.Duration(l.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(l.WriteTimeout) * time.Second,
			Addr:         fmt.Sprintf("%s:%d", l.IP, l.Port),
			Handler:      proxy,
			TLSConfig:    l.TLSConfig,
			ErrorLog:     logging.StandardLog("listener/https"),
			//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}

		httpsrv.ConnState = func(n net.Conn, c http.ConnState) {
			log.WithField("client_http_server_state", fmt.Sprintf("%+v", c)).WithField("client", n.RemoteAddr()).Debug("HTTP state")
		}

		http2.ConfigureServer(httpsrv, &http2.Server{})
		listener, err = net.Listen("tcp", httpsrv.Addr)

		if err != nil {
			log.WithField("error", err).Error("Error starting HTTPS proxy listener")
			return
		}

		l.socket = limitListenerConnections(listener.(*net.TCPListener), l.MaxConnections)
		tlsListener := tls.NewListener(l.socket, httpsrv.TLSConfig)
		if l.OCSPStapling == YES {
			httpsrv.TLSConfig.ServerName = fmt.Sprintf("%s:%d", l.IP, l.Port)
			go tlsconfig.OCSPHandler(httpsrv.TLSConfig, ocspQuit)
		}

		go httpsrv.Serve(tlsListener)

	case "udp":
		// TODO: not implemented yet
	}
	log.Debug("Proxy ready for clients")
	for {
		select {
		case _ = <-l.stop:
			switch l.ListenerMode {
			case "tcp":
				log.Debug("Stopping TCP Proxy on request")
				tcplistener.Close()

			case HTTP:
				fallthrough

			case HTTPS:
				log.Debug("Stopping HTTP(s) Proxy on request")
				err := httpsrv.Shutdown(nil)
				if err != http.ErrServerClosed {
					log.Debugf("Gracefull stop of Proxy failed: %s", err)
					listener.Close()
				}

				if l.OCSPStapling == YES {
					log.Debug("Stopping of Proxy finished, stopping ocsp")
					select {
					case ocspQuit <- true:
					default:
					}
				}

			case "udp":
				// TODO: not implemented yet
			}

			log.Debug("Stopping of Proxy finished, sending state back")
			l.stop <- true
			return
		}
	}

}

// Debug shows output for debugging
func (l *Listener) Debug() {
	log := logging.For("proxy/listener/debug").WithField("pool", l.Name).WithField("localip", l.IP).WithField("localport", l.Port).WithField("mode", l.ListenerMode)
	log.Debug("Active proxy")
}

// FindBackendByHost searches for matching backend by hostname requested
func (l *Listener) FindBackendByHost(req string) (string, *Backend) {
	var defaulthost string
	var defaultbackend *Backend
	for id, backend := range l.Backends {
		for _, host := range backend.Hostname {
			if strings.EqualFold(host, req) {
				return id, backend
			}

			if strings.EqualFold(host, "default") {
				defaulthost = id
				defaultbackend = backend
			}
		}
	}

	return defaulthost, defaultbackend
}

// FindAllHostNames searches for matching backend by hostname requested
func (l *Listener) FindAllHostNames() []string {
	var hostname []string
	for _, backend := range l.Backends {
		for _, host := range backend.Hostname {
			hostname = append(hostname, host)
		}
	}

	return hostname
}

// updateClients updates the statistics on connected clients
func (l *Listener) updateClients() {
	l.Statistics.ClientsConnectedSet(int64(l.socket.Clients()))
}

// Stop exits the proxy process for the listener
func (l *Listener) Stop() {
	log := logging.For("proxy/listener/stop").WithField("pool", l.Name).WithField("localip", l.IP).WithField("localport", l.Port).WithField("mode", l.ListenerMode)
	log.Info("Sending stop to proxy")
	l.stop <- true
	log.Info("Waiting for stopped state")
	<-l.stop
	log.Info("Proxy stopped")
}

// SetListener sets all listener config for the proxy
func (l *Listener) SetListener(mode string, sourceIP string, ip string, port int, maxConnections int, tlsConfig *tls.Config, readTimeout int, writeTimeout int, httpProto int, ocspStapling string) {
	log := logging.For("proxy/setlistener").WithField("mode", mode).WithField("sourceip", sourceIP).WithField("ip", ip).WithField("port", port).WithField("protocolversion", httpProto).WithField("maxconnections", maxConnections)
	log.WithField("readtimeout", readTimeout).WithField("writetimeout", writeTimeout).Debug("Setting Proxy Listener")

	l.ListenerMode = mode
	l.SourceIP = sourceIP
	l.IP = ip
	l.Port = port
	l.MaxConnections = maxConnections
	l.TLSConfig = tlsConfig
	l.HTTPProto = httpProto
	l.ReadTimeout = readTimeout
	l.WriteTimeout = writeTimeout
	l.OCSPStapling = ocspStapling
}

// UpdateBackend adds a backend to an existing proxy, or updates an existing one
func (l *Listener) UpdateBackend(uuid string, name string, balancemode string, connectmode string, hostname []string, maxconnections int, errorPage ErrorPage, maintenancePage ErrorPage) {
	if backend, ok := l.Backends[name]; ok {
		backend.BalanceMode = balancemode
		backend.ConnectMode = connectmode
		backend.Hostname = hostname
		backend.ErrorPage = errorPage
		backend.MaintenancePage = maintenancePage
		backend.ErrorPage.load()
		backend.MaintenancePage.load()
	} else {
		b := NewBackend(uuid, balancemode, connectmode, hostname, maxconnections, errorPage, maintenancePage)
		b.ErrorPage.load()
		b.MaintenancePage.load()
		l.Backends[name] = b
	}
}

// RemoveBackend removes a backend from the listener
func (l *Listener) RemoveBackend(name string) {
	if _, ok := l.Backends[name]; ok {
		delete(l.Backends, name)
	}
}

// LoadErrorPage preloads the error page
func (l *Listener) LoadErrorPage(e ErrorPage) error {
	l.ErrorPage = e
	return l.ErrorPage.load()
}

// LoadMaintenancePage preloads the error page
func (l *Listener) LoadMaintenancePage(e ErrorPage) error {
	l.MaintenancePage = e
	return l.MaintenancePage.load()
}

// GetBackendStats gets the combined statistics from all nodes of a backend
func (l *Listener) GetBackendStats(backendName string) *balancer.Statistics {
	l.Backends[backendName].sync.RLock()
	defer l.Backends[backendName].sync.RUnlock()

	backendStats := balancer.NewStatistics(l.Backends[backendName].UUID, 1)
	for _, node := range l.Backends[backendName].Nodes {
		backendStats.ClientsConnectedAdd(node.Statistics.ClientsConnectedGet())
		backendStats.ClientsConnectsAdd(node.Statistics.ClientsConnectsGet())
		backendStats.RXAdd(node.Statistics.RXGet())
		backendStats.RXAdd(node.Statistics.RXGet())
		backendStats.ResponseTimeValueMerge(node.Statistics.ResponseTimeValueGet())
	}

	return backendStats
}
