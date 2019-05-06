package main

import (
	core "github.com/schubergphilis/mercury.v2/internal/core.v2"
	"github.com/schubergphilis/mercury.v2/internal/logging"
)

// Only enabled for profiling

const (
	defaultLogLevel  = "warn"
	defaultLogTarget = "stdout"
)

// main start
func main() {
	logger, err := logging.NewZap()
	if err != nil {
		panic(err)
	}

	handler := core.New(
		core.WithLogger(logger),
	)

	handler.Log.Debugf("reading profile (credentials + api endpoint)")

	// with the api, we read the configured glb/cluster stuff
	// based on the api report, we do monitoring
	// api should provide all configured items + status

}
