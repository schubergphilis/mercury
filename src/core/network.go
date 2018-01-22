package core

import (
	"github.com/schubergphilis/mercury/src/config"
	"github.com/schubergphilis/mercury/src/logging"
	"github.com/schubergphilis/mercury/src/network"
)

// Netlink would be nice, but only works on linux, which makes testing on osx crap
//https://godoc.org/github.com/vishvananda/netlink
//import "github.com/vishvananda/netlink"

// CreateListeners for the pools
func CreateListeners() {
	log := logging.For("core/network/create")
	if config.Get().Settings.ManageNetworkInterfaces == "no" {
		log.WithField("manage_network_interfaces", "no").Info("Skipping binging the VIP's to the network interface")
		return
	}
	for poolName, pool := range config.Get().Loadbalancer.Pools {
		clog := log.WithField("ip", pool.Listener.IP).WithField("pool", poolName).WithField("interface", pool.Listener.Interface)
		clog.Debug("Binding IP")
		err := network.CreateListener(pool.Listener.Interface, pool.Listener.IP)
		if err != nil {
			log.WithField("error", err).Fatal("Error binding IP")
		}
	}
}

// RemoveListeners for the pools
func RemoveListeners() {
	log := logging.For("core/network/remove")
	if config.Get().Settings.ManageNetworkInterfaces == "no" {
		log.WithField("manage_network_interfaces", "no").Info("Skipping binging the VIP's to the network interface")
		return
	}
	for poolName, pool := range config.Get().Loadbalancer.Pools {
		clog := log.WithField("ip", pool.Listener.IP).WithField("pool", poolName).WithField("interface", pool.Listener.Interface)
		clog.Debug("Removing IP")
		err := network.RemoveListener(pool.Listener.Interface, pool.Listener.IP)
		if err != nil {
			log.WithField("error", err).Fatal("Error removing IP")
		}
	}
}
