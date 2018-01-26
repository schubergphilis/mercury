package cluster

import (
	"time"
)

func (m *Manager) handleAuthorizedConnection(node *Node) {
	// add authorized node if its uniq
	m.log("%s %s attempting to join (%s)", m.name, node.name, node.conn.RemoteAddr())

	oldNode, err := m.connectedNodes.nodeAdd(node)
	if err != nil { // err means we already have a node with this name, node was not added

		var oldConnection, oldDirection string
		var newConnection, newDirection string
		if oldNode.incomming == true {
			oldConnection = oldNode.conn.RemoteAddr().String()
			oldDirection = "incomming"
		} else {
			oldConnection = oldNode.conn.LocalAddr().String()
			oldDirection = "outgoing"
		}
		if node.incomming == true {
			newConnection = node.conn.RemoteAddr().String()
			newDirection = "incomming"
		} else {
			newConnection = node.conn.LocalAddr().String()
			newDirection = "outgoing"
		}
		// Always kill the 'lower' connection if double, the lower has to timeout before you can connect again
		if oldConnection < newConnection {
			m.log("%s we have 2 connections in auth state old:%s (%s) new:%s (%s) removing this one", m.name, oldConnection, oldDirection, newConnection, newDirection)
			node.close()
			return
		}

		m.log("%s we have 2 connections in auth state old:%s (%s) new:%s (%s) keeping this one", m.name, oldConnection, oldDirection, newConnection, newDirection)
		oldNode.close()
		m.connectedNodes.nodeRemove(oldNode)    // remove old node from connected list
		_, err = m.connectedNodes.nodeAdd(node) // again add new node to replace it
		if err != nil {
			m.log("%s %s failed to be re-added as the active node: %s", m.name, node.name, err)
		}
	}

	m.log("%s %s attempting to join (%s) - pending join delay", m.name, node.name, node.conn.RemoteAddr())
	// wait a second before advertizing the node, we might have simultainious connects we need to settle a winner for
	time.Sleep(m.getDuration("joindelay"))
	select {
	case <-node.quit:
		m.log("%s %s was replaced by another connection. closing the discarded connection (%s)", m.name, node.name, node.conn.RemoteAddr())
		return
	default:
	}

	// start pinger in the background
	m.log("%s %s Starting pinger (%s)", m.name, node.name, node.conn.RemoteAddr())
	go m.pinger(node)

	// send join
	m.internalMessage <- internalMessage{Type: "nodejoin", Node: node.name}
	// wait for data till connection is closed
	m.connectedNodes.setStatus(node.name, StatusOnline)
	m.connectedNodes.setStatusError(node.name, "")
	m.log("%s %s joined, started ioReader (%s) (timeout:%v)", m.name, node.name, node.conn.RemoteAddr(), m.getDuration("readtimeout"))
	err = node.ioReader(m.incommingPackets, m.getDuration("readtimeout"), node.quit)
	m.log("%s %s ioReader failed (%s) (%s)", m.name, node.name, node.conn.RemoteAddr(), err)
	m.connectedNodes.setStatus(node.name, StatusLeaving)
	m.connectedNodes.setStatusError(node.name, err.Error())

	// remove node from connectionPool
	m.log("%s %s left, removing from connected list (%s)", m.name, node.name, node.conn.RemoteAddr())
	m.connectedNodes.nodeRemove(node)
	node.close()

	// send leave
	m.internalMessage <- internalMessage{Type: "nodeleave", Node: node.name, Error: err.Error()}
}

func (m *Manager) pinger(node *Node) {
	for {
		select {
		case <-node.quit:
			m.log("%s Exiting pinger for %s (%s)", m.name, node.name, node.conn.RemoteAddr())
			return
		default:
		}

		p, _ := m.newPacket(&packetPing{Time: time.Now()})
		m.log("%s Sending ping to %s (%s)", m.name, node.name, node.conn.RemoteAddr())
		err := m.connectedNodes.writeSocket(node.conn, p)
		if err != nil {
			m.log("%s Failed to send ping to %s (%s). Error:%s", m.name, node.name, node.conn.RemoteAddr(), err.Error())
			node.close()
			return
		}

		time.Sleep(m.getDuration("pinginterval"))
	}
}

func (m *Manager) writeCluster(dataMessage interface{}) error {
	//nodes := connected.getActiveNodes()
	packet, err := m.newPacket(dataMessage)
	if err != nil {
		return err
	}

	err = m.connectedNodes.writeAll(packet)
	return err

}

func (m *Manager) writeClusterNode(node string, dataMessage interface{}) error {
	packet, err := m.newPacket(dataMessage)
	if err != nil {
		return err
	}

	err = m.connectedNodes.write(node, packet)
	return err
}
