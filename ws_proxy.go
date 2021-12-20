package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var u = url.URL{
	Scheme: "ws",
	Host:   "localhost:3001",
	Path:   "/api",
}

type wsProxy struct {
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

func newWsProxy(downstream *websocket.Conn, proxyAddr string) *wsProxy {
	w := &wsProxy{
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
		c, _, err := websocket.DefaultDialer.Dial(p.proxyAddr, nil)
		if err != nil {
			// log
			time.Sleep(1 * time.Second)
		} else {
			p.upstream = c
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
				fmt.Println(err)
				// log
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
				// log
				goto CONNECT
			}

		case <-connCloseCh:
			goto CONNECT

		case <-p.closeCh:
			return
		}
	}
}

/*
func (p *wsProxy) relayClientToBor(childConn *websocket.Conn) {

	for {

		mt, message, err := childConn.ReadMessage()
		if err != nil {
			childConn.Close()
			return
		}
		p.conn.WriteMessage(mt, message)

	}
}

func (p *wsProxy) connectToGui() {

	errTimer := make(chan bool, 5)
	errTimer <- true

	defer close(errTimer)

	for {
		select {
		case <-globalQuit:
			return
		case <-p.quitGuiConn:
			return
		case <-errTimer:

			if p.connectedToGui {
				return
			}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				fmt.Println("Retrying in 5 secs")
				time.Sleep(5 * time.Second)
				p.connectedToGui = false
				errTimer <- true
				continue
			} else {
				p.connectedToGui = true
				c.WriteMessage(websocket.TextMessage, p.authMsg)
			}

			defer func() {
				c.Close()
			}()

			go p.relayClientToBor(c)

			for {
				select {
				case message := <-messages:
					err := c.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						fmt.Println("Retrying in 5 secs")
						time.Sleep(5 * time.Second)
						p.reconnectToGui()
						return
					}
				case <-globalQuit:
					return
				case <-p.quitGuiConn:
					return
				}
			}
		}
	}
}

func (p *wsProxy) reconnectToGui() {
	fmt.Println("Trying to reconnect")

	p.connectedToGui = false

	go p.connectToGui()
}
*/
