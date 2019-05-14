package models

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/cnf/structhash"
	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/pkg/tlsconfig"
)

type HealthcheckService interface {
	// Start()
	Stop()

	//CreateHealthCheck(check) (uuid string, err error)
	//UpdateHealthCheck(check, checkUUID) (err error)
	//DeleteHealthCheck(checkUUID) (err error)

	ReceiveHealthCheckStatus() chan CheckResult // receive update of status

	AddHealthcheck(uuid string, check Healthcheck)
	RemoveHealthcheck(uuid string)

	WithLogger(logger logging.SimpleLogger)

	// SendHealthCheckStatus() chan string // force update of status <- handled internally not at library
}

// Healthcheck custom Healthcheck
type Healthcheck struct {
	Type               string              `json:"type" toml:"type"`                             // check type
	TCPRequest         string              `json:"tcprequest" toml:"tcprequest"`                 // tcp request to send
	TCPReply           string              `json:"tcpreply" toml:"tcpreply"`                     // tcp reply to expect
	HTTPRequest        string              `json:"httprequest" toml:"httprequest"`               // http request to send
	HTTPPostData       string              `json:"httppostdata" toml:"httppostdata"`             // http post data to send
	HTTPHeaders        []string            `json:"httpheaders" toml:"httpheaders"`               // http headers to send
	HTTPStatus         int                 `json:"httpstatus" toml:"httpstatus"`                 // http status expected
	HTTPReply          string              `json:"httpreply" toml:"httpreply"`                   // http reply expected
	HTTPFollowRedirect string              `json:"httpfollowredirect" toml:"httpfollowredirect"` // http follow redirects
	SSHUser            string              `json:"sshuser" toml:"sshuser"`                       // ssh user
	SSHPassword        string              `json:"sshpassword" toml:"sshpassword"`               // ssh password
	SSHKey             string              `json:"sshkey" toml:"sshkey"`                         // ssh key
	PINGpackets        int                 `json:"pingpackets" toml:"pingpackets"`               // ping packets to send
	PINGtimeout        int                 `json:"pingtimeout" toml:"pingtimeout"`               // ping timeout
	Interval           int                 `json:"interval" toml:"interval"`                     // how often to cechk
	Timeout            int                 `json:"timeout" toml:"timeout"`                       // timeout performing check
	ActivePassiveID    string              `json:"activepassiveid" toml:"activepassiveid"`       // used to link active/passive backends
	TLSConfig          tlsconfig.TLSConfig `json:"tls" toml:"tls"`                               // tls config
	DisableAutoCheck   bool                `json:"disableautocheck" toml:"disableautocheck"`     // only respond to check requests
	TargetIP           string              `json:"ip" toml:"ip"`                                 // specific ip
	TargetPort         int                 `json:"port" toml:"port"`                             // specific port
	SourceIP           string              `json:"sourceip" toml:"sourceip"`                     // specific ip
	OnlineState        StatusType          `json:"online_state" toml:"online_state"`             // alternative online_state - default: online / optional: offline / maintenance
	OfflineState       StatusType          `json:"offline_state" toml:"offline_state"`           // alternative offline_state - default: offline
}

// CheckResult holds the check result output
type CheckResult struct {
	UUID     string   `json:"uuid" toml:"uuid"`         // worker uuid who performed the check
	Status   Status   `json:"status" toml:"status"`     // status of the check after applying state processing
	ErrorMsg []string `json:"errormsg" toml:"errormsg"` // error message if any
}

func (h *Healthcheck) UUID() string {
	shasum := sha1.Sum(structhash.Dump(*h, 1))
	return fmt.Sprintf("%x", shasum)
}

// Description provides a description of the check that the worker is managing
func (h *Healthcheck) Description() string {
	switch h.Type {
	case "tcpconnect":
		return fmt.Sprintf("tcpconnect:%s:%d", h.TargetIP, h.TargetPort)

	case "tcpdata":
		return fmt.Sprintf("tcpdata:%s:%d:%s", h.TargetIP, h.TargetPort, h.TCPRequest)

	case "ssh":
		return fmt.Sprintf("ssh:%s:%d:%s", h.TargetIP, h.TargetPort, h.SSHUser)

	case "httpget":
		return fmt.Sprintf("httpget:%s:%d:%s", h.TargetIP, h.TargetPort, strings.Split(h.HTTPRequest, "?")[0])

	case "httppost":
		return fmt.Sprintf("httppost:%s:%d:%s", h.TargetIP, h.TargetPort, strings.Split(h.HTTPRequest, "?")[0])

	case "icmpping":
		return fmt.Sprintf("icmpping:%s", h.TargetIP)

	case "tcpping":
		return fmt.Sprintf("tcpping:%s:%d", h.TargetIP, h.TargetPort)

	case "udppping":
		return fmt.Sprintf("udpping:%s:%d", h.TargetIP, h.TargetPort)

	default:
		return fmt.Sprintf("unkown:%s:%s:%d", h.Type, h.TargetIP, h.TargetPort)
	}
}
