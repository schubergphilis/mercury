package dns

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/logging"

	dnssrv "github.com/miekg/dns"
	"github.com/rdoorn/dnsr"
)

// Domains is a collection of dns domains
type Domains struct {
	Domains map[string]Domain `toml:"domains" json:"domains"`
}

// Domain is a dns domain
type Domain struct {
	Records []Record `toml:"records" json:"records"`
	TTL     int      `json:"ttl"`
}

// Record of any type
type Record struct {
	Name          string               `toml:"name" json:"name"`                   // hostname
	Type          string               `toml:"type" json:"type"`                   // record type
	Target        string               `toml:"target" json:"target"`               // reply of record
	TTL           int                  `toml:"ttl" json:"ttl"`                     // time to live
	BalanceMode   string               `toml:"balancemode" json:"balancemode"`     // balance mode of dns
	ActivePassive string               `toml:"activepassive" json:"activepassive"` // used for monitoring only: record is active/passive setup
	ClusterNodes  int                  `toml:"clusternodes" json:"clusternodes"`   // ammount of cluster nodes that should serve this domain (defaults to len(clusternodes))
	LocalNetwork  string               `toml:"localnetwork" json:"localnetwork"`   // used by balance mode: topology
	Statistics    *balancer.Statistics `toml:"statistics" json:"statistics"`       // stats
	Status        Status               `toml:"status" json:"status"`               // is record online (do we serve it)
	Local         bool                 `toml:"local" json:"local"`                 // true if record is of the local dns server
	UUID          string               `toml:"uuid" json:"uuid"`                   // links record to check that added it,usefull for removing dead checks
}

// Config has the dns config
type Config struct {
	Domains         map[string]Domain `toml:"domains" json:"domains"`
	Binding         string            `toml:"binding" json:"binding"`
	AllowForwarding []string          `toml:"allow_forwarding" json:"allow_forwarding"`
	Port            int               `toml:"port" json:"port"`
	AllowedRequests []string          `toml:"allowed_requests" json:"allowed_requests"`
}

// reverse an array of strings
func reverse(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// dnsmanager holds all dns records known
var dnsmanager = struct {
	sync.RWMutex
	node            map[string]Domains
	stop            chan bool
	AllowedRequests []string
	proxyStats      bool
	TCPServer       *dnssrv.Server
	UDPServer       *dnssrv.Server
	Resolver        *dnsr.Resolver
	AllowForwarding []*net.IPNet
}{node: make(map[string]Domains), stop: make(chan bool, 1), AllowedRequests: []string{}, proxyStats: false, TCPServer: &dnssrv.Server{}, UDPServer: &dnssrv.Server{}, Resolver: dnsr.New(0)}

// Updates the counter of an dns record which was requested
func updateCounter(domain string, record Record) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()

	for nodename := range dnsmanager.node {
		if _, ok := dnsmanager.node[nodename].Domains[domain]; ok {
			// if records exist
			for id, rec := range dnsmanager.node[nodename].Domains[domain].Records {
				if rec.UUID == record.UUID {
					//dnsmanager.node[nodename].Domains[domain].Records[id].Statistics.ClientsConnected++
					dnsmanager.node[nodename].Domains[domain].Records[id].Statistics.ClientsConnectedAdd(1)
				}
			}
		}

	}
}

// resetCounters resets the query counters of a specific fqdn
func resetCounters(hostname, domain, request string) {
	for nodename := range dnsmanager.node {
		resetCounter(nodename, hostname, domain, request)
	}
}

// resetCounters resets the query counters of a specific fqdn on a specific loadbalancer node
func resetCounter(nodename, name, domain, request string) {
	log := logging.For("dns/server/resetcounter")
	log.WithField("cluster", nodename).WithField("hostname", name).WithField("domain", domain).WithField("type", request).Debug("Resetting Counters")
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	for id, rec := range dnsmanager.node[nodename].Domains[domain].Records {
		if rec.Name == name && rec.Type == request {
			dnsmanager.node[nodename].Domains[domain].Records[id].Statistics.ClientsConnectedSet(0)
		}
	}
}

