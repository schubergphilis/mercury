package cluster

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt"
)

var (
	// APITokenSigningKey is key used to sign jtw tokens
	APITokenSigningKey = rndKey()
	// APITokenDuration is how long the jwt token is valid
	APITokenDuration = 1 * time.Hour
	// APIEnabled defines wether or not the API is enabled
	APIEnabled = true
)

// Authentication middleware
type apiAuthentication struct {
	wrappedHandler http.Handler
	authKey        string
}

type apiMessage struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Data    interface{} `json:"data"`
}

// APIRequest is used to pass requests done to the cluster API to the client application
type APIRequest struct {
	Action  string `json:"action"`
	Manager string `json:"manager"`
	Node    string `json:"node"`
	Data    string `json:"data"`
}

func rndKey() []byte {
	token := make([]byte, 128)
	rand.Read(token)
	return token
}

func (h apiAuthentication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session")
	if cookie != nil {
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return APITokenSigningKey, nil
		})

		if err != nil {
			apiWriteData(w, 403, apiMessage{Success: false, Error: err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if time.Now().Unix() > int64(claims["expire"].(float64)) {
				apiWriteData(w, 403, apiMessage{Success: false, Error: "Token expired"})
				return
			}

			h.wrappedHandler.ServeHTTP(w, r)
		} else {
			apiWriteData(w, 403, apiMessage{Success: false, Error: err.Error()})
			return
		}
	}
}

// Authenticate user
func authenticate(h http.Handler, authKey string) apiAuthentication {
	return apiAuthentication{h, authKey}
}

func apiMakeKey(username, key string, epoch int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"username": username,
		"expire":   time.Now().Add(APITokenDuration).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(APITokenSigningKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func apiWriteData(w http.ResponseWriter, statusCode int, message apiMessage) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	messageData, err := json.Marshal(message.Data)
	message.Data = string(messageData)
	data, err := json.Marshal(message)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Failed to encode json on write"))
	}
	data = append(data, 10) // 10 = newline
	w.Write(data)

}
