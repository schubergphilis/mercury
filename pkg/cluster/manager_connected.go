package cluster

import (
	"time"
)

func (m *Manager) handleAuthorizedConnection(node *Node) {
	// add authorized node if its uniq
	m.log("%s attempting to join (%s)", node.name, node.conn.RemoteAddr())

	oldNode, err := m.connectedNodes.nodeAdd(node)
	if err != nil { // err means we already have a node with this name, node was not added
		if oldNode.joinTime.Before(node.joinTime) {
			// close the newest connection, the old one has to timeout before joining again
			m.log("%s failed to join, there is a older connection still active. closing this connection (%s)", node.name, node.conn.RemoteAddr())
			node.close()
			return
		}
		// we closed the old connection, so we should add this new correct one to the current list
		m.log("%s failed to join, there is a newer connection still active. replacing the old one (%s) with this one (%s)", node.name, oldNode.conn.RemoteAddr(), node.conn.RemoteAddr())
		oldNode.close()
		m.connectedNodes.nodeRemove(oldNode)    // remove old node from connected list
		_, err = m.connectedNodes.nodeAdd(node) // again add new node to replace it
		if err != nil {
			m.log("%s failed to be re-added as the active node: %s", node.name, err)
		}
	}

	// wait a second before advertizing the node, we might have simultainious connects we need to settle a winner for
	time.Sleep(m.getDuration("joindelay"))
	select {
	case <-node.quit:
		m.log("%s was replaced by another connection. closing the discarded connection (%s)", node.conn.RemoteAddr())
		return
	default:
	}

	// start pinger in the background
	go m.pinger(node)

	// send join
	m.internalMessage <- internalMessage{Type: "nodejoin", Node: node.name}
	// wait for data till connection is closed
	m.connectedNodes.setStatus(node.name, StatusOnline)
	m.connectedNodes.setStatusError(node.name, "")
	err = node.ioReader(m.incommingPackets, m.getDuration("readtimeout"), node.quit)
	m.connectedNodes.setStatus(node.name, StatusLeaving)
	m.connectedNodes.setStatusError(node.name, err.Error())

	// remove node from connectionPool
	m.log("%s left, removing from connected list (%s)", node.name, node.conn.RemoteAddr())
	m.connectedNodes.nodeRemove(node)
	node.close()

	// send leave
	m.internalMessage <- internalMessage{Type: "nodeleave", Node: node.name, Error: err.Error()}
}

func (m *Manager) pinger(node *Node) {
	for {
		select {
		case <-node.quit:
			m.log("Exiting pinger for %s", node.name)
			return
		default:
		}

		p, _ := m.newPacket(&packetPing{Time: time.Now()})
		m.log("Sending ping to %s (%s)", node.name, node.conn.RemoteAddr())
		err := m.connectedNodes.writeSocket(node.conn, p)
		if err != nil {
			m.log("Failed to send ping to %s (%s). Error:%s", node.name, node.conn.RemoteAddr(), err.Error())
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
