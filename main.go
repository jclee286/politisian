package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"politisian/app"
	ptypes "politisian/pkg/types"
	"politisian/server"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/config"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/types"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	politisianDir := filepath.Join(homeDir, "politisian")
	cometbftDir := filepath.Join(politisianDir, ".cometbft")

	cfg := config.DefaultConfig()
	cfg.SetRoot(cometbftDir)
	cfg.RPC.ListenAddress = "tcp://0.0.0.0:26657"

	config.EnsureRoot(cometbftDir)
	configFilePath := filepath.Join(cfg.RootDir, "config", "config.toml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(configFilePath, cfg)
	}

	pv := privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or gen node key: %w", err)
	}

	genesisFile := cfg.GenesisFile()
	if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("failed to get pubkey from priv validator: %w", err)
		}
		appState, err := json.Marshal(ptypes.GenesisState{})
		if err != nil {
			return fmt.Errorf("failed to marshal genesis app state: %w", err)
		}
		genDoc := &types.GenesisDoc{
			ChainID:         "politisian-chain-1",
			GenesisTime:     time.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
			Validators: []types.GenesisValidator{{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
			}},
			AppState: json.RawMessage(appState),
		}
		if err := genDoc.SaveAs(genesisFile); err != nil {
			return fmt.Errorf("failed to save genesis file: %w", err)
		}
	}

	dbPath := filepath.Join(cometbftDir, "data")
	db, err := dbm.NewDB("politisian_app", dbm.GoLevelDBBackend, dbPath)
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}
	abciApp := app.NewPoliticianApp(db)

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	genesisDocProvider := node.DefaultGenesisDocProviderFunc(cfg)
	dbProvider := node.DefaultDBProvider

	cometNode, err := node.NewNode(
		context.Background(),
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(abciApp),
		genesisDocProvider,
		dbProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create CometBFT node: %w", err)
	}

	if err := cometNode.Start(); err != nil {
		return fmt.Errorf("failed to start CometBFT node: %w", err)
	}

	// 웹 서버 시작
	go server.StartServer(cometNode)

	// 노드가 중지될 때까지 기다림
	cometNode.Wait()

	return nil
}
