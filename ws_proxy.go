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

type connection struct {
	conn           *websocket.Conn
	authMsg        []byte
	quitGuiConn    chan bool
	connectedToGui bool
}

func (p *connection) relayClientToBor(childConn *websocket.Conn) {

	for {

		mt, message, err := childConn.ReadMessage()
		if err != nil {
			childConn.Close()
			return
		}
		p.conn.WriteMessage(mt, message)

	}
}

func (p *connection) connectToGui() {

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

func (p *connection) reconnectToGui() {
	fmt.Println("Trying to reconnect")

	p.connectedToGui = false

	go p.connectToGui()
}
