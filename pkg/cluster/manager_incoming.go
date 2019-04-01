package cluster

func (m *Manager) handleIncommingConnections() {
	for {
		select {
		case conn := <-m.newSocket:
			m.log("%s new socket from %s", m.name, conn.RemoteAddr())
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
				m.log("%s sent an invalid authentication request: %s", conn.RemoteAddr(), err)
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: err.Error()})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}

			if authRequest.AuthKey != m.authKey {
				// auth failed
				m.log("%s sent an invalid authentication key", conn.RemoteAddr())
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: "invalid authentication key"})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}

			authResponse, _ := m.newPacket(packetAuthResponse{Status: true})
			err = m.connectedNodes.writeSocket(conn, authResponse)
			if err != nil {
				m.log("%s failed while trying to send an authentication response", conn.RemoteAddr())
				conn.Close()
				return
			}

			m.log("%s incomming auth completed by %s (%s)", m.name, packet.Name, conn.RemoteAddr())
			node := newNode(packet.Name, conn, true)
			go m.handleAuthorizedConnection(node)
		}
	}
}
