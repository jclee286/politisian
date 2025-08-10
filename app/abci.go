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
	app.logger.Info("Received Info request", "last_height", app.height, "last_app_hash", fmt.Sprintf("%X", app.appHash))
	return &types.ResponseInfo{
		LastBlockHeight: app.height,
		LastBlockAppHash: app.appHash,
	}, nil
}

// Query는 애플리케이션의 상태를 조회합니다.
func (app *PoliticianApp) Query(req *types.RequestQuery) (*types.ResponseQuery, error) {
	app.logger.Info("Received Query", "path", req.Path, "data", string(req.Data))
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
	app.logger.Debug("Received CheckTx", "tx", string(req.Tx))
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

// Commit은 블록의 모든 트랜잭션이 처리된 후, 최종 상태를 DB에 저장합니다.
func (app *PoliticianApp) Commit() (*types.ResponseCommit, error) {
	app.height++
	if err := app.saveState(); err != nil {
		app.logger.Error("Failed to save state on Commit", "error", err)
		// 여기서 패닉을 발생시켜 노드를 안전하게 중지시킬 수 있습니다.
		panic(err)
	}
	app.logger.Info("Committed state", "height", app.height, "appHash", fmt.Sprintf("%X", app.appHash))
	return &types.ResponseCommit{}, nil
}

// InitChain은 블록체인이 처음 시작될 때 한 번만 호출됩니다.
func (app *PoliticianApp) InitChain(req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	app.logger.Info("Initializing chain from genesis", "chain_id", req.ChainId, "app_state_bytes", len(req.AppStateBytes))
	var genesisState ptypes.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		app.logger.Error("Failed to parse genesis app state", "error", err)
		return nil, fmt.Errorf("failed to parse genesis state: %w", err)
	}
	if genesisState.Politicians != nil {
		app.politicians = genesisState.Politicians
		app.logger.Info("Loaded politicians from genesis", "count", len(app.politicians))
	}
	if genesisState.Accounts != nil {
		app.accounts = genesisState.Accounts
		app.logger.Info("Loaded accounts from genesis", "count", len(app.accounts))
	}
	return &types.ResponseInitChain{}, nil
}

// FinalizeBlock은 블록에 포함된 모든 트랜잭션을 실행하고, 상태 해시를 계산하여 반환합니다.
func (app *PoliticianApp) FinalizeBlock(req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	app.logger.Info("Finalizing block", "height", req.Height, "num_txs", len(req.Txs))
	respTxs := make([]*types.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		var txData ptypes.TxData
		if err := json.Unmarshal(tx, &txData); err != nil {
			logMsg := "Failed to parse transaction data"
			app.logger.Error(logMsg, "tx_raw", string(tx), "error", err)
			respTxs[i] = &types.ExecTxResult{Code: 1, Log: logMsg}
			continue
		}

		app.logger.Info("Processing tx", "action", txData.Action, "user_id", txData.UserID)
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
			logMsg := "Unknown action"
			app.logger.Error(logMsg, "action", txData.Action)
			respTxs[i] = &types.ExecTxResult{Code: 10, Log: logMsg}
		}
	}

	app.hashState() // 모든 트랜잭션 처리 후 상태 해시 업데이트
	app.logger.Debug("Finalized block state", "appHash", fmt.Sprintf("%X", app.appHash))

	// `Commit`이 이어서 호출되어 변경사항을 DB에 최종 저장합니다.
	return &types.ResponseFinalizeBlock{
		TxResults: respTxs,
		AppHash:   app.appHash,
	}, nil
}

// --- 핸들러 함수들 ---
func (app *PoliticianApp) handleCreateProfile(txData *ptypes.TxData) *types.ExecTxResult {
	if _, exists := app.accounts[txData.UserID]; exists {
		logMsg := "User ID already exists"
		app.logger.Info(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 2, Log: logMsg}
	}
	app.accounts[txData.UserID] = &ptypes.Account{
		Address: txData.UserID, Email: txData.Email, Politicians: txData.Politicians,
	}
	app.logger.Info("Created profile", "user_id", txData.UserID)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		logMsg := "Account not found for update"
		app.logger.Warn(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 30, Log: logMsg}
	}
	account.Politicians = txData.Politicians
	app.logger.Info("Updated supporters", "user_id", txData.UserID, "politician_count", len(txData.Politicians))
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
	proposalID := uuid.New().String()
	app.proposals[proposalID] = &ptypes.Proposal{
		ID: proposalID, Politician: ptypes.Politician{
			Name: txData.PoliticianName, Region: txData.Region, Party: txData.Party,
		}, Proposer: txData.UserID, Votes: make(map[string]bool),
	}
	app.logger.Info("Proposed new politician", "proposer", txData.UserID, "politician_name", txData.PoliticianName, "proposal_id", proposalID)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) handleVoteOnProposal(txData *ptypes.TxData) *types.ExecTxResult {
	proposal, exists := app.proposals[txData.ProposalID]
	if !exists {
		logMsg := "Proposal not found for vote"
		app.logger.Warn(logMsg, "proposal_id", txData.ProposalID)
		return &types.ExecTxResult{Code: 40, Log: logMsg}
	}
	if _, alreadyVoted := proposal.Votes[txData.UserID]; alreadyVoted {
		logMsg := "User has already voted"
		app.logger.Info(logMsg, "user_id", txData.UserID, "proposal_id", txData.ProposalID)
		return &types.ExecTxResult{Code: 41, Log: logMsg}
	}
	proposal.Votes[txData.UserID] = txData.Vote
	if txData.Vote {
		proposal.YesVotes++
	} else {
		proposal.NoVotes++
	}
	app.logger.Info("Vote cast", "user_id", txData.UserID, "proposal_id", txData.ProposalID, "vote", txData.Vote, "yes_votes", proposal.YesVotes, "no_votes", proposal.NoVotes)

	if proposal.YesVotes >= 10 {
		newPolitician := &proposal.Politician
		app.politicians[newPolitician.Name] = newPolitician
		delete(app.proposals, txData.ProposalID)
		app.logger.Info("Proposal approved and politician added", "proposal_id", txData.ProposalID, "politician_name", newPolitician.Name)
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
