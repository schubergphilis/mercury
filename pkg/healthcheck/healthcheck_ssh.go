package healthcheck

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

// tcpData does a simple tcp connect/reply check
func sshAuth(host string, port int, sourceIP string, healthCheck HealthCheck) (Status, error, string) {

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
				publicKeyFile(healthCheck.SSHKey),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

	}
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
	_, _, _, err = ssh.NewClientConn(conn, host, sshConfig)
	if err != nil {
		return Offline, err, fmt.Sprintf("failed to initiate ssh connection on %s with %+v", host, *sshConfig)
	}

	return Online, nil, "OK"
}

// publicKeyFile converts a string in to a ssh public key
func publicKeyFile(keyString string) ssh.AuthMethod {
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
