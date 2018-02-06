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
	Incoming  chan CheckResult
	Workers   []*Worker               `json:"workers" toml:"workers"`
	WorkerMap map[string]HealthStatus `json:"workermap" toml:"workermap"` // keeps the health of all items
	PoolMap   map[string]HealthPool   `json:"poolmap" toml:"poolmap"`     // keeps a list of uuids and what checks apply to them

	Worker sync.RWMutex
}

// CheckResult holds the check result output
type CheckResult struct {
	PoolName    string   `json:"poolname" toml:"poolname"`
	BackendName string   `json:"backendname" toml:"backendname"`
	NodeName    string   `json:"nodename" toml:"nodename"`
	NodeUUID    string   `json:"nodeuuid" toml:"nodeuuid"`
	WorkerUUID  string   `json:"workeruuid" toml:"workeruuid"`
	Description string   `json:"description" toml:"description"`
	Online      bool     `json:"online" toml:"online"`
	ErrorMsg    []string `json:"errormsg" toml:"errormsg"`
	SingleCheck bool     `json:"singlecheck" toml:"singlecheck"`
}

// HealthCheck custom HealthCheck
type HealthCheck struct {
	Type             string              `json:"type" toml:"type"`
	TCPRequest       string              `json:"tcprequest" toml:"tcprequest"`
	TCPReply         string              `json:"tcpreply" toml:"tcpreply"`
	HTTPRequest      string              `json:"httprequest" toml:"httprequest"`
	HTTPPostData     string              `json:"httppostdata" toml:"httppostdata"`
	HTTPHeaders      []string            `json:"httpheaders" toml:"httpheaders"`
	HTTPStatus       int                 `json:"httpstatus" toml:"httpstatus"`
	HTTPReply        string              `json:"httpreply" toml:"httpreply"`
	PINGpackets      int                 `json:"pingpackets" toml:"pingpackets"`
	PINGtimeout      int                 `json:"pingtimeout" toml:"pingtimeout"`
	Interval         int                 `json:"interval" toml:"interval"`
	Timeout          int                 `json:"timeout" toml:"timeout"`
	ActivePassiveID  string              `json:"activepassiveid" toml:"activepassiveid"` // used to link active/passive backends
	TLSConfig        tlsconfig.TLSConfig `json:"tls" toml:"tls"`
	DisableAutoCheck bool                `json:"disableautocheck" toml:"disableautocheck"` // only respond to check requests
	IP               string              `json:"ip" toml:"ip"`                             // specific ip
	SourceIP         string              `json:"sourceip" toml:"sourceip"`                 // specific ip
	Port             int                 `json:"port" toml:"port"`                         // specific port
	uuidStr          string
}

// UUID returns a uuid of a healthcheck
func (h HealthCheck) UUID() string {
	if h.uuidStr != "" {
		return h.uuidStr
	}

	sort.Strings(h.HTTPHeaders)
	s := fmt.Sprintf("%s%s%s%s%s%v%d%s%d%d%s%t", h.Type, h.TCPRequest, h.TCPReply, h.HTTPRequest, h.HTTPPostData, h.HTTPHeaders, h.HTTPStatus, h.HTTPReply, h.Interval, h.Timeout, h.ActivePassiveID, h.DisableAutoCheck)
	t := sha256.New()
	t.Write([]byte(s))
	h.uuidStr = fmt.Sprintf("%x", t.Sum(nil))
	return h.uuidStr
}

// Debug shows debug output for all workers
func (m *Manager) Debug() {
	for _, worker := range m.Workers {
		worker.Debug()
	}

	for wid, wm := range m.WorkerMap {
		fmt.Printf("Workermap -> worker:%s status:%+v\n", string(wid), wm)
	}

	for pmid, pm := range m.PoolMap {
		fmt.Printf("Poolmap -> node:%s map:%v\n", string(pmid), pm)
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
		Incoming:  make(chan CheckResult),
		WorkerMap: make(map[string]HealthStatus),
		PoolMap:   make(map[string]HealthPool),
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
	tmp.WorkerHealth = m.WorkerMap
	tmp.NodeMap = m.PoolMap
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
		if w.UuidStr == uuid {
			tmp.Workers = *w
		}
	}
	if _, ok := m.WorkerMap[uuid]; ok {
		tmp.WorkerHealth = m.WorkerMap[uuid]
	}
	for _, node := range m.PoolMap {
		fmt.Printf("NodeMap: %+v\n", node)
		for _, p := range node.Checks {
			if p == uuid {
				tmp.NodeMap = append(tmp.NodeMap, node.NodeName)
			}
		}
	}
	result, err := json.Marshal(tmp)
	return result, err
}
