package ethstats

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-hclog"
	_ "github.com/lib/pq"
)

type Config struct {
	CollectorAddr string
	Endpoint      string
	FrontendAddr  string
}

type Server struct {
	logger hclog.Logger
	config *Config
	state  *State
	srv    *http.Server
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
		logger:    s.logger.Named("collector"),
		manager:   s,
		proxyAddr: s.config.FrontendAddr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		collector.handle(conn)
	})

	srv := &http.Server{
		Addr:    s.config.CollectorAddr,
		Handler: mux,
	}
	s.srv = srv
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("error shutting down server", "err", err)
		}
	}()

	s.logger.Info("Collector ws server started", "addr", s.config.CollectorAddr, "secret", collector.secret)
	if s.config.FrontendAddr != "" {
		s.logger.Info("Frontend downstream enabled", "addr", s.config.FrontendAddr)
	}
}

func (s *Server) handleMessage(nodeID string, msg *Msg) {
	handle := func() error {
		switch msg.typ {
		case "hello":
			var info NodeInfo
			if err := msg.decodeMsg("info", &info); err != nil {
				return err
			}
			if err := s.state.WriteNodeInfo(&info); err != nil {
				return err
			}

		case "block":
			var block Block
			if err := msg.decodeMsg("block", &block); err != nil {
				return err
			}
			if err := s.state.WriteBlock(&block); err != nil {
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

		case "headEvent":
			var event HeadEvent
			if err := msg.decodeMsg("event", &event); err != nil {
				return err
			}
			if _, err := s.state.WriteHeadEvent(nodeID, &event); err != nil {
				return err
			}

		case "pending":
			// TODO?

		case "latency":
			// we do not track latency

		case "history":
			// we do not use history

		default:
			s.logger.Warn("unhandled message", "typ", msg.typ)
		}
		return nil
	}

	if err := handle(); err != nil {
		s.logger.Error("failed to handle message", "node", nodeID, "err", err)
	}
}

func (s *Server) Close() {
	s.state.db.Close()
	s.srv.Shutdown(context.Background())
}
