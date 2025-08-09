package app

import (
	"github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cometbft/cometbft-db"
	ptypes "politisian/pkg/types"
)

// PolitisianApp은 ABCI 애플리케이션의 상태를 저장합니다.
type PolitisianApp struct {
	types.BaseApplication
	db         dbm.DB
	accounts   map[string]*ptypes.Account   // 사용자 계정 정보 (주소 -> 계정)
	proposals  map[string]*ptypes.Proposal  // 제안 정보 (제안 ID -> 제안)
	politisian map[string]*ptypes.Politisian // 정치인 정보 (이름 -> 정치인)
}

func NewPolitisianApp(db dbm.DB) *PolitisianApp {
	app := &PolitisianApp{
		db:         db,
		accounts:   make(map[string]*ptypes.Account),
		proposals:  make(map[string]*ptypes.Proposal),
		politisian: make(map[string]ptypes.Politisian),
	}
	// 앱 시작 시 상태 로드
	if err := app.loadState(); err != nil {
		// 로드 실패 시, 새로운 상태로 시작
		// 이 경우, 제네시스 블록에서 초기 상태를 설정하게 됩니다.
	}
	return app
}
