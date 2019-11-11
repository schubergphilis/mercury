package proxy

import (
	"fmt"
	"net"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// processHeader calls the correct handler when editing headers
func (acl ACL) processCIDR(addr string) (match bool) {
	log := logging.For("proxy/processcidr")
	clientAddr := stringToClientIP(addr)

	for _, network := range acl.CIDRS {
		_, ipnetA, err := net.ParseCIDR(network)
		if err != nil {
			log.Printf("Error parsing CIRD:%s error: %s\n", network, err)
		}

		ipB, _, err := net.ParseCIDR(fmt.Sprintf("%s/32", clientAddr.IP))
		if err != nil {
			log.Printf("Error parsing CIRD:%s error: %s\n", fmt.Sprintf("%s/32", clientAddr.IP), err)
		}

		if ipnetA.Contains(ipB) {
			return true
		}
	}
	return false
}
