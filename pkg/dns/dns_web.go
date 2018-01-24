package dns

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// FindTargets returns all targets of matching records
func FindTargets(dnsmanager map[string]Domains, domain, name, request string) ([]string, []string, []string) {
	var records []string
	var faultyNodes []string
	var okNodes []string
	for nodename := range dnsmanager {
		if _, ok := dnsmanager[nodename].Domains[domain]; ok {
			for _, rec := range dnsmanager[nodename].Domains[domain].Records {
				if rec.Name == name && rec.Type == request {
					records = append(records, rec.Target)
					if rec.Online == true {
						okNodes = append(okNodes, nodename)
					} else {
						faultyNodes = append(faultyNodes, nodename)
					}
				}
			}
		}
	}

	return records, okNodes, faultyNodes
}

// WebGLBStatus Provides a status page for GLB
func WebGLBStatus(w http.ResponseWriter, r *http.Request) {
	dnscache := GetCache()
	switch r.Header.Get("Content-type") {
	case "application/json":
		data, err := json.Marshal(dnscache)
		if err != nil {
			fmt.Fprintf(w, "{ error:'%s' }", err)
		}

		fmt.Fprint(w, string(data))

	default:
		for nodename := range dnscache {
			for domainname := range dnscache[nodename].Domains {
				for _, rec := range dnscache[nodename].Domains[domainname].Records {
					targets, okNodes, faultyNodes := FindTargets(dnscache, domainname, rec.Name, rec.Type)
					fmt.Fprintf(w, "fqdn:%s.%s type:%s ttl:%d vips:%v vipcount:%d vipsonline:%d %v vipsoffline:%d %v method:%s\r\n", rec.Name, domainname, rec.Type, rec.TTL, targets, len(targets), len(okNodes), okNodes, len(faultyNodes), faultyNodes, rec.BalanceMode)
				}
			}
		}
	}

}
