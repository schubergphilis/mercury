package healthcheck

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/schubergphilis/mercury.v2/internal/logging"
	"github.com/schubergphilis/mercury.v2/internal/models"
)

// Worker type executes a healthcheck on a single node
type Worker struct {
	Check       models.Healthcheck `json:"check" toml:"check"`
	CheckResult models.Status      `json:"checkresult" toml:"checkresult"` //
	CheckError  string             `json:"checkerror" toml:"checkerror"`
	UUID        string             `json:"uuid" toml:"uuid"`
	update      chan models.CheckResult
	stop        chan bool
	log         logging.SimpleLogger
}

// NewWorker creates a new worker for healthchecks
func NewWorker(logger logging.SimpleLogger, uuid string, check models.Healthcheck, cr chan models.CheckResult) *Worker {
	return &Worker{
		Check:  check,
		update: cr,
		UUID:   uuid,
		stop:   make(chan bool, 1),
		log:    logger,
	}
}

// ErrorMsg provides a friendly version of the error message
func (w *Worker) ErrorMsg() string {
	if w.CheckResult == models.Online {
		return ""
	}

	return fmt.Sprintf("%s %s", w.Check.Description(), w.CheckError)
}

// Debug shows debug information for all workers
func (w *Worker) Debug() {
	//log := logging.For("healthcheck/worker/debug")
}

// Start worker, report check result to manager
func (w *Worker) Start() {
	w.log.Debugf("starting healthcheck", "uuid", w.UUID, "interval", w.Check.Interval)
	// Enter 3 second random delay before starting checks
	timer := time.NewTimer(time.Duration(rand.Intn(3000)) * time.Millisecond)
	go func() {
		for {
			select {
			/* new check interval has reached */
			case <-timer.C:
				result, err := executeCheck(w.Check)

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
					w.log.Warnf("healthcheck state changed", "uuid", w.UUID, "type", w.Check.Type, "status", result, "error", err)
					// Send the result to the cluster
					checkresult := models.CheckResult{
						Status: result,
						UUID:   w.UUID,
					}

					w.CheckResult = result
					w.CheckError = ""
					if err != nil {
						w.CheckError = err.Error()
						checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
					}

					w.update <- checkresult
				}
				timer = time.NewTimer(time.Duration(w.Check.Interval) * time.Second)

			case <-w.stop:
				checkresult := models.CheckResult{
					Status: models.Offline,
					UUID:   w.UUID,
				}

				w.CheckError = "healthcheck worker is exiting"
				checkresult.ErrorMsg = append(checkresult.ErrorMsg, w.ErrorMsg())
				w.update <- checkresult
				timer.Stop()
				w.log.Debugf("exiting healthcheck", "uuid", w.UUID)
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
func executeCheck(h models.Healthcheck) (models.Status, error) {
	var err error
	var result models.Status

	switch h.Type {
	case "tcpconnect":
		result, err = tcpConnect(h)

	case "tcpdata":
		result, err = tcpData(h)

	case "ssh":
		result, err = sshAuth(h)

	case "httpget":
		result, err = httpRequest("GET", h)

	case "httppost":
		result, err = httpRequest("POST", h)

	case "icmpping":
		result, err = ipPing("icmp", h)

	case "udpping":
		result, err = ipPing("udp", h)

	case "tcpping":
		result, err = ipPing("tcp", h)

	default:
		result = models.Online
	}
	return result, err
}
