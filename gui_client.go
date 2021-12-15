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

func (parentConn connection) relayClientToBor(childConn *websocket.Conn) {

	for {

		mt, message, err := childConn.ReadMessage()
		if err != nil {
			childConn.Close()
			return
		}
		parentConn.conn.WriteMessage(mt, message)

	}
}

func (parentConn connection) connectToGui() {

	errTimer := make(chan bool, 5)
	errTimer <- true

	defer close(errTimer)

	for {
		select {
		case <-globalQuit:
			return
		case <-parentConn.quitGuiConn:
			return
		case <-errTimer:

			if parentConn.connectedToGui {
				return
			}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				fmt.Println("Retrying in 5 secs")
				time.Sleep(5 * time.Second)
				parentConn.connectedToGui = false
				errTimer <- true
				continue
			} else {
				parentConn.connectedToGui = true
				c.WriteMessage(websocket.TextMessage, parentConn.authMsg)
			}

			defer func() {
				c.Close()
			}()

			go parentConn.relayClientToBor(c)

			for {
				select {
				case message := <-messages:
					err := c.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						fmt.Println("Retrying in 5 secs")
						time.Sleep(5 * time.Second)
						parentConn.reconnectToGui()
						return
					}
				case <-globalQuit:
					return
				case <-parentConn.quitGuiConn:
					return
				}
			}
		}
	}
}

func (parentConn connection) reconnectToGui() {
	fmt.Println("Trying to reconnect")

	parentConn.connectedToGui = false

	go parentConn.connectToGui()
}
