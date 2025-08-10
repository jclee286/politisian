package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	// 설정 디렉토리 및 기본 config.toml 파일 생성 (없을 경우)
	if err := config.EnsureRoot(cometbftDir); err != nil {
		return fmt.Errorf("failed to ensure root directory: %w", err)
	}
	configFilePath := filepath.Join(cometbftDir, "config", "config.toml")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		config.WriteConfigFile(configFilePath, cfg)
	}

	// private validator 와 node key 생성 (없을 경우)
	pv := privval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or gen node key: %w", err)
	}

	// genesis.json 파일 생성 (없을 경우)
	genesisFile := cfg.GenesisFile()
	if _, err := os.Stat(genesisFile); os.IsNotExist(err) {
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("failed to get pubkey from priv validator: %w", err)
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
		}
		if err := genDoc.SaveAs(genesisFile); err != nil {
			return fmt.Errorf("failed to save genesis file: %w", err)
		}
	}

	db, err := dbm.NewDB("politisian_app", dbm.GoLevelDBBackend, cometbftDir)
	if err != nil {
		return fmt.Errorf("데이터베이스 생성 실패: %w", err)
	}
	abciApp := app.NewPoliticianApp(db)

	return runNode(cfg, abciApp, pv, nodeKey)
}

func runNode(cfg *config.Config, app abci.Application, pv types.PrivValidator, nodeKey *p2p.NodeKey) error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	genesisDocProvider := node.GenesisDocProvider(func() (*types.GenesisDoc, error) {
		return types.GenesisDocFromFile(cfg.GenesisFile())
	})

	node, err := node.NewNode(
		cfg,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genesisDocProvider,
		config.DefaultDBProvider,
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
