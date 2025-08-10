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

	// "github.com/cometbft/cometbft/abci/types" // app.go 에서 사용하므로 main에서는 직접 필요 없음
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
	// 로거를 먼저 생성하여 전체 실행 과정의 로그를 기록합니다.
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	if err := run(logger); err != nil {
		logger.Error("Failed to run application", "error", err)
		os.Exit(1)
	}
}

func run(logger log.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	politisianDir := filepath.Join(homeDir, "politisian")
	cometbftDir := filepath.Join(politisianDir, ".cometbft")
	logger.Info("Base directory paths configured", "politisianDir", politisianDir, "cometbftDir", cometbftDir)

	cfg := config.DefaultConfig()
	cfg.SetRoot(cometbftDir)
	cfg.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	// 빠른 동기화를 위해 fast_sync 버전을 v0으로 설정
	cfg.Consensus.TimeoutCommit = 5 * time.Second

	config.EnsureRoot(cometbftDir)
	configFilePath := filepath.Join(cfg.RootDir, "config", "config.toml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		logger.Info("config.toml not found, creating a new one", "path", configFilePath)
		config.WriteConfigFile(configFilePath, cfg)
	} else {
		logger.Info("Using existing config.toml", "path", configFilePath)
	}

	logger.Info("Loading or generating private validator and node key", "pv_key_path", cfg.PrivValidatorKeyFile(), "node_key_path", cfg.NodeKeyFile())
	pv := privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or gen node key: %w", err)
	}

	genesisFile := cfg.GenesisFile()
	if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
		logger.Info("genesis.json not found, creating a new one", "path", genesisFile)
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
	} else {
		logger.Info("Using existing genesis.json", "path", genesisFile)
	}

	dbPath := filepath.Join(cometbftDir, "data")
	logger.Info("Opening application database", "backend", "goleveldb", "path", dbPath)
	db, err := dbm.NewDB("politisian_app", dbm.GoLevelDBBackend, dbPath)
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}
	abciApp := app.NewPoliticianApp(db, logger.With("module", "abci-app"))

	genesisDocProvider := node.DefaultGenesisDocProviderFunc(cfg)
	dbProvider := config.DefaultDBProvider

	logger.Info("Creating CometBft node...")
	cometNode, err := node.NewNode(
		context.Background(),
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(abciApp),
		genesisDocProvider,
		dbProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger.With("module", "cometbft"),
	)
	if err != nil {
		return fmt.Errorf("failed to create CometBFT node: %w", err)
	}

	logger.Info("Starting CometBFT node...")
	if err := cometNode.Start(); err != nil {
		return fmt.Errorf("failed to start CometBFT node: %w", err)
	}

	// 웹 서버 시작
	logger.Info("Starting web server...")
	go server.StartServer(cometNode)

	// 노드가 중지될 때까지 기다림
	logger.Info("Node started. Waiting for interrupt signal to shut down.")
	cometNode.Wait()

	return nil
}
