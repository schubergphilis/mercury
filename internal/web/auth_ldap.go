package web

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/schubergphilis/mercury/pkg/tlsconfig"
	ldap "gopkg.in/ldap.v2"
)

// AuthLDAP is the provider for LDAP based authentication for the web service
type AuthLDAP struct {
	Host      string              `json:"host" toml:"host" yaml:"host"`       // address to connect to
	Port      int                 `json:"port" toml:"port" yaml:"port"`       // port to connect to
	Method    string              `json:"method" toml:"method" yaml:"method"` // connect method (SSL/TLS)
	BindDN    string              `json:"binddn" toml:"binddn" yaml:"binddn"` // binddn
	TLSConfig tlsconfig.TLSConfig `json:"tls" toml:"tls" yaml:"tls"`
	addr      string
	tlsConfig *tls.Config
}

// Type is the authentication type
func (a *AuthLDAP) Type() string {
	return "LDAP"
}

// VerifyLogin validates a user/password combination and returns true or false accordingly
func (a *AuthLDAP) VerifyLogin(username, password string) (bool, error) {
	if a.tlsConfig == nil {
		a.tlsConfig = &tls.Config{InsecureSkipVerify: a.TLSConfig.InsecureSkipVerify}
	}
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
		err = l.StartTLS(a.tlsConfig)
		if err != nil {
			return false, err
		}
	case "SSL":
		l, err = ldap.DialTLS("tcp", a.addr, a.tlsConfig)
		if err != nil {
			return false, err
		}
	}

	err = l.Bind(fmt.Sprintf("cn=%s,%s", username, a.BindDN), password)
	if err != nil {
		return false, err
	}
	return true, nil
}
