package main

import (
	"fmt"
	"os"
	"path/filepath"

	"politisian/app"
	"politisian/server"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/config"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/types"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	homeDir, _ := os.UserHomeDir()
	politisianDir := filepath.Join(homeDir, "politisian")
	cometbftDir := filepath.Join(politisianDir, ".cometbft")
	
	cfg := config.DefaultConfig()
	cfg.SetRoot(cometbftDir)
	cfg.RPC.ListenAddress = "tcp://0.0.0.0:26657"

	db, err := dbm.NewDB("politisian_app", dbm.GoLevelDBBackend, cometbftDir)
	if err != nil {
		return fmt.Errorf("데이터베이스 생성 실패: %w", err)
	}
	abciApp := app.NewPoliticianApp(db)

	return runNode(cfg, abciApp)
}

func runNode(cfg *config.Config, app abci.Application) error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("노드 키 로드/생성 실패: %w", err)
	}

	pv := privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())

	genesisDocProvider := node.GenesisDocProvider(func() (*types.GenesisDoc, error) {
		return types.GenesisDocFromFile(cfg.GenesisFile())
	})

	node, err := node.NewNode(
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genesisDocProvider,
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
	)
	if err != nil {
		return fmt.Errorf("CometBFT 노드 생성 실패: %w", err)
	}

	if err := node.Start(); err != nil {
		return fmt.Errorf("CometBFT 노드 시작 실패: %w", err)
	}

	server.StartServer(node)

	node.Wait()
	return nil
}
