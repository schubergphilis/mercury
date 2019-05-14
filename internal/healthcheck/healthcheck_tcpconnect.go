package healthcheck

import (
	"fmt"
	"net"
	"time"

	"github.com/schubergphilis/mercury.v2/internal/models"
)

// tcpConnect only does a tcp connection check
func tcpConnect(h models.Healthcheck) (models.Status, error) {
	localAddr, errl := net.ResolveIPAddr("ip", h.SourceIP)
	if errl != nil {
		return models.Offline, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with timeouts
	dialer := &net.Dialer{
		LocalAddr: &localTCPAddr,
		Timeout:   time.Duration(h.Timeout) * time.Second,
		DualStack: true,
	}

	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", h.TargetIP, h.TargetPort))
	if err != nil {
		return models.Offline, err
	}

	conn.Close()
	return models.Online, nil
}
