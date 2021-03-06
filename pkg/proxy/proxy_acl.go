package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
)

// ACL is used by HTTP proxies for setting/removing headers, cookies or status code
type ACL struct {
	Action         string   `json:"action" toml:"action"`                   // remove, replace, add, deny
	HeaderKey      string   `json:"header_key" toml:"header_key"`           // header key
	HeaderValue    string   `json:"header_value" toml:"header_value"`       // header value
	CookieKey      string   `json:"cookie_key" toml:"cookie_key"`           // cookie key
	CookieValue    string   `json:"cookie_value" toml:"cookie_value"`       // cookie value
	CookiePath     string   `json:"cookie_path" toml:"cookie_path"`         // cookie path
	CookieExpire   duration `json:"cookie_expire" toml:"cookie_expire"`     // cookie expiry date
	CookieSecure   *bool    `json:"cookie_secure" toml:"cookie_secure"`     // cookie secure
	Cookiehttponly *bool    `json:"cookie_httponly" toml:"cookie_httponly"` // cookie httponly
	ConditionType  string   `json:"conditiontype" toml:"conditiontype"`     // header, cookie, other?
	ConditionMatch string   `json:"conditionmatch" toml:"conditionmatch"`   // header text (e.g. /^Content-Type: (.*)/(.*)$/i)
	URLMatch       string   `json:"urlmatch" toml:"urlmatch"`               // url match #^/(.*)#
	URLRewrite     string   `json:"urlrewrite" toml:"urlrewrite"`           // url rewrite /Other/Path/$1
	StatusCode     int      `json:"status_code" toml:"status_code"`         // status code
	URLPath        string   `json:"url_path" toml:"url_path"`               // request path to match this acl if provided
	CIDRS          []string `json:"cidrs" toml:"cidrs"`                     // network cidr
}

// ACLS contains a list of ACL
type ACLS []ACL

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}

func (d *duration) UnmarshalJSON(text []byte) (err error) {
	r, _ := regexp.Compile(`{"Duration":(\d+)}`)
	t := r.FindStringSubmatch(string(text))
	if len(t) == 2 {
		d.Duration, err = time.ParseDuration(t[1] + "ns")
	}
	return
}

const (
	headerMatch  = "header"
	cookieMatch  = "cookie"
	statusMatch  = "status"
	rewriteMatch = "rewrite"
	addMatch     = "add"
	replaceMatch = "replace"
	modifyMatch  = "modify"
	removeMatch  = "remove"
	denyMatch    = "deny"
	allowMatch   = "allow"
)

// ProcessRequest processes ACL's for request
func (acl ACL) ProcessRequest(req *http.Request) (deny bool) {

	// If we have a request path, see if we match this before processing this request
	if acl.URLPath != "" && req.URL != nil {
		regex, _ := regexp.Compile(acl.URLPath)
		if regex.MatchString(req.URL.Path) == false {
			return false
		}
	}

	switch acl.ConditionType {
	case headerMatch:
		return acl.processHeader(&req.Header)

	case cookieMatch:
		return acl.processCookie(&req.Header, nil, "Cookie")

	default: // always executed
		if acl.URLMatch != "" {
			return acl.processURI(req)
		}

		if acl.HeaderKey != "" {
			return acl.processHeader(&req.Header)
		}

		if acl.CookieKey != "" {
			return acl.processCookie(&req.Header, nil, "Cookie")
		}

		if len(acl.CIDRS) > 0 {
			return acl.processCIDR(req.RemoteAddr)
		}
	}
	return false
}

// ProcessResponse processes ACL's for response
func (acl ACL) ProcessResponse(res *http.Response) (deny bool) {

	// If we have a request path, see if we match this before processing this request
	if acl.URLPath != "" && res.Request != nil && res.Request.URL != nil {
		regex, _ := regexp.Compile(acl.URLPath)
		if regex.MatchString(res.Request.URL.Path) == false {
			return false
		}
	}

	if res == nil {
		return false
	}

	if res.Header == nil {
		return false
	}

	switch acl.ConditionType {
	case headerMatch:
		return acl.processHeader(&res.Header)

	case cookieMatch:
		if res.Request != nil {
			return acl.processCookie(&res.Request.Header, &res.Header, "Set-Cookie")
		}
		return acl.processCookie(nil, &res.Header, "Set-Cookie")

	case statusMatch:
		return acl.processStatus(res)

	default: // always executed
		if acl.HeaderKey != "" {
			return acl.processHeader(&res.Header)
		}

		if acl.CookieKey != "" {
			if res.Request != nil {
				return acl.processCookie(&res.Request.Header, &res.Header, "Set-Cookie")
			}
			return acl.processCookie(nil, &res.Header, "Set-Cookie")
		}

		if acl.StatusCode >= 100 {
			return acl.processStatus(res)
		}
	}

	return false
}

// CountActions returns the number of matches of action are in the ACL's
func (acls ACLS) CountActions(action string) (count int) {
	for _, acl := range acls {
		if acl.Action == action {
			count++
		}
	}

	return
}

// ProcessTCPRequest processes ACL's for tcp proxy
func (acl ACL) ProcessTCPRequest(clientIP string) (deny bool) {

	if len(acl.CIDRS) > 0 {
		return acl.processCIDR(clientIP)
	}
	return false
}

func (acl ACL) String() string {
	output := fmt.Sprintf("Action: %s", acl.Action)
	if acl.HeaderKey != "" {
		output += fmt.Sprintf(" Type:Header Key:%s Value:%s", acl.HeaderKey, acl.HeaderValue)
	}
	if acl.CookieKey != "" {
		output += fmt.Sprintf(" Type:Cookie Key:%s Value:%s Path:%s Expire:%s Secure:%t HttpOnly:%t",
			acl.CookieKey,
			acl.CookieValue,
			acl.CookiePath,
			acl.CookieExpire,
			*acl.CookieSecure,
			*acl.Cookiehttponly)
	}
	if acl.URLMatch != "" {
		output += fmt.Sprintf(" Type:URL Match:%s Rewrite:%s", acl.URLMatch, acl.URLRewrite)
	}
	if acl.URLPath != "" {
		output += fmt.Sprintf(" URLPath:%s", acl.URLPath)
	}
	if acl.ConditionType != "" {
		output += fmt.Sprintf(" ConditionType:%s ConditionMatch:%s", acl.ConditionType, acl.ConditionMatch)
	}
	if acl.StatusCode != 0 {
		output += fmt.Sprintf(" StatusCode:%d", acl.StatusCode)
	}
	if len(acl.CIDRS) > 0 {
		output += fmt.Sprintf(" CIDRS:%v", acl.CIDRS)
	}
	return output
}
