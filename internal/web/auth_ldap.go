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
	Domain    string              `json:"domain" toml:"domain" yaml:"domain"` // binddn
	Filter    string              `json:"filter" toml:"filter" yaml:"filter"` // binddn
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

	// Bind using provided credentials
	if a.Domain != "" {
		err = l.Bind(fmt.Sprintf("%s\\%s", a.Domain, username), password)
	} else {
		err = l.Bind(username, password)
	}
	if err != nil {
		return false, err
	}

	if a.BindDN == "" {
		return false, fmt.Errorf("Empty LDAP BindDN, cannot verify user exist in DN")
	}
	if a.Filter == "" {
		return false, fmt.Errorf("Empty LDAP Filter, cannot verify user exist in Filter")
	}

	// Search user in Filter
	search := fmt.Sprintf(a.Filter, ldap.EscapeFilter(username))
	searchRequest := ldap.NewSearchRequest(
		a.BindDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		search,
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return false, err
	}

	if len(sr.Entries) < 1 {
		return false, fmt.Errorf("User does not exist")
	}

	if len(sr.Entries) > 1 {
		return false, fmt.Errorf("Too many entries returned")
	}

	return true, nil
}
