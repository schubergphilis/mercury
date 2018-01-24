package cluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

// login page for manual testing
const loginPage = "<html><head><title>Login</title></head><body><form action=\"/login/managerFIVE\" method=\"post\"> <input type=\"username\" name=\"username\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"
const httpAddr = "127.0.0.1:9499"

type apiReadMessage struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Data    string `json:"data"`
}

func TestApi(t *testing.T) {
	t.Parallel()

	http.Handle("/login", apiLogin{authKey: "secret"})

	// Start Manager for API
	managerAPI := NewManager("managerAPI", "secret")
	managerAPI.AddNode("managerAPI2", "127.0.0.1:9599")
	err := managerAPI.ListenAndServe("127.0.0.1:9500")
	if err != nil {
		log.Fatal(err)
	}

	// Start HTTP
	srv := startHTTPServer(httpAddr)

	time.Sleep(500 * time.Millisecond)

	t.Run("apiCalls", func(t *testing.T) {
		t.Run("Cluster", testAPICluster)
		t.Run("ClusterPublic", testAPIClusterPublic)
		t.Run("ClusterAdmin", testAPIClusterAdmin)
	})

	if err := srv.Shutdown(nil); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}

func startHTTPServer(addr string) *http.Server {
	srv := &http.Server{Addr: addr}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()
	// returning reference so caller can call Shutdown()
	return srv
}

func getWithKey(authKey, url string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, nil)

	cookie := http.Cookie{Name: "session", Value: authKey}
	req.AddCookie(&cookie)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

// Login screen for manual testing
type apiLogin struct {
	authKey string
}

func (h apiLogin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Fprint(w, loginPage)
		return
	}

	if r.FormValue("username") == "" {
		fmt.Fprint(w, loginPage)
		return
	}

	tokenKey, err := apiMakeKey(r.FormValue("username"), h.authKey, 0)
	if err != nil {
		fmt.Fprintf(w, "Unable to create token")
		return
	}

	cookie := &http.Cookie{
		Name:    "session",
		Value:   tokenKey,
		Path:    "/",
		Expires: time.Now().Add(1 * time.Hour),
	}
	http.SetCookie(w, cookie)
}

func testAPICluster(t *testing.T) {
	// generate a new auth key
	url := "/api/v1/cluster"
	data, statusCode, err := getWithKey("nokey", "http://"+httpAddr+url)
	if err != nil {
		t.Errorf("failed to get %s, error:%s", url, err)
	}

	if statusCode != 200 {
		t.Errorf("incorrect status code for %s expected:200, got:%d", url, statusCode)
	}

	// decode response wrapper
	message := &apiReadMessage{}
	err = json.Unmarshal(data, message)
	if err != nil {
		t.Errorf("unable to parse output from %s data:%s error:%s", url, data, err)
	}

	// decode message data
	clusterNodes := []string{}
	err = json.Unmarshal([]byte(message.Data), &clusterNodes)
	if err != nil {
		t.Errorf("unable to parse output from %s data:%s error:%s", url, data, err)
	}
	var foundNode = false
	for _, node := range clusterNodes {
		if node == "managerAPI" {
			foundNode = true
		}
	}

	if foundNode == false {
		t.Errorf("unable to find ManagerAPI in result of %s data:%s", url, data)
	}
}

func testAPIClusterPublic(t *testing.T) {
	url := "/api/v1/cluster/managerAPI"
	data, statusCode, err := getWithKey("nokey", "http://"+httpAddr+url)
	if err != nil {
		t.Errorf("failed to get %s, error:%s", url, err)
	}

	if statusCode != 200 {
		t.Errorf("incorrect status code for %s expected:200, got:%d", url, statusCode)
	}

	// decode response wrapper
	message := &apiReadMessage{}
	err = json.Unmarshal(data, message)
	if err != nil {
		t.Errorf("unable to parse output from %s data:%s error:%s", url, data, err)
	}

	clusterNodes := &APIClusterNodeList{}
	err = json.Unmarshal([]byte(message.Data), &clusterNodes)
	if err != nil {
		t.Errorf("unable to parse output from %s data:%s error:%s", url, data, err)
	}

	if _, ok := clusterNodes.Nodes["managerAPI2"]; !ok {
		t.Errorf("expected managerAPI2 in output of %s, for %+v", url, data)
	}

}

func testAPIClusterAdmin(t *testing.T) {
	// Get requests of private interface without key
	url := "/api/v1/cluster/managerAPI/admin/managerAPI2/shutdown"
	_, statusCode, err := getWithKey("nokey", "http://"+httpAddr+url)
	if err != nil {
		t.Errorf("failed to get %s, error:%s", url, err)
	}

	if statusCode != 403 {
		t.Errorf("incorrect status code for %s, expected:403, got:%d", url, statusCode)
	}

	// generate a new auth key
	authKey, err := apiMakeKey("Test", "secret", 0)
	if err != nil {
		t.Errorf("authentication key creation failed, error:%s", err)
	}

	if authKey == "" {
		t.Errorf("authentication key is empty, expected some more")
	}
	// Get requests of private interface with key
	_, statusCode, err = getWithKey(authKey, "http://"+httpAddr+url)
	if err != nil {
		t.Errorf("failed to get url, error:%s", err)
	}

	if statusCode != 200 {
		t.Errorf("incorrect status code for %s expected:200, got:%d", url, statusCode)
	}
	//fmt.Printf("Got private data: %s", string(data))

}
