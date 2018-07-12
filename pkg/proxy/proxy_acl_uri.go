package proxy

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/schubergphilis/mercury/pkg/logging"
)

// processUri parses the request URL for modification, allow or deny actions
func (acl ACL) processURI(req *http.Request) (deny bool) {
	log := logging.For("proxy/acluri")
	re, err := regexp.Compile(acl.URLMatch)
	if err != nil {
		log.WithError(err).Warn("unable to parse regex for URLMatch")
		return false
	}
	if acl.URLRewrite != "" {
		requestURI := re.ReplaceAllString(req.URL.RequestURI(), acl.URLRewrite)
		parsedURI, err := url.Parse(requestURI)
		if err != nil {
			log.WithField("new", requestURI).WithError(err).Warn("failed to parse new url", err)
			return
		}
		original := req.RequestURI
		req.URL = parsedURI
		log.WithField("original", original).WithField("urlmatch", acl.URLMatch).WithField("urlrewrite", acl.URLRewrite).WithField("new", req.URL.RequestURI()).Debug("ACL rewrite")
	}
	if acl.Action == "deny" {
		// if we match, and action is deny => return true
		return re.MatchString(req.URL.RequestURI())
	}
	if acl.Action == "allow" {
		// if we match, and action is allow => return false
		return !re.MatchString(req.URL.RequestURI())
	}
	return false
}
