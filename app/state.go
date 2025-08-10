package app

import (
	"crypto/sha256"
	"encoding/json"

	dbm "github.com/cometbft/cometbft-db"
	ptypes "politisian/pkg/types"
)

var (
	stateKey = []byte("stateKey")
)

// AppState는 애플리케이션의 전체 상태를 나타내는 구조체입니다.
type AppState struct {
	Height      int64                        `json:"height"`
	AppHash     []byte                       `json:"app_hash"`
	Accounts    map[string]*ptypes.Account    `json:"accounts"`
	Proposals   map[string]*ptypes.Proposal   `json:"proposals"`
	Politicians map[string]*ptypes.Politician `json:"politicians"`
}

// saveState는 현재 애플리케이션 상태를 데이터베이스에 저장합니다.
func (app *PoliticianApp) saveState() ([]byte, error) {
	state := AppState{
		Height:      app.height,
		AppHash:     app.appHash,
		Accounts:    app.accounts,
		Proposals:   app.proposals,
		Politicians: app.politicians,
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	// 상태 해시 계산
	hash := sha256.Sum256(stateBytes)
	app.appHash = hash[:]

	// DB에 상태 저장
	err = app.db.SetSync(stateKey, stateBytes)
	if err != nil {
		return nil, err
	}
	return app.appHash, nil
}

// loadState는 데이터베이스에서 애플리케이션 상태를 불러옵니다.
func (app *PoliticianApp) loadState() error {
	stateBytes, err := app.db.Get(stateKey)
	if err != nil {
		// 키가 없는 경우(초기 상태)는 에러가 아님
		if err == dbm.ErrKeyNotFound {
			return nil
		}
		return err
	}
	// 데이터가 없는 경우 (초기 상태)
	if len(stateBytes) == 0 {
		return nil
	}

	var state AppState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return err
	}

	app.height = state.Height
	app.appHash = state.AppHash
	app.accounts = state.Accounts
	app.proposals = state.Proposals
	app.politicians = state.Politicians
	return nil
}
