package core

import (
	"github.com/rdoorn/old/glbv2/pkg/tlsconfig"
)

// Config for WEB
type WebConfig struct {
	Binding   string              `toml:"binding" json:"binding"`
	Port      int                 `toml:"port" json:"port"`
	Path      string              `toml:"path" json:"path"`
	TLSConfig tlsconfig.TLSConfig `json:"tls" toml:"tls" yaml:"tls"`
	Auth      WebConfigAuth       `json:"auth" toml:"auth" yaml:"auth"`
}

// AuthConfig contains the authentication configuration for web interface
type WebConfigAuth struct {
	Password *WebConfigAuthPassword `json:"password" toml:"password" yaml:"password"`
	LDAP     *WebConfigAuthLDAP     `json:"ldap" toml:"ldap" yaml:"ldap"`
}

type WebConfigAuthPassword struct {
	Users map[string]string `json:"users" toml:"users" yaml:"users"`
}

type WebConfigAuthLDAP struct {
	Host      string              `json:"host" toml:"host" yaml:"host"`       // address to connect to
	Port      int                 `json:"port" toml:"port" yaml:"port"`       // port to connect to
	Method    string              `json:"method" toml:"method" yaml:"method"` // connect method (SSL/TLS)
	Domain    string              `json:"domain" toml:"domain" yaml:"domain"` // binddn
	Filter    string              `json:"filter" toml:"filter" yaml:"filter"` // binddn
	BindDN    string              `json:"binddn" toml:"binddn" yaml:"binddn"` // binddn
	TLSConfig tlsconfig.TLSConfig `json:"tls" toml:"tls" yaml:"tls"`
	addr      string
	// tlsConfig *tls.Config
}
