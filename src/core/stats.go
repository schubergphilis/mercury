package core

/*
import (
	"fmt"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/src/config"
	"github.com/schubergphilis/mercury/src/dns"
	"github.com/schubergphilis/mercury/src/logging"
	"github.com/schubergphilis/mercury/src/stats"
)

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// remove dots and dashes
func underscore(l string) string {
	r := strings.NewReplacer(".", "_",
		"-", "_")
	return r.Replace(l)
}

// DNSStats gathers statistics for DNS
func dnsStats(s interface{}) {
	for {
		//log := logging.For("core/stats/dns")
		dnscache := dns.GetCache()
		//log.Debugf("Gathering DNS Statistics %T", dnscache)
		for nodename := range dnscache {
			for domainname := range dnscache[nodename].Domains {
				for _, rec := range dnscache[nodename].Domains[domainname].Records {
					targets, okNodes, _ := dns.FindTargets(dnscache, domainname, rec.Name, rec.Type)
					// Amount of queries done to DNS record
					s.(stats.Stat).Gauge(fmt.Sprintf("dns.%s.%s.query_count", underscore(domainname), rec.Name), fmt.Sprintf("%d", rec.Statistics.ClientsConnected))
					// Amount of targets returned on a request
					s.(stats.Stat).Gauge(fmt.Sprintf("dns.%s.%s.targets_total", underscore(domainname), rec.Name), fmt.Sprintf("%d", len(targets)))
					// Amount of targets online
					s.(stats.Stat).Gauge(fmt.Sprintf("dns.%s.%s.targets_online", underscore(domainname), rec.Name), fmt.Sprintf("%d", len(okNodes)))

				}
			}
		}
		interval := config.Get().Stats.Interval
		if interval == 0 {
			interval = 30
		}
		time.Sleep(interval * time.Second)
	}
}

// ClusterStats gathers statistics for Cluster
func clusterStats(s interface{}) {
	for {
		// Get local loadbalancer statistics
		loadbalancer := config.Get().Loadbalancer
		for poolname, pool := range loadbalancer.Pools {
			for backendname, backend := range pool.Backends {
				for _, node := range backend.Nodes {
					s.(stats.Stat).Gauge(fmt.Sprintf("cluster.%s.%s.%s.online_status", poolname, backendname, node.SafeName()), fmt.Sprintf("%d", btoi(node.Online)))
				}
			}
		}
		interval := config.Get().Stats.Interval
		if interval == 0 {
			interval = 30
		}
		time.Sleep(interval * time.Second)
	}
}

// gatherStats starts the gatherers in the background
func gatherStats(s interface{}) {
	go clusterStats(s)
	go dnsStats(s)
}

// InitializeStats starts the webserver
func InitializeStats(c stats.Config) interface{} {
	log := logging.For("core/stats/init").WithField("func", "stats")
	log.WithField("client", c.Client).Info("Connecting to stats server")
	switch c.Client {
	default:
		log.WithField("client", c.Client).Info("Unknown stats client, defaulting to carbon")
		fallthrough
	case "carbon":
		statsClient := &stats.Carbon{
			Prefix: "mercury",
		}
		go stats.Loop(c, statsClient)
		return statsClient
	case "statsd":
	}
	// we would never reach here
	return nil
}
*/
