package healthcheck

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// Worker type executes a healthcheck on a single node
type Worker struct {
	Pool        string
	Backend     string
	NodeName    string
	NodeUUID    string
	IP          string // IP used for check
	SourceIP    string
	Port        int // Port used for check
	Check       HealthCheck
	stop        chan bool
	CheckResult bool
	CheckError  error
	update      chan CheckResult
	uuidStr     string
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
	if w.CheckResult == true {
		return ""
	}
	return fmt.Sprintf("%s %s", w.Description(), w.CheckError)
	/*
		switch w.Check.Type {
		case "tcpconnect":
			return fmt.Sprintf("tcpconnect:%s", w.CheckError)
		case "tcpdata":
			return fmt.Sprintf("tcpdata:%s %s", w.Check.TCPRequest, w.CheckError)
		case "httpget":
			return fmt.Sprintf("httpget:%s %s", w.Check.HTTPRequest, w.CheckError)
		case "httppost":
			return fmt.Sprintf("httppost:%s %s", w.Check.HTTPRequest, w.CheckError)
		default:
			return ""
		}
	*/
}

// Description provides a description of the check that the worker is managing
func (w *Worker) Description() string {
	switch w.Check.Type {
	case "tcpconnect":
		return fmt.Sprintf("tcpconnect:%s:%d", w.IP, w.Port)
	case "tcpdata":
		return fmt.Sprintf("tcpdata:%s:%d:%s", w.IP, w.Port, w.Check.TCPRequest)
	case "httpget":
		return fmt.Sprintf("httpget:%s:%d:%s", w.IP, w.Port, w.Check.HTTPRequest)
	case "httppost":
		return fmt.Sprintf("httppost:%s:%d:%s", w.IP, w.Port, w.Check.HTTPRequest)
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
	if w.uuidStr != "" {
		return w.uuidStr
	}
	s := fmt.Sprintf("%s%s%s%s%s%s%d", w.Check.UUID(), w.Pool, w.Backend, w.NodeName, w.IP, w.SourceIP, w.Port)
	t := sha256.New()
	t.Write([]byte(s))
	//w.uuidStr = string(t.Sum(nil))
	w.uuidStr = fmt.Sprintf("%x", t.Sum(nil))

	return w.uuidStr

}

// Debug shows debug information for all workers
func (w *Worker) Debug() {
	log := logging.For("healthcheck/worker/debug")
	log.WithField("node", w.NodeName).WithField("result", w.CheckResult).WithError(w.CheckError).WithField("pool", w.Pool).WithField("backend", w.Backend).WithField("ip", w.IP).WithField("port", w.Port).WithField("type", w.Check.Type).Info("Active Healthchecks")
}

// SingleCheck executes and reports a single health check and then exits
func (w *Worker) SingleCheck() {
	result, err := w.executeCheck()
	checkresult := CheckResult{
		//						NodeID:      w.NodeID,
		PoolName:    w.Pool,
		BackendName: w.Backend,
		Online:      result,
		NodeName:    w.NodeName,
		Description: w.Description(),
		SingleCheck: true,
	}
	if err != nil {
		checkresult.ErrorMsg = append(checkresult.ErrorMsg, err.Error())
	}
	w.update <- checkresult
}

// Start worker, report check result to manager
func (w *Worker) Start() {
	log := logging.For("healthcheck/worker/start")
	log = log.WithField("pool", w.Pool).WithField("backend", w.Backend).WithField("ip", w.IP).WithField("port", w.Port).WithField("node", w.NodeName)
	// OPTIONAL: time.ParseDuration(w.Check.Interval)
	//log.Debugf("Starting worker with interval: %d (%+v)", w.Check.Interval, w)
	log.WithField("interval", w.Check.Interval).Debug("Starting healthcheck")
	// Enter 3 second random delay before starting checks
	timer := time.NewTimer(time.Duration(rand.Intn(3000)) * time.Millisecond)
	go func() {
		for {
			select {
			/* new check interval has reached */
			case <-timer.C:
				//log.Debugf("Received ticker, starting check for %s", w.UUID)
				result, err := w.executeCheck()
				//log.Debugf("State Report - Backend:%s node:%s port:%d type:%s online:%t error:%v", w.Backend, w.IP, w.Port, w.Check.Type, result, err)

				// Send update if check result or error changes
				var checkerror string
				var previouserror string
				if err != nil {
					checkerror = err.Error()
				}
				if w.CheckError != nil {
					previouserror = w.CheckError.Error()
				}
				if result != w.CheckResult || checkerror != previouserror {
					//log.Warnf("State Changed - Backend:%s node:%s port:%d type:%s online:%t error:%v UUID:%s", w.Backend, w.IP, w.Port, w.Check.Type, result, err, w.UUID)
					log.WithField("checktype", w.Check.Type).WithField("online", result).WithField("error", err).Warn("Healtcheck state changed")
					// Send the result to the cluster
					checkresult := CheckResult{
						//						NodeID:      w.NodeID,
						PoolName:    w.Pool,
						BackendName: w.Backend,
						Online:      result,
						NodeName:    w.NodeName,
						WorkerUUID:  w.UUID(),
						Description: w.Description(),
						SingleCheck: false,
					}
					w.CheckResult = result
					w.CheckError = err
					if err != nil {
						//checkresult.Error = err.Error()
						checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
					}
					w.update <- checkresult
				}
				timer = time.NewTimer(time.Duration(w.Check.Interval) * time.Second)
			case <-w.stop:
				checkresult := CheckResult{
					//NodeID:      w.NodeID,
					PoolName:    w.Pool,
					BackendName: w.Backend,
					Online:      false,
					//Error:       "healthcheck worker is exiting",
					NodeName:    w.NodeName,
					Description: w.Description(),
					WorkerUUID:  w.UUID(),
				}
				w.CheckError = fmt.Errorf("healthcheck worker is exiting")
				checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
				w.update <- checkresult
				timer.Stop()
				return
			}
		}
	}()
}

// Stop the worker
func (w *Worker) Stop() {
	w.stop <- true
}

// executeCheck directs the check to the executioner and returns the result
func (w *Worker) executeCheck() (bool, error) {
	//log := logging.For("healthcheck/executecheck")
	var err error
	var result = false
	//log.Debugf("Starting check for Backend:%s node:%s port:%d type:%s", w.Backend, w.IP, w.Port, w.Check.Type)

	switch w.Check.Type {
	case "tcpconnect":
		result, err = tcpConnect(w.IP, w.Port, w.SourceIP, w.Check)
	case "tcpdata":
		result, err = tcpData(w.IP, w.Port, w.SourceIP, w.Check)
	case "httpget":
		result, err = httpRequest("GET", w.IP, w.Port, w.SourceIP, w.Check)
	case "httppost":
		result, err = httpRequest("POST", w.IP, w.Port, w.SourceIP, w.Check)
	case "icmpping":
		result, err = ipPing("icmp", w.IP, 0, w.SourceIP, w.Check)
	case "udpping":
		result, err = ipPing("udp", w.IP, w.Port, w.SourceIP, w.Check)
	case "tcpping":
		result, err = ipPing("tcp", w.IP, w.Port, w.SourceIP, w.Check)
	default:
		result = true
	}
	return result, err
}
