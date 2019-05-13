package core

func (h *Handler) startHealthchecks() {
	// load new config
	h.reloadHealthchecks()
	h.healthcheck.Start()
}

func (h *Handler) stopHealthchecks() {
	h.healthcheck.Stop()
}

func (h *Handler) reloadHealthchecks() {
	for poolId, pool := range h.config.Loadbalancer.Pools {
		for backendId, backend := range pool.Backends {
			for healthcheckId, healthcheck := range backend.HealthChecks {

			}
		}
		for _, healthcheck := range pool.HealthChecks {

		}
	}
}

func (h *Handler) healthchecksReceiverHandler() {

}
