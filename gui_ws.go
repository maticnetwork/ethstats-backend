package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var guiWsAddr = "localhost:3001"

// type msgGui struct {
// 	Action string
// 	Data   map[string]json.RawMessage
// }

type guiBlock struct {
	Block
	Received    int64 `json:"received"`
	Arrived     int64 `json:"arrived"`
	Fork        int   `json:"fork"`
	Propagation int   `json:"propagation"`
	Time        int   `json:"time"`
	Trusted     bool  `json:"trusted"`
}

type guiData struct {
	ID             string   `json:"id"`
	Block          guiBlock `json:"block"`
	PropagationAvg int      `json:"propagationAvg"`
	History        [40]int  `json:"history"`
}

func populateGuiBlock(msg *Msg) (guiBlock, error) {

	now := time.Now()
	secs := now.Unix()

	var rawBlock guiBlock

	if err := msg.decodeMsg("block", &rawBlock); err != nil {
		return rawBlock, err
	}

	rawBlock.Arrived = secs
	rawBlock.Received = secs
	rawBlock.Fork = 0
	rawBlock.Propagation = 0
	rawBlock.Time = 1005
	rawBlock.Trusted = false

	return rawBlock, nil
}

func echoGui(w http.ResponseWriter, r *http.Request) {
	fmt.Print("LOLOLOLOLOL")
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	var err error
	var cGui *websocket.Conn
	cGui, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		panic(err)
		// return
	}
	defer cGui.Close()

	logged := false

	go func() {
		for {
			select {
			case message := <-messages:
				// log.Printf("%s", message)
				// m, _ := decodeMsg(message)
				// block2, _ := populateGuiBlock(m)
				// x := [40]int{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}
				// dataOut := &guiData{
				// 	Block:          block2,
				// 	ID:             "node1",
				// 	PropagationAvg: 0,
				// 	History:        x,
				// }
				// out, _ := json.Marshal(struct {
				// 	Action string   `json:"action"`
				// 	Data   *guiData `json:"data"`
				// }{
				// 	Action: m.typ,
				// 	Data:   dataOut,
				// })
				fmt.Print("LOL", message)
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

	http.HandleFunc("/primus/", echoGui)
	log.Fatal(http.ListenAndServe(guiWsAddr, nil))
	log.Info("Started server at 3001")
}
