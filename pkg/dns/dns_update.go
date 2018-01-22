package dns

import (
	"net"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// addRecord adds a dns record
func addRecord(node string, domain string, record Record) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	// Add new record
	//log := logging.For("dns/update")
	//log.Debugf("-------> Appending Record: %+v", record)
	// Create cluster node entry if not there yet
	if _, ok := dnsmanager.node[node]; !ok {
		dnsmanager.node[node] = Domains{
			Domains: make(map[string]Domain),
		}
	}

	// Create dns domains entry if not there yet
	if _, ok := dnsmanager.node[node].Domains[domain]; !ok {
		dnsmanager.node[node].Domains[domain] = Domain{
			Records: make([]Record, 0),
			//TTL:        ttl,
		}
	}

	y := dnsmanager.node[node].Domains[domain]
	y.Records = append(y.Records, record)
	dnsmanager.node[node].Domains[domain] = y
}

// updateRecord updates a dns record
func updateRecord(node string, domain string, recordid int, record Record) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	// Update new record
	dnsmanager.node[node].Domains[domain].Records[recordid] = record
}

// removeRecordByUUID removed a dns record
func removeRecordByUUID(node string, uuid string) {
	entries := GetCache()
	for domainName, domain := range entries[node].Domains {
		for recordID, record := range domain.Records {
			if record.UUID == uuid {
				removeRecord(node, domainName, recordID)
			}
		}
	}
}

// removeRecord removed a dns record
func removeRecord(node string, domain string, recordid int) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	// Remove new record
	y := dnsmanager.node[node].Domains[domain]
	y.Records = removeRecordArray(dnsmanager.node[node].Domains[domain].Records, recordid)
	dnsmanager.node[node].Domains[domain] = y
}

func removeRecordArray(s []Record, i int) []Record {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func createNode(name string) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	dnsmanager.node[name] = Domains{
		Domains: make(map[string]Domain),
	}
}

func createDomain(node, domain string) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	dnsmanager.node[node].Domains[domain] = Domain{
		Records: make([]Record, 0),
	}
}

// Update Updates a dns entry in a node
func Update(node string, domain string, record Record) {
	log := logging.For("dns/update").WithField("domain", domain).WithField("cluster", node).WithField("online", record.Online).WithField("name", record.Name).WithField("target", record.Target).WithField("mode", record.BalanceMode).WithField("uuid", record.UUID)
	log.Debug("Received DNS update")
	//log.Debugf("Got DNS update for node:%s domain:%s online:%t record:%+v", node, domain, record.Online, record)

	// Create cluster node entry if not there yet
	if _, ok := dnsmanager.node[node]; !ok {
		createNode(node)
		/*dnsmanager.node[node] = Domains{
			Domains: make(map[string]Domain),
		}*/
	}

	// Create dns domains entry if not there yet
	if _, ok := dnsmanager.node[node].Domains[domain]; !ok {
		createDomain(node, domain)
		/*dnsmanager.node[node].Domains[domain] = Domain{
			Records: make([]Record, 0),
		}*/
	}

	// This happens if there is no config anymore when removing a update, we remove by uuid
	if domain == "" && record.UUID != "" {
		log.Debug("Removing record with UUID")
		removeRecordByUUID(node, record.UUID)
		return
	}

	existingid := -1
	for id, rec := range dnsmanager.node[node].Domains[domain].Records {
		// match based on record name and type, anything else can be updated
		if rec.Name == record.Name && rec.Type == record.Type {
			existingid = id
		}
	}
	if existingid >= 0 {
		// We have an existing record, we need to update it
		//log.Debugf("Update existing record:%d on cluster node:%s (old:%v new:%v)", existingid, node, dnsmanager.node[node].Domains[domain].Records[existingid], record)
		//log.Infof("Update existing record:%s.%s on cluster node:%s (old:%v new:%v)", record.Name, domain, node, dnsmanager.node[node].Domains[domain].Records[existingid].Target, record.Target)
		log.WithField("oldtarget", dnsmanager.node[node].Domains[domain].Records[existingid].Target).Info("Updating existing DNS record")
		updateRecord(node, domain, existingid, record)

		if dnsmanager.proxyStats == false {
			// When updating an existing record, reset the counter inorder to keep loadbalancing mechanism working (e..g round robin counters etc)
			resetCounters(record.Name, domain, record.Type)
		}

	} else {
		// } else if record.Online {
		// We have a non-existing record, which is online or offline, add it.
		log.Warn("Adding new DNS record")
		addRecord(node, domain, record)
		// When joining an existing record, reset the counter inorder to keep loadbalancing mechanism working (e..g round robin counters etc)
		// this only matters when we have a new online records, not for offlines
		if dnsmanager.proxyStats == false && record.Online == true {
			resetCounters(record.Name, domain, record.Type)
		}
	}
}