// GetBackendNodeBalanced returns a single backend node, based on balancer proto
func getRecordsBalanced(r []Record, ip, balancemode string) ([]Record, error) {
	//log := logging.For("dns/getrecordbalanced")

	switch len(r) {
	case 0: // return error of no nodes
		return []Record{}, fmt.Errorf("Unable to find a record")

	case 1: // return node if there is only 1 present
		return r, nil

	default: // balance across N Nodes
		stats := getDNSStats(r)
		statrecords, err := balancer.MultiSort(stats, ip, "stickyness_not_supported_in_dns", balancemode)
		if err != nil {
			return r, fmt.Errorf("Unable to parse balance mode %s, err: %s", balancemode, err)
		}

		records, err := GetRecordsByStats(r, statrecords)
		if err != nil {
			return r, err
		}

		return records, nil
	}

}

// GetRecordsByStats returns the statistics of all records
func GetRecordsByStats(rec []Record, stat []balancer.Statistics) ([]Record, error) {
	var new []Record
	for _, statistics := range stat {
		for _, record := range rec {
			if record.Statistics.UUID == statistics.UUID {
				new = append(new, record)
			}
		}
	}

	if len(new) == 0 {
		return []Record{}, fmt.Errorf("Unable to find DNS record by IDs")
	}

	return new, nil
}

// BackendNodeStats gets statistics for backend nodes
func getDNSStats(n []Record) []balancer.Statistics {
	var s []balancer.Statistics
	for _, record := range n {
		s = append(s, *record.Statistics)
	}

	return s
}

// Returns an array of allowed request types
func getAllowedRequests() []string {
	dnsmanager.RLock()
	defer dnsmanager.RUnlock()
	return dnsmanager.AllowedRequests
}

func getRecordsByType(hostName string, domainName string, queryType uint16) (records []Record) {
	// Get all related records
	isAllowed := false
	allowedRequests := getAllowedRequests()
	for _, allowed := range allowedRequests {
		if queryType == dnssrv.StringToType[allowed] {
			isAllowed = true
			break
		}
	}

	// If we have limited the allowed requests, and we are not allowed, then return empty
	if len(allowedRequests) > 0 && isAllowed == false {
		return []Record{}
	}

	switch queryType {
	case dnssrv.TypeANY:
		// loop through available record types
		// 258 is last record used https://github.com/miekg/dns/blob/767422ac12884e2baed0afd7303cf06cff90fef6/types.go#L94
		for i := uint16(0); i <= 258; i++ {
			if dnssrv.TypeToString[i] != "" {
				anyR := getAllRecords(hostName, domainName, dnssrv.TypeToString[i])
				for _, r := range anyR {
					records = append(records, r)
				}
			}
		}

	default:
		// get specified record
		records = getAllRecords(hostName, domainName, dnssrv.TypeToString[queryType])
	}
	return records
}

