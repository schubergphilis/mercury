/*
Package cluster implements a interface for clustered services.
It allows you to interace to a cluster node, and messages sent across will be
received by the connected cluster nodes. You can interface freely between
these N nodes. Additionaly there is also an API interface for passing commands
to the cluster.

Interfacing

You can interface through the Cluster Manager using channels. Messages that
are not used specificly by the cluster are forwarded to the client application
using this cluster library.


 manager.ToCluster <- interface{} // send data to the cluster
 package := <-manager.FromCluster // recieve Package{} from the cluster containing the sent data (required)

This interface allows you to send data to the cluster, which will be
broadcasted across the connected nodes.

 state := <- manager.QuorumState // bool returning current quorum state, will update on node join/leave
 node := <- manager.NodeJoin  // string of node joining the cluster
 node := <- manager.NodeLeave // string of node leaving the cluster

These channels are available to read additional cluster status updates

 request := <-manager.FromClusterApi // recieve APIRequest{} send via the API interface by a client

With APIEnabled you can recieve API requests though an authenticated web interface

 log := <-manager.Log // recieve Logging from the debug package

Read the Log channel to receive cluster wide logging
*/
package cluster
