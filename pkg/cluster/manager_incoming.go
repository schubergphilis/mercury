package cluster

func (m *Manager) handleIncommingConnections() {
	for {
		select {
		case conn := <-m.newSocket:
			m.log.Infof("new connection", "handler", m.name, "addr", conn.RemoteAddr())
			packet, err := m.connectedNodes.readSocket(conn)
			if err != nil {
				m.log.Debugf("new connection failed to read from socket", "handler", m.name, "addr", conn.RemoteAddr(), "error", err)
				conn.Close()
				continue
			}
			// Receive authentication request
			authRequest := &packetAuthRequest{}
			err = packet.Message(authRequest)
			if err != nil {
				// Unable to decode authRequest, attempt to send an error
				m.log.Warnf("new connection failed authentication", "handler", m.name, "addr", conn.RemoteAddr(), "error", err)
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: err.Error()})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}

			if authRequest.AuthKey != m.authKey {
				// auth failed
				m.log.Warnf("new connection failed authentication", "handler", m.name, "addr", conn.RemoteAddr(), "error", "invalid authentication key")
				authRequest, _ := m.newPacket(packetAuthResponse{Status: true, Error: "invalid authentication key"})
				m.connectedNodes.writeSocket(conn, authRequest)
				conn.Close()
				return
			}

			authResponse, _ := m.newPacket(packetAuthResponse{Status: true})
			err = m.connectedNodes.writeSocket(conn, authResponse)
			if err != nil {
				m.log.Warnf("new connection failed authentication", "handler", m.name, "addr", conn.RemoteAddr(), "error", "failed to write authentication response")
				conn.Close()
				return
			}

			m.log.Debugf("new connection authenticated successful", "handler", m.name, "addr", conn.RemoteAddr())
			node := newNode(packet.Name, conn, true)
			go m.handleAuthorizedConnection(node)
		}
	}
}
