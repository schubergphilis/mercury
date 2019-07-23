package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/schubergphilis/mercury/pkg/healthcheck"
	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestWebSocketProxy(t *testing.T) {
	logging.Configure("stdout", "warn")

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		assert.Fail(t, err.Error())
	}
	defer l.Close()

	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := echoServer(w, r)
			if err != nil {
				assert.Fail(t, err.Error())
				//log.Printf("echo server: %v", err)
			}
		}),
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
	}
	defer s.Close()

	// This starts the echo server on the listener.
	go func() {
		err := s.Serve(l)
		if err != http.ErrServerClosed {
			assert.Fail(t, err.Error())
			//log.Fatalf("failed to listen and serve: %v", err)
		}
	}()

	wsHost, wsPortStr, err := net.SplitHostPort(l.Addr().String())
	assert.Nil(t, err)
	wsPort, _ := strconv.Atoi(wsPortStr)

	// setup Proxy
	proxyIP := "127.0.0.1"
	proxyPort := 51243
	proxy := New("listener-id", "Listener", 999)
	proxy.SetListener("http", proxyIP, proxyIP, proxyPort, 999, nil, 10, 10, 1, YES)
	go proxy.Start()
	defer proxy.Stop()

	errorPage := ErrorPage{}

	proxy.AddBackend("backend-id", "backend", "leastconnected", "http", []string{"default"}, 999, errorPage, errorPage)
	backend := proxy.Backends["backend"]
	//newProxy.UpdateBackend("backendpool.UUID", "backendname", "leastconnected", "http", []string{"default"}, 999, nil, nil)
	backendNode := NewBackendNode("backend-id", wsHost, "localhost", wsPort, 10, []string{}, 0, healthcheck.Online)
	backend.AddBackendNode(backendNode)

	time.Sleep(1 * time.Second)
	// Now we dial the server, send the messages and echo the responses.
	//err = client("ws://" + l.Addr().String())
	err = client("ws://" + fmt.Sprintf("%s:%d", proxyIP, proxyPort))
	if err != nil {
		assert.Fail(t, err.Error())
		//log.Fatalf("client failed: %v", err)
	}
}

func echoServer(w http.ResponseWriter, r *http.Request) error {
	//log.Printf("serving %v", r.RemoteAddr)

	c, err := websocket.Accept(w, r, websocket.AcceptOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		return err
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	if c.Subprotocol() != "echo" {
		c.Close(websocket.StatusPolicyViolation, "client must speak the echo subprotocol")
		return xerrors.Errorf("client does not speak echo sub protocol")
	}

	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)
	for {
		err = echo(r.Context(), c, l)
		if err != nil {
			return xerrors.Errorf("failed to echo with %v: %w", r.RemoteAddr, err)
		}
	}
}

func echo(ctx context.Context, c *websocket.Conn, l *rate.Limiter) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	err := l.Wait(ctx)
	if err != nil {
		return err
	}

	typ, r, err := c.Reader(ctx)
	if err != nil {
		return err
	}

	w, err := c.Writer(ctx, typ)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		return xerrors.Errorf("failed to io.Copy: %w", err)
	}

	err = w.Close()
	return err
}

// client dials the WebSocket echo server at the given url.
// It then sends it 5 different messages and echo's the server's
// response to each.
func client(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, url, websocket.DialOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		return err
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	for i := 0; i < 5; i++ {
		err = wsjson.Write(ctx, c, map[string]int{
			"i": i,
		})
		if err != nil {
			return err
		}

		v := map[string]int{}
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			return err
		}

		//fmt.Printf("received: %+v\n", v)
	}

	c.Close(websocket.StatusNormalClosure, "")
	return nil
}
