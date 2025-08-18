package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/google/uuid"
	ptypes "politisian/pkg/types"
)

// Info is called by CometBFT to query the application's last state.
func (app *PoliticianApp) Info(_ context.Context, req *types.RequestInfo) (*types.ResponseInfo, error) {
	app.logger.Info("Received Info request", "last_height", app.height, "last_app_hash", fmt.Sprintf("%X", app.appHash))
	return &types.ResponseInfo{
		LastBlockHeight:  app.height,
		LastBlockAppHash: app.appHash,
	}, nil
}

// Query queries the application state.
func (app *PoliticianApp) Query(_ context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	app.logger.Info("Received Query", "path", req.Path, "data", string(req.Data))
	switch req.Path {
	case "/politisian/list":
		res, err := json.Marshal(app.politicians)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politicians list"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil
	case "/proposals/list":
		res, err := json.Marshal(app.proposals)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal proposals list"}, nil
		}
		return &types.ResponseQuery{Value: res}, nil
	default:
		// Handle account queries with pattern /account?address=...
		if len(req.Path) >= 8 && req.Path[:8] == "/account" {
			// Extract address from query string
			address := ""
			if len(req.Path) > 16 && req.Path[8:16] == "?address" {
				address = req.Path[17:] // Skip "?address="
			}
			
			if address == "" {
				return &types.ResponseQuery{Code: 2, Log: "address parameter required"}, nil
			}
			
			account, exists := app.accounts[address]
			if !exists {
				return &types.ResponseQuery{Code: 3, Log: "account not found"}, nil
			}
			
			res, err := json.Marshal(account)
			if err != nil {
				return &types.ResponseQuery{Code: 4, Log: "failed to marshal account"}, nil
			}
			return &types.ResponseQuery{Value: res}, nil
		}
		return &types.ResponseQuery{Code: 1, Log: "unknown query path"}, nil
	}
}

// CheckTx validates a transaction for the mempool.
func (app *PoliticianApp) CheckTx(_ context.Context, req *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	app.logger.Debug("Received CheckTx", "tx", string(req.Tx))
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

// Commit saves the new state to the database.
func (app *PoliticianApp) Commit(_ context.Context, _ *types.RequestCommit) (*types.ResponseCommit, error) {
	app.height++
	if err := app.saveState(); err != nil {
		app.logger.Error("Failed to save state on Commit", "error", err)
		panic(err)
	}
	app.logger.Info("Committed state", "height", app.height, "appHash", fmt.Sprintf("%X", app.appHash))
	return &types.ResponseCommit{}, nil
}

// InitChain is called once upon genesis.
func (app *PoliticianApp) InitChain(_ context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
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

// FinalizeBlock executes all transactions in a block and returns the new app hash.
func (app *PoliticianApp) FinalizeBlock(_ context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
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
		case "claim_referral_reward":
			respTxs[i] = app.handleClaimReferralReward(&txData)
		default:
			logMsg := "Unknown action"
			app.logger.Error(logMsg, "action", txData.Action)
			respTxs[i] = &types.ExecTxResult{Code: 10, Log: logMsg}
		}
	}

	app.hashState() // Update app hash after all transactions
	app.logger.Debug("Finalized block state", "appHash", fmt.Sprintf("%X", app.appHash))

	return &types.ResponseFinalizeBlock{
		TxResults: respTxs,
		AppHash:   app.appHash,
	}, nil
}

