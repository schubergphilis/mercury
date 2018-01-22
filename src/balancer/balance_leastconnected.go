package balancer

// LeastConnected based loadbalancing interface for statistics
type LeastConnected struct{ statistics }

// Less implements LeastConnected based loadbalancing by sorting based on leastconnected counter
func (s LeastConnected) Less(i, j int) bool {
	return s.statistics[i].ClientsConnected < s.statistics[j].ClientsConnected
}
