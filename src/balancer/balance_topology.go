package balancer

import (
	"fmt"
	"log"
	"net"
)

// TODO: return and check error in upper function

// Topology Balance based on Topology, this only returns stats where the ip matches the topolology
func Topology(s []Statistics, ip string) []Statistics {
	var matches []Statistics
	for _, stats := range s {
		for _, network := range stats.Topology {
			_, ipnetA, err := net.ParseCIDR(network)
			if err != nil {
				log.Printf("Error parsing CIRD:%s error: %s\n", network, err)
			}
			ipB, _, err := net.ParseCIDR(fmt.Sprintf("%s/32", ip))
			if err != nil {
				log.Printf("Error parsing CIRD:%s error: %s\n", fmt.Sprintf("%s/32", ip), err)
			}
			if ipnetA.Contains(ipB) {
				matches = append(matches, stats)
			}
		}
	}
	if len(matches) > 0 {
		return matches
	}
	return s
}