// parseQuery parses the clients request, and formulates a reply
func parseQuery(m *dnssrv.Msg, client string) (int, error) {
	log := logging.For("dns/server/parse")

	var clientdata string
	if idx := strings.LastIndex(client, ":"); idx != -1 {
		clientdata = client[:idx]
		// ugly for ipv6 parsing
		clientdata = strings.Replace(clientdata, "[", "", -1)
		clientdata = strings.Replace(clientdata, "]", "", -1)
	}

	clientIP := net.ParseIP(clientdata)
	log.WithField("client", clientIP).WithField("orgclient", client).WithField("clientdata", clientdata).Debug("Client")

	exitcode := dnssrv.RcodeServerFailure

	for _, q := range m.Question {

		// Only deal with fqdn's
		if !dnssrv.IsFqdn(q.Name) || q.Name == "." {
			return dnssrv.RcodeRefused, nil
		}
		// Get hostname only if we didn't do a domain wide query type
		hostName := ""
		domainName := strings.TrimRight(q.Name, ".")

		// Check if its a reversable ip, or do a forward lookup if not
		reverseName, err := dnssrv.ReverseAddr(domainName)
		if err == nil {
			q.Name = strings.TrimRight(reverseName, ".")
			q.Qtype = dnssrv.TypePTR
		}

		if !localZone(domainName) { // if request is not our domain then its a fqdn
			log.WithField("fqdn", strings.ToLower(q.Name)).WithField("domain", strings.ToLower(domainName)).WithField("querytype", dnssrv.TypeToString[q.Qtype]).Debug("Non local zone request")
			s := dnssrv.SplitDomainName(q.Name)
			hostName = strings.Join(s[:1], ".")
			domainName = strings.Join(s[1:len(s)], ".")
		}

		clog := log.WithField("domain", strings.ToLower(domainName)).WithField("hostname", strings.ToLower(hostName)).WithField("querytype", dnssrv.TypeToString[q.Qtype]).WithField("client", clientIP.String()).WithField("0x20", q.Name != strings.ToLower(q.Name))
		clog.Info("DNS request from client")

		var records []Record
		records = getRecordsByType(hostName, domainName, q.Qtype)
		for id, r := range records {
			clog.WithField("prio", id).WithField("target", r.Target).WithField("recordtype", r.Type).Debug("DNS Records found")
		}

		// If we have no A records, check for CNAME
		if len(records) == 0 && q.Qtype == dnssrv.TypeA {
			records = getRecordsByType(hostName, domainName, dnssrv.TypeCNAME)
			for id, r := range records {
				clog.WithField("prio", id).WithField("target", r.Target).WithField("recordtype", r.Type).Debug("DNS Records found")
			}
		}

		if len(records) == 0 && !localZone(domainName) {
			if allowedToForward(clientIP) {
				clog.Debug("Relaying request for client")
				dnsForwarder(m, q)
				return -1, nil // copy error result
			}
			// no local zone, and nog allowed to forward
			return dnssrv.RcodeRefused, nil
		}

		// If we have more then 1 record, see how we should balance these records
		if len(records) > 1 {
			log.WithField("mode", records[0].BalanceMode).WithField("records", len(records)).Debug("Applying balancing")
			orderedrec, err := getRecordsBalanced(records, clientIP.String(), records[0].BalanceMode)
			if err != nil {
				clog.WithField("error", err).Warn("Unable to process the dns balancer, sending original records")
				orderedrec = records
			}
			records = orderedrec
		}

		var additionalrecords []Record
		switch q.Qtype {
		case dnssrv.TypeAAAA:
			if len(records) == 0 {
				aRecords := getRecordsByType(hostName, domainName, dnssrv.TypeA)
				if len(aRecords) > 0 {
					// we have AAAA record request, which doesn't exist, but we have A records that do exist.
					// so don't give an error that the domain doesn't exist, just nod and smile
					return dnssrv.RcodeSuccess, nil
				}
			}

		case dnssrv.TypeSOA:
			fallthrough

		case dnssrv.TypeNS, dnssrv.TypeMX:
			for _, currec := range records {
				first := strings.Split(currec.Target, " ")
				var s []string
				if q.Qtype == dnssrv.TypeMX {
					s = dnssrv.SplitDomainName(first[1])
				} else {
					s = dnssrv.SplitDomainName(first[0])
				}
				hostName := strings.ToLower(strings.Join(s[:1], "."))
				domainNameNS := strings.ToLower(strings.Join(s[1:len(s)], "."))

				nsrecords := getRecordsByType(hostName, domainNameNS, dnssrv.TypeA)
				log.Debugf("Got NS records: %+v", nsrecords)
				for _, ns := range nsrecords {
					additionalrecords = append(additionalrecords, ns)
				}
			}
		}

		// update counters if we return a record
		if dnsmanager.proxyStats == false && len(records) > 0 { // TODO: we only need to track counters if we are not using the build-in loadbalancer
			updateCounter(domainName, records[0])
		}

		// turn records in to reply
		if len(records) > 0 {
			m.Authoritative = true
			m.RecursionAvailable = true
		}

		for prio, record := range records {
			var newRecord string
			if record.TTL == 0 {
				record.TTL = 10
			}

			if record.Name == "" {
				newRecord = fmt.Sprintf("%s %d %s %s", q.Name, record.TTL, record.Type, record.Target)
			} else {
				newRecord = fmt.Sprintf("%s.%s %d %s %s", record.Name, domainName, record.TTL, record.Type, record.Target)
			}

			rr, err := dnssrv.NewRR(newRecord)
			if err == nil {
				clog.WithField("prio", prio).WithField("target", record.Target).WithField("recordtype", record.Type).Info("DNS reply to client")
				m.Answer = append(m.Answer, rr)
			} else {
				clog.WithField("error", err).WithField("target", record.Target).WithField("recordtype", record.Type).Error("DNS failed to add record")
			}
		}

		// Authoritive records
		if m.Authoritative == true {
			log.Debug("Authoritive answer, finding NS")
			authrecords := getRecordsByType("", domainName, dnssrv.TypeNS)
			for _, record := range authrecords {
				if record.TTL == 0 {
					record.TTL = 10
				}

				authrecordtxt := fmt.Sprintf("%s %d %s %s", domainName, record.TTL, record.Type, record.Target)
				nsrec, err := dnssrv.NewRR(authrecordtxt)
				if err == nil {
					m.Ns = append(m.Ns, nsrec)
				}
			}
		}

		// Additional records
		for prio, record := range additionalrecords {
			var newRecord string
			if record.TTL == 0 {
				record.TTL = 10
			}

			if record.Name == "" {
				newRecord = fmt.Sprintf("%s %d %s %s", q.Name, record.TTL, record.Type, record.Target)
			} else {
				newRecord = fmt.Sprintf("%s.%s %d %s %s", record.Name, domainName, record.TTL, record.Type, record.Target)
			}

			rr, err := dnssrv.NewRR(newRecord)
			if err == nil {
				clog.WithField("prio", prio).WithField("target", record.Target).WithField("recordtype", record.Type).Info("Additional DNS reply to client")
				m.Extra = append(m.Extra, rr)
			} else {
				clog.WithField("error", err).WithField("target", record.Target).WithField("recordtype", record.Type).Error("DNS failed to add additional record")

			}
		}

		switch q.Qtype {
		case dnssrv.TypeAAAA, dnssrv.TypeA:
			if len(m.Answer) == 0 {
				// Only for loadbalanced records (A/AAAA) do we show failure if there are no Records
				// This so that the 2nd loadbalancer will be queried
				exitcode = dnssrv.RcodeServerFailure
			} else {
				exitcode = dnssrv.RcodeSuccess
			}
		default:
			// For all other records, we know that the domain exists, but there is no host records
			// So return a successfull query with 0 records
			exitcode = dnssrv.RcodeSuccess
		}

	}

	log.WithField("exitcode", exitcode).Debugf("Request Finished")
	if exitcode == dnssrv.RcodeSuccess {
		return exitcode, nil
	}

	return exitcode, fmt.Errorf("No records returned")
}

