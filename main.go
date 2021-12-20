package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	defaultDBEndpoint = "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
)

func main() {
	var wsAddr, dbEndpoint string

	flag.StringVar(&dbEndpoint, "db-endpoint", defaultDBEndpoint, "")
	flag.StringVar(&wsAddr, "ws-addr", "localhost:3000", "ws service address for collector")
	flag.Parse()

	config := &Config{
		Endpoint:      dbEndpoint,
		CollectorAddr: wsAddr,
	}

	srv, err := NewServer(config)
	if err != nil {
		fmt.Printf("[ERROR]: %v", err)
		os.Exit(0)
	}

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	<-signalCh
	srv.Close()
}
