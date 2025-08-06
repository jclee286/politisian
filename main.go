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
	"github.com/cometbft/cometbft/rpc/client/local"

	"politician/app"
	"politician/server"
)

func main() {
	// --- 설정 및 초기화 ---
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("사용자 홈 디렉토리 찾기 실패: %v", err)
	}
	politicianDir := filepath.Join(homeDir, "politician")
	cometbftDir := filepath.Join(politicianDir, ".cometbft")
	config := cfg.DefaultConfig()
	config.SetRoot(cometbftDir)

	// --- ABCI 애플리케이션 생성 ---
	abciApp := app.NewPoliticianApp()

	// --- CometBFT 노드 생성 ---
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		log.Fatalf("노드 키 로드/생성 실패: %v", err)
	}

	logger := cmtlog.NewTMLogger(cmtlog.NewSyncWriter(os.Stdout))
	pv := privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())

	cometNode, err := node.NewNode(config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(abciApp),
		node.DefaultGenesisDocProviderFunc(config),
		cfg.DefaultDBProvider,
		node.DefaultMetricsProvider(config.Instrumentation),
		logger,
	)
	if err != nil {
		log.Fatalf("CometBFT 노드 생성 실패: %v", err)
	}

	// --- 노드 시작 및 RPC 준비 대기 ---
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

	// --- 노드 RPC 서버가 실제로 연결을 받을 때까지 대기 ---
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
			break // 연결 성공 시 루프 탈출
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("CometBFT RPC 서버가 성공적으로 준비되었습니다.")

	// --- RPC가 준비된 후 로컬 클라이언트 생성 ---
	localClient := local.New(cometNode)

	// --- HTTP 서버 시작 ---
	go server.StartHTTPServer(localClient, ":8080", politicianDir)

	// --- 종료 신호 대기 ---
	fmt.Println("통합 서버가 성공적으로 시작되었습니다. 종료하려면 Ctrl+C를 누르세요.")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("종료 신호 수신, 서버를 종료합니다...")
} 