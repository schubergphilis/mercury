package core

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/schubergphilis/mercury/internal/config"
)

var (
	// APIEnabled defines wether or not the API is enabled
	APIEnabled = true
	// APITokenSigningKey is key used to sign jtw tokens
	APITokenSigningKey = rndKey()
	// APITokenDuration is how long the jwt token is valid
	APITokenDuration = 1 * time.Hour
)

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

func apiWriteJSONData(w http.ResponseWriter, statusCode int, message apiMessage) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	data, err := json.Marshal(message)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Failed to encode json on write"))
	}
	data = append(data, 10) // 10 = newline
	w.Write(data)
}

func (m *Manager) setupAPI() {
	/*
		managers.Lock()
		defer managers.Unlock()
	*/
	titleHead := fmt.Sprintf("Mercury %s - ", config.Get().Cluster.Binding.Name)

	// HealthChecks are always used
	http.Handle("/api/v1/healthchecks/admin/", authenticate(apiHealthCheckAdminHandler{manager: m}, string(APITokenSigningKey)))
	http.Handle("/api/v1/healthchecks/", apiHealthCheckPublicHandler{manager: m})
	http.Handle("/healthchecks/", webHealthCheckHandler{
		title:         titleHead + "Checks",
		templateFiles: []string{"header.tmpl", "footer.tmpl", "healthchecks.tmpl"},
		template:      "healthchecks",
	})

	// Enable login
	http.Handle("/api/v1/login/", apiLoginHandler{manager: m})
	http.Handle("/login/", webLoginHandler{
		manager:       m,
		title:         titleHead + "Login",
		templateFiles: []string{"header.tmpl", "footer.tmpl", "login.tmpl"},
		template:      "login",
	})
	/*
		http.Handle("/api/v1/cluster/"+m.name+"/admin/", authenticate(apiClusterAdminHandler{manager: m}, m.authKey))
		http.Handle("/api/v1/cluster/"+m.name, apiClusterPublicHandler{manager: m})
		if managers.clusterAPISet == false {
			http.Handle("/api/v1/cluster", apiClusterHandler{})
			managers.clusterAPISet = true
		}
	*/
}
