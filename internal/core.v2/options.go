package core

import (
	"github.com/schubergphilis/mercury.v2/internal/logging"
)

type Option func(o *Handler)

func WithLogger(l logging.SimpleLogger) Option {
	return func(h *Handler) {
		h.Log = l
	}
}

func WithConfigFile(o string) Option {
	return func(h *Handler) {
		h.configFile = o
		if err := h.loadConfig(); err != nil {
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
