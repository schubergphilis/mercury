package cluster

import (
	"crypto/tls"
	"net"
)

// server defines the server part of the cluster service
type server struct {
	addr      string
	close     chan bool
	listener  net.Listener
	tlsConfig *tls.Config
}

func newServer(addr string, tlsConfig *tls.Config) *server {
	s := &server{
		addr:      addr,
		tlsConfig: tlsConfig,
	}
	return s
}

// Listen creates the listener for the cluster server
func (s *server) Listen() (ln net.Listener, err error) {
	if len(s.tlsConfig.Certificates) == 0 {
		s.listener, err = net.Listen("tcp", s.addr)
	} else {
		s.listener, err = tls.Listen("tcp", s.addr, s.tlsConfig)
	}
	if err != nil {
		return
	}
	return s.listener, nil
}

// Serve accepts connections and forwards these to the cluster server
func (s *server) Serve(newSocket chan net.Conn, quit chan bool) {
	defer s.listener.Close()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-quit:
				// quit was initiated outside of this function
				return
			default:
			}

			continue
		}
		newSocket <- conn
	}
}
