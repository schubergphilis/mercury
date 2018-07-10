package healthcheck

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

// tcpData does a simple tcp connect/reply check
func sshAuth(host string, port int, sourceIP string, healthCheck HealthCheck) (Status, error) {

	var sshConfig *ssh.ClientConfig
	if healthCheck.SSHPassword != "" {
		sshConfig = &ssh.ClientConfig{
			User: healthCheck.SSHUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(healthCheck.SSHPassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		sshConfig = &ssh.ClientConfig{
			User: healthCheck.SSHUser,
			Auth: []ssh.AuthMethod{
				PublicKeyFile(healthCheck.SSHKey),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return Offline, err
	}

	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return Offline, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with
	conn, err := net.DialTCP("tcp", &localTCPAddr, tcpAddr)
	if err != nil {
		return Offline, err
	}

	defer conn.Close()
	_, _, _, err = ssh.NewClientConn(conn, host, sshConfig)
	if err != nil {
		return Offline, err
	}

	return Online, nil
}

func PublicKeyFile(keyString string) ssh.AuthMethod {
	/*buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}*/

	key, err := ssh.ParsePrivateKey([]byte(keyString))
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
