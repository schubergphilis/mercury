package healthcheck

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/tlsconfig"
)

// Manager manages Health of Node
type Manager struct {
	Incoming  chan CheckResult
	Workers   []*Worker
	WorkerMap map[string]HealthStatus // keeps the health of all items
	PoolMap   map[string]HealthPool   // keeps a list of uuids and what checks apply to them

	Worker sync.RWMutex
}

// CheckResult holds the check result output
type CheckResult struct {
	PoolName    string
	BackendName string
	NodeName    string
	NodeUUID    string
	WorkerUUID  string
	Description string
	Online      bool
	ErrorMsg    []string
	SingleCheck bool
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
