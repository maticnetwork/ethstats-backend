package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"

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

func NewState(path string) (*State, error) {

	db, err := sql.Open("postgres", path)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	s := &State{
		db: db,
	}
	return s, nil
}

func cleanMessage(message []byte) *Msg {

	var msg map[string]interface{}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println(err)
	}

	m := &Msg{
		msg: msg,
	}

	return m
}

func (m Msg) decodeMsg() Block {

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

	decodedMsg := (m.msg["emit"]).([]interface{})[1].(map[string]interface{})["block"]

	out, _ := json.Marshal(decodedMsg)

	var rawBlock Block

	if err := json.Unmarshal(out, &rawBlock); err != nil {
		panic(err)
	}

	return rawBlock
}

type Msg struct {
	msg map[string]interface{}
}

func (s *State) WriteBlock(block Block, table string) {

	block_number := int(block.Number)
	block_hash := block.Hash
	parent_hash := block.ParentHash
	time_stamp := int(block.Timestamp)
	miner := block.Miner
	gas_used := int(block.GasUsed)
	gas_limit := int(block.GasLimit)

	difficulty, err := strconv.ParseInt(block.Diff, 10, 64)
	if err != nil {
		log.Println(err)
	}

	total_difficulty, err := strconv.ParseInt(block.TotalDiff, 10, 64)
	if err != nil {
		log.Println(err)
	}

	transactions_root := block.TxHash

	// txCount and UncleCount are arrays.
	transactions_count := len(block.Txs)
	uncles_count := len(block.Uncles)

	state_root := block.Root

	insertDynStmt := fmt.Sprintf(`insert into "%s"("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13 )`, table)
	_, e := s.db.Exec(insertDynStmt, block_number, block_hash, parent_hash, time_stamp, miner, gas_used, gas_limit, difficulty, total_difficulty, transactions_root, transactions_count, uncles_count, state_root)

	if e != nil {
		log.Println(err)
	}

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
		log.Printf("recv: %s", message)

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

			m := cleanMessage(message)
			block := m.decodeMsg()
			s.WriteBlock(block, "reorgblocks")

		} else if strings.Contains(string(message), "block") {

			m := cleanMessage(message)
			block := m.decodeMsg()
			s.WriteBlock(block, "blocks")

		} else if strings.Contains(string(message), "node-ping") {

			// send a pong

			if err := c.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		}
	}
}

func main() {
	flag.Parse()

	var err error
	s, err = NewState("host=localhost port=5432 user=postgres password=shivam dbname=postgres sslmode=disable")
	if err != nil {
		log.Info(err)
	}
	defer s.db.Close()

	http.HandleFunc("/", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
