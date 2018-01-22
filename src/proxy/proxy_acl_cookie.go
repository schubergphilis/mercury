package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/schubergphilis/mercury/src/logging"
)

// removeHeader removes matching headers
func removeCookie(header *http.Header, cookieHeader string, match string) {
	if cookieHeader == "Set-Cookie" {
		// Set-Cookies must have their own header for each cookie
		//header.Del(fmt.Sprintf("%s: %s")cookieHeader: match)
	} else {
	}
}

// removeHeader removes matching headers
func replaceCookie(header *http.Header, reqHeader *http.Header, cookieHeader string, match string, acl ACL) {
	// remove cookie
	removeCookie(header, cookieHeader, "")
	addCookie(header, reqHeader, cookieHeader, acl, true)
}

// addHeader adds a http header, only if cookie does not exist yet
func addCookie(header *http.Header, reqHeader *http.Header, cookieHeader string, acl ACL, force bool) {
	log := logging.For("proxy/addcookie")
	if force == false {
		if reqHeader != nil && cookieExists(reqHeader, "Cookie", acl.CookieKey) { // do nothing on existing cookies
			return
		} else if cookieExists(header, "Set-Cookie", acl.CookieKey) {
			return
		}
	}
	cookie := acl.newCookie()
	if cookieHeader == "Set-Cookie" {
		// Set-Cookies must have their own header for each cookie
		header.Add(cookieHeader, cookie.String())
	} else {
		// Cookies can have multiple values on 1 cookie: header
		if c := header.Get(cookieHeader); c != "" {
			header.Set(cookieHeader, c+"; "+cookie.String())
		} else {
			header.Set(cookieHeader, cookie.String())
		}
	}
	log.WithField("cookie", cookieHeader).WithField("value", cookie.Value).Debug("Adding Cookie")
}

// processHeader calls the correct handler when editing headers
func (acl ACL) processCookie(header *http.Header, reqHeader *http.Header, cookieName string) (deny bool) {
	switch acl.Action {
	case removeMatch:
		removeCookie(header, cookieName, acl.ConditionMatch)
	case replaceMatch:
		replaceCookie(header, reqHeader, cookieName, acl.ConditionMatch, acl)
	case addMatch:
		addCookie(header, reqHeader, cookieName, acl, false)
	}
	return false
}

// newCookie converts action to a cookie
func (acl ACL) newCookie() *http.Cookie {
	expire := time.Now().Add(acl.CookieExpire.Duration)
	cookie := &http.Cookie{
		Name:     acl.CookieKey,
		Value:    acl.CookieValue,
		Path:     acl.CookiePath,
		Expires:  expire,
		Secure:   acl.CookieSecure,
		HttpOnly: acl.Cookiehttponly}
	return cookie

}

func cookieExists(header *http.Header, cookieHeader string, cookieKey string) bool {
	search := fmt.Sprintf("%s: .*%s", cookieHeader, cookieKey)
	regex, err := regexp.Compile("(?i)" + search)
	if err != nil {
		return false
	}
	for key, hdr := range *header {
		for _, hdrstr := range hdr {
			if regex.MatchString(fmt.Sprintf("%s: %s", key, hdrstr)) == true {
				return true
			}
		}
	}
	return false
}
