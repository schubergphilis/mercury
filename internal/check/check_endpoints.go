package check

import (
	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// GLB Checks GLB status
func Endpoints() int {
	log := logging.For("check/endpoints")
	log.Info("checking endpoints")
	var expectedWorkers []*healthcheck.Worker

	for poolName, pool := range config.Get().Loadbalancer.Pools {

		// Create workers for pool checks
		var poolWorkers []*healthcheck.Worker
		for _, check := range pool.HealthChecks {
			sourceip := pool.Listener.IP
			if pool.Listener.SourceIP != "" {
				sourceip = pool.Listener.SourceIP
			}
			worker := healthcheck.NewWorker(poolName, "", "", "", check.IP, check.Port, sourceip, check, nil)
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
					worker := healthcheck.NewWorker(poolName, backendName, "", "", check.IP, check.Port, sourceip, check, nil)
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

						port := node.Port
						if check.Port != 0 {
							port = check.Port
						}
						// Create workers for node specific checks
						worker := healthcheck.NewWorker(poolName, backendName, node.Name(), node.UUID, node.IP, port, sourceip, check, nil)
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
				//h.SetCheckPool(node.UUID, poolName, backendName, node.Name(), backend.HealthCheckMode, nodeChecks)

				// Register worker for node checks
				expectedWorkers = append(expectedWorkers, nodeWorkers...)
			}
			// Register worker for backend checks
			expectedWorkers = append(expectedWorkers, backendWorkers...)
		}
		// Register worker for pool checks
		expectedWorkers = append(expectedWorkers, poolWorkers...)
	}

	// now execute the workers
	for _, worker := range expectedWorkers {
		s, e, d := worker.ExecuteCheck()
		log.Infof("worker: %+v \nstatus: %+v\n error: %+v\ndescription: %s\n\n", worker, s, e, d)
	}
	return OK
}
