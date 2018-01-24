package balancer

import (
	"fmt"
	"sort"
	"strings"
)

// Implements sort mechanisms
type statistics []Statistics

func (s statistics) Len() int      { return len(s) }
func (s statistics) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Sort sorts statistics based on value.
// ID can be a IP for ip based loadbalancing.
// ID van be sessionID for stickyness based loadbalancing.
func Sort(s []Statistics, ip string, sticky string, mode string) ([]Statistics, error) {
	switch mode {
	case "roundrobin":
		sort.Sort(RoundRobin{s})
	case "preference":
		sort.Sort(Preference{s})
	case "leastconnected":
		sort.Sort(LeastConnected{s})
	case "leasttraffic":
		sort.Sort(LeastTraffic{s})
	case "responsetime":
		sort.Sort(ResponseTime{s})
	case "topology":
		s = Topology(s, ip)
	case "sticky":
		s = Sticky(s, sticky)
	case "firstavailable":
		s = FirstAvailable(s)
	case "random":
		s = Random(s)
	default:
		return s, fmt.Errorf("Unknown balance mode: %s", mode)
	}
	return s, nil
}

// MultiSort sorts statistics based on multiple modes
func MultiSort(s []Statistics, ip string, sticky string, mode string) ([]Statistics, error) {
	modes := reverse(strings.Split(mode, ","))
	var err error
	for _, m := range modes {
		s, err = Sort(s, ip, sticky, m)
		if err != nil {
			return s, err
		}
	}
	return s, nil
}

// reverse an array of strings
func reverse(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
