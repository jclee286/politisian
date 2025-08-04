package main

import (
	"os"

	"github.com/cometbft/cometbft/abci/server"
	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
)

// Minimalist ABCI application
type MinimalApp struct {
	types.BaseApplication
}

func NewMinimalApp(logger log.Logger) *MinimalApp {
	return &MinimalApp{}
}

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	app := NewMinimalApp(logger)

	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		logger.Error("Error creating ABCI server", "error", err)
		os.Exit(1)
	}
	srv.SetLogger(logger)

	if err := srv.Start(); err != nil {
		logger.Error("Error starting ABCI server", "error", err)
		os.Exit(1)
	}
	defer srv.Stop()

	// Wait for termination signal
	select {}
}
