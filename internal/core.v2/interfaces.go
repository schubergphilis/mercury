package core

import "github.com/rdoorn/cluster"

type ClusterService interface {
	Start()
	Stop()

	ReceivedFromCluster() chan cluster.Packet
	ReceivedFromAPI() chan cluster.APIRequest
	SendToCluster() chan interface{}
	SendToNode() chan cluster.NodeMessage
	NodeJoin() chan string
	NodeLeave() chan string

	LogFeed() chan string
}

type DNSService interface {
	Start()
	Stop()

	CreateRecord(record) (uuid string, err error) // when multiple records, they get loadbalanced by lb type in record
	UpdateRecord(uuid string) error               // update record by uuid
	DeleteRecord(uuid string) error               // delete record by uuid

	// on node leave, delete all records of host by looping trough the active config, and calling deleterecord for all entried of host X

	ReceivedRecordStatistics() chan string
	SendRecordStatistics(uuid string) chan string
}

type ProxyService interface {
	Start()
	Stop()

	CreateListener(listener) (uuid string, err error)
	UpdateListener(listener, listenerUUID) error
	DeleteListener(listenerUUID) error

	CreateBackend(backend, listenerUUID string) (uuid string, err error)
	UpdateBackend(backend, backendUUID) error
	DeleteBackend(backendUUID) error

	CreateNode(node, backendUUID string) (uuid string, err error)
	UpdateNode(backend, backendUUID) error
	DeleteNode(nodeUUID) error

	ReceivedListenerStatistics() chan string // statistics received locally, to do something with in core
	ReceivedBackendStatistics() chan string
	ReceivedNodeStatistics() chan string

	UpdateListenerStatistics(listenerUUID string) chan string // send updates statistics, received remotely, to local node for update
	UpdateBackendStatistics(backendUUID string) chan string
	UpdateNodeStatistics(nodeUUID string) chan string
}

type HealthcheckService interface {
	Start()
	Stop()

	CreateHealthCheck(check) (uuid string, err error)
	UpdateHealthCheck(check, checkUUID) (err error)
	DeleteHealthCheck(checkUUID) (err error)

	ReceiveHealthCheckStatus() chan string // receive update of status

	// SendHealthCheckStatus() chan string // force update of status <- handled internally not at library
}
