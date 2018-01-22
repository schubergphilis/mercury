package cluster

import (
	"log"
	"time"
)

// Settings contains the adjustable setting for the cluster
type Settings struct {
	PingInterval    time.Duration // how over to ping a node
	JoinDelay       time.Duration // delay before announcing node (done to prevent duplicate join messages on simultainious connects) (must be shorter than ping timeout)
	ReadTimeout     time.Duration // timeout when to discard a node as broken if not read anything before this
	ConnectInterval time.Duration // how often we try to reconnect to lost cluster nodes
	ConnectTimeout  time.Duration // how long to try to connect to a node
}

func defaultSetting() Settings {
	s := Settings{
		PingInterval:    5 * time.Second,
		JoinDelay:       500 * time.Millisecond,
		ReadTimeout:     11 * time.Second,
		ConnectInterval: 2 * time.Second,
		ConnectTimeout:  10 * time.Second,
	}
	return s
}

// UpdateSettings allows you to update a running cluster node with new settings
func (m *Manager) UpdateSettings(settings Settings) {
	m.Lock()
	defer m.Unlock()
	m.settings = settings
}

func (m *Manager) getDuration(setting string) time.Duration {
	m.RLock()
	defer m.RUnlock()
	switch setting {
	case "pinginterval":
		return m.settings.PingInterval
	case "joindelay":
		return m.settings.JoinDelay
	case "connecttimeout":
		return m.settings.ConnectTimeout
	case "connectinterval":
		return m.settings.ConnectInterval
	case "readtimeout":
		return m.settings.ReadTimeout
	default:
		log.Fatalf("Unknown setting: %s", setting)
		return 0
	}
}
