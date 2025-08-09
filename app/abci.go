package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	ptypes "politisian/pkg/types"
	"github.com/google/uuid"
)

func (app *PolitisianApp) Info(_ context.Context, info *types.RequestInfo) (*types.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	return &types.ResponseInfo{
		LastBlockHeight:  app.lastBlockHeight,
		LastBlockAppHash: app.appHash,
	}, nil
}

func (app *PolitisianApp) Query(_ context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	switch req.Path {
	case "/account/exists":
		email := string(req.Data)
		if _, ok := app.accounts[email]; ok {
			return &types.ResponseQuery{Code: 0, Log: "exists"}, nil
		}
		return &types.ResponseQuery{Code: 1, Log: "does not exist"}, nil
	case "/account/profile-by-email":
		email := string(req.Data)
		account, ok := app.accounts[email]
		if !ok {
			return &types.ResponseQuery{Code: 3, Log: "account not found"}, nil
		}
		res, err := json.Marshal(account)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal account"}, nil
		}
		return &types.ResponseQuery{Code: 0, Value: res}, nil
	case "/politisian/list":
		res, err := json.Marshal(app.politisian)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politisian list"}, nil
		}
		return &types.ResponseQuery{Code: 0, Value: res}, nil
	case "/proposals":
		res, err := json.Marshal(app.proposals)
		if err != nil {
			return &types.ResponseQuery{Code: 5, Log: "failed to marshal proposals"}, nil
		}
		return &types.ResponseQuery{Code: 0, Value: res}, nil
	default:
		return &types.ResponseQuery{Code: 2, Log: "unknown query path"}, nil
	}
}

func (app *PolitisianApp) FinalizeBlock(_ context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	respTxs := make([]*types.ExecTxResult, len(req.Txs))
	for i, tx := range req.Txs {
		var txData ptypes.TxData
		if err := json.Unmarshal(tx, &txData); err != nil {
			log.Printf("[ABCI] FinalizeBlock: 트랜잭션 #%d 처리 중 오류 (JSON 파싱 실패): %v", i, err)
			respTxs[i] = &types.ExecTxResult{Code: 1, Log: "failed to unmarshal tx"}
			continue
		}

		switch txData.Action {
		case "create_profile":
			respTxs[i] = app.handleCreateProfile(txData)
		case "claim_reward":
			respTxs[i] = app.handleClaimReward(txData)
		case "update_supporters":
			respTxs[i] = app.updateSupporters(&txData)
		case "propose_politisian":
			respTxs[i] = app.proposePolitisian(&txData)
		case "vote_on_proposal":
			respTxs[i] = app.handleVoteOnProposal(txData)
		default:
			log.Printf("[ABCI] FinalizeBlock: 알 수 없는 액션('%s') 입니다.", txData.Action)
			respTxs[i] = &types.ExecTxResult{Code: 10, Log: "unknown action"}
		}
	}
	app.appHash = []byte(strconv.Itoa(len(app.accounts)))
	app.lastBlockHeight = req.Height
	log.Printf("[ABCI] FinalizeBlock: 블록 높이 %d 처리 완료", req.Height)
	return &types.ResponseFinalizeBlock{
		TxResults: respTxs,
		AppHash:   app.appHash,
	}, nil
}

