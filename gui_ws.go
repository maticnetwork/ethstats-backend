package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var guiWsAddr = "localhost:3001"

func echoGui(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	var err error
	var cGui *websocket.Conn
	cGui, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer cGui.Close()

	logged := false

	go func() {
		for {
			select {
			case message := <-messages:
				// log.Printf("%s", message)
				cGui.WriteMessage(1, message)
			case <-globalQuit: // will explain this in the last section
				return
			}
		}
	}()

	for {
		mt, message, err := cGui.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Print("recv", message)

		if !logged {
			// send auth message
			if err := cGui.WriteMessage(mt, loggedMessage); err != nil {
				log.Println("write:", err)
				break
			}
			logged = true
		}

		msg, err := decodeMsg(message)
		if err != nil {
			log.Println("failed to decode msg: %v", err)
			continue
		}

		if msg.msgType() == "node-ping" {
			// send a pong
			if err := cGui.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func startGui() {

	http.HandleFunc("/gui", echoGui)
	log.Fatal(http.ListenAndServe(guiWsAddr, nil))
	log.Info("Started server at 3001")
}
