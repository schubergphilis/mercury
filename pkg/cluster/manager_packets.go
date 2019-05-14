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
				m.log.Debugf("traffic to cluster node", "handler", m.name, "message", pm)
			}

			err := m.writeClusterNode(pm.Node, pm.Message)
			if err != nil {
				m.log.Debugf("failed to write message to remote node", "handler", m.name, "node", pm.Node, "error", err)
			}

		case message := <-m.ToCluster: // incomming from client application
			if LogTraffic {
				m.log.Debugf(" traffic to cluster", "handler", m.name, "message", message)
			}

			err := m.writeCluster(message)
			if err != nil {
				m.log.Debugf("failed to write message to remote node", "handler", m.name, "error", err)
			}

		case message := <-m.internalMessage: // incomming intenal messages (do not leave this library)
			switch message.Type {
			case "nodeadd":
				m.updateQuorum()

			case "noderemove":
				m.updateQuorum()

			case "nodejoin":
				m.log.Warnf("cluster node joined", "handler", m.name, "node", message.Node)
				select {
				case m.NodeJoin <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()

			case "nodeleave":
				m.log.Warnf("cluster node exited", "handler", m.name, "node", message.Node, "error", message.Error)
				select {
				case m.NodeLeave <- message.Node: // send node join to client application
				default:
				}
				m.updateQuorum()
			default:
				m.log.Debugf("unknown internal message", "handler", m.name, "message", message)
			}

		case packet := <-m.incommingPackets: // incomming packets from other cluster nodes
			if LogTraffic {
				m.log.Debugf("traffic received incomming packet", "handler", m.name, "message", packet)
			}

			m.connectedNodes.incPackets(packet.Name)

			switch packet.DataType {
			case "cluster.Auth": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusAuthenticating)

			case "cluster.packetNodeShutdown": // internal use
				m.connectedNodes.setStatus(packet.Name, StatusShutdown)
				m.log.Infof("got exit notice from node (shutdown)", "handler", m.name, "node", packet.Name)
				m.connectedNodes.close(packet.Name)

			case "cluster.packetPing": // internal use
				m.log.Debugf("got ping from node", "handler", m.name, "node", packet.Name, "time", time.Now().Sub(packet.Time))
				m.connectedNodes.setLag(packet.Name, time.Now().Sub(packet.Time))

			default:
				m.log.Debugf("recieved non-cluster packet", "handler", m.name, "messagetype", packet.DataType)
				select {
				case m.FromCluster <- packet: // outgoing to client application
				default:
					m.log.Warnf("unable to send data to FromCluster channel, channel full!", "handler", m.name)
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
		m.log.Warnf("unable to jsonfy data", "handler", m.name, "error", err)
	}

	packet.DataMessage = string(data)

	packetData, err := json.Marshal(packet)
	if err != nil {
		m.log.Warnf("unable to create json packet", "handler", m.name, "error", err)
	}

	packetData = append(packetData, 10) // 10 = newline
	return packetData, err
}
