package balancer

import (
	"testing"
	"time"
)

var topology = []string{"127.0.0.1/8"}
var testrecords = []Statistics{
	{UUID: "ID1", ClientsConnected: 1, ClientsConnects: 10, TX: 10, RX: 1, Preference: 1, Topology: topology, ResponseTimeValue: []float64{1.12, 3.12, 41}},
	{UUID: "ID2", ClientsConnected: 5, ClientsConnects: 1, TX: 50, RX: 1, Preference: 1, ResponseTimeValue: []float64{1.12, 3.62, 11}},
	{UUID: "ID3", ClientsConnected: 10, ClientsConnects: 8, TX: 10, RX: 5, Preference: 0, ResponseTimeValue: []float64{10.12, 3.72, 11}},
	{UUID: "ID4", ClientsConnected: 2, ClientsConnects: 2, TX: 20, RX: 30, Preference: 2, ResponseTimeValue: []float64{5.12, 3.12, 21}},
	{UUID: "ID5", ClientsConnected: 10, ClientsConnects: 2, TX: 3, RX: 1, Preference: 3, ResponseTimeValue: []float64{2.12, 3.92, 31}},
}

func getBalanceTests() []Statistics {
	var records []Statistics
	for _, val := range testrecords {
		statistic := NewStatistics(val.UUID, 100)
		statistic.ClientsConnectedSet(val.ClientsConnected)
		statistic.ClientsConnectsSet(val.ClientsConnects)
		//statistic.Selected = val.Selected
		statistic.TXAdd(val.TX)
		statistic.RXAdd(val.RX)
		statistic.Preference = val.Preference
		statistic.Topology = val.Topology
		for _, t := range val.ResponseTimeValue {
			statistic.ResponseTimeAdd(t)
		}

		records = append(records, *statistic)
	}
	// Add/sub a few to complicate things
	records[3].ClientsConnectedSub(1)
	records[0].ClientsConnectedAdd(2)
	return records
}

func TestBalancer(t *testing.T) {
	t.Logf("TestBalancer...")
	var err error

	tests := map[string]string{
		"roundrobin":     "ID2",
		"leastconnected": "ID4",
		"preference":     "ID3",
		"leasttraffic":   "ID5",
		"responsetime":   "ID2",
	}

	for mode, result := range tests {
		records := getBalanceTests()
		records, err = MultiSort(records, "127.0.0.1", "sticky", mode)
		//t.Logf("%s Result: %s", mode, records[0].ID)
		if err != nil {
			t.Errorf("%s Resulted in error: %s", mode, err)
		}
		if records[0].UUID != result {
			t.Errorf("%s Result: %s Expected: %s", mode, records[0].UUID, result)
		}
	}

	records := getBalanceTests()
	newrecords, _ := MultiSort(records, "127.0.0.1", "", "firstavailable")
	if newrecords[0].UUID != "ID1" {
		t.Errorf("Firstavailable Result: %s Expected: ID1", newrecords[0].UUID)
	}
	if len(newrecords) > 1 {
		t.Errorf("Firstavailable Entries: %d Expected: 1", len(newrecords))
	}

	records = getBalanceTests()
	records, _ = MultiSort(records, "127.0.0.1", "sticky", "random")
	id1 := records[0].UUID
	records, _ = MultiSort(records, "127.0.0.1", "sticky", "random")
	id2 := records[0].UUID
	records, _ = MultiSort(records, "127.0.0.1", "sticky", "random")
	id3 := records[0].UUID
	records, _ = MultiSort(records, "127.0.0.1", "sticky", "random")
	id4 := records[0].UUID
	if (id1 == id2) && (id2 == id3) && (id3 == id4) {
		t.Errorf("random Result: %s Expected: [random]", id1)
	}

	records = getBalanceTests()
	newrecords, _ = MultiSort(records, "127.0.0.1", "sticky", "topology")
	if newrecords[0].UUID != "ID1" {
		t.Errorf("Topology Result: %s Expected: ID1", newrecords[0].UUID)
	}
	if len(newrecords) > 1 {
		t.Errorf("Topology Entries: %d Expected: 1", len(newrecords))
	}

	records = getBalanceTests()
	newrecords, _ = MultiSort(records, "127.0.0.1", "ID4", "sticky")
	if newrecords[0].UUID != "ID4" {
		t.Errorf("Sticky Result: %s Expected: ID4", newrecords[0].UUID)
	}
	if len(newrecords) > 1 {
		t.Errorf("Sticky Entries: %d Expected: 1", len(newrecords))
	}

	// Test Reset
	records[0].Reset()
	if records[0].ClientsConnected != 0 {
		t.Errorf("Reset did not reset the values expected, got:%d expected:0", records[0].ClientsConnected)
	}

	// Test Timer
	records[0].TimeCounterAdd()
	time.Sleep(10 * time.Millisecond) // short delay due to go func
	if records[0].TimeCounterGet() != 1 {
		t.Errorf("TimeCounterAdd did not get added, got:%d expected:1", records[0].TimeCounterGet())
	}
}

var result []Statistics

func benchmarkBalancer(m string, b *testing.B) {
	records := getBalanceTests()
	for n := 0; n < b.N; n++ {
		records, _ = MultiSort(records, "127.0.0.1", "sticky", m)
	}
	result = records
}

func BenchmarkBalancerLeastConnected(b *testing.B) { benchmarkBalancer("leastconnected", b) }
func BenchmarkBalancerLeastTraffic(b *testing.B)   { benchmarkBalancer("leasttraffic", b) }
func BenchmarkBalancerPreference(b *testing.B)     { benchmarkBalancer("preference", b) }
func BenchmarkBalancerRandom(b *testing.B)         { benchmarkBalancer("random", b) }
func BenchmarkBalancerRoundRobin(b *testing.B)     { benchmarkBalancer("roundrobin", b) }
func BenchmarkBalancerSticky(b *testing.B)         { benchmarkBalancer("sticky", b) }
func BenchmarkBalancerTopology(b *testing.B)       { benchmarkBalancer("topology", b) }
func BenchmarkBalancerResponseTime(b *testing.B)   { benchmarkBalancer("responsetime", b) }
