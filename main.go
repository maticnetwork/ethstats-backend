package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"

	_ "github.com/lib/pq"

	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

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

	// connection string
	psqlconn := path

	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected!")

	s := &State{
		db: db,
	}
	return s, nil
}

func (m Msg) decodeMsg(field string) interface{} {

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

	decodedMsg := (m.msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})[field]
	return decodedMsg
}

type Msg struct {
	msg map[string]interface{}
}

func (s *State) WriteBlock(message []byte, table string) {
	var msg map[string]interface{}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println(err)
	}

	m := &Msg{
		msg: msg,
	}

	block_number := int(m.decodeMsg("number").(float64))
	block_hash := m.decodeMsg("hash").(string)
	parent_hash := m.decodeMsg("parentHash").(string)
	time_stamp := int(m.decodeMsg("timestamp").(float64))
	miner := m.decodeMsg("miner").(string)
	gas_used := int(m.decodeMsg("gasUsed").(float64))
	gas_limit := int(m.decodeMsg("gasLimit").(float64))

	difficulty, err := strconv.ParseInt(m.decodeMsg("difficulty").(string), 10, 64)
	if err != nil {
		log.Println(err)
	}

	total_difficulty, err := strconv.ParseInt(m.decodeMsg("totalDifficulty").(string), 10, 64)
	if err != nil {
		log.Println(err)
	}

	transactions_root := m.decodeMsg("transactionsRoot").(string)

	//txCount and UncleCount are arrays.
	transactions_count := len(m.decodeMsg("transactions").([]interface{}))
	uncles_count := len(m.decodeMsg("uncles").([]interface{}))

	state_root := m.decodeMsg("stateRoot").(string)

	insertDynStmt := fmt.Sprintf(`insert into "%s"("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13 )`, table)
	_, e := s.db.Exec(insertDynStmt, block_number, block_hash, parent_hash, time_stamp, miner, gas_used, gas_limit, difficulty, total_difficulty, transactions_root, transactions_count, uncles_count, state_root)

	if e != nil {
		log.Println(err)
	}

}

func echo(w http.ResponseWriter, r *http.Request) {

	// s, err := NewState("host=localhost port=5432 user=postgres password=shivam dbname=postgres sslmode=disable")

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
			s.WriteBlock(message, "reorgblocks")

		} else if strings.Contains(string(message), "block") {

			s.WriteBlock(message, "blocks")

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
