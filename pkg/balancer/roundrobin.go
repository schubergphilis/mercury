package balancer

// RoundRobin sorts records by Round robin
type RoundRobin struct{ statistics }

// Less implements RoundRobin based loadbalancing by sorting based on selected counter
func (s RoundRobin) Less(i, j int) bool {
	return s.statistics[i].ClientsConnects < s.statistics[j].ClientsConnects
}
