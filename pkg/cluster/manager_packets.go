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
				m.log("traffic to cluster node: %+v", pm)
			}
			err := m.writeClusterNode(pm.Node, pm.Message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case message := <-m.ToCluster: // incomming from client application
			if LogTraffic {
				m.log("traffic to cluster: %+v", message)
			}
			err := m.writeCluster(message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case message := <-m.apiRequest: // incomming messages from API
			if LogTraffic {
				m.log("traffic from cluster api: %+v", message)
			}
			switch message.Action {
			case "reconnect":
			case "admin":
			}
			m.log("Cluster API request: %s (%s)", message.Action, message.Node)
			select {
			case m.FromClusterAPI <- message:
			default:
				m.log("Unable to write API message to FromClusterAPI. Channel full!")
			}
		case message := <-m.internalMessage: // incomming intenal messages (do not leave this library)
			switch message.Type {
			case "nodeadd":
				m.updateQuorum()
			case "noderemove":
				m.updateQuorum()
			case "nodejoin":
				m.log("Cluster node joined: %s", message.Node)
				select {
				case m.NodeJoin <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()
			case "nodeleave":
				m.log("Cluster node left: %s (%s)", message.Node, message.Error)
				select {
				case m.NodeLeave <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()
			default:
				m.log("Unknown internal message %+v", message)
			}
		case packet := <-m.incommingPackets: // incomming packets from other cluster nodes
			if LogTraffic {
				m.log("traffic received incomming packet: %+v", packet)
			}
			m.connectedNodes.incPackets(packet.Name)
			switch packet.DataType {
			case "cluster.Auth": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusAuthenticating)
			case "cluster.packetNodeShutdown": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusShutdown)
				m.log("Got exit notice from node %s (shutdown)", packet.Name)
				m.connectedNodes.close(packet.Name)
			case "cluster.packetPing": // internal use
				m.log("Got ping from node %s (%v)", packet.Name, time.Now().Sub(packet.Time))
				m.connectedNodes.setLag(packet.Name, time.Now().Sub(packet.Time))
			default:
				m.log("Recieved non-cluster packet: %s", packet.DataType)
				select {
				case m.FromCluster <- packet: // outgoing to client application
				default:
					m.log("unable to send data to FromCluster channel, channel full!", packet.DataType)
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
		m.log("Unable to jsonfy data: %s", err)
	}
	packet.DataMessage = string(data)

	packetData, err := json.Marshal(packet)
	if err != nil {
		m.log("Unable to create json packet: %s", err)
	}

	packetData = append(packetData, 10) // 10 = newline
	return packetData, err
}
