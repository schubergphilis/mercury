package balancer

// ResponseTime based loadbalancing interface for statistics
type ResponseTime struct{ statistics }

// Less implements ResponseTime based loadbalancing by sorting based on ResponseTime counter
func (s ResponseTime) Less(i, j int) bool {
	return s.statistics[i].ResponseTimeGet() < s.statistics[j].ResponseTimeGet()
}
