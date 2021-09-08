package core

import (
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt"
)

// apiLoginHandler handles a login of a user, and gives back a cookie if successfull
type apiLoginHandler struct {
	manager *Manager
}

func (h apiLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("username") == "" {
		apiWriteData(w, 501, apiMessage{Success: false, Error: "username not supplied"})
		return
	}

	if r.FormValue("password") == "" {
		apiWriteData(w, 501, apiMessage{Success: false, Error: "password not supplied"})
		return
	}

	valid, err := h.manager.webAuthenticator.VerifyLogin(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		apiWriteData(w, 501, apiMessage{Success: false, Error: err.Error()})
		return
	}
	if !valid {
		apiWriteData(w, 501, apiMessage{Success: false, Error: "validation error"})
		return
	}

	expiration := time.Now().Add(APITokenDuration)
	apiKey, err := apiMakeKey(r.FormValue("username"), string(APITokenSigningKey), expiration.Unix())
	if err != nil {
		apiWriteData(w, 501, apiMessage{Success: false, Error: err.Error()})
		return
	}

	apiWriteData(w, http.StatusOK, apiMessage{Success: true, Data: apiKey})

}

func apiMakeKey(username, key string, expire int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"username": username,
		"expire":   expire,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(APITokenSigningKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
