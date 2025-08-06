package app

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"

	"github.com/cometbft/cometbft/abci/types"
)

// Account는 이제 사용자의 모든 프로필 정보를 저장합니다.
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

// TxData는 블록체인으로 전송될 트랜잭션의 데이터 구조입니다.
// server/http.go의 TxData와 일치해야 합니다.
type TxData struct {
	Email       string   `json:"email"`
	Nickname    string   `json:"nickname"`
	Wallet      string   `json:"wallet"`
	Country     string   `json:"country"`
	Gender      string   `json:"gender"`
	BirthYear   int      `json:"birthYear"`
	Politicians []string `json:"politicians"`
}

type PoliticianApp struct {
	mtx sync.Mutex
	// 이제 이메일 주소를 키로 사용하여 전체 Account 정보를 저장합니다.
	accounts map[string]Account
	appHash  []byte
}

func NewPoliticianApp() *PoliticianApp {
	return &PoliticianApp{
		accounts: make(map[string]Account),
		appHash:  []byte{},
	}
}

func (app *PoliticianApp) Info(_ context.Context, info *types.RequestInfo) (*types.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	return &types.ResponseInfo{
		LastBlockHeight:  0, // 임시
		LastBlockAppHash: app.appHash,
	}, nil
}

func (app *PoliticianApp) Query(_ context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	log.Printf("Query 수신: 경로=%s, 데이터=%s", req.Path, string(req.Data))

	// 서버에서 보낸 경로에 따라 분기 처리합니다.
	switch req.Path {
	case "/account/exists":
		email := string(req.Data)
		if _, ok := app.accounts[email]; ok {
			log.Printf("계정 '%s'가 존재합니다.", email)
			// 코드가 0이면 성공(존재함)을 의미합니다.
			return &types.ResponseQuery{Code: 0, Log: "exists"}, nil
		}
		log.Printf("계정 '%s'가 존재하지 않습니다.", email)
		return &types.ResponseQuery{Code: 1, Log: "does not exist"}, nil
	case "/account/profile":
		email := string(req.Data)
		if account, ok := app.accounts[email]; ok {
			log.Printf("프로필 조회 요청: '%s'", email)
			// 계정 정보를 JSON으로 변환합니다.
			res, err := json.Marshal(account)
			if err != nil {
				log.Printf("프로필 정보 JSON 변환 오류: %v", err)
				return &types.ResponseQuery{Code: 3, Log: "failed to marshal account data"}, nil
			}
			// 성공적으로 조회된 프로필 데이터를 반환합니다.
			return &types.ResponseQuery{Code: 0, Value: res}, nil
		}
		log.Printf("조회하려는 계정 '%s'가 존재하지 않습니다.", email)
		return &types.ResponseQuery{Code: 1, Log: "account not found"}, nil
	default:
		log.Printf("알 수 없는 쿼리 경로: %s", req.Path)
		return &types.ResponseQuery{Code: 2, Log: "unknown query path"}, nil
	}
}

func (app *PoliticianApp) FinalizeBlock(_ context.Context, req *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()
	log.Printf("FinalizeBlock: 수신된 트랜잭션 %d개", len(req.Txs))
	respTxs := make([]*types.ExecTxResult, len(req.Txs))

	for i, tx := range req.Txs {
		var txData TxData
		err := json.Unmarshal(tx, &txData)
		if err != nil {
			log.Printf("트랜잭션 Unmarshal 오류: %v", err)
			respTxs[i] = &types.ExecTxResult{Code: 1, Log: "failed to unmarshal tx"}
			continue
		}

		// 이메일이 이미 존재하는지 확인 (중복 등록 방지)
		if _, ok := app.accounts[txData.Email]; ok {
			log.Printf("이미 존재하는 이메일입니다: %s", txData.Email)
			respTxs[i] = &types.ExecTxResult{Code: 2, Log: "email already exists"}
			continue
		}

		// 새로운 계정 정보 생성
		newAccount := Account{
			Email:       txData.Email,
			Nickname:    txData.Nickname,
			Wallet:      txData.Wallet,
			Country:     txData.Country,
			Gender:      txData.Gender,
			BirthYear:   txData.BirthYear,
			Politicians: txData.Politicians,
			Balance:     1000, // 신규 사용자에게 초기 코인 지급
		}

		// 상태에 새로운 계정 저장
		app.accounts[txData.Email] = newAccount
		log.Printf("새 계정 생성 및 저장 완료: %s", txData.Email)

		respTxs[i] = &types.ExecTxResult{Code: types.CodeTypeOK}
	}

	// appHash를 간단하게 업데이트합니다. 실제로는 상태 전체를 해시해야 합니다.
	newAppHash := []byte(strconv.Itoa(len(app.accounts)))
	app.appHash = newAppHash

	return &types.ResponseFinalizeBlock{
		TxResults: respTxs,
		AppHash:   app.appHash,
	}, nil
}

// --- 나머지 ABCI 메소드들은 변경 없이 그대로 둡니다. ---

func (app *PoliticianApp) Commit(_ context.Context, _ *types.RequestCommit) (*types.ResponseCommit, error) {
	return &types.ResponseCommit{RetainHeight: 0}, nil
}

func (app *PoliticianApp) InitChain(context.Context, *types.RequestInitChain) (*types.ResponseInitChain, error) {
	return &types.ResponseInitChain{}, nil
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

// --- v0.38+ (ABCI++) 에서 상태 동기화를 위해 필요한 메소드들 ---
// 비어있는 상태로 구현해도 정상 작동합니다.

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