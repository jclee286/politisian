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
	
	// 가상의 정치인 데이터 초기화 (앱 시작 시 한번만)
	app.initializeDefaultPoliticians()
	
	return app
}

// initializeDefaultPoliticians는 가상의 정치인 데이터를 초기화합니다.
func (app *PoliticianApp) initializeDefaultPoliticians() {
	// 이미 정치인이 있다면 초기화하지 않음 (중복 방지)
	if len(app.politicians) > 0 {
		app.logger.Info("Politicians already exist, skipping initialization", "count", len(app.politicians))
		return
	}

	// 가상의 정치인 3명 데이터
	defaultPoliticians := []*ptypes.Politician{
		{
			Name:         "김민주",
			Region:       "서울특별시",
			Party:        "미래당",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "이정의",
			Region:       "부산광역시",
			Party:        "정의당",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "박희망",
			Region:       "인천광역시",
			Party:        "희망당",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
	}

	// 정치인들을 맵에 저장
	for _, politician := range defaultPoliticians {
		politicianID := politician.Name + "-" + politician.Region
		app.politicians[politicianID] = politician
		app.logger.Info("Initialized default politician", "name", politician.Name, "region", politician.Region, "party", politician.Party)
	}

	app.logger.Info("Default politicians initialization completed", "total_count", len(app.politicians))
}
