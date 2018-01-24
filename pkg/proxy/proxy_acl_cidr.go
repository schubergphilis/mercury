package proxy

import (
	"fmt"
	"net"
	"strings"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// processHeader calls the correct handler when editing headers
func (acl ACL) processCIDR(addr string) (match bool) {
	log := logging.For("proxy/processcidr")
	ip := strings.Split(addr, ":")[0]

	for _, network := range acl.CIDRS {
		_, ipnetA, err := net.ParseCIDR(network)
		if err != nil {
			log.Printf("Error parsing CIRD:%s error: %s\n", network, err)
		}

		ipB, _, err := net.ParseCIDR(fmt.Sprintf("%s/32", ip))
		if err != nil {
			log.Printf("Error parsing CIRD:%s error: %s\n", fmt.Sprintf("%s/32", ip), err)
		}

		if ipnetA.Contains(ipB) {
			return true
		}
	}
	return false
}
