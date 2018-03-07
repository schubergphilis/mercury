package healthcheck

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

// Manager manages Health of Node
type Manager struct {
	Incoming        chan CheckResult
	Workers         []*Worker               `json:"workers" toml:"workers"`
	HealthStatusMap map[string]HealthStatus `json:"healthstatusmap" toml:"healthstatusmap"` // keeps the health of all items
	HealthPoolMap   map[string]HealthPool   `json:"healthpoolmap" toml:"healthpoolmap"`     // keeps a list of uuids and what checks apply to them

	Worker sync.RWMutex
}

// CheckResult holds the check result output
type CheckResult struct {
	PoolName       string   `json:"poolname" toml:"poolname"`             // pool this check belongs to
	BackendName    string   `json:"backendname" toml:"backendname"`       // backend this check belongs to
	NodeName       string   `json:"nodename" toml:"nodename"`             // node name of the check
	NodeUUID       string   `json:"nodeuuid" toml:"nodeuuid"`             // node uuid for the check
	WorkerUUID     string   `json:"workeruuid" toml:"workeruuid"`         // worker uuid who performed the check
	Description    string   `json:"description" toml:"description"`       // description of the check (?)
	ActualStatus   Status   `json:"actualstatus" toml:"actualstatus"`     // status of the check as it is performed
	ReportedStatus Status   `json:"reportedstatus" toml:"reportedstatus"` // status of the check after applying state processing
	ErrorMsg       []string `json:"errormsg" toml:"errormsg"`             // error message if any
}

// HealthCheck custom HealthCheck
type HealthCheck struct {
	Type             string              `json:"type" toml:"type"`                         // check type
	TCPRequest       string              `json:"tcprequest" toml:"tcprequest"`             // tcp request to send
	TCPReply         string              `json:"tcpreply" toml:"tcpreply"`                 // tcp reply to expect
	HTTPRequest      string              `json:"httprequest" toml:"httprequest"`           // http request to send
	HTTPPostData     string              `json:"httppostdata" toml:"httppostdata"`         // http post data to send
	HTTPHeaders      []string            `json:"httpheaders" toml:"httpheaders"`           // http headers to send
	HTTPStatus       int                 `json:"httpstatus" toml:"httpstatus"`             // http status expected
	HTTPReply        string              `json:"httpreply" toml:"httpreply"`               // http reply expected
	PINGpackets      int                 `json:"pingpackets" toml:"pingpackets"`           // ping packets to send
	PINGtimeout      int                 `json:"pingtimeout" toml:"pingtimeout"`           // ping timeout
	Interval         int                 `json:"interval" toml:"interval"`                 // how often to cechk
	Timeout          int                 `json:"timeout" toml:"timeout"`                   // timeout performing check
	ActivePassiveID  string              `json:"activepassiveid" toml:"activepassiveid"`   // used to link active/passive backends
	TLSConfig        tlsconfig.TLSConfig `json:"tls" toml:"tls"`                           // tls config
	DisableAutoCheck bool                `json:"disableautocheck" toml:"disableautocheck"` // only respond to check requests
	IP               string              `json:"ip" toml:"ip"`                             // specific ip
	SourceIP         string              `json:"sourceip" toml:"sourceip"`                 // specific ip
	Port             int                 `json:"port" toml:"port"`                         // specific port
	OnlineState      StatusType          `json:"online_state" toml:"online_state"`         // alternative online_state - default: online / optional: offline / maintenance
	OfflineState     StatusType          `json:"offline_state" toml:"offline_state"`       // alternative offline_state - default: offline
	uuidStr          string
}

// UUID returns a uuid of a healthcheck
func (h HealthCheck) UUID() string {
	if h.uuidStr != "" {
		return h.uuidStr
	}

	sort.Strings(h.HTTPHeaders)
	s := fmt.Sprintf("%s%s%s%s%s%v%d%s%d%d%s%t%s%s", h.Type, h.TCPRequest, h.TCPReply, h.HTTPRequest, h.HTTPPostData, h.HTTPHeaders, h.HTTPStatus, h.HTTPReply, h.Interval, h.Timeout, h.ActivePassiveID, h.DisableAutoCheck, h.OfflineState, h.OnlineState)
	t := sha256.New()
	t.Write([]byte(s))
	h.uuidStr = fmt.Sprintf("%x", t.Sum(nil))
	return h.uuidStr
}

