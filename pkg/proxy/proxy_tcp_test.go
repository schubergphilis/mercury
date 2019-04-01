package proxy

import (
	"fmt"
	"log"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPProxyCopy(t *testing.T) {

	// client1 -> server1 -> server2 -> client2
	client1, server1 := net.Pipe()
	client2, server2 := net.Pipe()

	var w sync.WaitGroup
	w.Add(1)
	go func() {

		// copy data from server 1 to server 2
		in, out, firstByte := netPipe(server1, server2)
		log.Printf("in:%d out:%d firstByte:%s", in, out, firstByte)
		log.Printf("server 1 closed")
		server1.Close()
		w.Done()
	}()

	w.Add(1)
	go func() {
		// we send data to client 1, and close the connection. the result ends up in server1
		fmt.Fprintf(client1, "test message")
		log.Printf("client 1 closed")
		client1.Close()
		log.Printf("server 1 closed")
		server1.Close()
		w.Done()
	}()

	w.Add(1)
	go func() {
		buf := make([]byte, 12)
		client2.Read(buf)
		assert.Equal(t, "test message", string(buf))
		log.Printf("client 2 closed")
		client2.Close()
		w.Done()
	}()

	w.Wait()
	log.Printf("wait group finished\n")

}
