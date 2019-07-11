package core

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// HealthHandler gets all healthcheck feedback, and sends updates to the core manager
// We get all health checks here
func (manager *Manager) HealthHandler(healthCheck *healthcheck.Manager) {
	log := logging.For("core/healthcheck/handler").WithField("func", "healthcheck")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)

	for {
		select {
		case _ = <-signalChan:
			log.Debug("HealthHandler Debug triggered")
			healthCheck.Debug()
		case checkresult := <-healthCheck.Incoming:
			// Status change entity
			// pool + backend + node = node check Changed
			// pool + backend = backend check changed - applies to nodes
			// pool = pool check changed - applies to vip
			log.WithField("pool", checkresult.PoolName).WithField("backend", checkresult.BackendName).WithField("node", checkresult.NodeName).WithField("actualstatus", checkresult.ActualStatus.String()).WithField("reportedstatus", checkresult.ReportedStatus.String()).WithField("errormsg", checkresult.ErrorMsg).WithField("check", checkresult.Description).Info("Received health update from worker")

			// Set status in healh pool
			healthCheck.SetCheckStatus(checkresult.WorkerUUID, checkresult.ReportedStatus, checkresult.ErrorMsg)

			// Get all nodes using the check
			nodeUUIDs := healthCheck.GetPools(checkresult.WorkerUUID)
			log.WithField("nodeuuids", nodeUUIDs).WithField("workeruuid", checkresult.WorkerUUID).Debug("Pools to update")

			// and check each individual node using the above check, to see if status changes
			for _, nodeUUID := range nodeUUIDs {
				actualStatus, poolName, backendName, nodeName, errors := healthCheck.GetNodeStatus(nodeUUID)
				checkresult.ReportedStatus = actualStatus
				checkresult.ErrorMsg = errors
				checkresult.NodeUUID = nodeUUID
				if poolName != "" {
					checkresult.PoolName = poolName
				}

				if nodeName != "" {
					checkresult.NodeName = nodeName
				}

				if backendName != "" {
					checkresult.BackendName = backendName
				}

				log.WithField("pool", checkresult.PoolName).WithField("backend", checkresult.BackendName).WithField("node", checkresult.NodeName).WithField("reportedstatus", checkresult.ReportedStatus.String()).WithField("error", checkresult.ErrorMsg).Info("Sending status update to cluster")
				manager.healthchecks <- checkresult // do not send pointers, since pointer will change data
			}

		}
	}
}

// InitializeHealthChecks sets up the health checking
func (manager *Manager) InitializeHealthChecks(h *healthcheck.Manager) {
	log := logging.For("core/healthcheck/init").WithField("func", "healthcheck")
	var expectedWorkers []*healthcheck.Worker
	for poolName, pool := range config.Get().Loadbalancer.Pools {
		// Create workers for pool checks
		var poolWorkers []*healthcheck.Worker
		for _, check := range pool.HealthChecks {
			sourceip := pool.Listener.IP
			if pool.Listener.SourceIP != "" {
				sourceip = pool.Listener.SourceIP
			}
			worker := healthcheck.NewWorker(poolName, "", "", "", check.IP, check.Port, sourceip, check, h.Incoming)
			poolWorkers = append(poolWorkers, worker)
		}
		for backendName, backend := range config.Get().Loadbalancer.Pools[poolName].Backends {
			var backendWorkers []*healthcheck.Worker
			// Create workers for backend checks
			for _, check := range backend.HealthChecks {
				if check.IP != "" {
					sourceip := pool.Listener.IP
					if pool.Listener.SourceIP != "" {
						sourceip = pool.Listener.SourceIP
					}
					worker := healthcheck.NewWorker(poolName, backendName, "", "", check.IP, check.Port, sourceip, check, h.Incoming)
					backendWorkers = append(backendWorkers, worker)
				}
			}

			for _, node := range backend.Nodes {
				var nodeWorkers []*healthcheck.Worker
				// For each node
				for _, check := range backend.HealthChecks {
					if check.IP == "" {
						sourceip := pool.Listener.IP
						if pool.Listener.SourceIP != "" {
							sourceip = pool.Listener.SourceIP
						}
						// Create workers for node specific checks
						worker := healthcheck.NewWorker(poolName, backendName, node.Name(), node.UUID, node.IP, node.Port, sourceip, check, h.Incoming)
						nodeWorkers = append(nodeWorkers, worker)
					}
				}
				// Register all checks applicable to this node
				var nodeChecks []string
				for _, w := range nodeWorkers {
					nodeChecks = append(nodeChecks, w.UUID())
				}

				for _, w := range backendWorkers {
					nodeChecks = append(nodeChecks, w.UUID())
				}

				for _, w := range poolWorkers {
					nodeChecks = append(nodeChecks, w.UUID())
				}

				// Register all checks applicable to the node UUID
				h.SetCheckPool(node.UUID, poolName, backendName, node.Name(), backend.HealthCheckMode, nodeChecks)

				// Register worker for node checks
				expectedWorkers = append(expectedWorkers, nodeWorkers...)
			}
			// Register worker for backend checks
			expectedWorkers = append(expectedWorkers, backendWorkers...)
		}
		// Register worker for pool checks
		expectedWorkers = append(expectedWorkers, poolWorkers...)
	}

	// Clean up existing workers
	log.WithField("current", len(h.Workers)).WithField("expected", len(expectedWorkers)).Debug("Workers count")
	// current workers
	var removeWorkers []int
	current := h.Workers
	for cid, cworker := range current {
		// go through all current workers
		found := false

		// Get copy of existing worker, and trim them to keep removableProxy list
		var expected []*healthcheck.Worker
		for _, worker := range expectedWorkers {
			expected = append(expected, worker)
		}

		for eid, eworker := range expected {
			if cworker.UUID() == eworker.UUID() {
				log.Debugf("Existing worker check: current:%s new:%s", cworker.UUID(), eworker.UUID())
				// we have a matching current with expected check, no need to add it again
				expectedWorkers = append(expectedWorkers[:eid], expectedWorkers[eid+1:]...)
				found = true
			}
		}

		if found == false {
			// stop worker, its no longer expected
			log.Debugf("Nonexisting worker check: current:%s ", cworker.UUID())
			removeWorkers = append(removeWorkers, cid)
		}
	}

	log.WithField("count", len(removeWorkers)).Debug("Workers to be stopped")
	for _, id := range reverse(removeWorkers) {
		log.WithField("id", id).Debug("Stopping worker")
		h.StopWorker(id)
	}

	// expectedChecks no longer contains the active workers, so these need to be started regardless
	log.WithField("count", len(expectedWorkers)).Debug("Workers to be started")
	for _, worker := range expectedWorkers {
		h.RegisterWorker(worker)
		worker.Start()
	}

	log.WithField("count", len(expectedWorkers)).Debug("Workers running")
}

// reverse an array of strings
func reverse(s []int) []int {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
