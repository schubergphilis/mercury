package web

import (
	"crypto/tls"
	"fmt"
	"strings"

	ldap "gopkg.in/ldap.v2"
)

// AuthLDAP is the provider for LDAP based authentication for the web service
type AuthLDAP struct {
	Host      string // address to connect to
	Port      int    // port to connect to
	addr      string
	Method    string // connect method (SSL/TLS)
	TLSConfig *tls.Config
}

// Type is the authentication type
func (a *AuthLDAP) Type() string {
	return "LDAP"
}

// VerifyLogin validates a user/password combination and returns true or false accordingly
func (a *AuthLDAP) VerifyLogin(username, password string) (bool, error) {
	var l *ldap.Conn
	var err error
	if a.addr == "" {
		a.addr = fmt.Sprintf("%s:%d", a.Host, a.Port)
	}
	switch strings.ToUpper(a.Method) {
	case "TLS":
		l, err = ldap.Dial("tcp", a.addr)
		if err != nil {
			return false, err
		}
		fmt.Println("StartTLS")
		err = l.StartTLS(a.TLSConfig)
		if err != nil {
			return false, err
		}
	case "SSL":
		fmt.Println("DialTLS")
		l, err = ldap.DialTLS("tcp", a.addr, a.TLSConfig)
		if err != nil {
			return false, err
		}
	}

	err = l.Bind(username, password)
	if err != nil {
		return false, err
	}
	return true, nil
}

// NewAuthLDAP provides a authentication provider for LDAP
func NewAuthLDAP(host string, port int, method string, tlsconfig *tls.Config) *AuthLDAP {
	a := &AuthLDAP{
		Host:      host,
		Port:      port,
		addr:      fmt.Sprintf("%s:%d", host, port),
		Method:    method,
		TLSConfig: tlsconfig,
	}
	return a
}
