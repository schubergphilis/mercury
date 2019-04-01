package cluster

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func (m *Manager) handlePackets() {
	for {
		select {
		case pm := <-m.ToNode: // incomming from client application
			if LogTraffic {
				m.log("%s traffic to cluster node: %+v", m.name, pm)
			}

			err := m.writeClusterNode(pm.Node, pm.Message)
			if err != nil {
				m.log("%s Failed to write message to remote node. error: %s", m.name, err)
			}

		case message := <-m.ToCluster: // incomming from client application
			if LogTraffic {
				m.log("%s traffic to cluster: %+v", m.name, message)
			}

			err := m.writeCluster(message)
			if err != nil {
				m.log("%s Failed to write message to remote node. error: %s", m.name, err)
			}

		case message := <-m.apiRequest: // incomming messages from API
			if LogTraffic {
				m.log("%s traffic from cluster api: %+v", m.name, message)
			}

			switch message.Action {
			case "reconnect":
			case "admin":
			}

			m.log("%s Cluster API request: %s (%s)", m.name, message.Action, message.Node)
			select {
			case m.FromClusterAPI <- message:
			default:
				m.log("%s Unable to write API message to FromClusterAPI. Channel full!", m.name)
			}

		case message := <-m.internalMessage: // incomming intenal messages (do not leave this library)
			switch message.Type {
			case "nodeadd":
				m.updateQuorum()

			case "noderemove":
				m.updateQuorum()

			case "nodejoin":
				m.log("%s Cluster node joined: %s", m.name, message.Node)
				select {
				case m.NodeJoin <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()

			case "nodeleave":
				m.log("%s Cluster node left: %s (%s)", m.name, message.Node, message.Error)
				select {
				case m.NodeLeave <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()
			default:
				m.log("%s Unknown internal message %+v", m.name, message)
			}

		case packet := <-m.incommingPackets: // incomming packets from other cluster nodes
			if LogTraffic {
				m.log("%s traffic received incomming packet: %+v", m.name, packet)
			}

			m.connectedNodes.incPackets(packet.Name)

			switch packet.DataType {
			case "cluster.Auth": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusAuthenticating)

			case "cluster.packetNodeShutdown": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusShutdown)
				m.log("%s Got exit notice from node %s (shutdown)", m.name, packet.Name)
				m.connectedNodes.close(packet.Name)

			case "cluster.packetPing": // internal use
				m.log("%s Got ping from node %s (%v)", m.name, packet.Name, time.Now().Sub(packet.Time))
				m.connectedNodes.setLag(packet.Name, time.Now().Sub(packet.Time))

			default:
				m.log("%s Recieved non-cluster packet: %s", m.name, packet.DataType)
				select {
				case m.FromCluster <- packet: // outgoing to client application
				default:
					m.log("%s unable to send data to FromCluster channel, channel full!", m.name)
				}

			}
		}
	}
}

func (m *Manager) newPacket(dataMessage interface{}) ([]byte, error) {
	val := reflect.Indirect(reflect.ValueOf(dataMessage))
	packet := &Packet{
		Name:     m.name,
		DataType: fmt.Sprintf("%s", val.Type()),
		Time:     time.Now(),
	}

	data, err := json.Marshal(dataMessage)
	if err != nil {
		m.log("%s Unable to jsonfy data: %s", m.name, err)
	}

	packet.DataMessage = string(data)

	packetData, err := json.Marshal(packet)
	if err != nil {
		m.log("%s Unable to create json packet: %s", m.name, err)
	}

	packetData = append(packetData, 10) // 10 = newline
	return packetData, err
}
