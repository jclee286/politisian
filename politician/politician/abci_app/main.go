package main

import (
	"encoding/json"
	"os"

	"github.com/cometbft/cometbft/abci/server"
	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
)

// PoliticianApp은 우리 애플리케이션의 상태를 관리합니다.
type PoliticianApp struct {
	types.BaseApplication
	logger   log.Logger
	state    *AppState
}

// AppState는 애플리케이션의 모든 상태를 포함합니다.
type AppState struct {
	Accounts map[string]uint64         `json:"accounts"`
	Votes    map[string]map[string]bool `json:"votes"`
}

func NewPoliticianApp(logger log.Logger) *PoliticianApp {
	return &PoliticianApp{
		logger: logger,
		state: &AppState{
			Accounts: make(map[string]uint64),
			Votes:    make(map[string]map[string]bool),
		},
	}
}

// DeliverTx는 가장 중요한 메소드로, 실제 트랜잭션을 처리합니다.
func (app *PoliticianApp) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	var tx struct {
		Action      string   `json:"action"`
		User        string   `json:"user"`
		Politicians []string `json:"politicians,omitempty"`
		ProposalID  string   `json:"proposal_id,omitempty"`
		Choice      string   `json:"choice,omitempty"`
	}

	if err := json.Unmarshal(req.Tx, &tx); err != nil {
		return types.ResponseDeliverTx{Code: 1, Log: "Invalid TX format"}
	}

	switch tx.Action {
	case "signup":
		code, log := app.handleSignup(tx.User, tx.Politicians)
		return types.ResponseDeliverTx{Code: code, Log: log}
	case "vote":
		code, log := app.handleVote(tx.User, tx.ProposalID, tx.Choice)
		return types.ResponseDeliverTx{Code: code, Log: log}
	default:
		return types.ResponseDeliverTx{Code: 2, Log: "Unknown action"}
	}
}

// Commit은 블록의 모든 트랜잭션 처리가 끝난 후 호출됩니다.
func (app *PoliticianApp) Commit() types.ResponseCommit {
	return types.ResponseCommit{}
}

// --- 비즈니스 로직 핸들러 ---

func (app *PoliticianApp) handleSignup(user string, politicians []string) (uint32, string) {
	if _, exists := app.state.Accounts[user]; exists {
		msg := "User already exists"
		app.logger.Error(msg, "user", user)
		return 3, msg
	}
	// 초기 가입 보상은 3명의 정치인을 선택했을 때 300 코인입니다.
	if len(politicians) == 3 {
		app.state.Accounts[user] = 300
		app.logger.Info("Signup reward", "user", user, "balance", app.state.Accounts[user])
		return 0, "Signup successful"
	} else {
		msg := "Signup failed: must select 3 politicians"
		app.logger.Error(msg, "user", user, "count", len(politicians))
		return 4, msg
	}
}

func (app *PoliticianApp) handleVote(voter, proposalID, choice string) (uint32, string) {
	if _, exists := app.state.Votes[proposalID]; !exists {
		app.state.Votes[proposalID] = make(map[string]bool)
	}
	app.state.Votes[proposalID][voter] = (choice == "yes")
	app.logger.Info("Vote recorded", "proposal", proposalID, "voter", voter, "choice", choice)
	return 0, "Vote successful"
}

func main() {
	logFile, err := os.OpenFile("/home/jclee/politician/abci.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}
	logger := log.NewTMLogger(log.NewSyncWriter(logFile))

	app := NewPoliticianApp(logger)

	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		logger.Error("Error creating ABCI server", "error", err)
		os.Exit(1)
	}
	srv.SetLogger(logger)

	if err := srv.Start(); err != nil {
		logger.Error("Error starting ABCI server", "error", err)
		os.Exit(1)
	}
	defer srv.Stop()

	// 서버가 종료될 때까지 대기
	select {}
} 