// --- Handler Functions ---
func (app *PoliticianApp) handleCreateProfile(txData *ptypes.TxData) *types.ExecTxResult {
	if _, exists := app.accounts[txData.UserID]; exists {
		logMsg := "User ID already exists"
		app.logger.Info(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 2, Log: logMsg}
	}
	
	// 새 계정 생성
	newAccount := &ptypes.Account{
		Address:          txData.UserID,
		Email:            txData.Email,
		Wallet:           txData.WalletAddress,  // PIN 기반 지갑 주소
		Politicians:      txData.Politicians,
		ReferralCredits:  0, // 초기 크레딧은 0
		PoliticianCoins:  make(map[string]int64),  // 정치인별 코인 보유량
		ReceivedCoins:    make(map[string]bool),   // 정치인별 코인 수령 여부
		InitialSelection: false,                   // 초기 선택 아직 완료 안됨
	}
	
	// 추천인이 있는 경우 추천인에게 크레딧 지급
	if txData.Referrer != "" && txData.Referrer != txData.UserID {
		app.logger.Info("Processing referral", "new_user", txData.UserID, "referrer", txData.Referrer)
		
		// 추천인 계정 찾기
		if referrerAccount, exists := app.accounts[txData.Referrer]; exists {
			referrerAccount.ReferralCredits++
			app.logger.Info("Referral credit granted", "referrer", txData.Referrer, "new_credits", referrerAccount.ReferralCredits)
		} else {
			app.logger.Info("Referrer account not found", "referrer", txData.Referrer)
		}
	}
	
	app.accounts[txData.UserID] = newAccount
	app.logger.Info("Created profile", "user_id", txData.UserID, "wallet_address", txData.WalletAddress, "referrer", txData.Referrer)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		logMsg := "Account not found for update"
		app.logger.Info(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 30, Log: logMsg}
	}
	
	// 초기 선택인지 확인 (처음 3명 선택)
	if !account.InitialSelection && len(txData.Politicians) <= 3 {
		// 초기 3명 선택 시 각각 100개씩 코인 지급
		totalCoinsGiven := int64(0)
		
		for _, politicianName := range txData.Politicians {
			// 이미 받은 코인인지 확인
			if !account.ReceivedCoins[politicianName] {
				// 정치인이 존재하고 코인이 충분한지 확인
				if politician, exists := app.politicians[politicianName]; exists {
					if politician.RemainingCoins >= 100 {
						// 코인 지급
						account.PoliticianCoins[politicianName] += 100
						account.ReceivedCoins[politicianName] = true
						
						// 정치인의 남은 코인 수량 감소
						politician.RemainingCoins -= 100
						politician.DistributedCoins += 100
						
						totalCoinsGiven += 100
						
						app.logger.Info("Initial coin distribution", 
							"user", txData.UserID, 
							"politician", politicianName,
							"coins_given", 100,
							"politician_remaining", politician.RemainingCoins)
					}
				}
			}
		}
		
		account.InitialSelection = true
		app.logger.Info("Initial selection completed", 
			"user", txData.UserID, 
			"total_coins_given", totalCoinsGiven)
	}
	
	account.Politicians = txData.Politicians
	app.logger.Info("Updated supporters", "user_id", txData.UserID, "politician_count", len(txData.Politicians))
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) proposePolitician(txData *ptypes.TxData) *types.ExecTxResult {
	proposalID := uuid.New().String()
	app.proposals[proposalID] = &ptypes.Proposal{
		ID: proposalID, Politician: ptypes.Politician{
			Name: txData.PoliticianName, Region: txData.Region, Party: txData.Party, IntroUrl: txData.IntroUrl,
		}, Proposer: txData.UserID, Votes: make(map[string]bool),
	}
	app.logger.Info("Proposed new politician", "proposer", txData.UserID, "politician_name", txData.PoliticianName, "proposal_id", proposalID)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PoliticianApp) handleVoteOnProposal(txData *ptypes.TxData) *types.ExecTxResult {
	proposal, exists := app.proposals[txData.ProposalID]
	if !exists {
		logMsg := "Proposal not found for vote"
		app.logger.Info(logMsg, "proposal_id", txData.ProposalID)
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

	if proposal.YesVotes >= 1 {
		newPolitician := &proposal.Politician
		
		// 정치인 등록 시 1,000만개 코인 발행
		newPolitician.TotalCoinSupply = 10_000_000    // 1,000만개
		newPolitician.RemainingCoins = 10_000_000     // 초기에는 모두 남아있음
		newPolitician.DistributedCoins = 0            // 아직 배포 안됨
		
		app.politicians[newPolitician.Name] = newPolitician
		delete(app.proposals, txData.ProposalID)
		app.logger.Info("Politician approved with coin issuance", 
			"proposal_id", txData.ProposalID, 
			"politician_name", newPolitician.Name,
			"coin_supply", newPolitician.TotalCoinSupply)
	}
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// handleClaimReferralReward는 추천 크레딧을 사용하여 새 정치인의 코인 100개를 지급합니다.
func (app *PoliticianApp) handleClaimReferralReward(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		logMsg := "Account not found for referral reward claim"
		app.logger.Info(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 50, Log: logMsg}
	}
	
	// 사용 가능한 크레딧이 있는지 확인
	if account.ReferralCredits <= 0 {
		logMsg := "No referral credits available"
		app.logger.Info(logMsg, "user_id", txData.UserID, "credits", account.ReferralCredits)
		return &types.ExecTxResult{Code: 51, Log: logMsg}
	}
	
	// 선택한 정치인이 존재하는지 확인
	if txData.PoliticianName == "" {
		logMsg := "No politician specified for referral reward"
		app.logger.Info(logMsg, "user_id", txData.UserID)
		return &types.ExecTxResult{Code: 52, Log: logMsg}
	}
	
	// 이미 받은 정치인인지 확인
	if account.ReceivedCoins[txData.PoliticianName] {
		logMsg := "Already received coins from this politician"
		app.logger.Info(logMsg, "user_id", txData.UserID, "politician", txData.PoliticianName)
		return &types.ExecTxResult{Code: 53, Log: logMsg}
	}
	
	// 정치인이 존재하고 코인이 충분한지 확인
	politician, exists := app.politicians[txData.PoliticianName]
	if !exists {
		logMsg := "Politician not found"
		app.logger.Info(logMsg, "politician", txData.PoliticianName)
		return &types.ExecTxResult{Code: 54, Log: logMsg}
	}
	
	if politician.RemainingCoins < 100 {
		logMsg := "Not enough coins available from politician"
		app.logger.Info(logMsg, "politician", txData.PoliticianName, "remaining", politician.RemainingCoins)
		return &types.ExecTxResult{Code: 55, Log: logMsg}
	}
	
	// 크레딧 1개 차감
	account.ReferralCredits--
	
	// 코인 지급
	account.PoliticianCoins[txData.PoliticianName] += 100
	account.ReceivedCoins[txData.PoliticianName] = true
	
	// 정치인의 남은 코인 수량 감소
	politician.RemainingCoins -= 100
	politician.DistributedCoins += 100
	
	app.logger.Info("Referral reward coin distributed", 
		"user", txData.UserID, 
		"politician", txData.PoliticianName,
		"coins_given", 100,
		"remaining_credits", account.ReferralCredits,
		"politician_remaining", politician.RemainingCoins)
	
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// --- Required ABCI++ Methods (basic implementation) ---
func (app *PoliticianApp) PrepareProposal(_ context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil
}
func (app *PoliticianApp) ProcessProposal(_ context.Context, req *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}
func (app *PoliticianApp) ExtendVote(_ context.Context, req *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return &types.ResponseExtendVote{}, nil
}
func (app *PoliticianApp) VerifyVoteExtension(_ context.Context, req *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return &types.ResponseVerifyVoteExtension{Status: types.ResponseVerifyVoteExtension_ACCEPT}, nil
}
func (app *PoliticianApp) ListSnapshots(_ context.Context, req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return &types.ResponseListSnapshots{}, nil
}
func (app *PoliticianApp) OfferSnapshot(_ context.Context, req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return &types.ResponseOfferSnapshot{Result: types.ResponseOfferSnapshot_ABORT}, nil
}
func (app *PoliticianApp) LoadSnapshotChunk(_ context.Context, req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return &types.ResponseLoadSnapshotChunk{}, nil
}
func (app *PoliticianApp) ApplySnapshotChunk(_ context.Context, req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return &types.ResponseApplySnapshotChunk{Result: types.ResponseApplySnapshotChunk_ACCEPT}, nil
}
