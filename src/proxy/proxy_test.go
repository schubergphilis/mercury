package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/schubergphilis/mercury/src/logging"
)

var TestDuration = duration{}

func TestACLCookie(t *testing.T) {
	logging.Configure("stdout", "info")
	TestDuration.UnmarshalText([]byte("24h"))
	var TestACLForCookie = []ACL{
		{Action: "add", CookieKey: "testkey", CookieValue: "testvalue", CookieExpire: TestDuration, CookieSecure: true, Cookiehttponly: true},
	}
	var TestACLForCookieResult = []string{
		"testkey=testvalue; Expires=.*GMT; HttpOnly; Secure",
	}
	// ACL Cookie on Request
	req, _ := http.NewRequest("GET", "/", nil)
	for id, acl := range TestACLForCookie {
		//acl.processCookie(&req.Header, "Cookie-Set")
		acl.ProcessRequest(req)
		newcookie := req.Header.Get("Cookie")

		regex, _ := regexp.Compile(TestACLForCookieResult[id])
		if !regex.MatchString(newcookie) {
			t.Errorf("Wrong cookie result in request, result_string:'%s' does not match regex: %s", newcookie, TestACLForCookieResult[id])
		}
	}
	// ACL cookie on response
	req, _ = http.NewRequest("GET", "/", nil)
	res := http.Response{
		StatusCode: 200,
		Status:     "OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Request:    req,
	}
	for id, acl := range TestACLForCookie {
		acl.ProcessResponse(&res)
		newcookie := res.Header.Get("Set-Cookie")
		regex, _ := regexp.Compile(TestACLForCookieResult[id])
		if !regex.MatchString(newcookie) {
			t.Errorf("Wrong cookie result in response, result_string:'%s' does not match regex: %s", newcookie, TestACLForCookieResult[id])
		}
	}

}

func TestACLHeader(t *testing.T) {
	//logging.Configure("stdout", "debug")

	var TestACLForHeader = []ACL{
		{Action: "add", HeaderKey: "testkey", HeaderValue: "testvalue"},
		{Action: "add", HeaderKey: "testkey", HeaderValue: "testvalue2"},
		{Action: "replace", HeaderKey: "testkey", HeaderValue: "testvalue2"},
		{Action: "replace", ConditionType: "header", ConditionMatch: "test.*:", HeaderKey: "testkey", HeaderValue: "testvalue3"},
		{Action: "replace", HeaderKey: "secondtestkey", HeaderValue: "secondtestvalue"},
		{Action: "add", HeaderKey: "secondtestkey", HeaderValue: "secondtestvalue"},
		{Action: "add", HeaderKey: "thirdtestkey", HeaderValue: "thirdtestvalue"},
		{Action: "remove", ConditionType: "header", ConditionMatch: "^test.*:"},
		{Action: "remove", HeaderKey: "secondtestkey"},
	}

	var TestACLForHeaderResult = []map[string]string{
		{"testkey": "testvalue"},
		{"testkey": "testvalue"},                                                                        //  write should have failed on a add when already existing
		{"testkey": "testvalue2"},                                                                       //  write should work on a replace when existing
		{"testkey": "testvalue3"},                                                                       //  write should work on a replace match
		{"secondtestkey": "", "testkey": "testvalue3"},                                                  //  write fail on non existing with replace
		{"secondtestkey": "secondtestvalue", "testkey": "testvalue3"},                                   //  write should add second header
		{"thirdtestkey": "thirdtestvalue", "secondtestkey": "secondtestvalue", "testkey": "testvalue3"}, //  write should add third header
		{"thirdtestkey": "thirdtestvalue", "secondtestkey": "secondtestvalue", "testkey": ""},           //  remove should remove one header
		{"thirdtestkey": "thirdtestvalue", "secondtestkey": "", "testkey": ""},                          //  remove should remove one header
	}

	// ACL Header on Request
	req, _ := http.NewRequest("GET", "/", nil)
	for id, acl := range TestACLForHeader {
		acl.ProcessRequest(req)
		for key, value := range TestACLForHeaderResult[id] {
			//key := getKey(tk)
			newheader := req.Header.Get(key)
			if newheader != value {
				t.Errorf("Wrong header result for request header:%s (id:%d). got:%s expected:%s", key, id, newheader, value)
			}
		}
	}

	// ACL Header on Response
	res := http.Response{
		StatusCode: 200,
		Status:     "OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	for id, acl := range TestACLForHeader {
		acl.ProcessResponse(&res)
		for key, value := range TestACLForHeaderResult[id] {
			//key := getKey(tk)
			newheader := res.Header.Get(key)
			if newheader != value {
				t.Errorf("Wrong header result for response header:%s (id:%d). got:%s expected:%s", key, id, newheader, value)
			}
		}
	}
}

