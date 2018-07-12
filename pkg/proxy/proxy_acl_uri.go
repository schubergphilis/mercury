package proxy

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// processHeader calls the correct handler when editing headers
func (acl ACL) processUri(req *http.Request) (deny bool) {
	log := logging.For("proxy/acluri")
	//res.StatusCode = acl.StatusCode
	//res.Status = http.StatusText(acl.StatusCode)
	re, err := regexp.Compile(acl.URLMatch)
	if err != nil {
		log.WithError(err).Warn("unable to parse regex for URLMatch")
		return false
	}
	if acl.URLRewrite != "" {
		requestUri := re.ReplaceAllString(req.URL.RequestURI(), acl.URLRewrite)
		parsedUri, err := url.Parse(requestUri)
		if err != nil {
			log.WithField("new", requestUri).WithError(err).Warn("failed to parse new url", err)
			return
		}
		original := req.RequestURI
		req.URL = parsedUri
		log.WithField("original", original).WithField("urlmatch", acl.URLMatch).WithField("urlrewrite", acl.URLRewrite).WithField("new", req.URL.RequestURI()).Debug("ACL rewrite")
	}
	if acl.Action == "deny" {
		// if we match, and action is deny => return true
		//fmt.Printf("match of %s with %s = %t\n", acl.URLMatch, req.URL.RequestURI(), re.MatchString(req.URL.RequestURI()))
		return re.MatchString(req.URL.RequestURI())
	}
	if acl.Action == "allow" {
		// if we match, and action is allow => return false
		return !re.MatchString(req.URL.RequestURI())
	}
	return false
}
