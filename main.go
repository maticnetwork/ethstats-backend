package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// blockStats is the information to report about individual blocks.
type Block struct {
	Number     int       `json:"number"`
	Hash       string    `json:"hash"`
	ParentHash string    `json:"parentHash"`
	Timestamp  int       `json:"timestamp"`
	Miner      string    `json:"miner"`
	GasUsed    uint64    `json:"gasUsed"`
	GasLimit   uint64    `json:"gasLimit"`
	Diff       string    `json:"difficulty"`
	TotalDiff  string    `json:"totalDifficulty"`
	Txs        []TxStats `json:"transactions"`
	TxHash     string    `json:"transactionsRoot"`
	Root       string    `json:"stateRoot"`
	Uncles     []Block   `json:"uncles"`
}

var addr = flag.String("addr", "localhost:3000", "http service address")

var upgrader = websocket.Upgrader{} // use default options

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

var s *State

func extractMsg(message []byte) (*Msg, error) {

	var msg map[string]json.RawMessage

	if err := json.Unmarshal(message, &msg); err != nil {
		return nil, err
	}

	m := &Msg{
		msg: msg,
	}

	return m, nil
}

func (m *Msg) decodeMsg(field string, out interface{}) error {

	// FORMAT of msg
	// {
	// 	"emit": [
	// 	   "..",
	// 	   {
	// 		   "block": {
	//				number : xxxxx
	//				hash : xxxxx
	//				...
	//			}
	// 	   }
	// 	]
	// }

	var msg []json.RawMessage

	if err := json.Unmarshal(m.msg["emit"], &msg); err != nil {
		return err
	}

	var msg2 map[string]json.RawMessage

	if err := json.Unmarshal(msg[1], &msg2); err != nil {
		return err
	}

	data, ok := msg2[field]
	if !ok {
		return fmt.Errorf("message %s not found", field)
	}

	if err := json.Unmarshal(data, out); err != nil {
		return err
	}

	return nil
}

type Msg struct {
	msg map[string]json.RawMessage
}

func echo(w http.ResponseWriter, r *http.Request) {

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	logged := false

	var nodeID string

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		// log.Printf("recv: %d", mt)

		if !logged {
			// send auth message
			if err := c.WriteMessage(mt, loggedMessage); err != nil {
				log.Println("write:", err)
				break
			}
			logged = true
		}

		if strings.Contains(string(message), "pending") {

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println(err)
			}

		} else if strings.Contains(string(message), "REORGS DETECTED") {

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}
			var rawBlock Block
			m.decodeMsg("block", &rawBlock)

			s.WriteReorgEvents(&rawBlock, &nodeID)

		} else if strings.Contains(string(message), "block") {

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}
			var rawBlock Block
			m.decodeMsg("block", &rawBlock)

			s.WriteBlock(&rawBlock)

		} else if strings.Contains(string(message), "node-ping") {

			// send a pong
			if err := c.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		} else if strings.Contains(string(message), "stats") {

			var rawStats NodeStats

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}
			m.decodeMsg("stats", &rawStats)

			if err := s.WriteNodeStats(&rawStats, &nodeID); err != nil {
				log.Info(err)
			}

		} else if strings.Contains(string(message), "hello") {
			// First message sent by the user
			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}

			var rawInfo NodeInfo
			m.decodeMsg("info", &rawInfo)
			m.decodeMsg("id", &nodeID)

			if err := s.WriteNodeInfo(&rawInfo, &rawInfo.Name); err != nil {
				log.Info(err)
			}
		}
	}
}

func main() {
	flag.Parse()

	var err error
	path := fmt.Sprintf("host=localhost port=5432 user=postgres password=%s dbname=%s sslmode=disable", os.Getenv("DBPASS"), os.Getenv("DBNAME"))
	s, err = NewState(path)
	if err != nil {
		log.Info(err)
	}
	defer s.db.Close()
	log.Info("DB CONNECTED!")

	http.HandleFunc("/", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
