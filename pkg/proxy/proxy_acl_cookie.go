package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"
	"golang.org/x/net/http/httpguts"
)

// removeHeader removes matching requestHeaders
func removeCookie(requestHeader *http.Header, responseHeader *http.Header, cookieHeader string, match string) {
	log := logging.For("proxy/removecookie")
	if cookieHeader == "Set-Cookie" {
		cookies := readSetCookies(*responseHeader, cookieHeader)

		if len(cookies) == 0 {
			log.Debug("Delete cookie called but cookie doesn't exist")
			return
		}

		var newCookies []string
		for _, cookie := range cookies {
			if cookie.Name != match {
				newCookies = append(newCookies, cookie.String())
			}
		}
		// set cookies are 1 per line (old rfc)
		responseHeader.Del(cookieHeader)
		for _, cookie := range newCookies {
			responseHeader.Add(cookieHeader, cookie)
		}
	} else {
		// TODO: implement something usefull here
		// requestHeader.Del(fmt.Sprintf("%s: %s", cookieHeader, match))
	}
}

// removeHeader removes matching requestHeaders
func replaceCookie(requestHeader *http.Header, responseHeader *http.Header, cookieHeader string, match string, acl ACL) {
	// remove cookie
	removeCookie(requestHeader, responseHeader, cookieHeader, acl.CookieKey)
	addCookie(requestHeader, responseHeader, cookieHeader, acl, true)
}

// addHeader adds a http requestHeader, only if cookie does not exist yet
func addCookie(requestHeader *http.Header, responseHeader *http.Header, cookieHeader string, acl ACL, force bool) {
	log := logging.For("proxy/addcookie")
	if force == false {
		// if we are setting a response cookie (server -> mercury -[here]-> browser)
		if cookieHeader == "Set-Cookie" {
			// check if we have a request header, we verify if the cookie was already set in the request before adding it again
			if requestHeader != nil && cookieExists(requestHeader, "Cookie", acl.CookieKey) {
				// cookie key already existed in the request, so browser already has this key, we don't need to add the cookie
				return
			}
			// check if the cookie already exists in the response header
			if responseHeader != nil && cookieExists(responseHeader, "Set-Cookie", acl.CookieKey) {
				// cookie key already existed in the response, so the server already set this key, we are not going to add it again
				return
			}
		} else {
			// we are settings a request cookie (browser -> mercury -[here]-> server)
			if requestHeader != nil && cookieExists(requestHeader, "Cookie", acl.CookieKey) {
				// cookie key already existed in the request, so browser already has this key, we don't need to add the cookie
				return
			}
		}
	}

	cookie := acl.newCookie()
	if cookieHeader == "Set-Cookie" {
		// Set-Cookies must have their own requestHeader for each cookie, we write them to the request requestHeader
		responseHeader.Add(cookieHeader, cookie.String())

	} else {
		// Cookies can have multiple values on 1 cookie: requestHeader
		if c := requestHeader.Get(cookieHeader); c != "" {
			requestHeader.Set(cookieHeader, c+"; "+cookie.String())
		} else {
			requestHeader.Set(cookieHeader, cookie.String())
		}
	}

	log.WithField("cookie", cookieHeader).WithField(acl.CookieKey, cookie.Value).Debug("Adding Cookie")
}

// modifyCookie modifies a existing
func modifyCookie(requestHeader *http.Header, responseHeader *http.Header, cookieHeader string, acl ACL) {
	log := logging.For("proxy/modifycookie")
	log.Debug("Modify cookie called")
	if acl.CookieKey == "" {
		log.Warnf("attept to modify a cookie without a cookie key with acl: %+v", acl)
		return
	}

	var cookies []*http.Cookie
	if strings.Compare(cookieHeader, "Set-Cookie") == 0 {
		cookies = readSetCookies(*responseHeader, cookieHeader)
	}
	if strings.Compare(cookieHeader, "Cookie") == 0 {
		cookies = readSetCookies(*requestHeader, cookieHeader)
	}

	if len(cookies) == 0 {
		log.Debugf("Modify cookie called but cookie doesn't exist acl:%+v", acl)
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
		responseHeader.Del(cookieHeader)
		for _, cookie := range newCookies {
			responseHeader.Add(cookieHeader, cookie)
		}
	} else {
		// cookies are all on 1 line
		requestHeader.Set(cookieHeader, strings.Join(newCookies, ";"))
	}

	if strings.Compare(cookieHeader, "Set-Cookie") == 0 {
		cookies = readSetCookies(*responseHeader, cookieHeader)
	}
	if strings.Compare(cookieHeader, "Cookie") == 0 {
		cookies = readSetCookies(*requestHeader, cookieHeader)
	}
	log.WithField("cookie", cookieHeader).Debug("Modify Cookie Called")

}

// processHeader calls the correct handler when editing requestHeaders
func (acl ACL) processCookie(requestHeader *http.Header, responseHeader *http.Header, cookieName string) (deny bool) {
	switch acl.Action {
	case removeMatch:
		removeCookie(requestHeader, responseHeader, cookieName, acl.ConditionMatch)

	case replaceMatch:
		replaceCookie(requestHeader, responseHeader, cookieName, acl.ConditionMatch, acl)

	case addMatch:
		addCookie(requestHeader, responseHeader, cookieName, acl, false)
	case modifyMatch:
		modifyCookie(requestHeader, responseHeader, cookieName, acl)
	}

	return false
}

// newCookie converts action to a cookie
func (acl ACL) newCookie() *http.Cookie {
	expire := time.Now().Add(acl.CookieExpire.Duration)
	cookie := &http.Cookie{
		Name:    acl.CookieKey,
		Value:   acl.CookieValue,
		Path:    acl.CookiePath,
		Expires: expire,
	}
	if acl.CookieSecure != nil {
		cookie.Secure = *acl.CookieSecure
	}
	if acl.Cookiehttponly != nil {
		cookie.HttpOnly = *acl.Cookiehttponly
	}

	return cookie

}

func cookieExists(requestHeader *http.Header, cookieHeader string, cookieKey string) bool {
	// if no requestHeaders are set yet
	if requestHeader == nil {
		return false
	}

	search := fmt.Sprintf("%s: .*%s", cookieHeader, cookieKey)
	regex, err := regexp.Compile("(?i)" + search + `\W`)
	if err != nil {
		return false
	}

	for key, hdr := range *requestHeader {
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
	return !httpguts.IsTokenRune(r)
}
