package proxy

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestRules(t *testing.T) {

	// start webserver
	server := newWebserver()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			assert.FailNow(t, err.Error())
		}
	}()

	// start proxy (http)
	logging.Configure("stdout", "warn")
	// setup Proxy
	proxyIP := "127.0.0.1"
	proxyPort := 40183
	proxy := New("listener-id-rules", "Listener-rules", 998)
	proxy.SetListener("http", proxyIP, proxyIP, proxyPort, 998, nil, 10, 10, 1, YES)
	go proxy.Start()
	defer proxy.Stop()

	errorPage := ErrorPage{}
	proxy.AddBackend("backend-id-rules", "backend-rules", "leastconnected", "http", []string{"default"}, 998, errorPage, errorPage)
	backend := proxy.Backends["backend-rules"]
	//newProxy.UpdateBackend("backendpool.UUID", "backendname", "leastconnected", "http", []string{"default"}, 999, nil, nil)
	backendNode := NewBackendNode("backend-id-rules", "127.0.0.1", "localhost", 40182, 10, []string{}, 0, healthcheck.Online)
	backend.AddBackendNode(backendNode)

	time.Sleep(1 * time.Second)

	// do tests
	//t.Run("/goRuleScriptTests/preInboundTest"), preInboundTest)
	t.Run("goRuleScriptTests", func(t *testing.T) {
		t.Run("preInboundTest", func(t *testing.T) {
			preInboundTest(t, backend)
		})
		t.Run("inboundTest", func(t *testing.T) {
			inboundTest(t, backend)
		})
		t.Run("outboundTest", func(t *testing.T) {
			outboundTest(t, backend)
		})
	})

	// stop server
	server.SetKeepAlivesEnabled(false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	defer cancel()

}

func newWebserver() *http.Server {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "golang server")
		// echo client headers if they do not exist yet
		for k, v := range r.Header {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, v[0])
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:         "127.0.0.1:40182",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func preInboundTest(t *testing.T, backend *Backend) {
	client := &http.Client{}

	// test if the original works
	req, _ := http.NewRequest("GET", "http://localhost:40183", nil)
	req.Header.Add("custom-header", `custom-text`)
	resp, err := client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "custom-text", resp.Header.Get("custom-header"))

	// remove custom header
	backend.SetRules("prein", []string{
		"unset request.header.custom-header",
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	req.Header.Add("custom-header", `custom-text`)
	resp, err = client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "", resp.Header.Get("custom-header"))

	// add custom header
	backend.SetRules("prein", []string{
		"request.header.custom-header = \"hello world\"",
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	resp, err = client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "hello world", resp.Header.Get("custom-header"))

	// reset previous rules
	backend.SetRules("prein", []string{})

}

func inboundTest(t *testing.T, backend *Backend) {
	client := &http.Client{}

	// remove custom header
	backend.SetRules("in", []string{
		"unset request.header.custom-header",
	})

	req, _ := http.NewRequest("GET", "http://localhost:40183", nil)
	req.Header.Add("custom-header", `custom-text`)
	resp, err := client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "", resp.Header.Get("custom-header"))

	// add custom header
	backend.SetRules("in", []string{
		"request.header.custom-header = \"hello world\"",
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	resp, err = client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, "hello world", resp.Header.Get("custom-header"))

	// ip deny
	backend.SetRules("in", []string{
		`if $(client.ip) match_net "127.0.0.1/24" { response.statuscode = 404 }`,
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	resp, err = client.Do(req)
	assert.Nil(t, err)

	assert.Equal(t, 404, resp.StatusCode)

	// reset previous rules
	backend.SetRules("in", []string{})
}

func outboundTest(t *testing.T, backend *Backend) {
	client := &http.Client{}

	// remove custom header
	backend.SetRules("out", []string{
		"unset response.header.server",
	})

	req, _ := http.NewRequest("GET", "http://localhost:40183", nil)
	req.Header.Add("custom-header", `custom-text`)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	//log.Printf("resp: %+v err: %s", resp, err)

	assert.Equal(t, "", resp.Header.Get("server"))

	// add custom header
	backend.SetRules("out", []string{
		"response.header.custom-out-header = \"hello world\"",
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	resp, err = client.Do(req)
	assert.Nil(t, err)
	//log.Printf("resp: %+v err: %s", resp, err)

	assert.Equal(t, "hello world", resp.Header.Get("custom-out-header"))

	// change status based on client ip (does not modify contents)
	backend.SetRules("in", []string{
		`if $(client.ip) match_net "127.0.0.1/24" { response.statuscode = 204 }`,
	})

	req, _ = http.NewRequest("GET", "http://localhost:40183", nil)
	resp, err = client.Do(req)
	assert.Nil(t, err)
	//log.Printf("resp: %+v err: %s", resp, err)

	assert.Equal(t, 204, resp.StatusCode)

	// reset previous rules
	backend.SetRules("out", []string{})
}
