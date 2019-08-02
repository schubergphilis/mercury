package core

import "github.com/schubergphilis/mercury.v3/internal/logging"

// Domain is a dns domain
type DNSConfigDomain struct {
	Records []DNSConfigRecord `toml:"records" json:"records"`
	TTL     int               `json:"ttl"`
}

// Record of any type
type DNSConfigRecord struct {
	Name                string `toml:"name" json:"name"`                                   // hostname
	Type                string `toml:"type" json:"type"`                                   // record type
	Target              string `toml:"target" json:"target"`                               // reply of record
	TTL                 int    `toml:"ttl" json:"ttl"`                                     // time to live
	BalanceMode         string `toml:"balancemode" json:"balancemode"`                     // balance mode of dns
	ActivePassive       string `toml:"activepassive" json:"activepassive"`                 // used for monitoring only: record is active/passive setup
	ServingClusterNodes int    `toml:"serving_cluster_nodes" json:"serving_cluster_nodes"` // ammount of cluster nodes that should serve this domain (defaults to len(clusternodes))
	LocalNetwork        string `toml:"localnetwork" json:"localnetwork"`                   // used by balance mode: topology
}

// Config has the dns config
type DNSConfig struct {
	Domains         map[string]DNSConfigDomain `toml:"domains" json:"domains"`
	Binding         string                     `toml:"binding" json:"binding"`
	AllowForwarding []string                   `toml:"allow_forwarding" json:"allow_forwarding"`
	Port            int                        `toml:"port" json:"port"`
	AllowedRequests []string                   `toml:"allowed_requests" json:"allowed_requests"`
}

type DNS struct {
	log logging.SimpleLogger
	//manager *dnssrv.Server
	config *DNSConfig
	quit   chan struct{}
}

func NewDNSServer(config *DNSConfig) *DNS {
	return &DNS{}
}

func (d *DNS) start() {

}

func (d *DNS) stop() {

}

func (d *DNS) reload() {

}

func (d *DNS) WithLogger(l logging.SimpleLogger) {
	d.log = l
}