// MarkOffline marks all dns entries of a node as offline
func MarkOffline(node string) {
	log := logging.For("dns/update/markoffline")
	log.WithField("cluster", node).Warn("Marking all DNS records from cluster as Offline")
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	for domainName, domain := range dnsmanager.node[node].Domains {
		for id := range domain.Records {
			dnsmanager.node[node].Domains[domainName].Records[id].Online = false
		}
	}
}

// Discard discards all dns entries of a node
func Discard(node string) {
	log := logging.For("dns/update/discard")
	log.WithField("cluster", node).Warn("Discarding DNS records from cluster")
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	delete(dnsmanager.node, node)
}

// AddLocalRecord adds a local record
func AddLocalRecord(domain string, record Record) {
	record.Online = true
	record.Local = true
	addRecord("localdns", domain, record)
}

// RemoveLocalRecord adds a local record
func RemoveLocalRecord(domain string, recordid int) {
	removeRecord("localdns", domain, recordid)
}

// RemoveLocalRecordByContent remove content by record data
func RemoveLocalRecordByContent(domainName string, hostName string, TTL int, Target string, Type string) {
	records := dnsmanager.node["localdns"].Domains[domainName].Records
	for i := len(records) - 1; i > 0; i-- {
		if records[i].Name == hostName &&
			records[i].TTL == TTL &&
			records[i].Target == Target &&
			records[i].Type == Type {
			RemoveLocalRecord(domainName, i)
		}
	}
}

// GetAllLocalDomainRecords get all local records from domain
func GetAllLocalDomainRecords(domain string) []Record {
	dnsmanager.RLock()
	defer dnsmanager.RUnlock()
	records := make([]Record, len(dnsmanager.node["localdns"].Domains[domain].Records))
	for _, record := range dnsmanager.node["localdns"].Domains[domain].Records {
		records = append(records, record)
	}
	return records
	//return dnsmanager.node["localdns"].Domains[domain].Records
}

// UpdateStatistics for node
func UpdateStatistics(clusterNode string, domain string, s *balancer.Statistics) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	if _, ok := dnsmanager.node[clusterNode].Domains[domain]; ok {
		// if records exist
		for id, rec := range dnsmanager.node[clusterNode].Domains[domain].Records {
			if rec.UUID == s.UUID {
				dnsmanager.node[clusterNode].Domains[domain].Records[id].Statistics = s
			}
		}
	}

}

// EnableProxyStats set to true to use proxy stats, or to false to use internal stats of dns manager
func EnableProxyStats(b bool) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	dnsmanager.proxyStats = b
}

// AllowForwarding return true if client is allowed to forward dns requests
func AllowForwarding(cidr []string) {
	log := logging.For("dns/addforwarding")
	var cidrs []*net.IPNet
	for _, c := range cidr {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			log.WithField("cidr", c).Warn("Invalid cids in forwarding allow list")
		}
		cidrs = append(cidrs, ipnet)
	}
	dnsmanager.Lock()
	defer dnsmanager.Unlock()

	dnsmanager.AllowForwarding = cidrs
}
