package core

import (
	"fmt"
	"sync"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/dns"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// DNSHandler handles the DNS
func (manager *Manager) DNSHandler() {
	log := logging.For("core/dns/handler").WithField("func", "dns")
	log.Debug("Starting DNS Handler")
	for {
		select {
		case dnsupdate := <-manager.dnsupdates:
			log.Debugf("dnsupdates")

			//log.Debugf("Got: %+v", dnsupdate)
			stats := &balancer.Statistics{
				UUID:       dnsupdate.BackendUUID,
				Preference: dnsupdate.BalanceMode.Preference,
				Topology:   dnsupdate.BalanceMode.LocalNetwork,
				RWMutex:    new(sync.RWMutex),
			}
			record := dns.Record{
				Name:          dnsupdate.DNSEntry.HostName,
				TTL:           config.Get().DNS.Domains[dnsupdate.DNSEntry.Domain].TTL,
				BalanceMode:   dnsupdate.BalanceMode.Method,
				ActivePassive: dnsupdate.BalanceMode.ActivePassive,
				ClusterNodes:  dnsupdate.BalanceMode.ClusterNodes,
				Statistics:    stats,
				UUID:          dnsupdate.BackendUUID,
				Online:        dnsupdate.Online,
			}
			// TODO: pass record type along, and get rid of ipv6/ipv4 seperation
			clog := log.WithField("hostname", dnsupdate.DNSEntry.HostName).WithField("domain", dnsupdate.DNSEntry.Domain).WithField("cluster", dnsupdate.ClusterNode).WithField("backend", dnsupdate.BackendName).WithField("uuid", dnsupdate.BackendUUID).WithField("online", dnsupdate.Online)
			// Create IPv4 record if present
			if dnsupdate.DNSEntry.IP != "" {
				record.Type = "A"
				record.Target = dnsupdate.DNSEntry.IP
				//log.Infof("DNS Update for:%s.%s type:%s record:%s balancemode:%s online:%s uuid:%s", dnsupdate.DNSEntry.HostName, dnsupdate.DNSEntry.Domain, "A", dnsupdate.DNSEntry.IP, dnsupdate.BalanceMode.Method, dnsupdate.Online, dnsupdate.BackendUUID)
				clog.WithField("target", dnsupdate.DNSEntry.IP).Debug("Received DNS update from cluster")
				dns.Update(dnsupdate.ClusterNode, dnsupdate.DNSEntry.Domain, record)
			}
			// Create IPv6 record if present
			if dnsupdate.DNSEntry.IP6 != "" {
				record.Type = "AAAA"
				record.Target = dnsupdate.DNSEntry.IP6
				//log.Infof("DNS Update for:%s.%s type:%s record:%s balancemode:%s online:%s uuid:%s", dnsupdate.DNSEntry.HostName, dnsupdate.DNSEntry.Domain, "AAAA", dnsupdate.DNSEntry.IP6, dnsupdate.BalanceMode.Method, dnsupdate.Online, dnsupdate.BackendUUID)
				clog.WithField("target", dnsupdate.DNSEntry.IP6).Debug("Received DNS update from cluster")
				dns.Update(dnsupdate.ClusterNode, dnsupdate.DNSEntry.Domain, record)
			}
			log.Debugf("dnsupdates OK")
		case dnsStatistics := <-manager.clusterGlbalDNSStatisticsUpdate:
			log.Debugf("clusterGlbalDNSStatisticsUpdate")

			log.Debugf("Received DNS statistics from DNS manager")
			stats := balancer.NewStatistics(dnsStatistics.UUID, 1)
			stats.ClientsConnectedSet(dnsStatistics.ClientsConnected)
			stats.ClientsConnectsSet(dnsStatistics.ClientsConnects)
			stats.RXAdd(dnsStatistics.RX)
			stats.TXAdd(dnsStatistics.TX)
			stats.ResponseTimeValue = dnsStatistics.ResponseTimeValue
			dns.UpdateStatistics(dnsStatistics.ClusterNode, dnsStatistics.DNSEntry.Domain, stats)
			log.Debugf("clusterGlbalDNSStatisticsUpdate OK")
		case node := <-manager.dnsdiscard:
			log.Debugf("dnsdiscard")
			dns.Discard(node)
			log.Debugf("dnsdiscard OK")
		case node := <-manager.dnsoffline:
			log.Debugf("dnsoffline")
			dns.MarkOffline(node)
			log.Debugf("dnsoffline OK")
		}
	}
}

// InitializeDNSUpdates manages DNS records
func (manager Manager) InitializeDNSUpdates() {
	log := logging.For("core/dnsinit").WithField("func", "dns")
	log.Debug("Initializing DNS Updates")
	go manager.DNSHandler()
	UpdateDNSConfig()
	dns.EnableProxyStats(config.Get().Settings.EnableProxy == YES)
	log.Debug("Initializing DNS Updates OK")
}

// StartDNSServer starts the dns server
func (manager Manager) StartDNSServer() {
	go dns.Server(config.Get().DNS.Binding, config.Get().DNS.Port, config.Get().DNS.AllowedRequests)
}

// UpdateDNSConfig adds new records, and removes obsolete records
func UpdateDNSConfig() {
	log := logging.For("core/updatednsconfig").WithField("func", "dns")
	log.WithField("hosts", fmt.Sprintf("%v", config.Get().DNS.AllowForwarding)).Info("Initializing DNS Forwarder")
	dns.AllowForwarding(config.Get().DNS.AllowForwarding)

	log.Info("Initializing DNS Config Updates")
	// Loop through all manual entries in the config
	for domainName, domain := range config.Get().DNS.Domains {
		// Get all current records
		allRecords := dns.GetAllLocalDomainRecords(domainName)

		// Put them in the old records array, and loop through them
		var oldRecords []dns.Record
		for _, record := range allRecords {
			oldRecords = append(oldRecords, record)
		}
		//log.WithField("domain", domainName).WithField("records", oldRecords).Debug("Records before reload")

		// Add Records
		for _, record := range domain.Records {
			existing := 0
			// remove this record from the list to remove
			for i := len(oldRecords) - 1; i >= 0; i-- {
				if oldRecords[i].Name == record.Name &&
					oldRecords[i].TTL == record.TTL &&
					oldRecords[i].Target == record.Target &&
					oldRecords[i].Type == record.Type {
					log.Debug("Existing DNS record:%v,  marking as not to be removed", record)

					oldRecords = append(oldRecords[:i], oldRecords[i+1:]...)
					existing++
				}
			}

			// we have a new record, so add it
			if existing == 0 {
				log.WithField("domain", domainName).WithField("hostname", record.Name).WithField("target", record.Target).WithField("uuid", record.UUID).WithField("type", record.Type).Info("Adding new DNS record")
				dns.AddLocalRecord(domainName, record)
			}

		}

		// Remove obsolete records
		for _, record := range oldRecords {
			log.WithField("domain", domainName).WithField("hostname", record.Name).WithField("target", record.Target).WithField("uuid", record.UUID).WithField("type", record.Type).Info("Removing old DNS record")
			dns.RemoveLocalRecordByContent(domainName, record.Name, record.TTL, record.Target, record.Type)
		}

		// debugging
		//oldRecords = dns.GetAllLocalDomainRecords(domainName)
		//log.WithField("domain", domainName).WithField("records", oldRecords).Debug("Records after reload")

	}
	log.Info("Initializing DNS Config Updates OK")
}
