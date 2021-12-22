package ethstats

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
)

var upgrader = websocket.Upgrader{} // use default options

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

			msg, err := DecodeMsg(message)
			if err == nil {
				// downstream also sends messages that do not conform to the emit messages,
				// skip it since this was only meant to log outgoing messages
				p.logger.Debug("message from downstream", "type", msg.typ)
			}

			if err := p.downstream.WriteMessage(mt, message); err != nil {
				return
			}
		}
	}()

	p.logger.Debug("proxy connected")
	return connCloseCh
}

func (p *wsProxy) start(infoMsg []byte) {
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

	// send the infoMsg as initial message always after a connect. Only log if error,
	// and let the select group to handle any reconnects
	if err := p.upstream.WriteMessage(websocket.TextMessage, infoMsg); err != nil {
		p.logger.Error("failed to send info msg", "err", err)
	}

	for {
		select {
		case msg := <-p.msgCh:
			if err := p.upstream.WriteMessage(websocket.TextMessage, msg); err != nil {
				p.logger.Error("failed to write upstream message", "err", err)
				goto CONNECT
			}

		case <-connCloseCh:
			p.logger.Debug("proxy stopped")
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
	logger      hclog.Logger
	proxyAddr   string
	proxySecret string
	secret      string
	manager     sessionManager
}

func (c *wsCollector) handle(conn *websocket.Conn) {
	c.logger.Debug("new connection opened", "proxyEnabled", c.proxyAddr != "")

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
		if c.secret != "" && secret != c.secret {
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
			c.logger.Debug("failed to read msg", "err", err)
			break
		}

		msg, err := DecodeMsg(message)
		if err != nil {
			c.logger.Error("failed to decode msg", "err", err)
			continue
		}

		c.logger.Debug("new message", "node", nodeID, "typ", msg.typ)

		if !logged {
			// auth the node
			if err := handleAuth(msg); err != nil {
				c.logger.Error("failed to handle auth", "err", err)
				break
			}

			if c.proxyAddr != "" {
				proxy = newWsProxy(c.logger.Named("proxy_"+nodeID), conn, c.proxyAddr)

				// use the secret from the proxy
				proxyMsg := message
				if c.proxySecret != "" {
					proxyAuthMsg := msg.Copy()
					proxyAuthMsg.Set("secret", []byte(`"`+c.proxySecret+`"`))
					proxyMsg, _ = proxyAuthMsg.Marshal()
				}

				go proxy.start(proxyMsg)
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

		// deliver the message to the session
		if msg.typ != "node-ping" {
			// deliver the message to the proxy. We do not send neither:
			// - node-ping: since we do not want to proxy back pong.
			// - hello: since we have already sent hello ourselves to the proxy
			if proxy != nil && msg.typ != "hello" {
				proxy.Proxy(message)
			}

			c.manager.handleMessage(nodeID, msg)
		}
	}
}