func TestACLStatusCode(t *testing.T) {
	var TestACLForHeader = []ACL{
		{Action: "add", StatusCode: 501},
	}
	// ACL Header on Response
	for _, acl := range TestACLForHeader {
		res := http.Response{
			StatusCode: 200,
			Status:     "OK",
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
		}
		acl.ProcessResponse(&res)
		if res.StatusCode != acl.StatusCode {
			t.Errorf("Wrong statuscode result for response. got:%d expected:%d", res.StatusCode, acl.StatusCode)
		}
	}
}

func TestACLCIDRDeny(t *testing.T) {
	var TestACLForCIDR = []ACL{
		{Action: "deny", CIDRS: []string{"127.0.0.1/32"}},
	}
	// ACL Cookie on Request
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1"
	for _, acl := range TestACLForCIDR {
		match := acl.ProcessRequest(req)
		if !match {
			t.Errorf("ACL did not match deny of 127.0.0.1 while remote addr is %s", req.RemoteAddr)
		}
	}

	req.RemoteAddr = "127.0.0.2"
	for _, acl := range TestACLForCIDR {
		match := acl.ProcessRequest(req)
		if match {
			t.Errorf("ACL did match deny of 127.0.0.1 while remote addr is %s", req.RemoteAddr)
		}
	}

}

func TestACLCIDRAllow(t *testing.T) {
	var TestACLForCIDR = []ACL{
		{Action: "allow", CIDRS: []string{"127.0.0.1/32"}},
	}
	// ACL Cookie on Request
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1"
	for _, acl := range TestACLForCIDR {
		match := acl.ProcessRequest(req)
		if !match {
			t.Errorf("ACL did not match allow of 127.0.0.1 while remote addr is %s", req.RemoteAddr)
		}
	}

	req.RemoteAddr = "127.0.0.2"
	for _, acl := range TestACLForCIDR {
		match := acl.ProcessRequest(req)
		if match {
			t.Errorf("ACL did match allow of 127.0.0.1 while remote addr is %s", req.RemoteAddr)
		}
	}

}

func TestACLResponse(t *testing.T) {
	var acl = ACL{Action: "add", HeaderKey: "testkey", HeaderValue: "testvalue"}
	acl.ProcessResponse(nil) // this should not fatal, we can get an empty reply from host
	res := http.Response{
		StatusCode: 200,
		Status:     "OK",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	acl.ProcessResponse(&res) // this should not fatal, we can get an empty header from host
}

func TestTCPProxy(t *testing.T) {
	//logging.Configure("stdout", "debug")

	serverIP := "127.0.0.1"
	serverPort := 32323

	proxyIP := "127.0.0.1"
	proxyPort := 32324

	send := "TestData"

	exit := make(chan bool)
	go tcpDummyServer(serverIP, serverPort, exit, t)

	time.Sleep(100 * time.Millisecond) // give server time to start

	// Test self, see if we can do TCP connections
	received := tcpDummyClient(serverIP, serverPort, send, t)
	if received != send {
		t.Errorf("Failed to receive data from tcpDummyServer, send:[%+v] recieved:[%+v]", []byte(send), []byte(received))
	}

	// Create a TCP Proxy
	newProxy := New("UUIDP1", "tcpProxy", 1)
	newBackendNode := NewBackendNode("UUIDBN1", serverIP, serverIP, serverPort, 1, []string{}, 0)
	//newProxy.SetListener(pool.Listener.Mode, pool.Listener.IP, pool.Listener.Port, pool.Listener.MaxConnections, newTLS, pool.Listener.ReadTimeout, pool.Listener.WriteTimeout)
	newProxy.SetListener("tcp", proxyIP, proxyPort, 10, &tls.Config{}, 10, 10, 2, "yes")
	newProxy.AddBackend("UUIDB1", "tcpBackend", "leastconnected", "tcp", []string{}, 1, ErrorPage{})
	newProxy.Backends["tcpBackend"].AddBackendNode(newBackendNode)
	go newProxy.Start()

	time.Sleep(100 * time.Millisecond) // give server time to start

	// Test self, see if we can do TCP connections
	received = tcpDummyClient(proxyIP, proxyPort, send, t)
	if received != send {
		t.Errorf("Failed to receive data from tcpProxy, send:[%+v] recieved:[%+v]", []byte(send), []byte(received))
	}

	newProxy.Stop()

	//newProxy.UpdateBackend(backendname, backendpool.BalanceMode.Method, backendpool.ConnectMode, backendpool.HostNames)
	//				backend.SetACL("in", inboundACLs)

	exit <- true
}

func getKey(m map[string]string) string {
	for k := range m {
		return k
	}
	return ""
}

func tcpDummyClient(ip string, port int, send string, t *testing.T) string {
	// connect to server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// send data
	fmt.Fprintf(conn, send)
	buf := make([]byte, 256)
	_, err = conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	buf = bytes.Trim(buf, "\x00") // remove nul
	return string(buf)
}

func tcpDummyServer(ip string, port int, exit chan bool, t *testing.T) {
	// start listener
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_, err = conn.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(conn, string(buf))
			conn.Close()
		}
	}()
	// wait for exit signal to close listener
	for {
		select {
		case _ = <-exit:
			l.Close()
			return
		}
	}
}
