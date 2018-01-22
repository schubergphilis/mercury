package network

import (
	"fmt"
	"net"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// Iface is a interface that can have ip's
type Iface struct {
	Ipv4 []Address
	Ipv6 []Address
}

// Address contins the ip + netmask
type Address struct {
	IP      string
	Netmask int
}

// listenerAvailable checks if an ip is already attached to the system
func listenerAvailable(ip string) (bool, error) {
	log := logging.For("network/listeneravailable")
	config, err := getConfig()
	if err != nil {
		return false, err
	}
	for iface, addrs := range config {
		for _, addr := range addrs.Ipv4 {
			//log.Debugf("Interface: %s Address: %+v", iface, addr)

			if addr.IP == ip {
				log.WithField("interface", iface).WithField("ip", addr.IP).Debug("IPV4 found on interface")
				return true, nil
			}
		}
	}
	return false, nil
}

// Find the correct interface for string
func getInterface(ip string) (string, error) {
	log := logging.For("network/getinterface")
	config, err := getConfig()
	if err != nil {
		return "", err
	}
	for iface, addrs := range config {
		for _, addr := range addrs.Ipv4 {
			//log.Debugf("Interface: %s Address: %+v", iface, addr)
			_, ipnetA, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", addr.IP, addr.Netmask))
			ipB, _, _ := net.ParseCIDR(fmt.Sprintf("%s/32", ip))
			if ipnetA.Contains(ipB) {
				log.WithField("interface", iface).WithField("ip", addr.IP).Debug("Found matching interface for IP")
				return iface, nil
			}
		}
	}
	return "", fmt.Errorf("Could not detect what interface ip %s should be added to, please specify them by adding the `interface=\"eth1\"` configuration option to the listener.", ip)
}

// CreateListener Creates a listener
func CreateListener(iface, ip string) error {
	// Get interface if needed
	var err error
	if iface == "" {
		iface, err = getInterface(ip)
		if err != nil {
			return err
		}
	}
	// check if we already assigned the ip
	ready, err := listenerAvailable(ip)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}
	// Add ip to interface
	err = ifaceAdd(iface, ip)
	if err != nil {
		return err
	}
	return nil
}

// RemoveListener Removes a listener
func RemoveListener(iface, ip string) error {
	// Get interface if needed
	var err error
	if iface == "" {
		iface, err = getInterface(ip)
		if err != nil {
			return err
		}
	}
	// check if we already removed the ip
	ready, err := listenerAvailable(ip)
	if err != nil {
		return err
	}
	// Remove ip from interface
	if ready {
		err := ifaceRemove(iface, ip)
		if err != nil {
			return err
		}
	}
	return nil
}