func dnsForwarder(m *dnssrv.Msg, q dnssrv.Question) {
	log := logging.For("dns/server/forward")

	// Local resolving failed, if we have forwarding enabled, pass the request on
	log.WithField("name", q.Name).WithField("type", dnssrv.TypeToString[q.Qtype]).Infof("DNS Forwarding")
	rrs, err := dnsmanager.Resolver.ResolveErr(q.Name, dnssrv.TypeToString[q.Qtype])
	log.Debugf("Got forwarded DNS reply: %+v", rrs)
	if err != nil {
		log.WithField("name", q.Name).WithField("type", dnssrv.TypeToString[q.Qtype]).Warn("Failed to resolve forwarded dns")
		return
	}
	m.RecursionAvailable = true
	// Convert records to reply
	for _, dnsrr := range rrs {
		log.WithField("string", dnsrr.String()).Debugf("Forwarding request")
		rr, err := dnssrv.NewRR(dnsrr.String())
		if err != nil {
			log.WithField("reply", dnsrr.String()).Warn("Failed to create reply")
		} else {
			log.Debugf("Got forwarded DNS reply: %v", rr)
			m.Answer = append(m.Answer, rr)
		}
	}

}

// handleDNSRequest receives queries and sends replies
func handleDNSRequest(w dnssrv.ResponseWriter, r *dnssrv.Msg) {
	m := new(dnssrv.Msg)
	m.SetReply(r)
	m.Compress = false

	// go through the message requests
	switch r.Opcode {
	case dnssrv.OpcodeQuery:
		rcode, err := parseQuery(m, w.RemoteAddr().String())
		if err != nil {
			// No record found or other error, give server failure so resolv will move to next server for query
			m.SetRcode(r, dnssrv.RcodeServerFailure)
		}

		if rcode >= 0 {
			m.SetRcode(r, rcode)
		}
	default:
		m.SetRcode(r, dnssrv.RcodeRefused)
	}

	w.WriteMsg(m)
}

