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
	defaultDBEndpoint   = "postgres://postgres:postgrespassword@postgres:5432/postgres?sslmode=disable"
	persistDataDuration int
)

func main() {
	config := &ethstats.Config{}
	var logLevel string

	dbEndpoint := os.Getenv("DB_ENDPOINT")
	if dbEndpoint == "" {
		dbEndpoint = defaultDBEndpoint
	}

	serverCMD := flag.NewFlagSet("server", flag.ExitOnError)
	serverCMD.StringVar(&config.Endpoint, "db-endpoint", dbEndpoint, "")
	serverCMD.StringVar(&config.CollectorAddr, "collector.addr", "0.0.0.0:8000", "ws service address for collector")
	serverCMD.StringVar(&config.CollectorSecret, "collector.secret", os.Getenv("COLLECTOR_SECRET"), "")
	serverCMD.StringVar(&logLevel, "log-level", "Log level", "info")
	serverCMD.StringVar(&config.FrontendAddr, "frontend.addr", "", "")
	serverCMD.StringVar(&config.FrontendSecret, "frontend.secret", "", "")

	purgeCMD := flag.NewFlagSet("purge", flag.ExitOnError)
	purgeCMD.IntVar(&persistDataDuration, "persist-days", 0, "Data older than this days will be deleted")

	switch os.Args[1] {
	case "server":
		serverCMD.Parse(os.Args[2:])

	case "purge":
		purgeCMD.Parse(os.Args[2:])
		if persistDataDuration > 0 {
			state, err := ethstats.NewState(config.Endpoint)
			if err != nil {
				fmt.Printf("[ERROR]: %v", err)
				os.Exit(0)
			}
			persistDataDuration = persistDataDuration * 24 * 60 * 60

			//send persistData Duration in seconds
			err = state.DeleteOlderData(persistDataDuration)
			if err != nil {
				fmt.Printf("[ERROR]: %v", err)
				os.Exit(0)
			}
			fmt.Printf("[INFO]: Data older than %d days deleted\n", persistDataDuration)
		} else {
			fmt.Println("[ERROR]: persist-days must be greater than 0")
		}
		os.Exit(0)

	default:
		fmt.Println("expected 'server' or 'purge' subcommands")
		os.Exit(1)
	}

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
