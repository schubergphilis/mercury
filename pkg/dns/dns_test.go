package dns

import (
	"net"
	"strings"
	"sync"
	"testing"

	dnssrv "github.com/miekg/dns"
	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/logging"
)

var testRecordsServer1 = []Record{
	{UUID: "1-a", Name: "www-a", Type: "A", Target: "127.0.0.1", BalanceMode: "leastconnected", Online: true, Local: false},
	{UUID: "1-b", Name: "www-b", Type: "A", Target: "127.0.0.1", BalanceMode: "leastconnected", Online: false, Local: false},
	{UUID: "1-c", Name: "www-c", Type: "A", Target: "127.0.0.1", BalanceMode: "leastconnected", Online: false, Local: false},
}

var testRecordsServer2 = []Record{
	{UUID: "2-a", Name: "www-a", Type: "A", Target: "127.0.0.2", BalanceMode: "leastconnected", Online: true, Local: false},
	{UUID: "2-b", Name: "www-b", Type: "A", Target: "127.0.0.2", BalanceMode: "leastconnected", Online: true, Local: false},
	{UUID: "2-c", Name: "www-c", Type: "A", Target: "127.0.0.2", BalanceMode: "leastconnected", Online: false, Local: false},
}

const domain = "example.com"

func TestResolving(t *testing.T) {
	logging.Configure("stdout", "info")
	loadRecords("server1", domain, testRecordsServer1)
	loadRecords("server2", domain, testRecordsServer2)

	// With both nodes online we should return all records
	m := new(dnssrv.Msg)
	m.SetQuestion("www-a.example.com.", dnssrv.TypeA)
	rcode, _ := parseQuery(m, "127.0.0.1:12345")
	if !answerTarget(m, "127.0.0.1") || !answerTarget(m, "127.0.0.2") {
		t.Errorf("Expected 2 records, 127.0.0.1:%t 127.0.0.2:%t", answerTarget(m, "127.0.0.1"), answerTarget(m, "127.0.0.2"))
	}
	if rcode != 0 {
		t.Errorf("Return code incorrect, got:%d expected:%d", rcode, 0)

	}

	// Partially onlne should return 1 record
	m = new(dnssrv.Msg)
	m.SetQuestion("www-b.example.com.", dnssrv.TypeA)
	rcode, _ = parseQuery(m, "127.0.0.1:12345")
	if answerTarget(m, "127.0.0.1") || !answerTarget(m, "127.0.0.2") {
		t.Errorf("Expected 1 records, 127.0.0.1:%t 127.0.0.2:%t", answerTarget(m, "127.0.0.1"), answerTarget(m, "127.0.0.2"))
	}
	if rcode != 0 {
		t.Errorf("Return code incorrect, got:%d expected:%d", rcode, 0)
	}

	// Fully offlne should return 2 record
	m = new(dnssrv.Msg)
	m.SetQuestion("www-c.example.com.", dnssrv.TypeA)
	rcode, _ = parseQuery(m, "127.0.0.1:12345")
	if !answerTarget(m, "127.0.0.1") || !answerTarget(m, "127.0.0.2") {
		t.Errorf("Expected 2 records, 127.0.0.1:%t 127.0.0.2:%t", answerTarget(m, "127.0.0.1"), answerTarget(m, "127.0.0.2"))
	}
	if rcode != 0 {
		t.Errorf("Return code incorrect, got:%d expected:%d", rcode, 0)
	}

}

func TestRecursive(t *testing.T) {
	_, ipv4Net, _ := net.ParseCIDR("192.168.0.1/24")
	dnsmanager.AllowForwarding = append(dnsmanager.AllowForwarding, ipv4Net)

	// Check for allowed recursive lookup
	m := new(dnssrv.Msg)
	m.SetQuestion("www.google.com.", dnssrv.TypeA)
	rcode, _ := parseQuery(m, "192.168.0.2:12345")

	if !answerTarget(m, "172.217.19.196") {
		t.Errorf("Expected 1 records, 172.217.19.196:%t got:%+v", answerTarget(m, "172.217.19.196"), m)
	}
	if rcode != -1 {
		t.Errorf("Return code incorrect, got:%d expected:%d", rcode, 0)
	}

	// Check for denied recursive lookup
	m = new(dnssrv.Msg)
	m.SetQuestion("www.google.com.", dnssrv.TypeA)
	rcode, _ = parseQuery(m, "192.168.230.230:12345")

	if !answerCount(m, 0) {
		t.Errorf("Expected 0 records for denied recursive lookup, got:%d", len(m.Answer))
	}
	if rcode != 5 {
		t.Errorf("Return code incorrect, got:%d expected:%d", rcode, 0)
	}

}

func loadRecords(server string, domain string, records []Record) {
	// Prepare records
	for _, r := range records {
		stats := &balancer.Statistics{
			UUID:       r.UUID,
			Preference: 0,
			//Topology:   r.LocalNetwork,
			RWMutex: new(sync.RWMutex),
		}
		r.Statistics = stats
		Update(server, domain, r)
	}
}

func answerCount(m *dnssrv.Msg, c int) bool {
	return len(m.Answer) == c
}

func answerTarget(m *dnssrv.Msg, chars string) bool {
	for _, a := range m.Answer {
		if strings.Contains(a.String(), chars) {
			return true
		}
	}
	return false
}
