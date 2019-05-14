package healthcheck

import (
	"sync"

	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/models"
)

// Manager manages Health of Node
type Manager struct {
	Incoming chan models.CheckResult
	Workers  map[string]*Worker `json:"workers" toml:"workers"`
	Worker   sync.RWMutex
	log      logging.SimpleLogger
}

// Debug shows debug output for all workers
func (m *Manager) Debug() {
	for _, worker := range m.Workers {
		worker.Debug()
	}
}

// StopWorker stops and removes a worker
func (m *Manager) RemoveHealthcheck(uuid string) {
	m.Workers[uuid].Stop()
	m.Worker.Lock()
	defer m.Worker.Unlock()
	delete(m.Workers, uuid)
	//m.Workers = append(m.Workers[:id], m.Workers[id+1:]...)
}

// RegisterWorker adds the worker to the health manager
func (m *Manager) AddHealthcheck(uuid string, check models.Healthcheck) {
	w := NewWorker(m.log, uuid, check, m.Incoming)
	w.Start()
	m.Worker.Lock()
	defer m.Worker.Unlock()
	m.Workers[uuid] = w
	//m.Workers = append(m.Workers, w)
}

// NewManager creates a new healtheck manager
func NewManager() *Manager {
	manager := &Manager{
		Incoming: make(chan models.CheckResult),
		Workers:  make(map[string]*Worker),
	}

	return manager
}

func (m *Manager) Stop() {
	for uuid := range m.Workers {
		m.RemoveHealthcheck(uuid)
	}
}

func (m *Manager) WithLogger(s logging.SimpleLogger) {
	m.log = s
}

func (m *Manager) ReceiveHealthCheckStatus() chan models.CheckResult {
	return m.Incoming
}

/*
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
*/
