package healthcheck

import (
	"fmt"

	"github.com/schubergphilis/mercury/src/logging"
)

// HealthStatus keeps track of the status of each workers health check
type HealthStatus struct {
	CheckStatus bool     // map[checkuuid]status - status returned by worker
	AdminDown   bool     // manual override
	AdminUp     bool     // manual override
	ErrorMsg    []string // error message
	//Required     map[string]HealthRequired
}

// HealthPool contains a per nodeuuid information about all checks that apply to this node
type HealthPool struct {
	PoolName    string   // name of the vip pool
	BackendName string   // name of the backend
	NodeName    string   // name of the node
	Match       string   // all/any
	Checks      []string // []checkuuid
}

// SetCheckStatus sets the status of a worker check based on the health check result
func (m *Manager) SetCheckStatus(workerUUID string, status bool, errorMsg []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if _, ok := m.WorkerMap[workerUUID]; !ok {
		m.WorkerMap[workerUUID] = HealthStatus{}
	}
	s := m.WorkerMap[workerUUID]
	s.CheckStatus = status
	s.ErrorMsg = errorMsg
	m.WorkerMap[workerUUID] = s
}

// SetCheckPool sets which checks for a specified backend are applicable
func (m *Manager) SetCheckPool(nodeUUID string, poolName string, backendName string, nodeName string, match string, checks []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if _, ok := m.PoolMap[nodeUUID]; !ok {
		m.PoolMap[nodeUUID] = HealthPool{}
	}
	s := m.PoolMap[nodeUUID]
	s.Checks = checks
	s.PoolName = poolName
	s.BackendName = backendName
	s.NodeName = nodeName
	s.Match = match
	m.PoolMap[nodeUUID] = s
}

// GetNodeStatus returns the combined status of all checks applicable to a specific backend
func (m *Manager) GetNodeStatus(nodeUUID string) (bool, string, string, string, []string) {
	var errors []string
	log := logging.For("core/healthcheck/nodestatus").WithField("func", "healthcheck")
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if pool, ok := m.PoolMap[nodeUUID]; ok {
		ok := 0
		nok := 0
		for _, workerUUID := range pool.Checks {
			if worker, found := m.WorkerMap[workerUUID]; found {
				log.WithField("workeruuid", workerUUID).WithField("checkstatus", worker.CheckStatus).WithField("nodeuuid", nodeUUID).Debug("Status check for node")
				if worker.AdminDown {
					nok++
				} else if worker.AdminUp {
					ok++
				} else if worker.CheckStatus {
					ok++
				} else {
					nok++
				}
				errors = append(errors, worker.ErrorMsg...)
			} else {
				nok++
				log.WithField("workeruuid", workerUUID).WithField("checkstatus", worker.CheckStatus).WithField("nodeuuid", nodeUUID).Debug("Status check for node FAILED")
				errors = append(errors, fmt.Sprintf("Pending health check with worker:%s", workerUUID))
			}
		}
		log.WithField("ok", ok).WithField("nok", nok).WithField("nodeuuid", nodeUUID).WithField("match", pool.Match).WithField("pool", pool.PoolName).WithField("backend", pool.BackendName).WithField("node", pool.NodeName).Debug("Health Status Check")
		if pool.Match == "any" && ok > 0 {
			return true, pool.PoolName, pool.BackendName, pool.NodeName, errors
		}
		if pool.Match == "all" && nok == 0 {
			return true, pool.PoolName, pool.BackendName, pool.NodeName, errors
		}
		return false, pool.PoolName, pool.BackendName, pool.NodeName, errors
	}
	return false, "", "", "", []string{"no healthcheck result recorded yet"}
}

// GetPools returns all nodeUUID's of pools that are linked to a worker
func (m *Manager) GetPools(workerUUID string) (s []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	for nodeUUID, pools := range m.PoolMap {
		for _, worker := range pools.Checks {
			if worker == workerUUID {
				s = append(s, nodeUUID)
			}
		}
	}
	return
}
