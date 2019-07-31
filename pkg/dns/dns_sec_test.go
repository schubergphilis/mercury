package dns

import (
	"testing"
	"time"

	dnssrv "github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestKeyGeneration(t *testing.T) {
	key, err := NewPrivateKey(KeySigningKey, dnssrv.ECDSAP256SHA256)
	assert.Nil(t, err)
	assert.NotEmpty(t, key.privateKey)
}

func TestKeyRotation(t *testing.T) {
	store := NewKeyStore()

	key, err := NewPrivateKey(KeySigningKey, dnssrv.ECDSAP256SHA256)
	assert.Nil(t, err)
	store.SetRollover(KeySigningKey, "example.com", 24*time.Hour, key)

	// expect 1 active key
	assert.Equal(t, 1, len(store.DNSKEYS("example.com", time.Now())))

	key2, err := NewPrivateKey(KeySigningKey, dnssrv.RSASHA256)
	assert.Nil(t, err)
	store.SetRollover(KeySigningKey, "example.com", 24*time.Hour, key2)

	key3, err := NewPrivateKey(KeySigningKey, dnssrv.RSASHA256)
	assert.Nil(t, err)
	store.SetRollover(KeySigningKey, "example.com", 24*time.Hour, key3)

	// 2 active keys 10 hours later
	assert.Equal(t, 2, len(store.DNSKEYS("example.com", time.Now().Add(10*time.Hour))))

	// still 2 active keys 25 hours later
	assert.Equal(t, 2, len(store.DNSKEYS("example.com", time.Now().Add(25*time.Hour))))

	// no replacement key so , only 1 key after 49 hours
	assert.Equal(t, 1, len(store.DNSKEYS("example.com", time.Now().Add(49*time.Hour))))

}

func TestKeyLoading(t *testing.T) {
	store := NewKeyStore()
	store.Load("/tmp")

	//key, err := NewPrivateKey(KeySigningKey, dnssrv.ECDSAP256SHA256)
	//assert.Nil(t, err)
	//store.SetRollover(KeySigningKey, "example.com", 24*time.Hour, key)

	//store.Save("/tmp")
	assert.Equal(t, 1, len(store.DNSKEYS("example.com", time.Now())))

	//key := store.DNSKEYS("example.com", time.Now())[0]
	/*log.Printf("DNSKEY: %s", key)
	log.Printf("DS: %s", key.ToDS(2))*/
}

