package models

import (
	"crypto/tls"

	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/pkg/cluster"
)

type ClusterService interface {
	Start()
	WithName(name string)
	WithKey(key string)
	WithAddr(addr string)
	WithTLS(tls *tls.Config)
	WithLogger(s logging.SimpleLogger)
	Stop()

	AddNode(name, addr string)
	RemoveNode(name string)

	ReceivedFromCluster() chan cluster.Packet
	ReceivedNodeJoin() chan string
	ReceivedNodeLeave() chan string
	//ReceivedLogging() chan string

	SendToCluster() chan interface{}
	SendToNode() chan cluster.NodeMessage
}
