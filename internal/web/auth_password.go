package web

import (
	"crypto/sha256"
	"fmt"
)

// AuthPassword is the provider for Password based authentication for the web service
type AuthPassword struct {
	Users map[string]string `json:"users" toml:"users" yaml:"users"`
}

// Type is the authentication type
func (a *AuthPassword) Type() string {
	return "Local authentication"
}

// VerifyLogin validates a user/password combination and returns true or false accordingly
func (a *AuthPassword) VerifyLogin(username, password string) (bool, error) {
	if _, ok := a.Users[username]; !ok {
		return false, fmt.Errorf("incorrect username or password combination")
	}

	h := sha256.New()
	h.Write([]byte(password))
	shapass := fmt.Sprintf("%x", h.Sum(nil))

	if a.Users[username] != shapass {
		return false, fmt.Errorf("incorrect username or password combination(%s != %s)", a.Users[username], shapass)
	}
	return true, nil
}

// NewAuthPassword provides a authentication provider for Passwords
func NewAuthPassword(users map[string]string) *AuthPassword {
	a := &AuthPassword{
		Users: users,
	}
	return a
}
