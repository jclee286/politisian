package app

import (
	"encoding/json"
	"os"

	"politisian/pkg/types"
)

// AppState는 애플리케이션의 전체 상태를 나타내는 구조체입니다. (디버깅 및 상태 저장용)
type AppState struct {
	Accounts    map[string]*types.Account    `json:"accounts"`
	Proposals   map[string]*types.Proposal   `json:"proposals"`
	Politicians map[string]*types.Politician `json:"politicians"`
}

// saveState는 현재 애플리케이션 상태를 파일에 저장합니다.
func (app *PoliticianApp) saveState() error {
	state := AppState{
		Accounts:    app.accounts,
		Proposals:   app.proposals,
		Politicians: app.politicians,
	}
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("app_state.json", stateBytes, 0644)
}

// loadState는 파일에서 애플리케이션 상태를 불러옵니다.
func (app *PoliticianApp) loadState() error {
	stateBytes, err := os.ReadFile("app_state.json")
	if err != nil {
		if os.IsNotExist(err) {
			// 파일이 없으면 초기 상태로 시작
			app.accounts = make(map[string]*types.Account)
			app.proposals = make(map[string]*types.Proposal)
			app.politicians = make(map[string]*types.Politician)
			return nil
		}
		return err
	}
	var state AppState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return err
	}
	app.accounts = state.Accounts
	app.proposals = state.Proposals
	app.politicians = state.Politicians
	return nil
}
