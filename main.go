package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"net/http"

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

var messages = make(chan []byte, 1)
var globalQuit = make(chan struct{})

var s *State

func decodeMsg(message []byte) (*Msg, error) {
	// FORMAT of msg
	// {
	// 	"emit": [
	// 	   "<msg-type>",
	// 	   {
	// 		   "block": {
	//				number : xxxxx
	//				hash : xxxxx
	//				...
	//			}
	// 	   }
	// 	]
	// }

	var msg struct {
		Emit []json.RawMessage
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return nil, err
	}
	if len(msg.Emit) != 2 {
		return nil, fmt.Errorf("2 items expected")
	}

	// decode typename as string
	var typName string
	if err := json.Unmarshal(msg.Emit[0], &typName); err != nil {
		return nil, fmt.Errorf("failed to decode type: %v", err)
	}
	// decode data
	var data map[string]json.RawMessage
	if err := json.Unmarshal(msg.Emit[1], &data); err != nil {
		return nil, fmt.Errorf("failed to decode data: %v", err)
	}

	m := &Msg{
		typ: typName,
		msg: data,
	}
	return m, nil
}

type Msg struct {
	typ string
	msg map[string]json.RawMessage
}

func (m *Msg) msgType() string {
	return m.typ
}

func (m *Msg) decodeMsg(field string, out interface{}) error {
	data, ok := m.msg[field]
	if !ok {
		return fmt.Errorf("message %s not found", field)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}

func handleReorgMsg(nodeID string, msg *Msg) error {
	var rawBlock Block
	if err := msg.decodeMsg("block", &rawBlock); err != nil {
		return err
	}
	if err := s.WriteReorgEvents(&rawBlock, &nodeID); err != nil {
		return err
	}
	return nil
}

func handleBlockMsg(nodeID string, msg *Msg) error {
	var rawBlock Block
	if err := msg.decodeMsg("block", &rawBlock); err != nil {
		return err
	}
	if err := s.WriteBlock(&rawBlock); err != nil {
		return err
	}
	return nil
}

func handleStatsMsg(nodeID string, msg *Msg) error {
	var rawStats NodeStats
	if err := msg.decodeMsg("stats", &rawStats); err != nil {
		return err
	}
	if err := s.WriteNodeStats(&rawStats, &nodeID); err != nil {
		log.Info(err)
	}
	return nil
}

func handlePendingMsg(nodeID string, msg *Msg) error {

	return nil
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

	decoders := map[string]func(string, *Msg) error{
		"block":   handleBlockMsg,
		"stats":   handleStatsMsg,
		"reorg":   handleReorgMsg,
		"pending": handlePendingMsg,
	}

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		select {
		case messages <- message:

		default:

		}

		log.Printf("recv: %s", message)

		if !logged {
			// send auth message
			if err := c.WriteMessage(mt, loggedMessage); err != nil {
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
			if err := c.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		} else if msg.msgType() == "hello" {
			// gather the node info and keep the id during the session
			var rawInfo NodeInfo
			if err := msg.decodeMsg("info", &rawInfo); err != nil {
				log.Info(err)
				continue
			}
			if err := msg.decodeMsg("id", &nodeID); err != nil {
				log.Info(err)
				continue
			}
			if err := s.WriteNodeInfo(&rawInfo, &rawInfo.Name); err != nil {
				log.Info(err)
			}
		} else {
			// use one of the decoders
			decodeFn, ok := decoders[msg.msgType()]
			if !ok {
				log.Info("handler for msg '%s' not found", msg.msgType())
			} else {
				if err := decodeFn(nodeID, msg); err != nil {
					log.Info("failed to handle msg '%s': %v", msg.msgType(), err)
				}
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
	log.Print("DB CONNECTED!")

	go startGui()

	http.HandleFunc("/", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
	// close(globalQuit)
}