// GetCache Returns a copy current cached entries
func GetCache() map[string]Domains {
	dnsmanager.RLock()
	defer dnsmanager.RUnlock()
	d := make(map[string]Domains, len(dnsmanager.node))
	for id, domain := range dnsmanager.node {
		d[id] = domain
	}

	return d
}

// Listen starts the listener
func Listen(server *dnssrv.Server) error {
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("Failed to setup the "+server.Addr+" server: %s\n", err.Error())
	}

	defer server.Shutdown()
	return nil
}

// Server Process DNS Requests
func Server(host string, port int, allowedRequests []string) {
	log := logging.For("dns/server")
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	dnsmanager.AllowedRequests = allowedRequests
	dnssrv.HandleFunc(".", handleDNSRequest)

	clog := log.WithField("ip", host).WithField("port", port)
	clog.Debug("Serving DNS Requests")

	addr := fmt.Sprintf("%s:%d", host, port)
	var serverTCP *dnssrv.Server
	var serverUDP *dnssrv.Server

	// TCP Server address changed
	if dnsmanager.TCPServer.Addr != addr {
		clog.WithField("old", dnsmanager.TCPServer.Addr).Debug("New address for DNS TCP Listener")
		if dnsmanager.TCPServer.Addr != "" {
			clog.Debug("Stopping old listener")
			dnsmanager.stop <- true
		}

		tcpListener, err := net.Listen("tcp", addr)
		if err != nil {
			clog.WithField("error", err).Error("Failed to start DNS TCP listener")
			return
		}

		serverTCP = &dnssrv.Server{Addr: host + ":" + strconv.Itoa(port), Net: "TCP", Listener: tcpListener}
		go serverTCP.ActivateAndServe()
		dnsmanager.TCPServer = serverTCP
	}

	if dnsmanager.UDPServer.Addr != addr {
		clog.WithField("old", dnsmanager.UDPServer.Addr).Debug("New address for DNS UDP Listener")
		udpListener, err := net.ListenPacket("udp", addr)
		if err != nil {
			clog.WithField("error", err).Error("Failed to start DNS UDP listener")
			return
		}

		serverUDP = &dnssrv.Server{Addr: host + ":" + strconv.Itoa(port), Net: "UDP", PacketConn: udpListener}
		go serverUDP.ActivateAndServe()
		dnsmanager.UDPServer = serverUDP
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1)
	go func() {
		for {
			select {
			case _ = <-dnsmanager.stop:
				serverTCP.Shutdown()
				serverUDP.Shutdown()
				return
			case signal := <-signalChan:
				log.WithField("signal", signal).Debug("Signal detected")
				Debug()
			}
		}
	}()

}

