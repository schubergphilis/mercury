package cluster

import (
	"crypto/tls"
	"net"
	"time"
)

func (m *Manager) handleOutgoingConnections(tlsConfig *tls.Config) {
	for {
		select {
		case <-m.quit:
			// if manager exists, stop making outgoing connections
			m.log("EXIT of manager for outgoing connections")
			return
		default:
		}

		// Attempt to connect to non-connected nodes
		for _, node := range m.getConfiguredNodes() {
			if !m.connectedNodes.nodeExists(node.name) {
				// Connect to the remote cluster node
				m.log("Connecting to non-connected cluster node: %s", node.name)
				m.dial(node.name, node.addr, tlsConfig)
			}
		}
		//w ait before we try again
		time.Sleep(m.getDuration("connectinterval"))
	}
}

func (m *Manager) dial(name, addr string, tlsConfig *tls.Config) {
	m.log("Connecting to %s (%s)", name, addr)
	var conn net.Conn
	var err error
	if len(tlsConfig.Certificates) == 0 {
		m.log("Connecting to %s (%s) non-tls", name, addr)
		conn, err = net.DialTimeout("tcp", addr, m.getDuration("connecttimeout"))
	} else {
		m.log("Connecting to %s (%s) with-tls", name, addr)
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: m.getDuration("connecttimeout")}, "tcp", addr, tlsConfig)
	}
	if err == nil {
		// on dialing out, we need to send an auth
		authRequest, _ := m.newPacket(packetAuthRequest{AuthKey: m.authKey})
		m.connectedNodes.writeSocket(conn, authRequest)
		packet, err := m.connectedNodes.readSocket(conn)
		if err != nil {
			// close connection if someone is talking gibrish
			m.log("auth request failed on dial: %s", err)
			conn.Close()
			return
		}
		authResponse := &packetAuthResponse{}
		err = packet.Message(authResponse)
		if err != nil {
			// auth response unknown
			m.log("auth response failed on dial: %s", err)
			conn.Close()
			return
		}
		if authResponse.Status != true {
			m.log("auth failed on dial: %s", authResponse.Error)
			conn.Close()
			return
		}

		m.log("Connection to %s (%s) authorized", name, addr)
		node := newNode(packet.Name, conn)
		node.joinTime = authResponse.Time

		go m.handleAuthorizedConnection(node)
	}
}
