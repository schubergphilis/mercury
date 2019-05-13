package core

import (
	"github.com/nightlyone/lockfile"
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/profiler"
	"github.com/schubergphilis/mercury.v2/pkg/cluster"
)

// Handler is the core handler
type Handler struct {
	// log handler
	Log logging.SimpleLogger
	// log level
	LogLevel string
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
	cluster ClusterService
}

// New creates a new handler for the core
func New(opts ...Option) *Handler {
	handler := Handler{
		Log:    &logging.Default{},
		Quit:   make(chan struct{}),
		Reload: make(chan struct{}),

		runningConfig: &Config{}, // start with empty config
		cluster:       cluster.New(),
	}

	for _, o := range opts {
		o(&handler)
	}

	return &handler
}

func (h *Handler) Start() {
	// get a lock in the lock file
	lock, err := h.getLock()
	if err != nil {
		h.Log.Fatalf("failed to create pid file", "file", h.pidFile, "error", err)
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

	h.startCluster()
	defer h.stopCluster()
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
	dns.WithLogger(h.Log)
	go dns.start()   // starts the listener
	defer dns.stop() // stop the listener

	// wait for quit signal
	for {
		select {
		// Internal events
		case <-h.Reload:
			// attempt to load the new config
			if err := h.loadConfig(); err != nil {
				h.Log.Fatalf("reload of configuration failed", "error", err)
				continue
			}

			// apply config to cluster
			h.reloadCluster()
			// do reload action
		case <-h.Quit:
			return

			// cluster events
		case clusterLog := <-h.cluster.ReceivedLogging():
			h.Log.Debugf(clusterLog)
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
