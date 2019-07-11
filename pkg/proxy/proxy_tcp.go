package proxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// NewTCPProxy creates a new TCP proxy
func (l *Listener) NewTCPProxy() (net.Listener, error) {
	log := logging.For("proxy/tcp/new").WithField("ip", l.IP).WithField("port", l.Port)
	log.Debug("Starting TCP listener")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", l.IP, l.Port))
	if err != nil {
		return nil, fmt.Errorf("Error starting listener on %s:%d error:%s", l.IP, l.Port, err)
	}

	l.socket = limitListenerConnections(listener.(*net.TCPListener), l.MaxConnections)
	return l.socket, nil
}

// TCPProxy starts accepting connections
func (l *Listener) TCPProxy(n net.Listener) {
	log := logging.For("proxy/tcp/accept")
	if n == nil {
		log.Warn("No listener was connected, cannot accept its connections!")
		return
	}

	for {
		client, err := n.Accept()
		if err != nil {
			if v, ok := n.(*limitListener); ok && v.IsClosed() {
				return // Do nothing for we closed it.
			}

			log.WithField("error", err).Warn("Error accepting connection, closing listener")
			return
		}

		log.WithField("client", client.RemoteAddr()).Info("New TCP proxy client connected")
		go l.Handler(client)
	}
}

// Handler handles clients and connectors proxys
func (l *Listener) Handler(client net.Conn) {
	clientip := strings.Split(client.RemoteAddr().String(), ":")
	log := logging.For("proxy/tcp/handler").WithField("pool", l.Name).WithField("localip", l.IP).WithField("localport", l.Port).WithField("clientip", clientip[0]).WithField("clientaddr", client.RemoteAddr())
	if l.SourceIP != "" {
		log = log.WithField("sourceip", l.SourceIP)
	}
	log.Infof("Forwarding TCP client")

	l.Statistics.ClientsConnectsAdd(1)
	l.Statistics.ClientsConnectedAdd(1)

	l.updateClients()
	defer l.updateClients()

	// for TCP we only accept 1 backend, so return the first (any only) entry
	backend, err := l.GetBackend()
	if err != nil {
		log.WithField("connecttime", 0).WithField("transfertime", 0).WithError(err).Error("Forwarding TCP aborted")
		client.Close()
		return
	}

	// ACL
	aclAllows := backend.InboundACL.CountActions("allow")
	aclDenies := backend.InboundACL.CountActions("deny")
	// Process all ACL's and count hit's if any
	aclsHit := 0
	for _, inacl := range backend.InboundACL {
		if inacl.ProcessTCPRequest(clientip[0]) { // process request returns true if we match a allow/deny acl
			aclsHit++
		}
	}

	// Take actions based on allow/deny, you cannot combine allow and denies
	if aclDenies > 0 && aclAllows > 0 {
		log.Errorf("Found ALLOW and DENY ACL's in the same block, only allows will be processed")
	}

	if aclAllows > 0 && aclsHit == 0 { // setting an allow ACL, will deny all who do not match atleast 1 allow
		log.Infof("Client did not match allow acl")
		client.Close()
		return
	} else if aclAllows == 0 && aclDenies > 0 && aclsHit > 0 { // setting an deny ACL, will deny all who match 1 of the denies
		log.Infof("Client matched deny acl")
		client.Close()
		return
	}

	node, status, err := backend.GetBackendNodeBalanced(l.Name, clientip[0], "stickyness_not_supported_in_tcp_lb", backend.BalanceMode)
	if err != nil {
		if status == healthcheck.Maintenance {
			log.WithError(err).Error("No backend available")
			client.Close()
			return
		}
		log.WithField("connecttime", 0).WithField("transfertime", 0).WithError(err).Error("Forwarding TCP aborted")
		client.Close()
		return
	}

	clog := log.WithField("remoteip", node.IP).WithField("remoteport", node.Port)
	clog.Debug("Forwarding client to node")
	starttime := time.Now()

	var localAddr *net.IPAddr
	var errl error
	if l.SourceIP != "" {
		localAddr, errl = net.ResolveIPAddr("ip", l.SourceIP)
	} else {
		localAddr, errl = net.ResolveIPAddr("ip", l.IP)
	}
	if errl != nil {
		clog.WithError(errl).Error("Failed to bind to local ip for outbound connection")
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with timeouts
	dialer := &net.Dialer{
		LocalAddr: &localTCPAddr,
		Timeout:   60 * time.Second,
		//Deadline:  time.Now().Add(60 * time.Second),
		DualStack: true,
	}

	remote, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", node.IP, node.Port))
	if err != nil {
		clog.WithField("connecttime", 0).WithField("transfertime", 0).WithError(err).Error("Forwarding TCP aborted")
		client.Close()
		return
	}

	connecttime := time.Since(starttime)
	node.Statistics.ClientsConnectsAdd(1)
	node.Statistics.ClientsConnectedAdd(1)

	// do the copy of data
	in, out, firstByte := netPipe(client, remote)

	// only add first byte if its non nil
	if firstByte != nil {
		firstbytetime := firstByte.Sub(starttime)
		node.Statistics.ResponseTimeAdd(firstbytetime.Seconds())
		clog = clog.WithField("firstbyte", firstbytetime)
	}
	node.Statistics.ClientsConnectedSub(1)
	node.Statistics.RXAdd(in)
	node.Statistics.TXAdd(out)
	l.Statistics.ClientsConnectedSub(1)
	clog.WithField("statistics", fmt.Sprintf("%+v", node.Statistics)).Debug("Statistics updated")

	transfertime := time.Since(starttime)
	clog.WithField("connecttime", connecttime.Seconds()).WithField("transfertime", transfertime.Seconds()).Info("Forwarding TCP finished")
}

func copySourceToDestination(src io.ReadWriter, dst io.ReadWriter, datasent chan<- int64, firstbytereceived chan<- *time.Time) {
	buff := make([]byte, 0xffff)
	firstbytesReceived := false
	var sent int64
	for {
		n, err := src.Read(buff)
		if err != nil {
			if firstbytesReceived == false {
				firstbytereceived <- nil
			}
			break
		}

		// we got data, register first byte
		if len(firstbytereceived) == 0 {
			now := time.Now()
			firstbytereceived <- &now
			firstbytesReceived = true
		}

		b := buff[:n]
		sent += int64(len(b))

		_, err = dst.Write(b)
		if err != nil {
			if firstbytesReceived == false {
				firstbytereceived <- nil
			}
		}
	}
	src.(net.Conn).Close()
	dst.(net.Conn).Close()
	datasent <- sent
}

// netPipe src = client dst = remote, copy data bi-direction
func netPipe(src, dst io.ReadWriter) (in int64, out int64, firstByte *time.Time) {
	toRemoteFinished := make(chan int64)
	fromRemoteFinished := make(chan int64)

	firstByteFromRemote := make(chan *time.Time, 1)
	firstByteFromClient := make(chan *time.Time, 1)

	go copySourceToDestination(src, dst, toRemoteFinished, firstByteFromClient)
	go copySourceToDestination(dst, src, fromRemoteFinished, firstByteFromRemote)

	out = <-toRemoteFinished
	in = <-fromRemoteFinished
	firstByte = <-firstByteFromRemote
	return
}