// Debug Shows current state
func Debug() {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	log := logging.For("dns/debug")
	for nodename, node := range dnsmanager.node {
		for domainname, domain := range node.Domains {
			for _, record := range domain.Records {
				log.WithField("cluster", nodename).WithField("domain", domainname).WithField("name", record.Name).WithField("type", record.Type).WithField("target", record.Target).WithField("mode", record.BalanceMode).WithField("uuid", record.UUID).WithField("status", record.Status.String()).WithField("local", record.Local).Info("Active DNS records")
			}
		}
	}
}

// Get all records which match the fqdn and request
func getAllRecords(hostName, domainName, request string) (r []Record) {
	var offlineRecords []Record
	total := 0
	//fqdn := fmt.Sprintf("%s.%s", hostName, domainName)
	log := logging.For("dns/server/getallrecords").WithField("domain", strings.ToLower(domainName)).WithField("name", strings.ToLower(hostName)).WithField("type", request)
	searchDomain := strings.ToLower(domainName)
	searchHost := strings.ToLower(hostName)
	dnsmanager.RLock()
	defer dnsmanager.RUnlock()

	for nodeName := range dnsmanager.node {
		// Search domains for records such as NS, CAA, SOA
		// Search domains for records such as A, AAAA
		if _, ok := dnsmanager.node[nodeName].Domains[searchDomain]; ok {
			for _, record := range dnsmanager.node[nodeName].Domains[searchDomain].Records {
				if record.Name == searchHost && record.Type == request {
					log.WithField("cluster", nodeName).WithField("target", record.Target).WithField("mode", record.BalanceMode).WithField("uuid", record.UUID).Debug("Found record")

					if record.Type == "SOA" {
						reg, _ := regexp.Compile("###([A-Z_a-z]+)###")
						fn := func(m string) string {
							p := reg.FindStringSubmatch(m)
							switch p[1] {
							case "SERIAL":
								return fmt.Sprintf("%d", time.Now().Unix()-(time.Now().Unix()%10))
							}
							return m
						}
						record.Target = reg.ReplaceAllStringFunc(record.Target, fn)
					}

					if record.Status == Online {
						record.Name = hostName // set the original hostname requested for 0x20 bit
						r = append(r, record)
					} else {
						offlineRecords = append(offlineRecords, record)
					}

					total++
				}
			}
		}
	}
	log.WithField("records", total).WithField("online", len(r)).Debug("Matching dns records")
	if len(r) == 0 && len(offlineRecords) > 0 {
		log.WithField("records", total).WithField("online", len(r)).WithField("offline", len(offlineRecords)).Warn("No online dns records, using fallback")
		return offlineRecords
	}

	return
}

// allowedToForward returns true or fals if client ip matches networks allowed to do DNS forwarding
func allowedToForward(clientIP net.IP) bool {
	//log := logging.For("dns/server/allowedtoforward").WithField("clientip", clientIP)
	//log.Debug("Checking if client is allowed to forward")
	for _, cidr := range dnsmanager.AllowForwarding {
		//log.WithField("cidr", cidr.String()).Debug("Checking if client matches cidr")
		if cidr.Contains(clientIP) {
			return true
		}
	}

	return false
}

func localZone(domainName string) bool {
	dnsmanager.RLock()
	defer dnsmanager.RUnlock()
	searchDomain := strings.ToLower(domainName)
	for nodeName := range dnsmanager.node {
		if _, ok := dnsmanager.node[nodeName].Domains[searchDomain]; ok {
			return true
		}
	}

	return false
}
