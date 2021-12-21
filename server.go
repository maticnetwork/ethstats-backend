package main

import (
	"net/http"

	"github.com/hashicorp/go-hclog"
	_ "github.com/lib/pq"
)

type Config struct {
	CollectorAddr string
	Endpoint      string
}

type Server struct {
	logger hclog.Logger
	config *Config
	state  *State
}

func NewServer(logger hclog.Logger, config *Config) (*Server, error) {
	state, err := NewState(config.Endpoint)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		logger: logger,
		config: config,
		state:  state,
	}

	// start http/ws collector server
	srv.startCollectorServer()

	return srv, nil
}

func (s *Server) startCollectorServer() {
	collector := &wsCollector{
		manager: s,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		collector.handle(conn)
	})
	http.ListenAndServe(s.config.CollectorAddr, nil)
}

func (s *Server) handleMessage(nodeID string, msg *Msg) {
	handle := func() error {
		switch msg.typ {
		case "hello":
			var info *NodeInfo
			if err := msg.decodeMsg("info", info); err != nil {
				return err
			}
			if err := s.state.WriteNodeInfo(info); err != nil {
				return err
			}

		case "block":
			var block *Block
			if err := msg.decodeMsg("block", block); err != nil {
				return err
			}
			if err := s.state.WriteBlock(block); err != nil {
				return err
			}

		case "stats":
			var stats NodeStats
			if err := msg.decodeMsg("stats", &stats); err != nil {
				return err
			}
			if err := s.state.WriteNodeStats(nodeID, &stats); err != nil {
				return err
			}
		}
		return nil
	}

	if err := handle(); err != nil {
		s.logger.Error("failed to handle message", "node", nodeID, "err", err)
	}
}

func (s *Server) Close() {
	s.state.db.Close()
}
