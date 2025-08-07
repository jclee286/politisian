package app

import (
	"log"
	"sync"

	ptypes "politician/pkg/types"
)

// PoliticianApp은 ABCI 애플리케이션의 핵심 구조체입니다.
// 애플리케이션의 상태(계정, 정치인 정보 등)를 관리합니다.
type PoliticianApp struct {
	mtx             sync.Mutex
	accounts        map[string]ptypes.Account
	wallets         map[string]string // wallet_address -> email
	politicians     map[string]ptypes.Politician
	appHash         []byte
	lastBlockHeight int64
}

// NewPoliticianApp은 새로운 PoliticianApp 인스턴스를 생성하고 초기화합니다.
// 저장된 상태가 있으면 불러오고, 없으면 기본 정치인 정보로 초기화합니다.
func NewPoliticianApp() *PoliticianApp {
	app := &PoliticianApp{
		accounts:        make(map[string]ptypes.Account),
		wallets:         make(map[string]string),
		politicians:     make(map[string]ptypes.Politician),
		appHash:         []byte{},
		lastBlockHeight: 0,
	}

	// 서버 시작 시, 저장된 상태를 불러옵니다.
	if err := app.loadState(); err != nil {
		log.Printf("저장된 상태를 찾을 수 없음 (초기 상태로 시작): %v", err)
		// 초기 정치인 설정
		initialPoliticians := map[string]ptypes.Politician{
			"이순신": {Name: "이순신", TokensMinted: 0, MaxTokens: 10000000},
			"김구":  {Name: "김구", TokensMinted: 0, MaxTokens: 10000000},
			"세종대왕": {Name: "세종대왕", TokensMinted: 0, MaxTokens: 10000000},
		}
		app.politicians = initialPoliticians
	}
	return app
}
