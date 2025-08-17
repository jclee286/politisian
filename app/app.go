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

	// 실제 21대 국회의원 데이터 (주요 정치인들 샘플)
	defaultPoliticians := []*ptypes.Politician{
		{
			Name:         "이재명",
			Region:       "경기 계양구 갑",
			Party:        "더불어민주당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "한동훈",
			Region:       "비례대표",
			Party:        "국민의힘",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "조국",
			Region:       "서울 종로구",
			Party:        "조국혁신당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "안철수",
			Region:       "서울 관악구 을",
			Party:        "국민의당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "심상정",
			Region:       "비례대표",
			Party:        "정의당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "이낙연",
			Region:       "서울 종로구",
			Party:        "더불어민주당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "김기현",
			Region:       "울산 남구 을",
			Party:        "국민의힘",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "박홍근",
			Region:       "서울 중구·성동구 갑",
			Party:        "더불어민주당",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "추경호",
			Region:       "대구 수성구 갑",
			Party:        "국민의힘",
			IntroUrl:     "",
			Supporters:   []string{},
			TokensMinted: 0,
			MaxTokens:    1000000,
		},
		{
			Name:         "우상호",
			Region:       "서울 양천구 을",
			Party:        "더불어민주당",
			IntroUrl:     "",
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
