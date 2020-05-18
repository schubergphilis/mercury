package healthcheck

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// Worker type executes a healthcheck on a single node
type Worker struct {
	Pool        string      `json:"pool" toml:"pool"`
	Backend     string      `json:"backend" toml:"backend"`
	NodeName    string      `json:"nodename" toml:"nodename"`
	NodeUUID    string      `json:"nodeuuid" toml:"nodeuuid"`
	IP          string      `json:"ip" toml:"ip"` // IP used for check
	SourceIP    string      `json:"sourceip" toml:"sourceip"`
	Port        int         `json:"port" toml:"port"` // Port used for check
	Check       HealthCheck `json:"check" toml:"check"`
	CheckResult Status      `json:"checkresult" toml:"checkresult"` //
	CheckError  string      `json:"checkerror" toml:"checkerror"`
	UUIDStr     string      `json:"uuid" toml:"uuid"`
	update      chan CheckResult
	stop        chan bool
}

// NewWorker creates a new worker for healthchecks
func NewWorker(pool string, backend string, nodeName string, nodeUUID string, ip string, port int, sourceIP string, check HealthCheck, cr chan CheckResult) *Worker {
	return &Worker{
		Pool:     pool,
		Backend:  backend,
		NodeName: nodeName,
		IP:       ip, // Optional alternative IP
		SourceIP: sourceIP,
		Port:     port, // Optional alternative Port
		Check:    check,
		update:   cr,
		stop:     make(chan bool, 1),
		NodeUUID: nodeUUID,
	}
}

// ErrorMsg provides a friendly version of the error message
func (w *Worker) ErrorMsg() string {
	if w.CheckResult == Online {
		return ""
	}

	return fmt.Sprintf("%s %s", w.Description(), w.CheckError)
}

// Description provides a description of the check that the worker is managing
func (w *Worker) Description() string {
	switch w.Check.Type {
	case "tcpconnect":
		return fmt.Sprintf("tcpconnect:%s:%d", w.IP, w.Port)

	case "tcpdata":
		return fmt.Sprintf("tcpdata:%s:%d:%s", w.IP, w.Port, w.Check.TCPRequest)

	case "ssh":
		return fmt.Sprintf("ssh:%s:%d:%s", w.IP, w.Port, w.Check.SSHUser)

	case "httpget":
		return fmt.Sprintf("httpget:%s:%d:%s", w.IP, w.Port, strings.Split(w.Check.HTTPRequest, "?")[0])

	case "httppost":
		return fmt.Sprintf("httppost:%s:%d:%s", w.IP, w.Port, strings.Split(w.Check.HTTPRequest, "?")[0])

	case "icmpping":
		return fmt.Sprintf("icmpping:%s", w.IP)

	case "tcpping":
		return fmt.Sprintf("tcpping:%s:%d", w.IP, w.Port)

	case "udppping":
		return fmt.Sprintf("udpping:%s:%d", w.IP, w.Port)

	default:
		return fmt.Sprintf("unkown:%s:%s:%d", w.Check.Type, w.IP, w.Port)
	}
}

// UUID returns a uniq ID for the worker
func (w *Worker) UUID() string {
	// UUID returns a uuid of a healthcheck
	if w.UUIDStr != "" {
		return w.UUIDStr
	}

	s := fmt.Sprintf("%s%s%s%s%s%s%d", w.Check.UUID(), w.Pool, w.Backend, w.NodeName, w.IP, w.SourceIP, w.Port)
	t := sha256.New()
	t.Write([]byte(s))
	w.UUIDStr = fmt.Sprintf("%x", t.Sum(nil))

	return w.UUIDStr

}

// Debug shows debug information for all workers
func (w *Worker) Debug() {
	log := logging.For("healthcheck/worker/debug")
	log.WithField("node", w.NodeName).WithField("result", w.CheckResult).WithField("error", w.CheckError).WithField("pool", w.Pool).WithField("backend", w.Backend).WithField("ip", w.IP).WithField("port", w.Port).WithField("type", w.Check.Type).Info("Active Healthchecks")
}

