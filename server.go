package main

import (
	_ "github.com/lib/pq"

	"net/http"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	CollectorAddr string
	Endpoint      string
}

type Server struct {
	config *Config
	state  *State
}

func NewServer(config *Config) (*Server, error) {
	state, err := NewState(config.Endpoint)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		config: config,
		state:  state,
	}

	// start http/ws collector server
	srv.startCollectorServer()

	return srv, nil
}

func (s *Server) startCollectorServer() {
	http.HandleFunc("/", s.echo)
	log.Fatal(http.ListenAndServe(s.config.CollectorAddr, nil))
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
		"pending": s.handlePendingMsg,
	}

	defer func() {
		c.Close()
	}()

	proxy := newWsProxy(nil, c, "")
	defer proxy.close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		// fmt.Print(string(message))

		proxy.Proxy(message)

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
				log.Infof("handler for msg '%s' not found : ", msg.msgType())
			} else {
				if err := decodeFn(nodeID, msg); err != nil {
					log.Infof("failed to handle msg '%s': %v", msg.msgType(), err)
				}
			}
		}
	}
}

func (s *Server) Close() {
	s.state.db.Close()
}
