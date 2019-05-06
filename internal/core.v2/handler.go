package core

import (
	"github.com/nightlyone/lockfile"
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/profiler"
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
	// pid file to write
	pidFile string
	// profile address
	profilerAddr string

	// quit is called on exit
	Quit chan struct{}
	// reload is called on reload
	Reload chan struct{}
}

type Option func(o *Handler)

// New creates a new handler for the core
func New(opts ...Option) *Handler {
	handler := Handler{
		Log:    &logging.Default{},
		Quit:   make(chan struct{}),
		Reload: make(chan struct{}),
	}

	for _, o := range opts {
		o(&handler)
	}

	return &handler
}

func WithLogger(l logging.SimpleLogger) Option {
	return func(h *Handler) {
		h.Log = l
	}
}

func WithConfigFile(o string) Option {
	return func(h *Handler) {
		h.configFile = o
		var err error
		if h.config, err = h.loadConfig(); err != nil {
			h.Log.Fatalf("failed to load config file", "error", err, "file", o)
		}
	}
}

func WithPidFile(o string) Option {
	return func(h *Handler) {
		h.pidFile = o
	}
}

func WithLogLevel(o string) Option {
	return func(h *Handler) {
		h.LogLevel = o
	}
}

func WithProfiler(o string) Option {
	return func(h *Handler) {
		h.profilerAddr = o
	}
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

	// wait for quit signal
	for {
		select {
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
