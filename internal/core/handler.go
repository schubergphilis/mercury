package core

import (
	"fmt"
	"time"

	"github.com/rdoorn/gohelper/app"
	"github.com/rdoorn/gohelper/logging"
	"github.com/spf13/viper"
)

const (
	// Name is the name of the application
	Name string = "mercury"
)

var (
	// Version of application
	Version string
	// VersionBuild number
	VersionBuild string
	// VersionSha git commit of build
	VersionSha string
	// StartTime of application
	StartTime time.Time
	// ReloadTime last time a reload was successfull
	ReloadTime time.Time
	// FailedReloadTime last time a reload failed
	FailedReloadTime time.Time
	// FailedReloadError last time a reload failed
	FailedReloadError string
)

type Handler struct {
	app.App
	stop chan struct{}
}

func New() *Handler {
	return &Handler{
		stop: make(chan struct{}),
	}
}

// Enable is called on start and reload
func (h *Handler) Enable(config *Config) error {
	if err := h.setLogging(viper.GetString("log_level"), viper.GetStringSlice("log_output")...); err != nil {
		return err
	}
	h.Infof("Enabling configuration")
	return nil
}

// Stop stops the handler
func (h *Handler) Stop() {
	close(h.stop)
}

func (h *Handler) setLogging(level string, output ...string) error {
	logger, _ := logging.NewZap(output...)
	logLevel, err := logging.ToLevel(level)
	if err != nil {
		return fmt.Errorf("error setting log level: %s\n", err)
	}
	logWrapper := &logging.Wrapper{
		Log:   logger,
		Level: logLevel,
	}
	h.WithLogging(logWrapper)
	return nil
}
