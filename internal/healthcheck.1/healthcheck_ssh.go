package healthcheck

import (
	"fmt"
	"net"

	"github.com/schubergphilis/mercury.v3/internal/models"
	"golang.org/x/crypto/ssh"
)

// tcpData does a simple tcp connect/reply check
func sshAuth(h models.Healthcheck) (models.Status, error) {

	var sshConfig *ssh.ClientConfig
	if h.SSHPassword != "" {
		sshConfig = &ssh.ClientConfig{
			User: h.SSHUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(h.SSHPassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		sshConfig = &ssh.ClientConfig{
			User: h.SSHUser,
			Auth: []ssh.AuthMethod{
				publicKeyFile(h.SSHKey),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

	}
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
	_, _, _, err = ssh.NewClientConn(conn, h.TargetIP, sshConfig)
	if err != nil {
		return models.Offline, err
	}

	return models.Online, nil
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
