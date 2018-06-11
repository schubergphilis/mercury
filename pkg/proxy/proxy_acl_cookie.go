package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"
	"golang.org/x/net/lex/httplex"
)

// removeHeader removes matching headers
func removeCookie(header *http.Header, cookieHeader string, match string) {
	if cookieHeader == "Set-Cookie" {
		// Set-Cookies must have their own header for each cookie
		// header.Del(fmt.Sprintf("%s: %s")cookieHeader: match)
	} else {
		// TODO: put something meaning full here
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
		if reqHeader != nil && cookieExists(reqHeader, "Cookie", acl.CookieKey) && cookieHeader == "Cookie" { // do nothing on existing cookies
			return
		} else if cookieExists(header, "Set-Cookie", acl.CookieKey) && cookieHeader == "Set-Cookie" {
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

	log.WithField("cookie", cookieHeader).WithField(acl.CookieKey, cookie.Value).Debug("Adding Cookie")
}

// modifyCookie modifies a existing
func modifyCookie(header *http.Header, reqHeader *http.Header, cookieHeader string, acl ACL) {
	log := logging.For("proxy/modifycookie")
	log.Debug("Modify cookie called")
	if acl.CookieKey == "" {
		log.Warn("attept to modify a cookie without a cookie key with acl: %+v", acl)
		return
	}
	cookies := readSetCookies(*header, cookieHeader)

	if len(cookies) == 0 {
		log.Debug("Modify cookie called but cookie doesn't exist acl:%+v", acl)
		return
	}

	var newCookies []string
	for _, cookie := range cookies {
		if cookie.Name == acl.CookieKey {
			//toModify = cookie
			if acl.CookieSecure != nil {
				cookie.Secure = *acl.CookieSecure
			}
			if acl.Cookiehttponly != nil {
				cookie.Secure = *acl.Cookiehttponly
			}
			if acl.CookiePath != "" {
				cookie.Domain = acl.CookiePath
			}
			if acl.CookieValue != "" {
				cookie.Value = acl.CookieValue
			}
		}
		newCookies = append(newCookies, cookie.String())
	}
	if cookieHeader == "Set-Cookie" {
		// set cookies are 1 per line (old rfc)
		header.Del(cookieHeader)
		for _, cookie := range newCookies {
			header.Add(cookieHeader, cookie)
		}
	} else {
		// cookies are all on 1 line
		header.Set(cookieHeader, strings.Join(newCookies, ";"))
	}

	cookies = readSetCookies(*header, cookieHeader)
	log.WithField("cookie", cookieHeader).Debug("Modify Cookie Called")

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
	case modifyMatch:
		modifyCookie(header, reqHeader, cookieName, acl)
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
		Secure:   *acl.CookieSecure,
		HttpOnly: *acl.Cookiehttponly,
	}

	return cookie

}

func cookieExists(header *http.Header, cookieHeader string, cookieKey string) bool {
	search := fmt.Sprintf("%s: .*%s", cookieHeader, cookieKey)
	regex, err := regexp.Compile("(?i)" + search + `\W`)
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

func readSetCookies(h http.Header, cookieName string) []*http.Cookie {
	cookieCount := len(h[cookieName])
	if cookieCount == 0 {
		return []*http.Cookie{}
	}
	cookies := make([]*http.Cookie, 0, cookieCount)
	for _, line := range h[cookieName] {
		parts := strings.Split(strings.TrimSpace(line), ";")
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		parts[0] = strings.TrimSpace(parts[0])
		j := strings.Index(parts[0], "=")
		if j < 0 {
			continue
		}
		name, value := parts[0][:j], parts[0][j+1:]
		if !isCookieNameValid(name) {
			continue
		}
		value, ok := parseCookieValue(value, true)
		if !ok {
			continue
		}
		c := &http.Cookie{
			Name:  name,
			Value: value,
			Raw:   line,
		}
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}

			attr, val := parts[i], ""
			if j := strings.Index(attr, "="); j >= 0 {
				attr, val = attr[:j], attr[j+1:]
			}
			lowerAttr := strings.ToLower(attr)
			val, ok = parseCookieValue(val, false)
			if !ok {
				c.Unparsed = append(c.Unparsed, parts[i])
				continue
			}
			switch lowerAttr {
			case "secure":
				c.Secure = true
				continue
			case "httponly":
				c.HttpOnly = true
				continue
			case "domain":
				c.Domain = val
				continue
			case "max-age":
				secs, err := strconv.Atoi(val)
				if err != nil || secs != 0 && val[0] == '0' {
					break
				}
				if secs <= 0 {
					secs = -1
				}
				c.MaxAge = secs
				continue
			case "expires":
				c.RawExpires = val
				exptime, err := time.Parse(time.RFC1123, val)
				if err != nil {
					exptime, err = time.Parse("Mon, 02-Jan-2006 15:04:05 MST", val)
					if err != nil {
						c.Expires = time.Time{}
						break
					}
				}
				c.Expires = exptime.UTC()
				continue
			case "path":
				c.Path = val
				continue
			}
			c.Unparsed = append(c.Unparsed, parts[i])
		}
		cookies = append(cookies, c)
	}
	return cookies
}

func parseCookieValue(raw string, allowDoubleQuote bool) (string, bool) {
	// Strip the quotes, if present.
	if allowDoubleQuote && len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	for i := 0; i < len(raw); i++ {
		if !validCookieValueByte(raw[i]) {
			return "", false
		}
	}
	return raw, true
}

func isCookieNameValid(raw string) bool {
	if raw == "" {
		return false
	}
	return strings.IndexFunc(raw, isNotToken) < 0
}

func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

func isNotToken(r rune) bool {
	return !httplex.IsTokenRune(r)
}
