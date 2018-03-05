package healthcheck

import (
	"fmt"
	"net"
	"time"
)

// tcpConnect only does a tcp connection check
func tcpConnect(host string, port int, sourceIP string, healthCheck HealthCheck) (Status, error) {
	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return Offline, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with timeouts
	dialer := &net.Dialer{
		LocalAddr: &localTCPAddr,
		Timeout:   time.Duration(healthCheck.Timeout) * time.Second,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return Offline, err
	}

	conn.Close()
	return Online, nil
}
