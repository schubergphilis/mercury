package core

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Authentication middleware, we expect a token and verify this
type apiAuthentication struct {
	wrappedHandler http.Handler
	authKey        string
}

func (h apiAuthentication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		bearer := strings.Split(auth, " ")
		if bearer[0] != "BEARER" {
			apiWriteData(w, 405, apiMessage{Success: false, Error: "Invalid authorization header type"})
			return
		}

		if len(bearer) < 2 {
			apiWriteData(w, 405, apiMessage{Success: false, Error: "Invalid authorization header length"})
			return
		}

		token, err := jwt.Parse(bearer[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return APITokenSigningKey, nil
		})

		// More user friendly message
		if err != nil {
			if err.Error() == jwt.ErrSignatureInvalid.Error() {
				apiWriteData(w, 403, apiMessage{Success: false, Error: "Login token expired"})
				return
			}
			apiWriteData(w, 403, apiMessage{Success: false, Error: err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if time.Now().Unix() > int64(claims["expire"].(float64)) {
				apiWriteData(w, 403, apiMessage{Success: false, Error: "Login token expired"})
				return
			}

			h.wrappedHandler.ServeHTTP(w, r)
			return
		}
		apiWriteData(w, 403, apiMessage{Success: false, Error: err.Error()})
		return
	}
	apiWriteData(w, 403, apiMessage{Success: false, Error: "Login required"})
}

// Authenticate user
func authenticate(h http.Handler, authKey string) apiAuthentication {
	return apiAuthentication{h, authKey}
}

// authenticateUser returns authenticationStatus, username and error
func authenticateUser(r *http.Request) (bool, string, error) {
	cookie, _ := r.Cookie("session")
	if cookie != nil {
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return APITokenSigningKey, nil
		})

		if err != nil {
			return false, "", err
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if time.Now().Unix() > int64(claims["expire"].(float64)) {
				return false, "", fmt.Errorf("token expired")
			}
			return true, fmt.Sprintf("%s", claims["username"]), nil
		} else {
			return false, "", err
		}
	}
	return false, "", fmt.Errorf("no session token")
}
