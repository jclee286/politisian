package app

import (
	"encoding/json"
	"log"
	"os"
)

const (
	stateFilePath = "app_state.json"
)

// AppState는 애플리케이션의 전체 상태를 나타냅니다.
type AppState struct {
	Accounts        map[string]Account    `json:"accounts"`
	Politicians     map[string]Politician `json:"politicians"`
	AppHash         []byte                `json:"appHash"`
	LastBlockHeight int64                 `json:"lastBlockHeight"`
}

// Account는 사용자의 모든 프로필 정보를 저장합니다.
type Account struct {
	Email       string   `json:"email"`
	Nickname    string   `json:"nickname"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country"`
	Gender      string   `json:"gender"`
	BirthYear   int      `json:"birthYear"`
	Politicians []string `json:"politicians"`
	Balance     int64    `json:"balance"`
}

// Politician은 정치인의 이름과 남은 토큰 발행량을 저장합니다.
type Politician struct {
	Name         string `json:"name"`
	TokensMinted int64  `json:"tokensMinted"`
	MaxTokens    int64  `json:"maxTokens"`
}

// saveState는 현재 애플리케이션 상태를 stateFilePath에 JSON 형식으로 저장합니다.
func (app *PoliticianApp) saveState() error {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	state := AppState{
		Accounts:        app.accounts,
		Politicians:     app.politicians,
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
	app.appHash = state.AppHash
	app.lastBlockHeight = state.LastBlockHeight
	log.Println("저장된 애플리케이션 상태를 성공적으로 불러왔습니다.")
	return nil
}