// Start worker, report check result to manager
func (w *Worker) Start() {
	log := logging.For("healthcheck/worker/start")
	log = log.WithField("pool", w.Pool).WithField("backend", w.Backend).WithField("ip", w.IP).WithField("port", w.Port).WithField("node", w.NodeName)
	// OPTIONAL: time.ParseDuration(w.Check.Interval)
	log.WithField("interval", w.Check.Interval).Debug("Starting healthcheck")
	// Enter 3 second random delay before starting checks
	timer := time.NewTimer(time.Duration(rand.Intn(3000)) * time.Millisecond)
	go func() {
		for {
			select {
			/* new check interval has reached */
			case <-timer.C:
				result, err, _ := w.ExecuteCheck()

				// Send update if check result or error changes
				var checkerror string
				var previouserror string
				if err != nil {
					checkerror = err.Error()
				}

				if w.CheckError != "" {
					previouserror = w.CheckError
				}

				if result != w.CheckResult || checkerror != previouserror {
					log.WithField("checktype", w.Check.Type).WithField("online", result).WithField("error", err).Warn("Healtcheck state changed")
					// Send the result to the cluster
					/*
						checkresult := CheckResult{
							PoolName:       w.Pool,
							BackendName:    w.Backend,
							ActualStatus:   result,
							ReportedStatus: w.reportState(result),
							NodeName:       w.NodeName,
							WorkerUUID:     w.UUID(),
							Description:    w.Description(),
						}

						w.CheckResult = result
						w.CheckError = ""
						if err != nil {
							w.CheckError = err.Error()
							checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
						}

						w.update <- checkresult
					*/
					w.CheckResult = result
					w.CheckError = ""

					var errorMsg []string
					if err != nil {
						w.CheckError = err.Error()
						errorMsg = append(errorMsg, w.ErrorMsg())
					}

					w.SendUpdate(result, errorMsg)
				}
				timer = time.NewTimer(time.Duration(w.Check.Interval) * time.Second)

			case <-w.stop:
				checkresult := CheckResult{
					PoolName:       w.Pool,
					BackendName:    w.Backend,
					ActualStatus:   Offline,
					ReportedStatus: Offline,
					NodeName:       w.NodeName,
					Description:    w.Description(),
					WorkerUUID:     w.UUID(),
				}

				w.CheckError = "healthcheck worker is exiting"
				checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
				w.update <- checkresult
				timer.Stop()
				return
			}
		}
	}()
}

// Poll sends the updated status to the channel
func (w *Worker) SendUpdate(result Status, errMsg []string) {
	checkresult := CheckResult{
		PoolName:       w.Pool,
		BackendName:    w.Backend,
		ActualStatus:   result,
		ReportedStatus: w.reportState(result),
		NodeName:       w.NodeName,
		WorkerUUID:     w.UUID(),
		Description:    w.Description(),
		ErrorMsg:       errMsg,
	}

	w.update <- checkresult
}

// Poll sends the current status to the channel
func (w *Worker) Poll() {
	var errorMsg []string
	if w.CheckError != "" {
		errorMsg = append(errorMsg, w.CheckError)
	}

	w.SendUpdate(w.CheckResult, errorMsg)
}

// Stop the worker
func (w *Worker) Stop() {
	w.stop <- true
}

// executeCheck directs the check to the executioner and returns the result
func (w *Worker) ExecuteCheck() (Status, error, string) {
	var err error
	var result = Offline
	var description string

	switch w.Check.Type {
	case "tcpconnect":
		result, err, description = tcpConnect(w.IP, w.Port, w.SourceIP, w.Check)

	case "tcpdata":
		result, err, description = tcpData(w.IP, w.Port, w.SourceIP, w.Check)

	case "ssh":
		result, err, description = sshAuth(w.IP, w.Port, w.SourceIP, w.Check)

	case "httpget":
		result, err, description = httpRequest("GET", w.IP, w.Port, w.SourceIP, w.Check)

	case "httppost":
		result, err, description = httpRequest("POST", w.IP, w.Port, w.SourceIP, w.Check)

	case "icmpping":
		result, err, description = ipPing("icmp", w.IP, 0, w.SourceIP, w.Check)

	case "udpping":
		result, err, description = ipPing("udp", w.IP, w.Port, w.SourceIP, w.Check)

	case "tcpping":
		result, err, description = ipPing("tcp", w.IP, w.Port, w.SourceIP, w.Check)

	default:
		result = Online
	}
	return result, err, description
}

func (w *Worker) filterWorker() (n Worker) {
	n = *w
	n.Check.HTTPHeaders = []string{}
	n.Check.HTTPPostData = ""
	n.Check.HTTPRequest = strings.Split(n.Check.HTTPRequest, "?")[0]
	return
}

func (w *Worker) sendUpdate(result Status) {
	checkresult := CheckResult{
		PoolName:       w.Pool,
		BackendName:    w.Backend,
		ActualStatus:   result,
		ReportedStatus: result,
		NodeName:       w.NodeName,
		WorkerUUID:     w.UUID(),
		Description:    w.Description(),
	}
	w.update <- checkresult
}

func (w *Worker) reportState(result Status) Status {
	switch result {
	case Online:
		return w.Check.OnlineState.Status
	case Offline:
		return w.Check.OfflineState.Status
	}
	return result
}
