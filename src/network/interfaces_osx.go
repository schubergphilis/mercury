// +build darwin

package network

import (
	"encoding/hex"
	"os/exec"
	"regexp"
	"strings"

	"github.com/schubergphilis/mercury/src/logging"
)

// IfAdd adds a ip to an interface
func ifaceAdd(iface, ip string) error {
	if err := exec.Command("ifconfig", iface, "alias", ip).Run(); err != nil {
		return err
	}
	return nil
}

// IfRemove removes a ip from an interface
func ifaceRemove(iface, ip string) error {
	if err := exec.Command("ifconfig", iface, "-alias", ip).Run(); err != nil {
		return err
	}
	return nil
}

/* example ifconfig output on osx
lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384
	options=1203<RXCSUM,TXCSUM,TXSTATUS,SW_TIMESTAMP>
	inet 127.0.0.1 netmask 0xff000000
	inet6 ::1 prefixlen 128
	inet6 fe80::1%lo0 prefixlen 64 scopeid 0x1
*/

// simpleMaskLength convers osx 0xff000000 netmask output to a number
func simpleMaskLength(mask []byte) int {
	var n int
	for i, v := range mask {
		if v == 0xff {
			n += 8
			continue
		}
		// found non-ff byte
		// count 1 bits
		for v&0x80 != 0 {
			n++
			v <<= 1
		}
		// rest must be 0 bits
		if v != 0 {
			return -1
		}
		for i++; i < len(mask); i++ {
			if mask[i] != 0 {
				return -1
			}
		}
		break
	}
	return n
}

// getConfig parses the output of ifconfig and adds it to the nerwork.Config
func getConfig() (map[string]Iface, error) {
	out, err := exec.Command("ifconfig", "-a").Output()
	if err != nil {
		return nil, err
	}
	oldiface := ""
	newiface := Iface{}
	ifaces := make(map[string]Iface)

	log := logging.For("network/osx/config")
	ire := regexp.MustCompile("^([a-z]+[0-9]+)")
	ipv4re := regexp.MustCompile("[[:space:]]+inet ([0-9]+.[0-9]+.[0-9]+.[0-9]+) netmask 0x([a-f0-9]+)")
	for _, line := range strings.Split(string(out), "\n") {
		iface := ire.FindString(line)
		if iface != "" {
			// found new interface
			if oldiface != iface && oldiface != "" {
				//log.Debug("Found ip(s):%+v for interface:%s", oldiface, newiface)
				ifaces[oldiface] = newiface
			}
			//log.Debugf("Found Interface:%s", iface)

			newiface = Iface{}
			oldiface = iface
		}
		ipv4 := ipv4re.FindStringSubmatch(line)
		if len(ipv4) > 0 {
			// found new ip on interface
			log.WithField("interface", ipv4[1]).Debug("IPV4 found on interface")
			hex, _ := hex.DecodeString(ipv4[2])
			netmask := simpleMaskLength(hex)
			addr := Address{IP: ipv4[1], Netmask: netmask}
			newiface.Ipv4 = append(newiface.Ipv4, addr)
		}
	}
	ifaces[oldiface] = newiface
	//log.Infof("All interfaces discovered:%+v", ifaces)
	return ifaces, nil

}
