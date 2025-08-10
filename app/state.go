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
func (app *PoliticianApp) saveState() error {
	state := AppState{
		Height:      app.height,
		AppHash:     app.appHash,
		Accounts:    app.accounts,
		Proposals:   app.proposals,
		Politicians: app.politicians,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return app.db.SetSync(stateKey, stateBytes)
}

// loadState는 데이터베이스에서 애플리케이션 상태를 불러옵니다.
func (app *PoliticianApp) loadState() error {
	stateBytes, err := app.db.Get(stateKey)
	if err != nil {
		if err == dbm.ErrKeyNotFound { // DB에 아직 상태가 없으면 초기 상태로 시작
			return nil
		}
		return err
	}
	if len(stateBytes) == 0 {
		return nil // 데이터가 비어있어도 초기 상태
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

// hashState는 현재 상태의 해시를 계산하여 app.appHash를 업데이트합니다.
func (app *PoliticianApp) hashState() {
	state := AppState{
		Height:      app.height,
		Accounts:    app.accounts,
		Proposals:   app.proposals,
		Politicians: app.politicians,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		// 이 에러는 발생해서는 안됩니다. 발생 시 시스템적인 문제입니다.
		panic(err)
	}
	hash := sha256.Sum256(stateBytes)
	app.appHash = hash[:]
}
