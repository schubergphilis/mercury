package healthcheck

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// tcpConnect only does a tcp connection check
func ipPing(proto string, host string, port int, sourceIP string, healthCheck HealthCheck) (bool, error) {
	errorcount := 0
	errormsg := ""
	for i := 0; i < healthCheck.PINGpackets; i++ {
		_, _, err := pingAddr(proto, host, port, sourceIP, i, 64, time.Duration(healthCheck.PINGtimeout)*time.Second)
		if err != nil {
			errorcount++
			errormsg = err.Error()
		}
		time.Sleep(1 * time.Second)
	}

	if errorcount == healthCheck.PINGpackets {
		return false, fmt.Errorf("%s ping lost 100%% packets: %s", proto, errormsg)
	}

	return true, nil
}

func pingAddr(proto string, host string, port int, sourceIP string, seq int, dataSize int, timeout time.Duration) (bool, int, error) {

	var conn net.Conn
	var err error
	switch proto {
	case "icmp":
		localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
		if errl != nil {
			return false, 0, errl
		}

		// Custom dialer with timeouts
		dialer := &net.Dialer{
			LocalAddr: localAddr,
			Timeout:   timeout,
			DualStack: true,
		}
		conn, err = dialer.Dial("ip4:icmp", host)
		if err != nil {
			return false, 0, err
		}
	case "udp":
		localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
		if errl != nil {
			return false, 0, errl
		}

		localUDPAddr := net.UDPAddr{
			IP: localAddr.IP,
		}

		// Custom dialer with timeouts
		dialer := &net.Dialer{
			LocalAddr: &localUDPAddr,
			Timeout:   timeout,
			DualStack: true,
		}
		conn, err = dialer.Dial("udp4", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			return false, 0, err
		}
	case "tcp":
		localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
		if errl != nil {
			return false, 0, errl
		}

		localTCPAddr := net.TCPAddr{
			IP: localAddr.IP,
		}

		// Custom dialer with timeouts
		dialer := &net.Dialer{
			LocalAddr: &localTCPAddr,
			Timeout:   timeout,
			DualStack: true,
		}
		conn, err = dialer.Dial("tcp4", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			return false, 0, err
		}
	}

	defer conn.Close()
	pingMsg := getEchoMsg(seq, []byte(strings.Repeat("h", dataSize)))
	conn.SetWriteDeadline(time.Now().Add(timeout))
	size, err := conn.Write(pingMsg)
	if err != nil {
		return false, 0, err
	}
	if size != len(pingMsg) {
		return false, 0, errors.New("send ping data err")
	}
	beginTime := time.Now()
	for time.Now().Sub(beginTime) < timeout {
		allData := make([]byte, 20+size)
		conn.SetReadDeadline(time.Now().Add(timeout))
		_, err := conn.Read(allData)
		if err != nil {
			return false, 0, err
		}
		header, err := ipv4.ParseHeader(allData)
		if err != nil {
			return false, 0, err
		}
		var msg *icmp.Message
		msg, err = icmp.ParseMessage(1, allData[header.Len:header.TotalLen])
		if err != nil {
			return false, 0, nil
		}
		switch msg.Type {
		case ipv4.ICMPTypeEcho:
			continue
		case ipv4.ICMPTypeEchoReply:
			msg.Body.Marshal(1)
			if _, ok := msg.Body.(*icmp.Echo); !ok {
				return false, 0, errors.New("ping recv err data")
			}
			return true, header.TTL, nil
		default:
			continue
		}
	}
	return false, 0, errors.New("ping addr" + host + "timeout")
}

func getEchoMsg(seq int, data []byte) []byte {
	timeNow := time.Now().UnixNano()
	timeData := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeData, uint64(timeNow))
	data = append(timeData, data...)
	echoMsg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seq,
			Data: data,
		},
	}
	echoData, err := echoMsg.Marshal(nil)
	if err != nil {
		panic(err)
	}
	return echoData
}
