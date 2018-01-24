// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proxy

import (
	"net"
	"sync"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// LimitListenerConnections returns a Listener that accepts at most n simultaneous
// connections from the provided Listener.
func limitListenerConnections(l *net.TCPListener, n int) *limitListener {
	return &limitListener{l, false, make(chan struct{}, n)}
}

type limitListener struct {
	*net.TCPListener
	closed bool
	sem    chan struct{}
}

// Clients returns the number of clients currently connected to a socket
func (l *limitListener) Clients() int {
	return len(l.sem) - 1
}

// acquire accepts new connections, if we reached the maximum allowed, it puts connections on hold
func (l *limitListener) acquire() {
	log := logging.For("proxy/limitaquire").WithField("listener", l.Addr()).WithField("clients", len(l.sem)).WithField("max", cap(l.sem))
	log.Debug("Client acquire")
	select {
	case l.sem <- struct{}{}:
		log.Debug("Client allowed")
	default:
		log.Warn("Max connections reached")
		// let socket wait anyway for new free one
		l.sem <- struct{}{}
	}
}

// release release new connections, removing them from the pool of connected clients
func (l *limitListener) release() {
	log := logging.For("proxy/limitrelease").WithField("listener", l.Addr()).WithField("clients", len(l.sem)).WithField("max", cap(l.sem))
	log.Debug("Client release")
	<-l.sem
}

// Accepts accepts a tcp connection
func (l limitListener) Accept() (c net.Conn, err error) {
	log := logging.For("proxy/limit").WithField("listener", l.Addr())
	l.acquire()
	log.Debug("Waiting to accept new client")
	c, err = l.AcceptTCP()

	if err != nil {
		log.Error("Client failed to connect")
		l.release()
		return
	}

	c = &limitListenerConn{Conn: c, release: l.release}
	log.WithField("client", c.RemoteAddr()).Debug("Client connected")
	return
}

func (l *limitListener) Close() error {
	l.closed = true
	return nil
}

func (l *limitListener) IsClosed() bool {
	return l.closed
}

type limitListenerConn struct {
	net.Conn
	releaseOnce sync.Once
	release     func()
}

// Close closes a clients tcp connection
func (l *limitListenerConn) Close() error {
	log := logging.For("proxy/limitclose")
	log.WithField("client", l.RemoteAddr()).Debug("Client connection closed")
	err := l.Conn.Close()
	l.releaseOnce.Do(l.release)
	return err
}
