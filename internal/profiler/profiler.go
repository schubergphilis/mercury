package profiler

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"
)

type Handler struct {
	addr string
	srv  *http.Server
}

func New(addr string) *Handler {
	return &Handler{
		addr: addr,
	}
}

func (h *Handler) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", pprof.Index)

	h.srv = &http.Server{
		Addr:    h.addr,
		Handler: mux,
	}

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	//http.ListenAndServe(addr, mux)

	go func() {
		// returns ErrServerClosed on graceful close
		if err := h.srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Profiler ListenAndServe(): %s", err)
		}
	}()
}

func (h *Handler) Stop() {
	if h.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.srv.Shutdown(ctx)
	h.srv = nil
}
