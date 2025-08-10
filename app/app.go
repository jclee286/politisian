package app

import (
	"github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	ptypes "politisian/pkg/types"
)

// PoliticianApp은 ABCI 애플리케이션의 상태를 저장합니다.
type PoliticianApp struct {
	types.BaseApplication
	logger      log.Logger
	db          dbm.DB
	height      int64
	appHash     []byte
	accounts    map[string]*ptypes.Account    // 사용자 계정 정보
	proposals   map[string]*ptypes.Proposal   // 제안 정보
	politicians map[string]*ptypes.Politician // 정치인 정보
}

func NewPoliticianApp(db dbm.DB, logger log.Logger) *PoliticianApp {
	app := &PoliticianApp{
		logger:      logger,
		db:          db,
		accounts:    make(map[string]*ptypes.Account),
		proposals:   make(map[string]*ptypes.Proposal),
		politicians: make(map[string]*ptypes.Politician),
	}
	// DB에서 마지막 상태를 불러옵니다.
	if err := app.loadState(); err != nil {
		// 로드 실패 시 애플리케이션을 중단해야 합니다.
		panic("Failed to load state: " + err.Error())
	}
	return app
}
