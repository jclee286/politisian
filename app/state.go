package app

import (
	"encoding/json"
	"log"
	"os"

	"politician/pkg/types"
)

const (
	stateFilePath = "app_state.json"
)

// AppState는 애플리케이션의 전체 상태를 나타냅니다.
type AppState struct {
	Accounts        map[string]types.Account    `json:"accounts"`
	Politicians     map[string]types.Politician `json:"politicians"`
	Proposals       map[string]types.Proposal   `json:"proposals"`
	AppHash         []byte                      `json:"appHash"`
	LastBlockHeight int64                       `json:"lastBlockHeight"`
}

// saveState는 현재 애플리케이션 상태를 stateFilePath에 JSON 형식으로 저장합니다.
func (app *PoliticianApp) saveState() error {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	state := AppState{
		Accounts:        app.accounts,
		Politicians:     app.politicians,
		Proposals:       app.proposals,
		AppHash:         app.appHash,
		LastBlockHeight: app.lastBlockHeight,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFilePath, data, 0644)
}

// loadState는 stateFilePath에서 JSON 형식의 상태를 불러와 애플리케이션에 적용합니다.
func (app *PoliticianApp) loadState() error {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return err
	}
	var state AppState
	if err = json.Unmarshal(data, &state); err != nil {
		return err
	}
	app.mtx.Lock()
	defer app.mtx.Unlock()
	app.accounts = state.Accounts
	app.politicians = state.Politicians
	app.proposals = state.Proposals
	app.appHash = state.AppHash
	app.lastBlockHeight = state.LastBlockHeight

	// wallets 맵 재생성
	app.wallets = make(map[string]string)
	for email, account := range app.accounts {
		if account.Wallet != "" {
			app.wallets[account.Wallet] = email
		}
	}

	log.Println("저장된 애플리케이션 상태를 성공적으로 불러왔습니다.")
	return nil
}
