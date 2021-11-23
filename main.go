package main

import (
	"database/sql"
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

type TxStats struct {
	Hash string `json:"hash"`
}

// nodeInfo is the collection of meta information about a node that is displayed
// on the monitoring page.
type NodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"client"`
	History  bool   `json:"canUpdateHistory"`
}

var rawInfo NodeInfo

// nodeStats is the information to report about the local node.
type NodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Mining   bool `json:"mining"`
	Hashrate int  `json:"hashrate"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

var rawStats NodeStats

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

type State struct {
	db *sql.DB
}

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

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		// log.Printf("recv: %s", message)

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

			tx, err := s.db.Begin()
			if err != nil {
				log.Info(err)
			}
			s.WriteReorgEvents(tx, &rawBlock)

			if err := tx.Commit(); err != nil {
				log.Info(err)
			}

		} else if strings.Contains(string(message), "block") {

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}
			var rawBlock Block
			m.decodeMsg("block", &rawBlock)

			tx, err := s.db.Begin()
			if err != nil {
				log.Info(err)
			}
			s.WriteBlock(tx, &rawBlock)

			if err := tx.Commit(); err != nil {
				log.Info(err)
			}

		} else if strings.Contains(string(message), "node-ping") {

			// send a pong
			if err := c.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		} else if strings.Contains(string(message), "stats") {

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}
			m.decodeMsg("stats", &rawStats)

			tx, err := s.db.Begin()
			if err != nil {
				log.Info(err)
			}
			if err := s.WriteNodeStats(tx, &rawStats, rawInfo.Name); err != nil {
				log.Info(err)
			}

			if err := tx.Commit(); err != nil {
				log.Info(err)
			}

		} else if strings.Contains(string(message), "hello") {

			m, err := extractMsg(message)
			if err != nil {
				log.Info(err)
			}

			m.decodeMsg("info", &rawInfo)

			tx, err := s.db.Begin()
			if err != nil {
				log.Info(err)
			}
			if err := s.WriteNodeInfo(tx, &rawInfo); err != nil {
				log.Info(err)
			}

			if err := tx.Commit(); err != nil {
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
