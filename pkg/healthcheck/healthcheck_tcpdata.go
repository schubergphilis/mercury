package healthcheck

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"time"
)

// tcpData does a simple tcp connect/reply check
func tcpData(host string, port int, sourceIP string, healthCheck HealthCheck) (bool, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false, err
	}

	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return false, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with
	conn, err := net.DialTCP("tcp", &localTCPAddr, tcpAddr)
	if err != nil {
		return false, err
	}

	defer conn.Close()

	fmt.Fprintf(conn, healthCheck.TCPRequest)
	r, err := regexp.Compile(healthCheck.TCPReply)
	if err != nil {
		return false, err
	}

	conn.SetReadDeadline(time.Now().Add(time.Duration(healthCheck.Timeout) * time.Second))
	for {
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return false, err
		}

		if r.MatchString(line) {
			return true, nil
		}
	}
}