/*
var address = "127.0.0.1:15355"

var _, net1, _ = net.ParseCIDR("127.0.0.1/32")
var _, net2, _ = net.ParseCIDR("127.0.0.2/32")

var recordsAdd = map[string][]cache.Record{
	"example.com.": []cache.Record{
		{Name: "", Type: "SOA", Target: "ns1.example.com. hostmaster.example.com. ###SERIAL### 3600 10 30 30", ClusterID: "localhost1", Online: true},
		{Name: "www", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true},
		{Name: "www", Type: "A", Target: "1.2.3.5", ClusterID: "localhost1", Online: true},
		{Name: "www", Type: "A", Target: "1.2.3.6", ClusterID: "localhost1", Online: false},
		{Name: "www3", Type: "CNAME", Target: "www.example.com", ClusterID: "localhost1", Online: true},
		{Name: "www2", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true},
		{Name: "", Type: "NS", Target: "ns1.example.com.", ClusterID: "localhost1", Online: true},
		{Name: "", Type: "NS", Target: "ns2.example.com.", ClusterID: "localhost1", Online: true},
		{Name: "", Type: "MX", Target: "10 mx1.example.com.", ClusterID: "localhost1", Online: true},
		{Name: "ns1", Type: "A", Target: "1.2.3.5", ClusterID: "localhost1", Online: true},
		{Name: "ns2", Type: "A", Target: "1.2.3.6", ClusterID: "localhost1", Online: true},
		{Name: "mx1", Type: "A", Target: "1.2.3.6", ClusterID: "localhost1", Online: true},
		{Name: "mx1", Type: "A", Target: "1.2.3.6", ClusterID: "localhost1", Online: true},
		{Name: "leastconnected", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{Connected: 2}, BalanceMode: "leastconnected"},
		{Name: "leastconnected", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{Connected: 1}, BalanceMode: "leastconnected"},
		{Name: "roundrobin", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{Requests: 2}, BalanceMode: "roundrobin"},
		{Name: "roundrobin", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{Requests: 1}, BalanceMode: "roundrobin"},
		{Name: "leasttraffic", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{TX: 2, RX: 2}, BalanceMode: "leasttraffic"},
		{Name: "leasttraffic", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, Statistics: cache.Statistics{TX: 1, RX: 1}, BalanceMode: "leasttraffic"},
		{Name: "firstavailable", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, BalanceMode: "firstavailable"},
		{Name: "firstavailable", Type: "A", Target: "1.2.3.5", ClusterID: "localhost1", Online: true, BalanceMode: "firstavailable"},
		{Name: "preference", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true, Preference: 2, BalanceMode: "preference"},
		{Name: "preference", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, Preference: 1, BalanceMode: "preference"},
		{Name: "topology", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true, LocalNetworks: []net.IPNet{*net2}, BalanceMode: "topology"},
		{Name: "topology", Type: "A", Target: "127.0.0.1", ClusterID: "localhost1", Online: true, LocalNetworks: []net.IPNet{*net1}, BalanceMode: "topology"},
	},
}

var recordsRemove = map[string][]cache.Record{
	"example.com.": []cache.Record{
		{Name: "www2", Type: "A", Target: "1.2.3.4", ClusterID: "localhost1", Online: true},
	},
}

func TestDNSServer(t *testing.T) {

	m := New()

	// Add DNS Records to server
	for domain, records := range recordsAdd {
		for _, record := range records {
			m.Cache.AddRecord(domain, record)
		}
	}

	// Remove DNS Records
	for domain, records := range recordsRemove {
		for _, record := range records {
			m.Cache.RemoveRecord(domain, record)
		}
	}

	//m.Start()
	t.Run("queryCalls", func(t *testing.T) {
		t.Run("StaticQueries", m.testStaticQueries)
		t.Run("BalancedQueries", m.testBalancedQueries)
	})

}

func (m *Master) testStaticQueries(t *testing.T) {
	t.Parallel()

	c := new(dns.Client)
	var d *dns.Msg

	// MX
	d = new(dns.Msg)
	d.SetEdns0(4096, true)
	d.SetQuestion("example.com.", dns.TypeMX)
	m.ServeRequest(d, d.Question[0].Name, "example.com", d.Question[0].Qtype, net.IP{}, 4096)
	checkResult(t, d, 1, 2, 4) // request, answers, auth, extra

	// NS
	d = new(dns.Msg)
	d.SetQuestion("example.com.", dns.TypeNS)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 2, 0, 2) // request, answers, auth, extra

	// SOA
	d = new(dns.Msg)
	d.SetQuestion("example.com.", dns.TypeSOA)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 1, 2, 2) // request, answers, auth, extra

	// A
	d = new(dns.Msg)
	d.SetQuestion("www.example.com.", dns.TypeA)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 2, 2, 2) // request, answers, auth, extra

	// CNAME -> A + 0x20 encoding
	d = new(dns.Msg)
	d.SetQuestion("Www3.ExAmpLe.CoM.", dns.TypeA)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 3, 2, 2) // request, answers, auth, extra
	if strings.Index(r.Answer[0].String(), "Www3.ExAmpLe.CoM.") < 0 {
		t.Errorf("Request of Www3.ExAmpLe.CoM. did not return the exact case back (0x20 encoding)\n: %+v", r)
	}
}

func (m *Master) testBalancedQueries(t *testing.T) {
	t.Parallel()

	// Test all balance modes
	c := new(dns.Client)
	for _, records := range recordsAdd {
		record := records[0]
		var m *dns.Msg
		if record.BalanceMode != "" {
			m = new(dns.Msg)
			m.SetQuestion(record.BalanceMode+".example.com.", dns.TypeA)
			r, _, err := c.Exchange(m, address)
			if err != nil {
				t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
			}
			if strings.Index(r.Answer[0].String(), "127.0.0.1") < 0 {
				t.Errorf("Result of %s failed, expected 127.0.0.1 to be the record, got: %+v\n", record.BalanceMode, r.Answer[0])
			}
		}
	}
}

func testForwardQueries(t *testing.T) {
	t.Parallel()

	c := new(dns.Client)
	var m *dns.Msg
	var r *dns.Msg
	var err error

	// google.com
	m = new(dns.Msg)
	m.SetQuestion("wwW.gOOgle.com.", dns.TypeA)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 1, 0, 0) // request, answers, auth, extra

	// nu.nl
	m = new(dns.Msg)
	m.SetQuestion("wwW.nu.nl.", dns.TypeA)
	r, _, err = c.Exchange(m, address)
	if err != nil {
		t.Errorf("Lookup failed for %+v: %s\n", m.Question, err)
	}
	checkResult(t, r, 9, 0, 0) // request, answers, auth, extra
}

// channelReadStrings reads a array of strings for the duration of timeout
func channelReadStrings(channel chan string, timeout time.Duration) (results []string) {
	for {
		select {
		case result := <-channel:
			results = append(results, result)
		case <-time.After(timeout * time.Second):
			return
		}
	}
}

func checkResult(t *testing.T, m *dns.Msg, answers int, authoritive int, extra int) { // request, answers, auth, extra
	//fmt.Printf("got:%+v", m)
	if m == nil {
		t.Errorf("Request returned nil!")
		return
	}
	if len(m.Answer) != answers || len(m.Ns) != authoritive || len(m.Extra) != extra {
		t.Errorf("Request did not return the correct reply (answers:%d/%d authotitive:%d/%d extra:%d/%d)\n", answers, len(m.Answer), authoritive, len(m.Ns), extra, len(m.Extra))
		t.Errorf("Request in error: %+v\n", m)
	}
}

func getKey() (*dns.DNSKEY, crypto.PrivateKey, error) {
	key := new(dns.DNSKEY)
	key.Hdr.Rrtype = dns.TypeDNSKEY
	key.Hdr.Name = "example.com."
	key.Hdr.Class = dns.ClassINET
	key.Hdr.Ttl = 14400
	key.Flags = 256
	key.Protocol = 3
	key.Algorithm = dns.ECDSAP384SHA384
	// RSASHA256/2048
	// ECDSAP384SHA384/384
	privkey, err := key.Generate(384)
	if err != nil {
		return nil, nil, err
	}

	newPrivKey, err := key.NewPrivateKey(key.PrivateKeyString(privkey))
	if err != nil {
		return nil, nil, err
	}

	switch newPrivKey := newPrivKey.(type) {
	case *rsa.PrivateKey:
		newPrivKey.Precompute()
	}
	return key, newPrivKey, nil

}
*/
