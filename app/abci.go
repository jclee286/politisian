package app

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/google/uuid"
	ptypes "politisian/pkg/types"
)

// Info는 CometBFT가 노드 시작/재시작 시 앱의 마지막 상태를 질의하기 위해 호출합니다.
func (app *PoliticianApp) Info(req *types.RequestInfo) (*types.ResponseInfo, error) {
	return &types.ResponseInfo{
		LastBlockHeight: app.height,
		LastBlockAppHash: app.appHash,
	}, nil
}

// Query는 애플리케이션의 상태를 조회합니다.
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

// CheckTx는 트랜잭션이 유효한지 기본적인 검사를 수행합니다.
func (app *PoliticianApp) CheckTx(req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

// Commit은 블록의 모든 트랜잭션이 처리된 후, 최종 상태를 DB에 저장하고 AppHash를 반환합니다.
func (app *PoliticianApp) Commit() (*types.ResponseCommit, error) {
	app.height++
	appHash, err := app.saveState()
	if err != nil {
		log.Printf("CRITICAL: Failed to save state: %v", err)
		// 여기서 패닉을 발생시켜 노드를 안전하게 중지시킬 수 있습니다.
		panic(err)
	}
	return &types.ResponseCommit{Data: appHash}, nil
}

// InitChain은 블록체인이 처음 시작될 때 한 번만 호출됩니다.
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

// FinalizeBlock은 블록에 포함된 모든 트랜잭션을 실행합니다.
func (app *PoliticianApp) FinalizeBlock(req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	respTxs := make([]*types.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		var txData ptypes.TxData
		if err := json.Unmarshal(tx, &txData); err != nil {
			respTxs[i] = &types.ExecTxResult{Code: 1, Log: "failed to parse transaction data"}
			continue
		}

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
	// `Commit`이 이어서 호출되어 변경사항을 DB에 최종 저장합니다.
	return &types.ResponseFinalizeBlock{TxResults: respTxs}, nil
}

// --- 핸들러 함수들 ---
func (app *PoliticianApp) handleCreateProfile(txData *ptypes.TxData) *types.ExecTxResult {
	if _, exists := app.accounts[txData.UserID]; exists {
		return &types.ExecTxResult{Code: 2, Log: "user ID already exists"}
	}
	app.accounts[txData.UserID] = &ptypes.Account{
		Address: txData.UserID, Email: txData.Email, Politicians: txData.Politicians,
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists { return &types.ExecTxResult{Code: 30, Log: "account not found"} }
	account.Politicians = txData.Politicians
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
	proposalID := uuid.New().String()
	app.proposals[proposalID] = &ptypes.Proposal{
		ID: proposalID, Politician: ptypes.Politician{
			Name: txData.PoliticianName, Region: txData.Region, Party: txData.Party,
		}, Proposer: txData.UserID, Votes: make(map[string]bool),
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) handleVoteOnProposal(txData *ptypes.TxData) *types.ExecTxResult {
	proposal, exists := app.proposals[txData.ProposalID]
	if !exists { return &types.ExecTxResult{Code: 40, Log: "proposal not found"} }
	if _, alreadyVoted := proposal.Votes[txData.UserID]; alreadyVoted {
		return &types.ExecTxResult{Code: 41, Log: "user has already voted"}
	}
	proposal.Votes[txData.UserID] = txData.Vote
	if txData.Vote { proposal.YesVotes++ } else { proposal.NoVotes++ }

	if proposal.YesVotes >= 10 {
		newPolitician := &proposal.Politician
		app.politicians[newPolitician.Name] = newPolitician
		delete(app.proposals, txData.ProposalID)
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// --- ABCI++ 필수 메서드 (기본 구현) ---
func (app *PoliticianApp) PrepareProposal(req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil
}
func (app *PoliticianApp) ProcessProposal(req *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}
func (app *PoliticianApp) ExtendVote(req *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return &types.ResponseExtendVote{}, nil
}
func (app *PoliticianApp) VerifyVoteExtension(req *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return &types.ResponseVerifyVoteExtension{Status: types.ResponseVerifyVoteExtension_ACCEPT}, nil
}
func (app *PoliticianApp) ListSnapshots(req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return &types.ResponseListSnapshots{}, nil
}
func (app *PoliticianApp) OfferSnapshot(req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return &types.ResponseOfferSnapshot{Result: types.ResponseOfferSnapshot_ABORT}, nil
}
func (app *PoliticianApp) LoadSnapshotChunk(req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return &types.ResponseLoadSnapshotChunk{}, nil
}
func (app *PoliticianApp) ApplySnapshotChunk(req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return &types.ResponseApplySnapshotChunk{Result: types.ResponseApplySnapshotChunk_ACCEPT}, nil
}
