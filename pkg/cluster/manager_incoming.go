package cluster

import (
	"time"
)

func (m *Manager) handleIncommingConnections() {
	for {
		select {
		case conn := <-m.newSocket:
			packet, err := m.connectedNodes.readSocket(conn)
			if err != nil {
				m.log("%s failed while trying to read from socket: %s", conn.RemoteAddr(), err)
				conn.Close()
				continue
			}
			// Receive authentication request
			authRequest := &packetAuthRequest{}
			err = packet.Message(authRequest)
			if err != nil {
				// Unable to decode authRequest, attempt to send an error
				m.log("%s sent an invalid authentication request: %s", err)
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: err.Error()})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}
			if authRequest.AuthKey != m.authKey {
				// auth failed
				m.log("%s sent an invalid authentication key")
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: "invalid authentication key"})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}
			authTime := time.Now()
			authResponse, _ := m.newPacket(packetAuthResponse{Status: true, Time: authTime})
			err = m.connectedNodes.writeSocket(conn, authResponse)
			if err != nil {
				m.log("%s failed while trying to send an authentication response")
				conn.Close()
				return
			}

			node := newNode(packet.Name, conn)
			node.joinTime = authTime
			go m.handleAuthorizedConnection(node)
		}
	}
}
