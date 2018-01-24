package check

import (
	"encoding/json"
	"fmt"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/dns"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// otherNodes Find the other nodes, not the nodes given
func otherNodes(nodes []string) (diff []string) {
	m := map[string]int{}
	for _, val := range nodes {
		m[val] = 1
	}

	for _, val := range config.Get().Cluster.Nodes {
		m[val.Name] = m[val.Name] + 1
	}

	for key, val := range m {
		if val == 1 {
			diff = append(diff, key)
		}
	}

	return diff
}

// checkLoadbalancerCount checks the ammount of loadbalancers vs the ammount of loadbalancers with a registered dns entry
func checkLoadbalancerCount(dnsmanager map[string]dns.Domains) (int, error) {
	clusterNodeCount := len(config.Get().Cluster.Nodes)
	delete(dnsmanager, "localdns")                        // ignore local dns in GLB check
	delete(dnsmanager, config.Get().Cluster.Binding.Name) // remove self
	dnsManagerNodeCount := len(dnsmanager)
	var faultyNodes []string
	for _, clusterNode := range config.Get().Cluster.Nodes {
		if len(dnsmanager[clusterNode.Name].Domains) == 0 {
			faultyNodes = append(faultyNodes, clusterNode.Name)
		}
	}

	if clusterNodeCount != dnsManagerNodeCount {
		return CRITICAL, fmt.Errorf("Cluster node(s) %v have not reported in yet! (%d/%d online)", faultyNodes, dnsManagerNodeCount, clusterNodeCount)
	}

	return OK, nil
}

// checkEntryOnAllLoadbalancers checks if a dne record has a entry in all loadbalancers
func checkEntryOnAllLoadbalancers(dnsmanager map[string]dns.Domains) (int, error) {
	var faultyTargets []string
	nodename := config.Get().Cluster.Binding.Name
	if _, ok := dnsmanager[nodename]; ok {
		// Only check local cluster, the other cluster will check its self
		for domainname := range dnsmanager[nodename].Domains {
			for _, rec := range dnsmanager[nodename].Domains[domainname].Records {
				targets, okNodes, _ := dns.FindTargets(dnsmanager, domainname, rec.Name, rec.Type)
				if rec.ActivePassive == YES {
					if len(okNodes) == 0 {
						faultyTargets = append(faultyTargets, fmt.Sprintf("%s.%s in error: No backends online on any cluster! (%d/%d)", rec.Name, domainname, len(okNodes), len(targets)))
					}
					if len(okNodes) > 1 {
						faultyTargets = append(faultyTargets, fmt.Sprintf("%s.%s in error: More then 1 pool online of a active/standby backend! (%d/%d)", rec.Name, domainname, len(okNodes), len(targets)))
					}
				} else if len(okNodes) == 0 {
					// Completely offline
					faultyTargets = append(faultyTargets, fmt.Sprintf("%s.%s in error: No backends online on any cluster! (%d/%d)", rec.Name, domainname, len(okNodes), len(targets)))
				} else if len(okNodes) < rec.ClusterNodes {
					// we do not have all ok nodes, faultyNodes however might not know all nodes in error, so lets report all not OK
					faultyTargets = append(faultyTargets, fmt.Sprintf("%s.%s in error: Entry not available on all clusters (ok:%v, faulty:%v expected number of nodes ok:%v)", rec.Name, domainname, okNodes, otherNodes(okNodes), rec.ClusterNodes))
				}
			}
		}
	}

	if faultyTargets != nil {
		return CRITICAL, fmt.Errorf("%v\n", faultyTargets)
	}
	return OK, nil

}

// GLB Checks GLB status
func GLB() int {
	log := logging.For("check/glb")
	log.Debugf("Connecting to https://%s:%s/glb", config.Get().Web.Binding, config.Get().Web.Port)
	body, err := GetBody(fmt.Sprintf("https://%s:%d/glb", config.Get().Web.Binding, config.Get().Web.Port))
	if err != nil {
		fmt.Printf("Error connecting to Mercury at %s:%d. Is the service running? (error:%s)\n", config.Get().Web.Binding, config.Get().Web.Port, err)
		return CRITICAL
	}
	var dnsentries map[string]dns.Domains
	err = json.Unmarshal(body, &dnsentries)
	if err != nil {
		fmt.Printf("Error parsing json given by the Mercury service: %s\n", err)
		return CRITICAL
	}
	// Prepare data
	var criticals []string
	var warnings []string
	// Execute Checks
	log.Debug("Checking loadbalancer count")
	if exitcode, err := checkLoadbalancerCount(dnsentries); err != nil {
		switch exitcode {
		case CRITICAL:
			criticals = append(criticals, err.Error())
		case WARNING:
			warnings = append(warnings, err.Error())
		}
	}
	log.Debug("Checking dns entries exist on all known loadbalancers")
	if exitcode, err := checkEntryOnAllLoadbalancers(dnsentries); err != nil {
		switch exitcode {
		case CRITICAL:
			criticals = append(criticals, err.Error())
		case WARNING:
			warnings = append(warnings, err.Error())
		}
	}
	if len(criticals) > 0 {
		fmt.Printf("CRITICAL: %+v\n", criticals)
		return CRITICAL
	}
	if len(warnings) > 0 {
		fmt.Printf("WARNING: %v\n", warnings)
		return WARNING
	}
	fmt.Println("OK: All checks are fine!")
	return OK
}
