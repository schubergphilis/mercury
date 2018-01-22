package balancer

// LeastTraffic based loadbalancing
type LeastTraffic struct{ statistics }

// Less implements LeastTraffic based loadbalancing by sorting based on leasttraffic counter
func (s LeastTraffic) Less(i, j int) bool {
	return s.statistics[i].RX+s.statistics[i].TX < s.statistics[j].RX+s.statistics[j].TX
}
