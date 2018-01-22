package cluster

import (
	"encoding/json"
	"fmt"
	"time"
)

// Packet is a cluster communication packet
type Packet struct {
	Name        string    `json:"name"`
	DataType    string    `json:"datatype"`
	DataMessage string    `json:"datamessage"`
	Time        time.Time `json:"time"`
}

// Some predefined packets //

// AuthRequestPacket defines an authorization request
type packetAuthRequest struct {
	AuthKey string `json:"authkey"`
}

// AuthResponsePacket defines an authorization response
type packetAuthResponse struct {
	Status bool      `json:"status"`
	Error  string    `json:"error"`
	Time   time.Time `json:"time"`
}

// PingPacket defines a ping
type packetPing struct {
	Time time.Time `json:"time"`
}

// NodeShutdownPacket defines a node shutting down the cluster
type packetNodeShutdown struct{}

// Message returns the message of a packet
func (packet *Packet) Message(message interface{}) error {
	if packet == nil {
		return fmt.Errorf("Unable to decrypt nil packet")
	}
	err := json.Unmarshal(json.RawMessage(packet.DataMessage), &message)
	if err != nil {
		return fmt.Errorf("Failed to decrypt dataMessage:%v", err)
	}
	return nil
}

// UnpackPacket unpacks a packet and returns its structure
func UnpackPacket(data []byte) (packet *Packet, err error) {
	err = json.Unmarshal(json.RawMessage(data), &packet)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt packet header:%v", err)
	}
	return
}
