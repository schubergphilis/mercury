package healthcheck

import (
	"fmt"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// HealthStatus keeps track of the status of each workers health check
type HealthStatus struct {
	CheckStatus  Status   `json:"checkstatus" toml:"checkstatus"`   // map[checkuuid]status - status returned by worker
	ManualStatus Status   `json:"manualstatus" toml:"manualstatus"` // manual override
	ErrorMsg     []string `json:"errormsg" toml:"errormsg"`         // error message
}

// HealthPool contains a per nodeuuid information about all checks that apply to this node
type HealthPool struct {
	PoolName    string   `json:"poolname" toml:"poolname"`       // name of the vip pool
	BackendName string   `json:"backendname" toml:"backendname"` // name of the backend
	NodeName    string   `json:"nodename" toml:"nodename"`       // name of the node
	Match       string   `json:"match" toml:"match"`             // all/any
	Checks      []string `json:"checks" toml:"checks"`           // []checkuuid
}

// SetCheckStatus sets the status of a worker check based on the health check result
func (m *Manager) SetCheckStatus(workerUUID string, status Status, errorMsg []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if _, ok := m.HealthStatusMap[workerUUID]; !ok {
		m.HealthStatusMap[workerUUID] = HealthStatus{}
	}
	s := m.HealthStatusMap[workerUUID]
	s.CheckStatus = status
	if s.CheckStatus != Online {
		s.ErrorMsg = errorMsg
	} else {
		s.ErrorMsg = []string{}
	}
	m.HealthStatusMap[workerUUID] = s
}

// SetCheckPool sets which checks for a specified backend are applicable
func (m *Manager) SetCheckPool(nodeUUID string, poolName string, backendName string, nodeName string, match string, checks []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if _, ok := m.HealthPoolMap[nodeUUID]; !ok {
		m.HealthPoolMap[nodeUUID] = HealthPool{}
	}
	s := m.HealthPoolMap[nodeUUID]
	s.Checks = checks
	s.PoolName = poolName
	s.BackendName = backendName
	s.NodeName = nodeName
	s.Match = match
	m.HealthPoolMap[nodeUUID] = s
}

// GetNodeStatus returns the combined status of all checks applicable to a specific backend
func (m *Manager) GetNodeStatus(nodeUUID string) (Status, string, string, string, []string) {
	var errors []string
	log := logging.For("core/healthcheck/nodestatus").WithField("func", "healthcheck")
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if pool, ok := m.HealthPoolMap[nodeUUID]; ok {
		ok := 0
		nok := 0
		maintenance := 0
		for _, workerUUID := range pool.Checks {
			if worker, found := m.HealthStatusMap[workerUUID]; found {
				log.WithField("workeruuid", workerUUID).WithField("checkstatus", worker.CheckStatus).WithField("nodeuuid", nodeUUID).Debug("Status check for node")
				switch worker.ManualStatus {
				case Online:
					ok++
				case Offline:
					nok++
				case Maintenance:
					maintenance++
				default: // (Automatic)
					switch worker.CheckStatus {
					case Online:
						ok++
					case Offline:
						nok++
					case Maintenance:
						maintenance++
					}
				}

				errors = append(errors, worker.ErrorMsg...)
			} else {
				nok++
				log.WithField("workeruuid", workerUUID).WithField("checkstatus", worker.CheckStatus).WithField("nodeuuid", nodeUUID).Debug("Status check for node FAILED")
				errors = append(errors, fmt.Sprintf("Pending health check with worker:%s", workerUUID))
			}
		}

		log.WithField("ok", ok).WithField("nok", nok).WithField("maintenance", maintenance).WithField("nodeuuid", nodeUUID).WithField("match", pool.Match).WithField("pool", pool.PoolName).WithField("backend", pool.BackendName).WithField("node", pool.NodeName).Debug("Health Status Check")
		if maintenance > 0 {
			return Maintenance, pool.PoolName, pool.BackendName, pool.NodeName, errors
		}

		if pool.Match == "any" && ok > 0 {
			return Online, pool.PoolName, pool.BackendName, pool.NodeName, errors
		}

		if pool.Match == "all" && nok == 0 {
			return Online, pool.PoolName, pool.BackendName, pool.NodeName, errors
		}

		return Offline, pool.PoolName, pool.BackendName, pool.NodeName, errors
	}

	return Offline, "", "", "", []string{"no healthcheck result recorded yet"}
}

// GetPools returns all nodeUUID's of pools that are linked to a worker
func (m *Manager) GetPools(workerUUID string) (s []string) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	for nodeUUID, pools := range m.HealthPoolMap {
		for _, worker := range pools.Checks {
			if worker == workerUUID {
				s = append(s, nodeUUID)
			}
		}
	}

	return
}
