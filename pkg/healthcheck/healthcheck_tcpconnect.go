package healthcheck

import (
	"fmt"
	"net"
	"time"
)

// tcpConnect only does a tcp connection check
func tcpConnect(host string, port int, sourceIP string, healthCheck HealthCheck) (bool, error) {
	//log := logging.For("healthcheck/tcpconnect")
	//log.Debugf("Connecting to %s:%d", host, port)

	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return false, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with timeouts
	dialer := &net.Dialer{
		LocalAddr: &localTCPAddr,
		Timeout:   time.Duration(healthCheck.Timeout) * time.Second,
		//Deadline:  time.Now().Add(time.Duration(healthCheck.Timeout) * time.Second),
		DualStack: true,
	}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		//log.Debugf("Connect to %s:%d failed:%+v", host, port, err)
		return false, err
	}
	conn.Close()
	/*
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Duration(healthCheck.Timeout)*time.Second)
		if err != nil {
			//log.Debugf("Connect to %s:%d failed:%+v", host, port, err)
			return false, err
		}
		//log.Debugf("Connect to %s:%d ok:%+v", host, port, conn)
		conn.Close()
	*/
	return true, nil
}
