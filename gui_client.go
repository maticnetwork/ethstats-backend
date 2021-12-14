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

func relayClientToBor(childConn *websocket.Conn, parentConn *websocket.Conn, errTimer chan bool) {

	for {

		mt, message, err := childConn.ReadMessage()
		if err != nil {
			fmt.Println("w1", err)
			break
		}
		parentConn.WriteMessage(mt, message)

	}
}

func connectToGui(authMsg []byte, quitGuiConn chan bool, parentConn *websocket.Conn) {

	errTimer := make(chan bool, 5)
	errTimer <- true

	for {
		select {
		case <-globalQuit:
			return
		case <-quitGuiConn:
			return
		case <-errTimer:
			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				fmt.Println("w2", err)
				fmt.Println("Retrying in 5 secs")
				time.Sleep(5 * time.Second)
				errTimer <- true
				continue
			}

			defer func() {
				fmt.Println("Closing gui conn")
				c.Close()
			}()

			go relayClientToBor(c, parentConn, errTimer)

			c.WriteMessage(websocket.TextMessage, authMsg)

			for {
				select {
				case message := <-messages:
					c.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						fmt.Println("w3", err)
						fmt.Println("Retrying in 5 secs")
						time.Sleep(5 * time.Second)
						errTimer <- true
						continue
					}
				case <-globalQuit:
					return
				case <-quitGuiConn:
					return
				}
			}
		}
	}
}
