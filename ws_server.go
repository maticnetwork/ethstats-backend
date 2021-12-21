package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
)

var upgrader = websocket.Upgrader{} // use default options

var u = url.URL{
	Scheme: "ws",
	Host:   "localhost:3001",
	Path:   "/api",
}

type wsProxy struct {
	logger hclog.Logger

	// connection with the Bor client
	downstream *websocket.Conn

	// connection to the proxy frontend server
	upstream *websocket.Conn

	// proxyAddr is the address to the proxy server
	proxyAddr string

	// close channel to stop the proxy
	closeCh chan struct{}

	// channel to proxy messages
	msgCh chan []byte
}

func newWsProxy(logger hclog.Logger, downstream *websocket.Conn, proxyAddr string) *wsProxy {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	w := &wsProxy{
		logger:     logger,
		downstream: downstream,
		closeCh:    make(chan struct{}),
		msgCh:      make(chan []byte, 1000),
		proxyAddr:  proxyAddr,
	}
	return w
}

func (p *wsProxy) Proxy(data []byte) {
	// we do not want to block the server if the proxy
	// cannot rely data (i.e. the frontend is down).
	select {
	case p.msgCh <- data:
	default:
	}
}

func (p *wsProxy) close() {
	close(p.closeCh)
}

func (p *wsProxy) connect() chan struct{} {
	for {
		conn, _, err := websocket.DefaultDialer.Dial(p.proxyAddr, nil)
		if err != nil {
			p.logger.Error("failed to dial upstream", "addr", p.proxyAddr, "err", err)
			time.Sleep(1 * time.Second)
		} else {
			p.upstream = conn
			break
		}
	}

	connCloseCh := make(chan struct{})

	// read any message from upstream server and relay back to Bor
	go func() {
		for {
			mt, message, err := p.upstream.ReadMessage()
			if err != nil {
				close(connCloseCh)
				return
			}
			if err := p.downstream.WriteMessage(mt, message); err != nil {
				return
			}
		}
	}()

	return connCloseCh
}

func (p *wsProxy) start() {
	defer func() {
		// close the websocket connection (if open)
		if p.upstream != nil {
			if err := p.upstream.Close(); err != nil {
				p.logger.Error("failed to close upstream", "err", err)
			}
		}
	}()

CONNECT:
	// try to connect with the frontend node
	connCloseCh := p.connect()

	for {
		select {
		case msg := <-p.msgCh:
			if err := p.upstream.WriteMessage(websocket.TextMessage, msg); err != nil {
				p.logger.Error("failed to write upstream message", "err", err)
				goto CONNECT
			}

		case <-connCloseCh:
			goto CONNECT

		case <-p.closeCh:
			return
		}
	}
}

type sessionManager interface {
	handleMessage(nodeID string, msg *Msg)
}

var loggedMessage = []byte(`{
	"emit": ["ready"]
}`)

// pong message needs to send two messages (second is not read)
var pongMessage = []byte(`{
	"emit": [
		"node-pong",
		{}
	]
}`)

type wsCollector struct {
	logger    hclog.Logger
	proxyAddr string
	password  string
	manager   sessionManager
}

func (c *wsCollector) handle(conn *websocket.Conn) {
	// start the proxy to the upstream repo (if any)
	var proxy *wsProxy

	logged := false
	var nodeID string

	defer func() {
		conn.Close()
	}()

	handleAuth := func(msg *Msg) error {
		// first message has to be a 'hello'
		if msg.msgType() != "hello" {
			return fmt.Errorf("bad auth message type: %s", msg.msgType())
		}

		// decode the secret and get node info for the session
		var secret string
		if err := msg.decodeMsg("secret", &secret); err != nil {
			return err
		}
		if c.password != "" && secret != c.password {
			return fmt.Errorf("secret is not correct: %s", secret)
		}

		var info NodeInfo
		if err := msg.decodeMsg("info", &info); err != nil {
			return err
		}

		if err := conn.WriteMessage(websocket.TextMessage, loggedMessage); err != nil {
			return err
		}
		nodeID = info.Name
		return nil
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Error("failed to read msg", "err", err)
			break
		}

		msg, err := DecodeMsg(message)
		if err != nil {
			c.logger.Error("failed to decode msg", "err", err)
			continue
		}

		if !logged {
			// auth the node
			if err := handleAuth(msg); err != nil {
				c.logger.Error("failed to handle auth", "err", err)
				break
			}

			if c.proxyAddr != "" {
				proxy = newWsProxy(c.logger.Named("proxy_"+nodeID), conn, "")
				defer proxy.close()
			}
			logged = true
		}

		if msg.msgType() == "node-ping" {
			// send a pong
			if err := conn.WriteMessage(websocket.TextMessage, pongMessage); err != nil {
				c.logger.Error("failed to write message", "err", err)
				break
			}
		}

		// deliver the message to the proxy
		if proxy != nil {
			proxy.Proxy(message)
		}

		// deliver the message to the session
		if msg.typ != "hello" {
			c.manager.handleMessage(nodeID, msg)
		}
	}
}
