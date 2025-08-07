package app

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/cometbft/cometbft/abci/types"
	ptypes "politician/pkg/types"
)

func (app *PoliticianApp) Info(_ context.Context, info *types.RequestInfo) (*types.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	return &types.ResponseInfo{
		LastBlockHeight:  app.lastBlockHeight,
		LastBlockAppHash: app.appHash,
	}, nil
}

func (app *PoliticianApp) Query(_ context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
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
	case "/politicians/list":
		res, err := json.Marshal(app.politicians)
		if err != nil {
			return &types.ResponseQuery{Code: 4, Log: "failed to marshal politicians list"}, nil
		}
		return &types.ResponseQuery{Code: 0, Value: res}, nil
	default:
		return &types.ResponseQuery{Code: 2, Log: "unknown query path"}, nil
	}
}

func (app *PoliticianApp) FinalizeBlock(_ context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
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

func (app *PoliticianApp) handleCreateProfile(txData ptypes.TxData) *types.ExecTxResult {
	if _, ok := app.accounts[txData.Email]; ok {
		return &types.ExecTxResult{Code: 2, Log: "email already exists"}
	}

	totalReward, res := app.processInitialReward(txData.Politicians)
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
		Politicians: txData.Politicians,
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

func (app *PoliticianApp) processInitialReward(politicianNames []string) (int64, *types.ExecTxResult) {
	var totalReward int64 = 0
	for _, politicianName := range politicianNames {
		politician, ok := app.politicians[politicianName]
		if !ok {
			return 0, &types.ExecTxResult{Code: 3, Log: "selected politician does not exist"}
		}
		if politician.TokensMinted+100 > politician.MaxTokens {
			return 0, &types.ExecTxResult{Code: 4, Log: "token minting limit exceeded"}
		}
		politician.TokensMinted += 100
		app.politicians[politicianName] = politician
		totalReward += 100
	}
	return totalReward, nil
}

func (app *PoliticianApp) handleClaimReward(txData ptypes.TxData) *types.ExecTxResult {
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

func (app *PoliticianApp) Commit(_ context.Context, _ *types.RequestCommit) (*types.ResponseCommit, error) {
	if err := app.saveState(); err != nil {
		log.Printf("[ABCI] Commit: 상태 저장 실패! %v", err)
		log.Printf("심각한 오류: 상태 저장 실패: %v", err)
	}
	return &types.ResponseCommit{RetainHeight: 0}, nil
}

func (app *PoliticianApp) InitChain(_ context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	log.Println("[ABCI] InitChain: 체인 초기화 시작")

	if len(req.AppStateBytes) > 0 {
		var initialState struct {
			Politicians map[string]ptypes.Politician `json:"politicians"`
		}
		if err := json.Unmarshal(req.AppStateBytes, &initialState); err != nil {
			log.Printf("[ABCI] InitChain: app_state 파싱 실패: %v", err)
			return nil, err
		}

		if len(initialState.Politicians) > 0 {
			app.politicians = initialState.Politicians
			log.Printf("[ABCI] InitChain: 제네시스에서 %d명의 정치인 정보를 로드했습니다: %v", len(app.politicians), initialState.Politicians)
		}
	} else {
		log.Println("[ABCI] InitChain: AppStateBytes가 비어있어, 별도의 상태 초기화 없이 진행합니다.")
	}

	return &types.ResponseInitChain{
		Validators: req.Validators,
	}, nil
}

func (app *PoliticianApp) CheckTx(_ context.Context, _ *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	return &types.ResponseCheckTx{Code: types.CodeTypeOK}, nil
}

func (app *PoliticianApp) PrepareProposal(_ context.Context, req *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	return &types.ResponsePrepareProposal{Txs: req.Txs}, nil
}

func (app *PoliticianApp) ProcessProposal(_ context.Context, _ *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	return &types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}, nil
}

func (app *PoliticianApp) ExtendVote(_ context.Context, _ *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	return &types.ResponseExtendVote{}, nil
}

func (app *PoliticianApp) VerifyVoteExtension(_ context.Context, _ *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	return &types.ResponseVerifyVoteExtension{Status: types.ResponseVerifyVoteExtension_ACCEPT}, nil
}

func (app *PoliticianApp) ListSnapshots(_ context.Context, _ *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return &types.ResponseListSnapshots{}, nil
}

func (app *PoliticianApp) OfferSnapshot(_ context.Context, _ *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return &types.ResponseOfferSnapshot{Result: types.ResponseOfferSnapshot_ABORT}, nil
}

func (app *PoliticianApp) LoadSnapshotChunk(_ context.Context, _ *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return &types.ResponseLoadSnapshotChunk{}, nil
}

func (app *PoliticianApp) ApplySnapshotChunk(_ context.Context, _ *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return &types.ResponseApplySnapshotChunk{Result: types.ResponseApplySnapshotChunk_ACCEPT}, nil
}
