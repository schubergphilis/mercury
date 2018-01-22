package cluster

import (
	"fmt"
)

var (
	// LogTraffic true/false to log all traffic sent to and received from nodes
	LogTraffic = false
)

func (m *Manager) log(message string, args ...interface{}) {
	select {
	case m.Log <- fmt.Sprintf(message, args...):
	default:
	}
}
