package healthcheck

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/schubergphilis/mercury.v2/internal/models"
)

// tcpData does a simple tcp connect/reply check
func tcpData(h models.Healthcheck) (models.Status, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", h.TargetIP, h.TargetPort))
	if err != nil {
		return models.Offline, err
	}

	localAddr, errl := net.ResolveIPAddr("ip", h.SourceIP)
	if errl != nil {
		return models.Offline, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with
	conn, err := net.DialTCP("tcp", &localTCPAddr, tcpAddr)
	if err != nil {
		return models.Offline, err
	}

	defer conn.Close()

	fmt.Fprintf(conn, h.TCPRequest)
	r, err := regexp.Compile(h.TCPReply)
	if err != nil {
		return models.Offline, err
	}

	conn.SetReadDeadline(time.Now().Add(time.Duration(h.Timeout) * time.Second))
	for {
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return models.Offline, err
		}

		if r.MatchString(line) {
			return models.Online, nil
		}
	}
}
