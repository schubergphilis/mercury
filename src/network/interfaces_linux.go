// +build linux

package network

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/schubergphilis/mercury/src/logging"
)

// IfAdd adds a ip to an interface
func ifaceAdd(iface, ip string) error {
	if err := exec.Command("ip", "addr", "add", ip+"/32", "dev", iface).Run(); err != nil {
		return err
	}
	return nil
}

// IfRemove removes a ip from an interface
func ifaceRemove(iface, ip string) error {
	if err := exec.Command("ip", "addr", "del", ip+"/32", "dev", iface).Run(); err != nil {
		return err
	}
	return nil
}

/* Example ip add list output on linux
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
*/

// getConfig parses the output of ifconfig and adds it to the nerwork.Config
func getConfig() (map[string]Iface, error) {
	out, err := exec.Command("ip", "addr", "list").Output()
	if err != nil {
		return nil, err
	}
	oldiface := ""
	newiface := Iface{}
	ifaces := make(map[string]Iface)

	log := logging.For("network/linux/config")
	ire := regexp.MustCompile("^[0-9]+:[[:space:]]+([a-z0-9]+):")
	ipv4re := regexp.MustCompile("[[:space:]]+inet ([0-9]+.[0-9]+.[0-9]+.[0-9]+)/([0-9]+)[[:space:]]")
	for _, line := range strings.Split(string(out), "\n") {
		iface := ire.FindStringSubmatch(line)
		if len(iface) > 0 {
			// found new interface
			if oldiface != iface[1] && oldiface != "" {
				//log.Debug("Found ip(s):%+v for interface:%s", oldiface, newiface)
				ifaces[oldiface] = newiface
			}
			//log.Debugf("Found Interface:%s", iface[1])

			newiface = Iface{}
			oldiface = iface[1]
		}
		ipv4 := ipv4re.FindStringSubmatch(line)
		if len(ipv4) > 0 {
			// found new ip for interface
			log.WithField("interface", ipv4[1]).Debug("IPV4 found on interface")
			netmask, _ := strconv.Atoi(ipv4[2])
			addr := Address{IP: ipv4[1], Netmask: netmask}
			newiface.Ipv4 = append(newiface.Ipv4, addr)
		}
	}
	ifaces[oldiface] = newiface
	//log.Infof("All interfaces discovered:%+v", ifaces)
	return ifaces, nil
}