func (app *PolitisianApp) handleCreateProfile(txData ptypes.TxData) *types.ExecTxResult {
	if _, ok := app.accounts[txData.Email]; ok {
		return &types.ExecTxResult{Code: 2, Log: "email already exists"}
	}

	totalReward, res := app.processInitialReward(txData.Politisians)
	if res != nil {
		return res
	}

	newAccount := ptypes.Account{
		Email:       txData.Email,
		Nickname:    txData.Nickname,
		Wallet:      txData.Wallet,
		Country:     txData.Country,
		Gender:      txData.Gender,
		BirthYear:   txData.BirthYear,
		Politisians: txData.Politisians,
		Balance:     totalReward,
		Referrer:    txData.Referrer,
	}

	if txData.Referrer != "" {
		if referrerEmail, ok := app.wallets[txData.Referrer]; ok {
			if referrerAccount, ok := app.accounts[referrerEmail]; ok {
				referrerAccount.ReferralCredits++
				app.accounts[referrerEmail] = referrerAccount
			}
		}
	}

	app.accounts[txData.Email] = newAccount
	if txData.Wallet != "" {
		app.wallets[txData.Wallet] = txData.Email
	}

	log.Printf("새 계정 생성: %s (추천인: %s)", txData.Email, txData.Referrer)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PolitisianApp) processInitialReward(politisianNames []string) (int64, *types.ExecTxResult) {
	var totalReward int64 = 0
	for _, politisianName := range politisianNames {
		politisian, ok := app.politisian[politisianName]
		if !ok {
			return 0, &types.ExecTxResult{Code: 3, Log: "selected politisian does not exist"}
		}
		if politisian.TokensMinted+100 > politisian.MaxTokens {
			return 0, &types.ExecTxResult{Code: 4, Log: "token minting limit exceeded"}
		}
		politisian.TokensMinted += 100
		app.politisian[politisianName] = politisian
		totalReward += 100
	}
	return totalReward, nil
}

