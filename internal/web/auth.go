package web

// Auth provides a interface for Authentication
type Auth interface {
	VerifyLogin(username, password string) (bool, error)
}

/*
func NewAuthProvider(method string) (Auth, error) {
	switch method {
	case "ldap":
		return NewAuthLDAP(host, port, method, tlsconfig), nil
	case "password":
		return NewAuthPassword(users), nil
	}
	return nil, fmt.Errorf("Unknown authentication provider")
}
*/
