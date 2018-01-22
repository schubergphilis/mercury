package config

import (
	"bytes"
	"testing"

	"github.com/schubergphilis/mercury/src/proxy"
)

type CloseableBuffer struct {
	bytes.Buffer
}

func (b *CloseableBuffer) Close() error {
	return nil
}

func TestLoadbalancerConfig(t *testing.T) {
	proxy := proxy.NewBackendNode("UUID1", "192.168.1.1", "server1", 22, 10, []string{}, 0)
	addr := &BackendNode{
		BackendNode: proxy,
		Online:      false,
	}
	if Get().Loadbalancer.Pools["INTERNAL_VIP"].Backends["myapp"].Nodes[0].Name() != addr.Name() {
		t.Errorf("Expected pool:INTERNAL_VIP backend:myapp node:0 to be %+v (got:%+v)", addr, Get().Loadbalancer.Pools["INTERNAL_VIP"].Backends["myapp"].Nodes[0])
	}

	addr.Hostname = "server-1"
	if val := addr.Name(); val != "server_1_22" {
		t.Errorf("Expected addr.Name() of %+v to return server-1 (got:%s)", addr, val)
	}

	if val := addr.ServerName(); val != "server-1" {
		t.Errorf("Expected addr.Name() of %+v to return server-1 (got:%s)", addr, val)
	}

	if val := addr.SafeName(); val != "server_1" {
		t.Errorf("Expected addr.SafeName() of %+v to return server_1 (got:%s)", addr, val)
	}

	addr.Hostname = ""
	if val := addr.Name(); val != "192_168_1_1_22" {
		t.Errorf("Expected addr.Name() of %+v to return 192.168.1.1 (got:%s)", addr, val)
	}

	if val := addr.ServerName(); val != "192.168.1.1" {
		t.Errorf("Expected addr.Name() of %+v to return 192.168.1.1 (got:%s)", addr, val)
	}

	if val := addr.SafeName(); val != "192_168_1_1" {
		t.Errorf("Expected addr.SafeName() of %+v to return 192_168_1_1 (got:%s)", addr, val)
	}
}
