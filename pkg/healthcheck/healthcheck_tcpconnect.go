package healthcheck

import (
	"fmt"
	"net"
	"time"
)

// tcpConnect only does a tcp connection check
func tcpConnect(host string, port int, sourceIP string, healthCheck HealthCheck) (Status, error, string) {
	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return Offline, errl, fmt.Sprintf("failed to resolve to an ip adress: %s", sourceIP)
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
		return Offline, err, fmt.Sprintf("failed to dial %s:%d", host, port)
	}

	conn.Close()
	return Online, nil, "OK"
}
