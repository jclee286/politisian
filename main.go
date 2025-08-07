package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/types"

	"politician/app"
	"politician/server"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("사용자 홈 디렉토리 찾기 실패: %v", err)
	}
	politicianDir := filepath.Join(homeDir, "politician")
	cometbftDir := filepath.Join(politicianDir, ".cometbft")
	
	config := cfg.DefaultConfig()
	config.SetRoot(cometbftDir)

	config.RPC.ListenAddress = "tcp://0.0.0.0:26657"

	abciApp := app.NewPoliticianApp()

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		log.Fatalf("노드 키 로드/생성 실패: %v", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	pv := privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	pubKey, err := pv.GetPubKey()
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}
	
	genesisProvider := func() (*types.GenesisDoc, error) {
		genDoc, err := types.GenesisDocFromFile(config.GenesisFile())
		if err != nil {
			return nil, fmt.Errorf("failed to read genesis file: %w", err)
		}
		if len(genDoc.Validators) == 0 {
			genDoc.Validators = []types.GenesisValidator{{
				Address: pubKey.Address(),
				PubKey:  pubKey,
				Power:   10,
				Name:    "genesis-validator",
			}}
		}
		return genDoc, nil
	}

	cometNode, err := node.NewNode(config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(abciApp),
		genesisProvider,
		cfg.DefaultDBProvider,
		node.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
	if err != nil {
		log.Fatalf("CometBFT 노드 생성 실패: %v", err)
	}

	if err := cometNode.Start(); err != nil {
		log.Fatalf("CometBFT 노드 시작 실패: %v", err)
	}
	defer func() {
		if cometNode.IsRunning() {
			if err := cometNode.Stop(); err != nil {
				log.Printf("CometBFT 노드 종료 오류: %v", err)
			}
		}
	}()
	
	rpcAddr := config.RPC.ListenAddress
	parts := strings.SplitN(rpcAddr, "://", 2)
	if len(parts) != 2 {
		log.Fatalf("잘못된 RPC 주소 형식: %s", rpcAddr)
	}
	networkAddr := parts[1]

	log.Printf("CometBFT RPC 서버(%s)가 준비될 때까지 대기 중...", networkAddr)
	for {
		conn, err := net.DialTimeout("tcp", networkAddr, 1*time.Second)
		if err == nil && conn != nil {
			conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("CometBFT RPC 서버가 성공적으로 준비되었습니다. HTTP 서버를 시작합니다.")

	// ### 수정된 부분 ###
	// RPC 준비가 완료된 후, HTTP 서버를 별도의 고루틴에서 시작합니다.
	go server.StartServer(cometNode)

	// --- 종료 신호 대기 ---
	fmt.Println("통합 서버가 성공적으로 시작되었습니다. 종료하려면 Ctrl+C를 누르세요.")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("종료 신호 수신, 서버를 종료합니다...")
}
