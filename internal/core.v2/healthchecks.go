package core

import (
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/models"
)

func (h *Handler) startHealthchecks() {
	if alternativeLogLevel, err := logging.ToLevel(h.config.LoggingConfig.HealthcheckLevel); err == nil {
		h.log.Infof("Healtcheck alternative log level", "level", h.config.LoggingConfig.HealthcheckLevel)
		var prefix []interface{}
		prefix = append(prefix, "func")
		prefix = append(prefix, "healthcheck")

		h.healthcheck.WithLogger(&logging.Wrapper{Log: h.LogProvider, Level: alternativeLogLevel, Prefix: prefix})
	}

	// load new config
	h.reloadHealthchecks()
	//h.healthcheck.Start()
}

func (h *Handler) stopHealthchecks() {
	h.healthcheck.Stop()
}

func (h *Handler) reloadHealthchecks() {
	// health checks we have configured
	requestedHealthchecks := make(map[string]models.Healthcheck)
	for _, pool := range h.config.Loadbalancer.Pools {
		for _, backend := range pool.Backends {
			for _, healthcheck := range backend.Healthchecks {
				requestedHealthchecks[healthcheck.UUID()] = healthcheck
			}
		}
		for _, healthcheck := range pool.Healthchecks {
			requestedHealthchecks[healthcheck.UUID()] = healthcheck
		}
	}

	// health checks we have active
	runningHealthchecks := make(map[string]models.Healthcheck)
	for _, pool := range h.runningConfig.Loadbalancer.Pools {
		for _, backend := range pool.Backends {
			for _, healthcheck := range backend.Healthchecks {
				runningHealthchecks[healthcheck.UUID()] = healthcheck
			}
		}
		for _, healthcheck := range pool.Healthchecks {
			runningHealthchecks[healthcheck.UUID()] = healthcheck
		}
	}

	added, deleted := healthcheckAddedAndDeleted(runningHealthchecks, requestedHealthchecks)
	h.log.Infof("healthchecks monitor starting", "added", len(added), "removed", len(deleted))
	for uuid := range deleted {
		h.healthcheck.RemoveHealthcheck(uuid)
	}

	for uuid, check := range added {
		h.healthcheck.AddHealthcheck(uuid, check)
	}

	// update log level if needed
	if h.config.LoggingConfig.HealthcheckLevel != h.runningConfig.LoggingConfig.HealthcheckLevel {
		if alternativeLogLevel, err := logging.ToLevel(h.config.LoggingConfig.HealthcheckLevel); err == nil {
			h.log.Infof("Healtcheck alternative log level", "level", h.config.LoggingConfig.HealthcheckLevel)
			var prefix []interface{}
			prefix = append(prefix, "func")
			prefix = append(prefix, "healthcheck")

			h.healthcheck.WithLogger(&logging.Wrapper{Log: h.LogProvider, Level: alternativeLogLevel, Prefix: prefix})
			// update running config
			h.runningConfig.LoggingConfig.HealthcheckLevel = h.config.LoggingConfig.HealthcheckLevel
		}
	}
}

func healthcheckAddedAndDeleted(old, new map[string]models.Healthcheck) (added map[string]models.Healthcheck, deleted map[string]models.Healthcheck) {
	return healthcheckAdded(old, new), healthcheckAdded(new, old)
}

// find added items
func healthcheckAdded(old, new map[string]models.Healthcheck) map[string]models.Healthcheck {
	added := make(map[string]models.Healthcheck)
	for i, check := range new {
		if _, ok := old[i]; !ok {
			added[i] = check
		}
	}
	return added
}
