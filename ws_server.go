package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

var u = url.URL{
	Scheme: "ws",
	Host:   "localhost:3001",
	Path:   "/api",
}

type wsProxy struct {
	logger *log.Logger

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

func newWsProxy(logger *log.Logger, downstream *websocket.Conn, proxyAddr string) *wsProxy {
	if logger == nil {
		logger = log.New(ioutil.Discard, "", 0)
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
			p.logger.Printf("[ERROR]: Failed to dial upstream %s: %v", p.proxyAddr, err)
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
				p.logger.Printf("[ERROR]: Failed to close upstream: %v", err)
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
				p.logger.Printf("[ERROR]: Failed to write upstream message: %v", err)
				goto CONNECT
			}

		case <-connCloseCh:
			goto CONNECT

		case <-p.closeCh:
			return
		}
	}
}

type session struct {
	info    *NodeInfo
	closeCh chan struct{}
	msgCh   chan *Msg
}

type sessionManager interface {
	handleSession(s *session)
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
	logger    *log.Logger
	proxyAddr string
	password  string
	manager   sessionManager
}

func (c *wsCollector) handle(conn *websocket.Conn) {
	// start the proxy to the upstream repo (if any)
	var proxy *wsProxy
	if c.proxyAddr != "" {
		proxy = newWsProxy(c.logger, conn, "")
		defer proxy.close()
	}

	logged := false
	var ss *session

	defer func() {
		conn.Close()
		if ss != nil {
			close(ss.closeCh)
		}
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
		ss = &session{
			info:    &info,
			closeCh: make(chan struct{}),
			msgCh:   make(chan *Msg, 100),
		}
		c.manager.handleSession(ss)
		return nil
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		msg, err := DecodeMsg(message)
		if err != nil {
			log.Printf("failed to decode msg: %v", err)
			continue
		}

		if !logged {
			// auth the node
			if err := handleAuth(msg); err != nil {
				log.Printf("failed to handle auth: %v", err)
				continue
			}
			logged = true
		}

		if msg.msgType() == "node-ping" {
			// send a pong
			if err := conn.WriteMessage(websocket.TextMessage, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		}

		// deliver the message to the proxy
		if proxy != nil {
			proxy.Proxy(message)
		}

		// deliver the message to the session
		if msg.typ != "hello" && msg.typ != "node-ping" {
			select {
			case ss.msgCh <- msg:
			default:
			}
		}
	}
}
