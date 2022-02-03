package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/maticnetwork/ethstats-backend/ethstats"
)

var (
	defaultDBEndpoint = "postgres://postgres:postgrespassword@postgres:5432/postgres?sslmode=disable"
)

func main() {
	config := &ethstats.Config{}
	var logLevel string

	flag.StringVar(&config.Endpoint, "db-endpoint", defaultDBEndpoint, "")
	flag.StringVar(&config.CollectorAddr, "collector.addr", "localhost:8000", "ws service address for collector")
	flag.StringVar(&config.CollectorSecret, "collector.secret", "", "")
	flag.StringVar(&logLevel, "log-level", "Log level", "info")
	flag.StringVar(&config.FrontendAddr, "frontend.addr", "", "")
	flag.StringVar(&config.FrontendSecret, "frontend.secret", "", "")
	flag.Parse()

	logger := hclog.New(&hclog.LoggerOptions{Level: hclog.LevelFromString(logLevel)})
	srv, err := ethstats.NewServer(logger, config)
	if err != nil {
		fmt.Printf("[ERROR]: %v", err)
		os.Exit(0)
	}

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	<-signalCh
	srv.Close()
}
