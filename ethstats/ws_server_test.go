package ethstats

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

var (
	initialWsPort = uint64(4569)
)

func getNextPort() int {
	return int(atomic.AddUint64(&initialWsPort, 1))
}

type mockWsServer struct {
	t *testing.T

	addr     string
	srv      *http.Server
	cancelFn context.CancelFunc
}

func (m *mockWsServer) close() {
	m.cancelFn()

	if err := m.srv.Shutdown(context.Background()); err != nil {
		m.t.Fatal(err)
	}
}

type wsChHandler struct {
	sendCh chan []byte
	recvCh chan []byte
}

func (m *wsChHandler) handle(ctx context.Context, c *websocket.Conn) {
	go func() {
		for {
			select {
			case msg := <-m.sendCh:
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					panic(err)
				}
			case <-ctx.Done():
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

func newMockWsServer(t *testing.T, addr string, handler func(ctx context.Context, conn *websocket.Conn)) *mockWsServer {
	if addr == "" {
		addr = "0.0.0.0:" + strconv.Itoa(getNextPort())
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	m := &mockWsServer{
		t:        t,
		addr:     "ws://" + addr,
		cancelFn: cancelFn,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(ctx, c)
	})

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

func (m *mockWsClient) readMsg() *Msg {
	msg, err := DecodeMsg(m.read())
	if err != nil {
		m.t.Fatal(err)
	}
	return msg
}

func (m *mockWsClient) read() []byte {
	_, msg, err := m.conn.ReadMessage()
	if err != nil {
		m.t.Fatal(err)
	}
	return msg
}

func (m *mockWsClient) emit(typ, msg string) {
	m.Write([]byte(`{
		"emit": [
			"` + typ + `",
			` + msg + `
		]
	}`))
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
		msg1    = []byte{0x1, 0x2}
		msg2    = []byte{0x3, 0x4}
		infoMsg = []byte{0x5, 0x6}
	)

	echoCh := &wsChHandler{
		recvCh: make(chan []byte, 10),
	}
	recv := func(timeout time.Duration) []byte {
		select {
		case msg := <-echoCh.recvCh:
			return msg
		case <-time.After(timeout):
			t.Fatal("timeout")
		}
		return nil
	}

	upstream := newMockWsServer(t, "", echoCh.handle)

	doneCh := make(chan struct{})
	middleman := newMockWsServer(t, "", func(ctx context.Context, conn *websocket.Conn) {
		proxy := newWsProxy(nil, conn, upstream.addr)
		defer proxy.close()

		go proxy.start(infoMsg)
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
	assert.Equal(t, recv(1*time.Second), infoMsg)
	assert.Equal(t, recv(1*time.Second), msg1)

	// restart upstream connection
	upstream.close()
	upstream = newMockWsServer(t, strings.TrimPrefix(upstream.addr, "ws://"), echoCh.handle)

	downstream.Write(msg2)

	// wait for the second downstream message
	assert.Equal(t, recv(1*time.Second), infoMsg)
	assert.Equal(t, recv(1*time.Second), msg2)
}

type mockSessionManager struct {
	ch chan *Msg
}

func newMockSessionManager() *mockSessionManager {
	return &mockSessionManager{ch: make(chan *Msg, 10)}
}

func (m *mockSessionManager) handleMessage(nodeID string, msg *Msg) {
	m.ch <- msg
}

func TestWsCollector_Session(t *testing.T) {
	sm := newMockSessionManager()

	ws := &wsCollector{
		manager: sm,
		logger:  hclog.NewNullLogger(),
	}

	srv := newMockWsServer(t, "", func(ctx context.Context, conn *websocket.Conn) {
		ws.handle(conn)
	})

	clt := newMockWsClient(t, srv.addr)
	clt.emit("hello", `{
		"secret": "secret",
		"info": {}
	}`)

	clt.emit("msg1", `{}`)
	clt.emit("msg2", `{}`)

	assert.Equal(t, (<-sm.ch).typ, "hello")
	assert.Equal(t, (<-sm.ch).typ, "msg1")
	assert.Equal(t, (<-sm.ch).typ, "msg2")
}

func TestWsCollector_PingPong(t *testing.T) {
	sm := newMockSessionManager()

	ws := &wsCollector{
		manager: sm,
		logger:  hclog.NewNullLogger(),
	}
	srv := newMockWsServer(t, "", func(ctx context.Context, conn *websocket.Conn) {
		ws.handle(conn)
	})

	clt := newMockWsClient(t, srv.addr)
	clt.emit("hello", `{
		"secret": "",
		"info": {}
	}`)

	clt.emit("node-ping", `{}`)

	// expect a ready message
	assert.Equal(t, clt.readMsg().typ, "ready")

	// expect a pong message
	assert.Equal(t, clt.readMsg().typ, "node-pong")
}
