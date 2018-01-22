package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/schubergphilis/mercury/src/logging"
)

// removeHeader removes matching headers
func removeHeader(header *http.Header, match string) {
	log := logging.For("proxy/removeheader")
	var matches = 0
	//log.Debugf("Remove header -> regex match: %s", match)
	reg, err := regexp.Compile("(?i)" + match)
	if err != nil {
		log.WithField("match", match).WithError(err).Warn("Invalid regex while matching headers")
	}
	for s, m := range *header {
		line := fmt.Sprintf("%s: %s", s, strings.Join(m, " "))
		if reg.MatchString(line) {
			log.WithField("match", match).WithField("header", line).Debug("Removing header")
			header.Del(s)
			matches++
		}
	}
	if matches == 0 {
		log.WithField("match", match).Debug("Could not remove non-existing header")
	}
	//log.Debugf("Headers after remove:", header)
}

// removeHeader removes matching headers
func replaceHeader(header *http.Header, match string, key string, value string) {
	var addheader int
	var old string
	log := logging.For("proxy/replaceheader")
	reg, err := regexp.Compile("(?i)" + match) // Case insensetive match
	if err != nil {
		log.WithField("match", match).WithError(err).Warn("Invalid regex while matching headers")
	}
	for s, m := range *header {
		line := fmt.Sprintf("%s: %s", s, strings.Join(m, " "))
		if reg.MatchString(line) {
			old = line
			header.Del(s)
			addheader = 1
		}
		//new := reg.ReplaceAllString(line, action)
	}
	if addheader == 1 {
		log.WithField("match", match).WithField("old", old).WithField("new", fmt.Sprintf("%s: %s", key, value)).Debug("Replacing header")
		header.Add(key, value)
	} else {
		log.WithField("match", match).WithField("new", fmt.Sprintf("%s: %s", key, value)).Debug("Not replacing nonexisting header")
	}
	return
}

// addHeader adds a http header
func addHeader(header *http.Header, key string, value string) {
	log := logging.For("proxy/addheader")
	exists := header.Get(key)
	if exists != "" {
		log.WithField("old", fmt.Sprintf("%s: %s", key, exists)).WithField("new", fmt.Sprintf("%s: %s", key, value)).Debug("Skipping add of header, it already exists")
	} else {
		log.WithField("new", fmt.Sprintf("%s: %s", key, value)).Debug("Adding header")
		header.Add(key, value)
	}
}

// matchHeader returns true or false if a header matches
func matchHeader(header *http.Header, match string) bool {
	log := logging.For("proxy/matchheader")
	//log.Debugf("UserAgent XXX: %s", header.Get("User-Agent"))
	var matches = 0
	log.Debugf("Match header -> regex match: %s", match)
	reg, err := regexp.Compile("(?i)" + match)
	if err != nil {
		log.WithField("match", match).WithError(err).Warn("Invalid regex while matching headers")
	}
	for s, m := range *header {
		line := fmt.Sprintf("%s: %s", s, strings.Join(m, " "))
		if reg.MatchString(line) {
			log.WithField("match", match).WithField("header", line).Debug("Matched header")
			matches++
		} else {
			log.Debugf("line '%s' does not match regex '%s'", line, "(?i)"+match)
		}
	}
	log.Debugf("Match header -> regex match(%d): %s", matches, match)
	return matches > 0
}

// processHeader calls the correct handler when editing headers
// returns true if we need to deny
func (acl ACL) processHeader(header *http.Header) (deny bool) {
	switch acl.Action {
	case removeMatch:
		if acl.HeaderKey != "" {
			removeHeader(header, fmt.Sprintf("^%s:", acl.HeaderKey))
		} else if acl.ConditionMatch != "" {
			removeHeader(header, acl.ConditionMatch)
		}
	case replaceMatch:
		if acl.ConditionMatch != "" {
			replaceHeader(header, acl.ConditionMatch, acl.HeaderKey, acl.HeaderValue)
		} else {
			replaceHeader(header, fmt.Sprintf("^%s:", acl.HeaderKey), acl.HeaderKey, acl.HeaderValue)
		}
	case addMatch:
		addHeader(header, acl.HeaderKey, acl.HeaderValue)
	case denyMatch:
		if acl.ConditionMatch != "" {
			return matchHeader(header, acl.ConditionMatch)
		}
		return matchHeader(header, fmt.Sprintf("^%s:%s", acl.HeaderKey, acl.HeaderValue))
	}
	return false
}
