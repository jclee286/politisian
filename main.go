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

	"politisian/app"
	"politisian/server"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("사용자 홈 디렉토리 찾기 실패: %v", err)
	}
	politisianDir := filepath.Join(homeDir, "politisian")
	cometbftDir := filepath.Join(politisianDir, ".cometbft")
	
	config := cfg.DefaultConfig()
	config.SetRoot(cometbftDir)

	config.RPC.ListenAddress = "tcp://0.0.0.0:26657"

	// ABCI 애플리케이션 인스턴스 생성
	db, err := dbm.NewDB("politisian_app", dbm.GoLevelDBBackend, cometbftDir)
	if err != nil {
		return fmt.Errorf("데이터베이스 생성 실패: %w", err)
	}
	abciApp := app.NewPolitisianApp(db)

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		log.Fatalf("노드 키 로드 실패: %v. 'cometbft init'을 실행했는지 확인하세요.", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	
	pv := privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())

	genesisProvider := func() (*types.GenesisDoc, error) {
		return types.GenesisDocFromFile(config.GenesisFile())
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

	go server.StartServer(cometNode)

	fmt.Println("통합 서버가 성공적으로 시작되었습니다. 종료하려면 Ctrl+C를 누르세요.")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("종료 신호 수신, 서버를 종료합니다...")
}
