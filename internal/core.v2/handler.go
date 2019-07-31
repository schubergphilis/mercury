package core

import (
	"github.com/nightlyone/lockfile"
	"github.com/schubergphilis/mercury.v2/internal/healthcheck"
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/models"
	"github.com/schubergphilis/mercury.v2/internal/profiler"
	"github.com/schubergphilis/mercury.v2/pkg/cluster"
)

// Handler is the core handler
type Handler struct {
	// log provider
	LogProvider logging.SimpleLogger
	// local handler
	log logging.SimpleLogger
	// log level
	DefaultLevel string
	// config file to load from (including reloads)
	configFile string
	// last loaded config
	config *Config
	// active config
	runningConfig *Config
	// pid file to write
	pidFile string
	// profile address
	profilerAddr string

	// quit is called on exit
	Quit chan struct{}
	// reload is called on reload
	Reload chan struct{}

	// interfaces
	cluster     models.ClusterService
	healthcheck models.HealthcheckService

	// state maintainer
	state models.StateService
}

// New creates a new handler for the core
func New(opts ...Option) *Handler {
	logProvider, _ := logging.NewDefault()
	handler := Handler{
		LogProvider: logProvider,
		Quit:        make(chan struct{}),
		Reload:      make(chan struct{}),

		runningConfig: &Config{}, // start with empty config
		cluster:       cluster.New(),
		healthcheck:   healthcheck.NewManager(),
	}
	handler.setLogLevel("info")

	for _, o := range opts {
		o(&handler)
	}

	// set log level
	handler.setLogLevel(handler.config.LoggingConfig.Level)

	return &handler
}

func (h *Handler) setLogLevel(level string) {
	// set log level
	logLevel, _ := logging.ToLevel(level)
	var prefix []interface{}
	prefix = append(prefix, "func")
	prefix = append(prefix, "main")
	h.log = (&logging.Wrapper{Log: h.LogProvider, Level: logLevel, Prefix: prefix})
}

func (h *Handler) Start() {
	// get a lock in the lock file
	lock, err := h.getLock()
	if err != nil {
		h.log.Fatalf("failed to create pid file", "file", h.pidFile, "error", err)
		close(h.Quit)
		return
	}
	defer lock.Unlock()

	// start memory profiler if requested
	if h.profilerAddr != "" {
		p := profiler.New(h.profilerAddr)
		go p.Start()
		defer p.Stop()
	}

	// start cluster listener
	h.startCluster()
	defer h.stopCluster()

	// start health checks
	h.startHealthchecks()
	defer h.stopHealthchecks()

	//h.cluster.Wi
	// start cluster service
	/*cluster := h.newCluster(h.cluster, &h.config.ClusterConfig)
	cluster.WithLogger(h.Log)
	cluster.start()
	defer cluster.stop()
	*/
	/*
		cluster := NewCluster(&h.config.ClusterConfig)
		cluster.WithLogger(h.Log)
		go cluster.start()        // starts the listener
		go cluster.connectNodes() // connects to the nodes
		defer cluster.stop()
		go cluster.Handler() // starts the handler
	*/

	// start dns service
	dns := NewDNSServer(&h.config.DNSConfig)
	dns.WithLogger(h.LogProvider)
	go dns.start()   // starts the listener
	defer dns.stop() // stop the listener

	// start all handlers
	go h.clusterReceiverHandler()
	go h.healthcheckReceiverHandler()

	// wait for quit signal
	for {
		select {
		// Internal events
		case <-h.Reload:
			// attempt to load the new config
			if err := h.loadConfig(); err != nil {
				h.log.Fatalf("reload of configuration failed", "error", err)
				continue
			}

			// apply config to cluster
			h.reloadCluster()
			h.reloadHealthchecks()
			// do reload action
		case <-h.Quit:
			return
		}
	}

}

func (h *Handler) getLock() (lock lockfile.Lockfile, err error) {
	// get a lock, or die trying
	lock, err = lockfile.New(h.pidFile)
	if err != nil {
		return
	}
	return lock, lock.TryLock()
}

func (h *Handler) clusterReceiverHandler() {
	// wait for quit signal
	for {
		select {
		case <-h.Quit:
			return

			// cluster events
		//case clusterLog := <-h.cluster.ReceivedLogging():
		//h.log.Debugf(clusterLog)
		case <-h.cluster.ReceivedFromCluster():
			// application based packet received, take related action
		case <-h.cluster.ReceivedNodeJoin():
			//cl.ToNode <- cluster.NodeMessage{Node: node, Message: config.ClusterPacketConfigRequest{}}
			//manager.dnsdiscard <- node
			// go clusterDNSUpdateSingleBroadcastAll(cl, node)
		case <-h.cluster.ReceivedNodeLeave():
			//go manager.BackendNodeDiscard(node)
			//manager.dnsoffline <- node
		}

	}
}

func (h *Handler) healthcheckReceiverHandler() {
	// wait for quit signal
	for {
		select {
		case <-h.Quit:
			return

		case healthcheck := <-h.healthcheck.ReceiveHealthCheckStatus():
			h.log.Infof("Received healhcheck update", "uuid", healthcheck.UUID, "status", healthcheck.Status, "error", healthcheck.ErrorMsg)
			//
		}

	}
}
