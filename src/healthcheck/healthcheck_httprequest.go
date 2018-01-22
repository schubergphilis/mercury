package healthcheck

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/src/tlsconfig"
)

/* dateParseFunc parses the ###DATE+3mFORMAT### string and returns specified date */
func dataParseFunc(t time.Time, mod string, duration string, format string, utc string) string {
	//var t time.Time
	//t = time.Now()
	if utc == "|UTC" {
		t = t.UTC()
	}
	if duration != "" {
		d, err := time.ParseDuration(duration)
		if err != nil {
			return fmt.Sprintf("date parse error:%s", err.Error())
		}
		switch mod {
		case "+":
			t = t.Add(d)
		case "-":
			t = t.Add(-d)
		}
	}
	if t.IsZero() {
		return "INVALID TIME"
	}
	if format == "" {
		format = time.RFC3339
	}
	return t.Format(format)
}

func postDataParser(t time.Time, data string) string {
	r, err := regexp.Compile("###(DATE)(\\+|\\-)*([0-9]+[a-zA-Z])*([a-zA-Z0-9\\+\\-:\\.]+)*(\\|UTC)*###")
	if err != nil {
		return data
	}
	newdata := r.ReplaceAllStringFunc(data,
		func(m string) string {
			p := r.FindStringSubmatch(m)
			switch p[1] {
			case "DATE":
				return dataParseFunc(t, p[2], p[3], p[4], p[5])
			}
			// return same if no correct match
			return p[0]
		})

	if newdata != data {
		return newdata
	}

	return data
}

// httpRequest does a http request check
func httpRequest(method string, host string, port int, sourceIP string, healthCheck HealthCheck) (bool, error) {
	//log := logging.For("healthcheck/httprequest")
	var err error

	localAddr, errl := net.ResolveIPAddr("ip", sourceIP)
	if errl != nil {
		return false, errl
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	// Custom dialer with timeouts
	dialer := &net.Dialer{
		LocalAddr: &localTCPAddr,
		Timeout:   time.Duration(healthCheck.Timeout) * time.Second,
		KeepAlive: 10 * time.Second,
		//Deadline:  time.Now().Add(10 * time.Second),
		DualStack: true,
	}

	// Parse TLS config if provided
	//tlsConfig := &tls.Config{}
	//err = healthCheck.TLSConfig.LoadConfig(tlsConfig)
	tlsConfig, err := tlsconfig.LoadCertificate(healthCheck.TLSConfig)
	if err != nil {
		return false, fmt.Errorf("Unable to setup TLS:%s", err)
	}

	// Overwrite default transports with our own for checking the correct node
	tr := &http.Transport{
		TLSClientConfig:       tlsConfig,
		DisableCompression:    true,
		ResponseHeaderTimeout: time.Duration(healthCheck.Timeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(healthCheck.Timeout) * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// force adress to be our node, and do not resolve it
			addr = fmt.Sprintf("%s:%d", host, port)
			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := &http.Client{Transport: tr}

	//log.Debugf("Creating new %s request on:%s (%s:%d)", method, healthCheck.Request, host, port)
	var postData *bytes.Buffer
	var req *http.Request
	t := time.Now()
	if healthCheck.HTTPPostData != "" {
		//log.Debugf("REQUEST with POST DATA")
		postData = bytes.NewBufferString(postDataParser(t, healthCheck.HTTPPostData))
		req, err = http.NewRequest(method, healthCheck.HTTPRequest, postData)
	} else {
		//log.Debugf("REQUEST without post data")
		req, err = http.NewRequest(method, healthCheck.HTTPRequest, nil)

	}
	if err != nil {
		return false, err
	}

	// Process headers to add
	for _, header := range healthCheck.HTTPHeaders {
		hdr := strings.SplitN(header, ":", 2)
		key := strings.TrimSpace(hdr[0])
		value := strings.TrimSpace(hdr[1])
		req.Header.Set(key, value)
	}

	req.Header.Set("User-Agent", "mercury/1.0")
	//log.Debugf("Executing check:%+v", req)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	//log.Debugf("Check result for:%s (%s:%d) result:%d", healthCheck.Request, host, port, healthCheck.HTTPStatus)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("Error reading HTTP Body: %s", err)
	}

	// Check health status
	if healthCheck.HTTPStatus > 0 {
		if resp.StatusCode != healthCheck.HTTPStatus {
			//return false, fmt.Errorf("HTTP Response code incorrect (got:%d %s expected:%d) Body:%s", resp.StatusCode, resp.Status, healthCheck.HTTPStatus, html.EscapeString(string(body)))
			//return false, fmt.Errorf("HTTP Response code incorrect (got:%d %s expected:%d)<br>Body:%s<br>PostData:%s", resp.StatusCode, resp.Status, healthCheck.HTTPStatus, string(body), postDataParser(t, healthCheck.PostData))
			//reqDump, _ := httputil.DumpRequest(req, true)
			//log.Debugf("Request: %s", reqDump)
			return false, fmt.Errorf("HTTP Response code incorrect (got:%d %s expected:%d)", resp.StatusCode, resp.Status, healthCheck.HTTPStatus)
		}
	}

	// check body
	r, err := regexp.Compile(healthCheck.HTTPReply)
	if err != nil {
		return false, err
	}

	if len(healthCheck.HTTPReply) != 0 {
		//log.Debugf("Body check for:%s", healthCheck.Request)
		if !r.MatchString(string(body)) {
			return false, fmt.Errorf("Reply '%s' not found in body", healthCheck.HTTPReply)
		}
	}
	// http and body check were ok
	return true, nil
}
