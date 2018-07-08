package balancer

import (
	"fmt"
	"net"
)

// Topology Balance based on Topology, this only returns stats where the ip matches the topolology
func Topology(s []Statistics, ip string) []Statistics {
	var matches []Statistics
	for _, stats := range s {

		for _, network := range stats.Topology {
			_, ipnetA, err := net.ParseCIDR(network)
			if err != nil {
				continue
			}
			ipB, _, err := net.ParseCIDR(fmt.Sprintf("%s/32", ip))
			if err != nil {
				continue
			}
			if ipnetA.Contains(ipB) {
				matches = append(matches, stats)
			} else {
			}
		}
	}
	if len(matches) > 1 {
		// keep closest topology match (highest subnet = smallest network)
		closestMatch := 0
		matchresult := 0
		for id, match := range matches {
			for _, network := range match.Topology {
				_, ipnetA, err := net.ParseCIDR(network)
				if err != nil {
					continue
				}
				prefixSize, _ := ipnetA.Mask.Size()
				if closestMatch < prefixSize {
					closestMatch = prefixSize
					matchresult = id
				}
			}
		}
		return []Statistics{matches[matchresult]}
	}
	if len(matches) == 1 {
		return matches
	}
	return s
}
