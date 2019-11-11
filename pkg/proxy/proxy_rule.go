package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/rdoorn/gorule"
)

// ProcessOutboundRules runs the rules script, and modified the given request/response accordingly
// Outbound rules are applied when contacting the backend, or replying the response to a client
// res cannot be nil at this point
func (l *Listener) ProcessOutboundRules(rules []string, req *http.Request, res *http.Response) error {
	if len(rules) == 0 {
		return nil
	}

	// extract remote ip from request
	client := stringToClientIP(req.RemoteAddr)

	// apply inbound rules if any
	for _, rule := range rules {
		err := gorule.Parse(map[string]interface{}{
			"request":  req,
			"response": res,
			"client":   client,
		}, []byte(rule))
		if err != nil {
			return err
		}
	}
	return nil
}

// ProcessInboundRules runs the rules script, and modified the given request accordingly
// Inbound rules apply only on the initial request before passing it on to a backend
// res can be nil, we need to ensure its only filled if we did set a response on the inbound rule
func (l *Listener) ProcessInboundRules(rules []string, req *http.Request, res *http.Response) error {
	if len(rules) == 0 {
		return nil
	}
	var originalresponse *http.Response

	if res == nil {
		res = &http.Response{
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			Request:    req,
		}
		originalresponse = &http.Response{
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			Request:    req,
		}

	}

	// extract remote ip from request
	client := stringToClientIP(req.RemoteAddr)

	// apply inbound rules if any
	for _, rule := range rules {
		err := gorule.Parse(map[string]interface{}{
			"request":  req,
			"response": res,
			"client":   client,
		}, []byte(rule))
		if err != nil {
			return err
		}
	}

	// only if we have an original response, we started with an empty response
	if originalresponse != nil {
		// if the rules changed the response, we need to ensure we respond a proper response by setting the missing fields
		if reflect.DeepEqual(res, originalresponse) {
			// if there was no change, then empty the response, we'll set it later
			res = nil
			return nil
		}
		// add empty body
		if res.Body == nil {
			nbody := &bytes.Buffer{}
			nbody.Write([]byte{})
			res.Body = ioutil.NopCloser(nbody)
		}
	}
	return nil
}

// ProcessPreInboundRules runs the rules script, and modified the given request accordingly
// PreInbound rules apply only on the initial request before selecting a backend
func (l *Listener) ProcessPreInboundRules(rules []string, req *http.Request) error {
	if len(rules) == 0 {
		return nil
	}

	// extract remote ip from request
	client := stringToClientIP(req.RemoteAddr)

	// apply inbound rules if any
	for _, rule := range rules {
		err := gorule.Parse(map[string]interface{}{
			"request": req,
			"client":  client,
		}, []byte(rule))
		if err != nil {
			return err
		}
	}
	return nil
}

/*
if len(l.Backends[backendname].InboundRule) > 0 {
	response := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Request:    req,
	}
	originalresponse := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Request:    req,
	}
	// apply inbound rules if any
	for _, rule := range l.Backends[backendname].InboundRule {
		err := gorule.Parse(map[string]interface{}{
			"request":  req,
			"response": response,
		}, []byte(rule))
		if err != nil {
			clog.WithError(err).Warnf("error in inbound rule")
		}
	}

	if !reflect.DeepEqual(response, originalresponse) {
		// add empty body
		nbody := &bytes.Buffer{}
		nbody.Write([]byte{})
		response.Body = ioutil.NopCloser(nbody)
		req.URL.Scheme = "error//" + backendname + "403//Access denied - matched DENY ACL"

		// eehhh how to pass response ?!?!?! RDOORN
	}

}
*/

type clientIP struct {
	IP   string
	Port int
}

func stringToClientIP(addr string) *clientIP {
	client := &clientIP{}
	remoteAddr := strings.Split(addr, ":")
	if len(remoteAddr) > 1 {
		client.IP = strings.Join(remoteAddr[:len(remoteAddr)-1], ":")
		if port, err := strconv.Atoi(remoteAddr[len(remoteAddr)-1]); err == nil {
			client.Port = port
		}
	}
	return client
}
