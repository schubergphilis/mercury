package proxy

import (
	"strconv"
	"strings"
)

type clientIP struct {
	IP   string
	Port int
}

func stringToClientIP(addr string) *clientIP {
	client := &clientIP{}
	remoteAddr := strings.Split(addr, ":")
	// if we have no port, assume its an IP only
	if len(remoteAddr) == 1 {
		client.IP = addr
		return client
	}
	// if we have a port definition check wether its ipv6 or a real port number
	if len(remoteAddr) > 1 {
		client.IP = strings.Join(remoteAddr[:len(remoteAddr)-1], ":")
		if port, err := strconv.Atoi(remoteAddr[len(remoteAddr)-1]); err == nil {
			client.Port = port
		}
	}
	return client
}
