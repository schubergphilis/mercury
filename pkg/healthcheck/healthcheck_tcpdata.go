package healthcheck

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"time"
)

// tcpData does a simple tcp connect/reply check
func tcpData(host string, port int, sourceIP string, healthCheck HealthCheck) (Status, error, string) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return Offline, err, fmt.Sprintf("failed to resolve to an address: %s:%d", host, port)
	}

	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return Offline, errl, fmt.Sprintf("failed to resolve to an ip adress: %s", sourceIP)
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with
	conn, err := net.DialTCP("tcp", &localTCPAddr, tcpAddr)
	if err != nil {
		return Offline, err, fmt.Sprintf("failed to dail from source: %+v target: %+v", localTCPAddr, *tcpAddr)
	}

	defer conn.Close()

	fmt.Fprintf(conn, healthCheck.TCPRequest)
	r, err := regexp.Compile(healthCheck.TCPReply)
	if err != nil {
		return Offline, err, fmt.Sprintf("regex Compile failed on %s", healthCheck.TCPReply)
	}

	conn.SetReadDeadline(time.Now().Add(time.Duration(healthCheck.Timeout) * time.Second))
	for {
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return Offline, err, fmt.Sprintf("failed with last input %s", line)
		}

		if r.MatchString(line) {
			return Online, nil, "OK"
		}
	}
}
