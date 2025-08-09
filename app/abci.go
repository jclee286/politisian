package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/google/uuid"
	ptypes "politisian/pkg/types"
)

// Info, Query, CheckTx, Commit, InitChain 등 다른 ABCI 함수들은 이전과 동일하게 유지합니다.

func (app *PoliticianApp) Info(req *types.RequestInfo) (*types.ResponseInfo, error) {
	return &types.ResponseInfo{}, nil
}

func (app *PoliticianApp) Query(req *types.RequestQuery) (*types.ResponseQuery, error) {
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

func (app *PoliticianApp) InitChain(req *types.RequestInitChain) (*types.ResponseInitChain, error) {
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


// FinalizeBlock은 ABCI++의 핵심 메서드로, 블록에 포함된 모든 트랜잭션을 처리합니다.
func (app *PoliticianApp) FinalizeBlock(ctx context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	respTxs := make([]*types.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		var txData ptypes.TxData
		if err := json.Unmarshal(tx, &txData); err != nil {
			log.Printf("트랜잭션 파싱 실패: %v", err)
			respTxs[i] = &types.ExecTxResult{Code: 1, Log: "failed to parse transaction data"}
			continue
		}

		// 각 액션에 맞는 핸들러 호출
		switch txData.Action {
		case "create_profile":
			respTxs[i] = app.handleCreateProfile(&txData)
		case "update_supporters":
			respTxs[i] = app.updateSupporters(&txData)
		case "propose_politician":
			respTxs[i] = app.proposePolitician(&txData)
		case "vote_on_proposal":
			respTxs[i] = app.handleVoteOnProposal(&txData)
		default:
			respTxs[i] = &types.ExecTxResult{Code: 10, Log: "unknown action"}
		}
	}

	return &types.ResponseFinalizeBlock{TxResults: respTxs}, nil
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
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		return &types.ExecTxResult{Code: 30, Log: "account not found"}
	}
	account.Politicians = txData.Politicians
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
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
	if proposal.YesVotes >= 10 { // 10표 이상 찬성 시 등록
		newPolitician := &proposal.Politician
		app.politicians[newPolitician.Name] = newPolitician
		delete(app.proposals, txData.ProposalID)
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// PrepareProposal과 ProcessProposal은 ABCI++의 일부로, 기본 구현을 제공합니다.
func (app *PoliticianApp) PrepareProposal(ctx context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil
}

func (app *PoliticianApp) ProcessProposal(ctx context.Context, req *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}
