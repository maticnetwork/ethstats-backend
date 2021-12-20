package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var (
	initialWsPort = uint64(4569)
)

func getNextPort() int {
	return int(atomic.AddUint64(&initialWsPort, 1))
}

type mockWsServer struct {
	t            *testing.T
	sendCh       chan []byte
	recvCh       chan []byte
	closeCh      chan struct{}
	addr         string
	proxyHandler func(c *websocket.Conn)
	srv          *http.Server
}

func (m *mockWsServer) close() {
	close(m.closeCh)
	if err := m.srv.Shutdown(context.Background()); err != nil {
		m.t.Fatal(err)
	}
}

func (m *mockWsServer) write(data []byte) {
	m.sendCh <- data
}

func (m *mockWsServer) handle(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.t.Fatal(err)
		return
	}

	if m.proxyHandler != nil {
		m.proxyHandler(c)
		return
	}

	go func() {
		for {
			select {
			case msg := <-m.sendCh:
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					panic(err)
				}
			case <-m.closeCh:
				// Gorilla websocket does not seem to close connection after http server shutdown
				c.Close()
				return
			}
		}
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		} else {
			m.recvCh <- msg
		}
	}
}

func newMockWsServer(t *testing.T, addr string, proxyHandler ...func(*websocket.Conn)) *mockWsServer {
	if addr == "" {
		addr = "0.0.0.0:" + strconv.Itoa(getNextPort())
	}
	m := &mockWsServer{
		t:       t,
		sendCh:  make(chan []byte),
		recvCh:  make(chan []byte),
		closeCh: make(chan struct{}),
		addr:    "ws://" + addr,
	}
	if len(proxyHandler) != 0 {
		m.proxyHandler = proxyHandler[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handle)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	m.srv = srv
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)
	return m
}

type mockWsClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func (m *mockWsClient) close() {
	if err := m.conn.Close(); err != nil {
		m.t.Fatal(err)
	}
}

func (m *mockWsClient) Write(data []byte) {
	if err := m.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		m.t.Fatal(err)
	}
}

func newMockWsClient(t *testing.T, addr string) *mockWsClient {
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &mockWsClient{
		t:    t,
		conn: conn,
	}
	return c
}

func TestWsProxy(t *testing.T) {
	var (
		msg1 = []byte{0x1, 0x2}
		msg2 = []byte{0x3, 0x4}
	)

	upstream := newMockWsServer(t, "")

	doneCh := make(chan struct{})
	middleman := newMockWsServer(t, "", func(conn *websocket.Conn) {
		proxy := newWsProxy(conn, upstream.addr)
		defer proxy.close()

		go proxy.start()
		close(doneCh)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				// closed conn
				return
			} else {
				proxy.Proxy(msg)
			}
		}
	})
	defer middleman.close()

	// connect a websocket client (downstream) to middleman
	downstream := newMockWsClient(t, middleman.addr)
	defer downstream.close()

	<-doneCh
	downstream.Write(msg1)

	// wait for the first downstream message
	select {
	case msg := <-upstream.recvCh:
		assert.Equal(t, msg, msg1)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout in 1st message")
	}

	// restart upstream connection
	upstream.close()
	upstream = newMockWsServer(t, strings.TrimPrefix(upstream.addr, "ws://"))

	downstream.Write(msg2)

	// wait for the second downstream message
	select {
	case msg := <-upstream.recvCh:
		assert.Equal(t, msg, msg2)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout in 2st message")
	}
}
