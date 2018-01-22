package proxy

import (
	"net/http"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// processHeader calls the correct handler when editing headers
func (acl ACL) processStatus(res *http.Response) (deny bool) {
	log := logging.For("proxy/addstatus")
	log.WithField("statuscode", acl.StatusCode).Debug("ACL Write status code")
	res.StatusCode = acl.StatusCode
	res.Status = http.StatusText(acl.StatusCode)
	return false
}
