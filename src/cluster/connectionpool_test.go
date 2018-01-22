package cluster

import "testing"

var c = newConnectionPool()

func TestConnectionPool(t *testing.T) {
	node1 := &Node{name: "node1"}
	node2 := &Node{name: "node2"}

	if c.nodeExists("node1") {
		t.Errorf("node1 exists in connectionPool, but should not yet exist")
	}

	c.nodeAdd(node1)

	if !c.nodeExists("node1") {
		t.Errorf("node1 does not exists in connectionPool, but should exist")
	}

	c.nodeAdd(node2)

	if !c.nodeExists("node2") {
		t.Errorf("node2 does not exists in connectionPool, but should exist")
	}

}
