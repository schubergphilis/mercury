package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestCookieReplace(t *testing.T) {
	logging.Configure("stdout", "error")

	handler := func(w http.ResponseWriter, r *http.Request) {

		expires := time.Now().AddDate(1, 0, 0)
		mercID := http.Cookie{
			Name:    "mercid",
			Domain:  "foo.com",
			Path:    "/",
			Expires: expires,
			Value:   "USERID",
		}

		http.SetCookie(w, &mercID)
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	res := w.Result()
	defer res.Body.Close()
	// body, _ := ioutil.ReadAll(res.Body)

	secure := true
	httpOnly := true

	aclAddMercID := ACL{
		Action:         "add",
		CookieKey:      "mercid",
		CookieValue:    "USERID1",
		CookieExpire:   duration{time.Second * 10},
		CookieSecure:   &secure,
		Cookiehttponly: &httpOnly,
	}

	// try to add the cookie 10 times
	for i := 0; i < 10; i++ {
		aclAddMercID.CookieValue = fmt.Sprintf("USERID%d", i)
		addCookie(nil, &res.Header, "Set-Cookie", aclAddMercID, false)
	}

	addcount := 0
	addValue := ""
	for _, cookie := range res.Cookies() {
		if cookie.Name == aclAddMercID.CookieKey {
			addcount++
			addValue = cookie.Value
		}
	}
	assert.Equal(t, 1, addcount, "Addcookie generated more then 1 cookie with the same name")
	assert.Equal(t, "USERID", addValue, "Addcookie should not overwrite existing cookie")

	aclReplaceMercID := ACL{
		Action:         "replace",
		CookieKey:      "mercid",
		CookieValue:    "USERID1",
		CookieExpire:   duration{time.Second * 10},
		CookieSecure:   &secure,
		Cookiehttponly: &httpOnly,
		CookiePath:     "/",
	}

	// try to replace the cookie 10 times
	for i := 0; i < 10; i++ {
		aclReplaceMercID.CookieValue = fmt.Sprintf("USERID%d", i)
		replaceCookie(nil, &res.Header, "Set-Cookie", aclReplaceMercID.ConditionMatch, aclReplaceMercID)
	}

	addcount = 0
	addValue = ""
	for _, cookie := range res.Cookies() {
		if cookie.Name == aclReplaceMercID.CookieKey {
			addcount++
			addValue = cookie.Value
		}
	}
	assert.Equal(t, 1, addcount, "Replacecookie generated more then 1 cookie with the same name")
	assert.Equal(t, "USERID9", addValue, "Replcecookie should overwrite existing cookie")

}