// Debug shows debug output for all workers
func (m *Manager) Debug() {
	log := logging.For("healthcheck/worker/debug")
	for _, worker := range m.Workers {
		worker.Debug()
	}

	for wid, wm := range m.HealthStatusMap {
		log.Infof("Workermap -> worker:%s status:%+v\n", string(wid), wm)
	}

	for pmid, pm := range m.HealthPoolMap {
		log.Infof("Poolmap -> node:%s map:%v\n", string(pmid), pm)
	}
}

// StopWorker stops and removes a worker
func (m *Manager) StopWorker(id int) {
	m.Workers[id].Stop()
	m.Worker.Lock()
	defer m.Worker.Unlock()
	m.Workers = append(m.Workers[:id], m.Workers[id+1:]...)
}

// RegisterWorker adds the worker to the health manager
func (m *Manager) RegisterWorker(w *Worker) {
	log := logging.For("healthcheck/worker/register")
	//log.Debugf("Registering worker: %+v", w)
	log.WithField("pool", w.Pool).WithField("backend", w.Backend).WithField("ip", w.IP).WithField("port", w.Port).WithField("node", w.NodeName).Info("Adding new healthcheck")
	m.Worker.Lock()
	defer m.Worker.Unlock()
	m.Workers = append(m.Workers, w)
}

// NewManager creates a new healtheck manager
func NewManager() *Manager {
	manager := &Manager{
		Incoming:        make(chan CheckResult),
		HealthStatusMap: make(map[string]HealthStatus),
		HealthPoolMap:   make(map[string]HealthPool),
	}

	return manager
}

// StartWorkers starts the workers to do the checking
func (m *Manager) StartWorkers() {
	for _, worker := range m.Workers {
		worker.Start()
	}
}

// StopWorkers stops the workers
func (m *Manager) StopWorkers() {
	for _, worker := range m.Workers {
		worker.Stop()
	}

	m.Worker.Lock()
	defer m.Worker.Unlock()
	m.Workers = []*Worker{}
}

// JSON returns the healtheck status of the manager in json format
func (m *Manager) JSON() ([]byte, error) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	tmp := struct {
		Workers      []Worker                `json:"workers" toml:"workers"`           // all workers that do health checks
		WorkerHealth map[string]HealthStatus `json:"workerhealth" toml:"workerhealth"` // health status for each worker
		NodeMap      map[string]HealthPool   `json:"nodemap" toml:"nodemap"`           // map of node ID, and their healthchecks
	}{}
	for _, w := range m.Workers {
		tmp.Workers = append(tmp.Workers, w.filterWorker())
	}
	tmp.WorkerHealth = m.HealthStatusMap
	tmp.NodeMap = m.HealthPoolMap
	result, err := json.Marshal(tmp)
	return result, err
}

// JSONAuthorized returns unfiltered the healtheck status of the manager in json format
func (m *Manager) JSONAuthorized(uuid string) ([]byte, error) {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	tmp := struct {
		Workers      Worker       `json:"worker" toml:"worker"`             // all workers that do health checks
		WorkerHealth HealthStatus `json:"workerhealth" toml:"workerhealth"` // health status for each worker
		NodeMap      []string     `json:"nodemap" toml:"nodemap"`           // map of node ID, and their healthchecks
	}{}
	for _, w := range m.Workers {
		if w.UUIDStr == uuid {
			tmp.Workers = *w
		}
	}
	if _, ok := m.HealthStatusMap[uuid]; ok {
		tmp.WorkerHealth = m.HealthStatusMap[uuid]
	}
	for _, node := range m.HealthPoolMap {
		for _, p := range node.Checks {
			if p == uuid {
				tmp.NodeMap = append(tmp.NodeMap, node.NodeName)
			}
		}
	}
	result, err := json.Marshal(tmp)
	return result, err
}

// SetStatus sets the status of a uuid to status
func (m *Manager) SetStatus(uuid string, status Status) error {
	m.Worker.Lock()
	defer m.Worker.Unlock()
	if node, ok := m.HealthStatusMap[uuid]; ok {
		if _, ok := StatusTypeToString[status]; !ok {
			return fmt.Errorf("unknown status to set: %s", status)
		}
		node.ManualStatus = status
		m.HealthStatusMap[uuid] = node
		m.sendWorkerUpdate(uuid)
		return nil
	}
	return fmt.Errorf("unkown uuid: %s", uuid)
}

func (m *Manager) sendWorkerUpdate(uuid string) {
	for _, worker := range m.Workers {
		if worker.UUIDStr == uuid {
			worker.sendUpdate(worker.CheckResult) // send update with no change
		}
	}
}