func (app *PolitisianApp) handleClaimReward(txData ptypes.TxData) *types.ExecTxResult {
	log.Printf("[ABCI] FinalizeBlock: 'claim_reward' 액션 수신. 이메일: %s", txData.Email)
	account, ok := app.accounts[txData.Email]
	if !ok {
		return &types.ExecTxResult{Code: 11, Log: "account not found"}
	}

	if account.ReferralCredits <= 0 {
		return &types.ExecTxResult{Code: 12, Log: "no referral credits to claim"}
	}

	account.ReferralCredits--
	account.Balance += 100
	app.accounts[txData.Email] = account

	log.Printf("[ABCI] FinalizeBlock: 계정(%s)에 보상 100 코인 지급 완료. 남은 크레딧: %d, 현재 잔액: %d", txData.Email, account.ReferralCredits, account.Balance)
	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PolitisianApp) proposePolitisian(txData *ptypes.TxData) *types.ExecTxResult {
	if txData.PolitisianName == "" || txData.Region == "" || txData.Party == "" {
		return &types.ExecTxResult{Code: 20, Log: "politisian name, region, and party are required"}
	}

	// 동일한 이름의 정치인이 이미 존재하는지 확인
	if _, exists := app.politisian[txData.PolitisianName]; exists {
		return &types.ExecTxResult{Code: 21, Log: "politisian with this name already exists"}
	}

	// 새로운 정치인 객체 생성
	newPolitisian := ptypes.Politisian{
		Name:         txData.PolitisianName,
		Region:       txData.Region,
		Party:        txData.Party,
		Supporters:   []string{txData.Email}, // 제안자를 지원자로 추가
		TokensMinted: 0,
		MaxTokens:    1000000, // 예시: 최대 발행량
	}

	// 상태에 새로운 정치인 추가
	app.politisian[newPolitisian.Name] = newPolitisian
	log.Printf("[ABCI] 새로운 정치인 '%s'가 사용자 '%s'에 의해 제안되었습니다.", newPolitisian.Name, txData.Email)

	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

func (app *PolitisianApp) handleVoteOnProposal(txData ptypes.TxData) *types.ExecTxResult {
	proposal, exists := app.proposals[txData.ProposalID]
	if !exists {
		return &types.ExecTxResult{Code: 30, Log: "proposal not found"}
	}

	// 중복 투표 방지
	if _, alreadyVoted := proposal.Votes[txData.Email]; alreadyVoted {
		return &types.ExecTxResult{Code: 31, Log: "user has already voted on this proposal"}
	}

	proposal.Votes[txData.Email] = txData.Vote
	if txData.Vote {
		proposal.YesVotes++
	} else {
		proposal.NoVotes++
	}

	log.Printf("투표 기록: 제안ID(%s), 투표자(%s), 찬성(%v)", txData.ProposalID, txData.Email, txData.Vote)

	// 찬성 50표 이상이면 공식 등록
	if proposal.YesVotes >= 50 {
		log.Printf("찬성 50표 달성! 정치인 '%s'을 공식 등록합니다.", proposal.Politisian.Name)
		
		newPolitisian := proposal.Politisian
		newPolitisian.MaxTokens = 10000000 // 1천만개 발행 한도 설정
		
		app.politisian[newPolitisian.Name] = newPolitisian
		delete(app.proposals, txData.ProposalID)
		
		log.Printf("정치인 '%s' 등록 완료.", newPolitisian.Name)
	} else {
		app.proposals[txData.ProposalID] = proposal
	}

	return &types.ExecTxResult{Code: types.CodeTypeOK}
}

// updateSupporters는 사용자가 지지하는 정치인 목록을 업데이트합니다.
func (app *PolitisianApp) updateSupporters(txData *ptypes.TxData) *types.ExecTxResult {
	account, exists := app.accounts[txData.UserID]
	if !exists {
		// 계정이 없으면 새로 생성할 수도 있습니다. 여기서는 에러 처리합니다.
		return &types.ExecTxResult{Code: 30, Log: "account not found"}
	}

	// 지지하는 정치인 목록 업데이트
	account.Politisian = txData.Politisian
	app.accounts[txData.UserID] = account

	log.Printf("[ABCI] 사용자 '%s'의 지지 정치인 목록이 업데이트되었습니다: %v", txData.UserID, txData.Politisian)
	return &types.ExecTxResult{Code: 0, Log: "supporters updated successfully"}
}


func (app *PolitisianApp) Commit(_ context.Context, _ *types.RequestCommit) (*types.ResponseCommit, error) {
	if err := app.saveState(); err != nil {
		log.Printf("[ABCI] Commit: 상태 저장 실패! %v", err)
		log.Printf("심각한 오류: 상태 저장 실패: %v", err)
	}
	return &types.ResponseCommit{RetainHeight: 0}, nil
}

func (app *PolitisianApp) InitChain(_ context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	log.Println("[ABCI] InitChain: 체인 초기화 시작")

	if len(req.AppStateBytes) > 0 {
		var initialState ptypes.GenesisState
		if err := json.Unmarshal(req.AppStateBytes, &initialState); err != nil {
			log.Printf("[ABCI] InitChain: app_state 파싱 실패: %v", err)
			return nil, err
		}

		if initialState.Politisian != nil {
			app.politisian = initialState.Politisian
			log.Printf("[ABCI] InitChain: 제네시스에서 %d명의 정치인 정보를 로드했습니다: %v", len(app.politisian), initialState.Politisian)
		}
	} else {
		log.Println("[ABCI] InitChain: AppStateBytes가 비어있어, 별도의 상태 초기화 없이 진행합니다.")
	}

	return &types.ResponseInitChain{
		Validators: req.Validators,
	}, nil
}

func (app *PolitisianApp) CheckTx(_ context.Context, _ *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

func (app *PolitisianApp) PrepareProposal(_ context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil
}

func (app *PolitisianApp) ProcessProposal(_ context.Context, _ *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}

func (app *PolitisianApp) ExtendVote(_ context.Context, _ *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return &types.ResponseExtendVote{}, nil
}

func (app *PolitisianApp) VerifyVoteExtension(_ context.Context, _ *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return &types.ResponseVerifyVoteExtension{Status: types.ResponseVerifyVoteExtension_ACCEPT}, nil
}

func (app *PolitisianApp) ListSnapshots(_ context.Context, _ *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return &types.ResponseListSnapshots{}, nil
}

func (app *PolitisianApp) OfferSnapshot(_ context.Context, _ *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return &types.ResponseOfferSnapshot{Result: types.ResponseOfferSnapshot_ABORT}, nil
}

func (app *PolitisianApp) LoadSnapshotChunk(_ context.Context, _ *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return &types.ResponseLoadSnapshotChunk{}, nil
}

func (app *PolitisianApp) ApplySnapshotChunk(_ context.Context, _ *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return &types.ResponseApplySnapshotChunk{Result: types.ResponseApplySnapshotChunk_ACCEPT}, nil
}
