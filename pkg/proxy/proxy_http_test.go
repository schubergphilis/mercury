package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/stretchr/testify/assert"
)

const certPEM = "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"

var cert = mustParsePEMCertificate(certPEM)

func TestProcessACLVariables(t *testing.T) {
	logging.Configure("stdout", "error")

	acl := []ACL{
		ACL{
			Action:      "add",
			HeaderKey:   "key-a",
			HeaderValue: "###CLIENT_IP###",
		},
		ACL{
			Action:      "add",
			CookieKey:   "key-a",
			CookieValue: "###CLIENT_IP###",
		},
		ACL{
			Action:      "add",
			CookieKey:   "key-a",
			CookieValue: "###UNKNOWN###",
		},
	}

	listener := New("listener-id", "Listener", 999)
	backendNode := BackendNode{}
	request := createHTTPRequest()

	newACL := processACLVariables(acl, listener, backendNode, request)
	assert.Equal(t, "192.0.2.1", newACL[0].HeaderValue)
	assert.Equal(t, "192.0.2.1", newACL[1].CookieValue)
	assert.Equal(t, "UNKNOWN", newACL[2].CookieValue)
}

func TestGetVariableValue(t *testing.T) {
	listener := New("listener-id", "Listener", 999)
	listener.IP = "192.168.0.2"
	listener.Port = 8080

	backendNode := NewBackendNode("backend-id", "192.168.1.1", "server1", 22, 10, []string{}, 0, healthcheck.Online)

	httpRequest := createHTTPRequest()
	httpsRequest := createHTTPSRequest()
	httpsRequestWithCertificate := createHTTPSRequestWithClientCertificate()
	httpsRequestWithMultipleCertificates := createHTTPSRequestWithMultipleClientCertificates()

	var testData = []struct {
		name          string
		expectedValue string
		expectedError error
		request       *http.Request
	}{
		{
			name:          "UNKNOWN",
			expectedValue: "",
			expectedError: errors.New("Unknown variable: UNKNOWN"),
			request:       httpRequest,
		},
		{
			name:          "NODE_ID",
			expectedValue: "backend-id",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "NODE_IP",
			expectedValue: "192.168.1.1",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "LB_IP",
			expectedValue: "192.168.0.2",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "LB_PORT",
			expectedValue: "8080",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_URL",
			expectedValue: "example.com:8080/foo",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_QUERY",
			expectedValue: "key-a=value-a&key-b",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_PATH",
			expectedValue: "/foo",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_HOST",
			expectedValue: "example.com",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_IP",
			expectedValue: "example.com",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_IP",
			expectedValue: "127.0.0.1",
			expectedError: nil,
			request:       httptest.NewRequest("GET", "http://127.0.0.1:4443/", nil),
		},
		{
			name:          "REQ_PROTO",
			expectedValue: "http",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "REQ_PROTO",
			expectedValue: "https",
			expectedError: nil,
			request:       httpsRequest,
		},
		{
			name:          "CLIENT_IP",
			expectedValue: "192.0.2.1",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "CLIENT_CERT",
			expectedValue: "",
			expectedError: nil,
			request:       httpRequest,
		},
		{
			name:          "CLIENT_CERT",
			expectedValue: "",
			expectedError: nil,
			request:       httpsRequest,
		},
		{
			name:          "CLIENT_CERT",
			expectedValue: certPEM,
			expectedError: nil,
			request:       httpsRequestWithCertificate,
		},
		{
			name:          "CLIENT_CERT",
			expectedValue: certPEM + "," + certPEM,
			expectedError: nil,
			request:       httpsRequestWithMultipleCertificates,
		},
	}

	for index, data := range testData {
		description := fmt.Sprintf("%v %v", index, data.name)
		t.Run(description, func(t *testing.T) {
			value, err := getVariableValue(data.name, listener, backendNode, data.request)
			assert.Equal(t, data.expectedValue, value)
			assert.Equal(t, data.expectedError, err)
		})
	}
}

func mustParsePEMCertificate(pemContent string) *x509.Certificate {
	block, _ := pem.Decode([]byte("-----BEGIN CERTIFICATE-----\n" + pemContent + "\n-----END CERTIFICATE-----"))
	if block == nil || block.Bytes == nil {
		panic("Failed to parse PEM content")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}

	return cert
}

func createHTTPRequest() *http.Request {
	return httptest.NewRequest("GET", "http://example.com:8080/foo?key-a=value-a&key-b", nil)
}

func createHTTPSRequest() *http.Request {
	req := httptest.NewRequest("GET", "https://example.com:4443/foo?key-a=value-a&key-b", nil)
	req.TLS = &tls.ConnectionState{}
	return req
}

func createHTTPSRequestWithClientCertificate() *http.Request {
	req := createHTTPSRequest()
	req.TLS.PeerCertificates = []*x509.Certificate{
		mustParsePEMCertificate(certPEM),
	}
	return req
}

func createHTTPSRequestWithMultipleClientCertificates() *http.Request {
	req := createHTTPSRequest()
	req.TLS.PeerCertificates = []*x509.Certificate{
		mustParsePEMCertificate(certPEM),
		mustParsePEMCertificate(certPEM),
	}
	return req
}
