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

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "<password>"
	dbname   = "postgres"
)

var db *sql.DB = nil

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

			// fmt.Println("lol : ", (msg["emit"]).([]interface{})[1])

		} else if strings.Contains(string(message), "REORGS DETECTED") {

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println(err)
			}

			block_number := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["number"].(float64))
			block_hash := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["hash"].(string)
			parent_hash := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["parentHash"].(string)
			time_stamp := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["timestamp"].(float64))
			miner := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["miner"].(string)
			gas_used := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["gasUsed"].(float64))
			gas_limit := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["gasLimit"].(float64))

			difficulty, err := strconv.ParseInt((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["difficulty"].(string), 10, 64)
			if err != nil {
				log.Println(err)
			}

			total_difficulty, err := strconv.ParseInt((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["totalDifficulty"].(string), 10, 64)
			if err != nil {
				log.Println(err)
			}

			transactions_root := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["transactionsRoot"].(string)
			transactions_count := len((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["transactions"].([]interface{}))
			uncles_count := len((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["uncles"].([]interface{}))
			state_root := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["stateRoot"].(string)

			// connection string
			psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

			// open database
			db, err := sql.Open("postgres", psqlconn)
			CheckError(err)

			// close database
			defer db.Close()

			// check db
			err = db.Ping()
			CheckError(err)

			fmt.Println("Connected!")

			insertDynStmt := `insert into "blocks"("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13 )`
			_, e := db.Exec(insertDynStmt, block_number, block_hash, parent_hash, time_stamp, miner, gas_used, gas_limit, difficulty, total_difficulty, transactions_root, transactions_count, uncles_count, state_root)
			CheckError(e)

			fmt.Println("reorg : ", (msg["emit"]).([]interface{})[1])

		} else if strings.Contains(string(message), "block") {

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println(err)
			}

			block_number := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["number"].(float64))
			block_hash := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["hash"].(string)
			parent_hash := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["parentHash"].(string)
			time_stamp := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["timestamp"].(float64))
			miner := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["miner"].(string)
			gas_used := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["gasUsed"].(float64))
			gas_limit := int((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["gasLimit"].(float64))

			difficulty, err := strconv.ParseInt((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["difficulty"].(string), 10, 64)
			if err != nil {
				log.Println(err)
			}

			total_difficulty, err := strconv.ParseInt((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["totalDifficulty"].(string), 10, 64)
			if err != nil {
				log.Println(err)
			}

			transactions_root := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["transactionsRoot"].(string)
			transactions_count := len((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["transactions"].([]interface{}))
			uncles_count := len((msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["uncles"].([]interface{}))
			state_root := (msg["emit"]).([]interface{})[1].(map[string]interface{})["block"].(map[string]interface{})["stateRoot"].(string)

			insertDynStmt := `insert into "blocks"("block_number", "block_hash", "parent_hash", "time_stamp", "miner", "gas_used", "gas_limit", "difficulty", "total_difficulty", "transactions_root", "transactions_count", "uncles_count", "state_root") values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13 )`
			_, e := db.Exec(insertDynStmt, block_number, block_hash, parent_hash, time_stamp, miner, gas_used, gas_limit, difficulty, total_difficulty, transactions_root, transactions_count, uncles_count, state_root)
			CheckError(e)

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
	// log.SetFlags(0)

	// connection string
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	// open database
	var err error
	db, err = sql.Open("postgres", psqlconn)
	CheckError(err)

	// close database
	defer db.Close()

	// check db
	err = db.Ping()
	CheckError(err)

	fmt.Println("Connected!")

	http.HandleFunc("/", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func CheckError(err error) {
	if err != nil {
		log.Info(err)
	}
}
