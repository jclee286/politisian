package app

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/google/uuid"
	ptypes "politisian/pkg/types"
)

// Info, Query, CheckTx, Commit 함수는 이전과 동일하게 유지

func (app *PoliticianApp) Info(req *types.RequestInfo) (*types.ResponseInfo, error) {
	return &types.ResponseInfo{}, nil
}

func (app *PoliticianApp) Query(req *types.RequestQuery) (*types.ResponseQuery, error) {
	log.Printf("[ABCI] Query: 경로 '%s'에 대한 쿼리 수신", req.Path)
	switch req.Path {
	case "/politisian/list":
		res, err := json.Marshal(app.politicians)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politicians list"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil
	default:
		return &types.ResponseQuery{Code: 1, Log: "unknown query path"}, nil
	}
}

func (app *PoliticianApp) CheckTx(req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

func (app *PoliticianApp) Commit() (*types.ResponseCommit, error) {
	if err := app.saveState(); err != nil {
		log.Printf("심각한 오류: 상태 저장 실패: %v", err)
	}
	return &types.ResponseCommit{}, nil
}

// DeliverTx는 트랜잭션을 올바른 핸들러로 라우팅합니다.
func (app *PoliticianApp) DeliverTx(req *types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	var txData ptypes.TxData
	if err := json.Unmarshal(req.Tx, &txData); err != nil {
		return &types.ResponseDeliverTx{Code: 1, Log: "failed to parse transaction data"}, nil
	}

	log.Printf("[ABCI] DeliverTx: 사용자 '%s'로부터 액션 '%s' 처리 중", txData.UserID, txData.Action)

	switch txData.Action {
	case "create_profile":
		return app.handleCreateProfile(&txData), nil
	case "update_supporters":
		return app.updateSupporters(&txData), nil
	case "propose_politician":
		return app.proposePolitician(&txData), nil
	case "vote_on_proposal":
		return app.handleVoteOnProposal(&txData), nil
	default:
		return &types.ResponseDeliverTx{Code: 10, Log: "unknown action"}, nil
	}
}

// --- 트랜잭션 핸들러 함수들 ---

func (app *PoliticianApp) handleCreateProfile(txData *ptypes.TxData) *types.ExecTxResult {
	if _, exists := app.accounts[txData.UserID]; exists {
		return &types.ExecTxResult{Code: 2, Log: "user ID already exists"}
	}
	newAccount := &ptypes.Account{
		Address:     txData.UserID,
		Email:       txData.Email,
		Politicians: txData.Politicians,
	}
	app.accounts[txData.UserID] = newAccount
	log.Printf("[ABCI] 새로운 계정 생성: %s", txData.UserID)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		return &types.ExecTxResult{Code: 30, Log: "account not found"}
	}
	account.Politicians = txData.Politicians
	log.Printf("[ABCI] 사용자 '%s'의 지지 정치인 목록 업데이트: %v", txData.UserID, txData.Politicians)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
	// 제안(Proposal) 로직을 다시 구현합니다.
	proposalID := uuid.New().String()
	newProposal := &ptypes.Proposal{
		ID: proposalID,
		Politician: ptypes.Politician{
			Name:   txData.PoliticianName,
			Region: txData.Region,
			Party:  txData.Party,
		},
		Proposer: txData.UserID,
		Votes:    make(map[string]bool),
	}
	app.proposals[proposalID] = newProposal
	log.Printf("[ABCI] 새로운 정치인 제안: %s (제안자: %s)", txData.PoliticianName, txData.UserID)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) handleVoteOnProposal(txData *ptypes.TxData) *types.ExecTxResult {
	proposal, exists := app.proposals[txData.ProposalID]
	if !exists {
		return &types.ExecTxResult{Code: 40, Log: "proposal not found"}
	}
	if _, alreadyVoted := proposal.Votes[txData.UserID]; alreadyVoted {
		return &types.ExecTxResult{Code: 41, Log: "user has already voted"}
	}
	proposal.Votes[txData.UserID] = txData.Vote
	if txData.Vote {
		proposal.YesVotes++
	} else {
		proposal.NoVotes++
	}

	// 간단한 예시: 10표 이상 찬성 시 정치인으로 등록
	if proposal.YesVotes >= 10 {
		log.Printf("[ABCI] 제안 통과! 정치인 '%s' 등록", proposal.Politician.Name)
		newPolitician := &proposal.Politician
		app.politicians[newPolitician.Name] = newPolitician
		delete(app.proposals, txData.ProposalID)
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// InitChain는 이전과 동일
func (app *PoliticianApp) InitChain(req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	log.Println("[ABCI] InitChain: 체인 초기화 시작...")
	var genesisState ptypes.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, fmt.Errorf("failed to parse genesis state: %w", err)
	}
	if genesisState.Politicians != nil {
		app.politicians = genesisState.Politicians
	}
	if genesisState.Accounts != nil {
		app.accounts = genesisState.Accounts
	}
	return &types.ResponseInitChain{}, nil
}
