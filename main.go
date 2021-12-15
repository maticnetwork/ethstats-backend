package main

import (
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

func (s *Server) handleReorgMsg(nodeID string, msg *Msg) error {
	var rawBlock Block
	if err := msg.decodeMsg("block", &rawBlock); err != nil {
		return err
	}
	if err := s.state.WriteReorgEvents(&rawBlock, &nodeID); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleBlockMsg(nodeID string, msg *Msg) error {
	var rawBlock Block
	if err := msg.decodeMsg("block", &rawBlock); err != nil {
		return err
	}
	if err := s.state.WriteBlock(&rawBlock); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleStatsMsg(nodeID string, msg *Msg) error {
	var rawStats NodeStats
	if err := msg.decodeMsg("stats", &rawStats); err != nil {
		return err
	}
	if err := s.state.WriteNodeStats(&rawStats, &nodeID); err != nil {
		log.Info(err)
	}
	return nil
}

func (s *Server) handlePendingMsg(nodeID string, msg *Msg) error {

	return nil
}

func (s *Server) echo(w http.ResponseWriter, r *http.Request) {

	quitGuiConn := make(chan bool)

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
		"block":   s.handleBlockMsg,
		"stats":   s.handleStatsMsg,
		"reorg":   s.handleReorgMsg,
		"pending": s.handlePendingMsg,
	}

	defer func() {
		c.Close()
		quitGuiConn <- true
	}()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		// fmt.Print(string(message))

		select {
		case messages <- message:

		default:

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

		msg, err := DecodeMsg(message)
		if err != nil {
			log.Printf("failed to decode msg: %v", err)
			continue
		}

		if msg.msgType() == "node-ping" {
			// send a pong
			if err := c.WriteMessage(mt, pongMessage); err != nil {
				log.Println("write:", err)
				break
			}
		} else if msg.msgType() == "hello" {
			go connectToGui(message, quitGuiConn, c)
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
			if err := s.state.WriteNodeInfo(&rawInfo, &rawInfo.Name); err != nil {
				log.Info(err)
			}
		} else {
			// use one of the decoders
			decodeFn, ok := decoders[msg.msgType()]
			if !ok {
				log.Info("handler for msg '%s' not found : ", msg.msgType())
			} else {
				if err := decodeFn(nodeID, msg); err != nil {
					log.Infof("failed to handle msg '%s': %v", msg.msgType(), err)
				}
			}
		}
	}
}

type Server struct {
	state *State
}

func (s *Server) Close() {
	s.state.db.Close()
}

func main() {
	flag.Parse()

	var err error

	path := fmt.Sprintf("host=localhost port=5432 user=postgres password=%s dbname=%s sslmode=disable", os.Getenv("DBPASS"), os.Getenv("DBNAME"))
	state, err := NewState(path)
	if err != nil {
		log.Info(err)
	}
	srv := &Server{
		state: state,
	}
	defer srv.Close()

	log.Info("DB CONNECTED!")

	http.HandleFunc("/", srv.echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
	close(globalQuit)
}
